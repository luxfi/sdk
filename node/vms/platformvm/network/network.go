// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package network

import (
	"context"
	"sync"
	"time"

	"github.com/luxfi/log"
	"github.com/luxfi/metric"

	"github.com/luxfi/ids"
	"github.com/luxfi/container"
	"github.com/luxfi/p2p"
	"github.com/luxfi/p2p/gossip"
	validators "github.com/luxfi/consensus/validator"
	"github.com/luxfi/sdk/node/vms/platformvm/config"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/platformvm/warp"
	"github.com/luxfi/sdk/node/vms/txs/mempool"
	extwarp "github.com/luxfi/warp"
)

type Network struct {
	*p2p.Network

	log                       log.Logger
	mempool                   *gossipMempool
	partialSyncPrimaryNetwork bool

	txPushGossiper        *gossip.PushGossiper[*txs.Tx]
	txPushGossipFrequency time.Duration
	txPullGossiper        gossip.Gossiper
	txPullGossipFrequency time.Duration
}

// warpSignerAdapter adapts warp.Signer (node's internal) to extwarp.Signer (external warp)
type warpSignerAdapter struct {
	signer warp.Signer
}

// Sign implements extwarp.Signer interface
func (a *warpSignerAdapter) Sign(msg *extwarp.UnsignedMessage) ([]byte, error) {
	// Convert external warp message to internal warp message
	// msg.SourceChainID is already ids.ID type
	internalMsg, err := warp.NewUnsignedMessage(msg.NetworkID, msg.SourceChainID, msg.Payload)
	if err != nil {
		return nil, err
	}
	// Sign using internal signer and return raw signature bytes
	sig, err := a.signer.Sign(internalMsg)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

func New(
	log log.Logger,
	nodeID ids.NodeID,
	netID ids.ID,
	vdrs validators.State,
	txVerifier TxVerifier,
	mempool mempool.Mempool[*txs.Tx],
	partialSyncPrimaryNetwork bool,
	appSender extwarp.Sender,
	stateLock sync.Locker,
	state state.Chain,
	signer warp.Signer,
	registerer metric.Registerer,
	config config.Network,
) (*Network, error) {
	p2pNetwork, err := p2p.NewNetwork(log, appSender, registerer, "p2p")
	if err != nil {
		return nil, err
	}

	marshaller := txMarshaller{}
	validators := p2p.NewValidators(
		p2pNetwork.Peers,
		log,
		netID,
		vdrs,
		config.MaxValidatorSetStaleness,
	)
	txGossipClient := p2pNetwork.NewClient(
		p2p.TxGossipHandlerID,
		p2p.WithValidatorSampling(validators),
	)
	txGossipMetrics, err := gossip.NewMetrics(registerer, "tx")
	if err != nil {
		return nil, err
	}

	gossipMempool, err := newGossipMempool(
		mempool,
		registerer,
		log,
		txVerifier,
		config.ExpectedBloomFilterElements,
		config.ExpectedBloomFilterFalsePositiveProbability,
		config.MaxBloomFilterFalsePositiveProbability,
	)
	if err != nil {
		return nil, err
	}

	txPushGossiper, err := gossip.NewPushGossiper[*txs.Tx](
		marshaller,
		gossipMempool,
		validators,
		txGossipClient,
		txGossipMetrics,
		gossip.BranchingFactor{
			StakePercentage: config.PushGossipPercentStake,
			Validators:      config.PushGossipNumValidators,
			Peers:           config.PushGossipNumPeers,
		},
		gossip.BranchingFactor{
			Validators: config.PushRegossipNumValidators,
			Peers:      config.PushRegossipNumPeers,
		},
		config.PushGossipDiscardedCacheSize,
		config.TargetGossipSize,
		config.PushGossipMaxRegossipFrequency,
	)
	if err != nil {
		return nil, err
	}

	var txPullGossiper gossip.Gossiper = gossip.NewPullGossiper[*txs.Tx](
		log,
		marshaller,
		gossipMempool,
		txGossipClient,
		txGossipMetrics,
		config.PullGossipPollSize,
	)

	// Gossip requests are only served if a node is a validator
	txPullGossiper = gossip.ValidatorGossiper{
		Gossiper:   txPullGossiper,
		NodeID:     nodeID,
		Validators: validators,
	}

	handler := gossip.NewHandler[*txs.Tx](
		log,
		marshaller,
		gossipMempool,
		txGossipMetrics,
		config.TargetGossipSize,
		nil, // BloomChecker - optional
	)

	validatorHandler := p2p.NewValidatorHandler(
		p2p.NewThrottlerHandler(
			handler,
			p2p.NewSlidingWindowThrottler(
				config.PullGossipThrottlingPeriod,
				config.PullGossipThrottlingLimit,
			),
			log,
		),
		validators,
		log,
	)

	// We allow pushing txs between all peers, but only serve gossip requests
	// from validators
	txGossipHandler := txGossipHandler{
		appGossipHandler:  handler,
		appRequestHandler: validatorHandler,
	}

	if err := p2pNetwork.AddHandler(p2p.TxGossipHandlerID, txGossipHandler); err != nil {
		return nil, err
	}

	// We allow all peers to request warp messaging signatures
	verifier := signatureRequestVerifier{
		stateLock: stateLock,
		state:     state,
	}
	// Create a cache for signature requests (100 entries)
	signatureCache := &cache.LRU[ids.ID, []byte]{Size: 100}
	// Wrap signer to adapt node's warp.Signer to extwarp.Signer
	signerAdapter := &warpSignerAdapter{signer: signer}
	cachedHandler := extwarp.NewCachedSignatureHandler(signatureCache, verifier, signerAdapter)
	signatureHandler := extwarp.NewSignatureHandlerAdapter(cachedHandler)

	if err := p2pNetwork.AddHandler(extwarp.SignatureHandlerID, signatureHandler); err != nil {
		return nil, err
	}

	return &Network{
		Network:                   p2pNetwork,
		log:                       log,
		mempool:                   gossipMempool,
		partialSyncPrimaryNetwork: partialSyncPrimaryNetwork,
		txPushGossiper:            txPushGossiper,
		txPushGossipFrequency:     config.PushGossipFrequency,
		txPullGossiper:            txPullGossiper,
		txPullGossipFrequency:     config.PullGossipFrequency,
	}, nil
}

func (n *Network) PushGossip(ctx context.Context) {
	gossip.Every(ctx, n.log, n.txPushGossiper, n.txPushGossipFrequency)
}

func (n *Network) PullGossip(ctx context.Context) {
	// If the node is running partial sync, we do not perform any pull gossip
	// because we should never be a validator.
	if n.partialSyncPrimaryNetwork {
		return
	}

	gossip.Every(ctx, n.log, n.txPullGossiper, n.txPullGossipFrequency)
}

func (n *Network) AppGossip(ctx context.Context, nodeID ids.NodeID, msgBytes []byte) error {
	if n.partialSyncPrimaryNetwork {
		n.log.Debug("dropping AppGossip message",
			log.String("reason", "primary network is not being fully synced"),
		)
		return nil
	}

	return n.Network.Gossip(ctx, nodeID, msgBytes)
}

func (n *Network) IssueTxFromRPC(tx *txs.Tx) error {
	if err := n.mempool.Add(tx); err != nil {
		return err
	}
	n.txPushGossiper.Add(tx)
	return nil
}
