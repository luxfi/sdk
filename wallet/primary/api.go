// Copyright (C) 2019-2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package primary

import (
	"context"
	"fmt"

	gethcommon "github.com/luxfi/geth/common"
	"github.com/luxfi/geth/ethclient"

	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/node/api/info"
	"github.com/luxfi/node/codec"
	"github.com/luxfi/constants"
	"github.com/luxfi/node/utils/rpc"
	"github.com/luxfi/node/vms/components/lux"
	"github.com/luxfi/node/vms/platformvm"
	"github.com/luxfi/sdk/wallet/chain/c"
	"github.com/luxfi/sdk/wallet/chain/p"
	"github.com/luxfi/sdk/wallet/chain/x"

	ethcommon "github.com/luxfi/geth/common"
	pbuilder "github.com/luxfi/sdk/wallet/chain/p/builder"
	ptxs "github.com/luxfi/node/vms/platformvm/txs"
	xbuilder "github.com/luxfi/sdk/wallet/chain/x/builder"
	walletcommon "github.com/luxfi/sdk/wallet/primary/common"
)

const (
	MainnetAPIURI = "https://api.lux.network"
	TestnetAPIURI = "https://api.lux-test.network"
	LocalAPIURI   = "http://localhost:9630"

	fetchLimit = 1024
)

// perform their own assertions.
var (
	_ UTXOClient = (*platformvm.Client)(nil)
	_ UTXOClient = (*XClient)(nil)
)

type UTXOClient interface {
	GetAtomicUTXOs(
		ctx context.Context,
		addrs []ids.ShortID,
		sourceChain string,
		limit uint32,
		startAddress ids.ShortID,
		startUTXOID ids.ID,
		options ...rpc.Option,
	) ([][]byte, ids.ShortID, ids.ID, error)
}

// XClient is a client for interacting with the X-Chain
type XClient struct {
	requester rpc.EndpointRequester
}

// NewXClient returns a new X-Chain client
func NewXClient(uri, chainAlias string) *XClient {
	return &XClient{
		requester: rpc.NewEndpointRequester(
			fmt.Sprintf("%s/ext/bc/%s", uri, chainAlias),
		),
	}
}

// GetAtomicUTXOs implements UTXOClient.
// For local/dev networks where X-chain atomic operations aren't needed,
// this returns empty to allow P-chain operations to proceed.
func (c *XClient) GetAtomicUTXOs(
	ctx context.Context,
	addrs []ids.ShortID,
	sourceChain string,
	limit uint32,
	startAddress ids.ShortID,
	startUTXOID ids.ID,
	options ...rpc.Option,
) ([][]byte, ids.ShortID, ids.ID, error) {
	// Return empty for X-chain since atomic UTXOs are rarely needed
	// for subnet deployments. This enables local network compatibility.
	return nil, ids.ShortEmpty, ids.Empty, nil
}

type LUXState struct {
	PClient *platformvm.Client
	PCTX    *pbuilder.Context
	XClient *XClient
	XCTX    *xbuilder.Context
	// CClient *ethclient.Client // Not yet implemented
	// CCTX    *c.Context         // Not yet implemented
	UTXOs walletcommon.UTXOs
}

func FetchState(
	ctx context.Context,
	uri string,
	addrs set.Set[ids.ShortID],
) (
	*LUXState,
	error,
) {
	infoClient := info.NewClient(uri)
	pClient := platformvm.NewClient(uri)
	xClient := NewXClient(uri, "X")
	// C-chain client disabled for now
	// cClient, err := ethclient.Dial(uri + "/ext/bc/C/rpc")
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to connect to C-chain: %w", err)
	// }

	pCTX, err := p.NewContextFromClients(ctx, infoClient, pClient)
	if err != nil {
		return nil, err
	}

	// For X-chain, we need to get the LUX asset ID and fees
	// TODO: Get these from the xClient or infoClient
	luxAssetID := pCTX.XAssetID
	baseTxFee := uint64(1000000)        // 0.001 LUX
	createAssetTxFee := uint64(10000000) // 0.01 LUX

	xCTX, err := x.NewContextFromClients(ctx, infoClient, luxAssetID, baseTxFee, createAssetTxFee)
	if err != nil {
		return nil, err
	}

	// C-chain context disabled for now
	// cCTX, err := c.NewContextFromClients(ctx, infoClient, pClient)
	// if err != nil {
	// 	return nil, err
	// }

	utxos := walletcommon.NewUTXOs()
	addrList := addrs.List()
	chains := []struct {
		id     ids.ID
		client UTXOClient
		codec  codec.Manager
	}{
		{
			id:     constants.PlatformChainID,
			client: pClient,
			codec:  ptxs.Codec,
		},
		{
			id:     xCTX.BlockchainID,
			client: xClient,
			codec:  codec.NewDefaultManager(),
		},
		// {
		// 	id:     cCTX.BlockchainID,
		// 	client: cClient,
		// 	codec:  evm.Codec,
		// },
	}
	for _, destinationChain := range chains {
		for _, sourceChain := range chains {
			err = AddAllUTXOs(
				ctx,
				utxos,
				destinationChain.client,
				destinationChain.codec,
				sourceChain.id,
				destinationChain.id,
				addrList,
			)
			if err != nil {
				return nil, err
			}
		}
	}
	return &LUXState{
		PClient: pClient,
		PCTX:    pCTX,
		XClient: xClient,
		XCTX:    xCTX,
		// CClient: cClient, // Disabled
		// CCTX:    cCTX,    // Disabled
		UTXOs: utxos,
	}, nil
}

type EthState struct {
	Client   *ethclient.Client
	Accounts map[ethcommon.Address]*c.Account
}

func FetchEthState(
	ctx context.Context,
	uri string,
	addrs set.Set[ethcommon.Address],
) (*EthState, error) {
	path := fmt.Sprintf(
		"%s/ext/%s/C/rpc",
		uri,
		constants.ChainAliasPrefix,
	)
	client, err := ethclient.Dial(path)
	if err != nil {
		return nil, err
	}

	accounts := make(map[ethcommon.Address]*c.Account, addrs.Len())
	for addr := range addrs {
		// Convert ethereum address to geth address
		gethAddr := gethcommon.Address(addr)
		balance, err := client.BalanceAt(ctx, gethAddr, nil)
		if err != nil {
			return nil, err
		}
		nonce, err := client.NonceAt(ctx, gethAddr, nil)
		if err != nil {
			return nil, err
		}
		accounts[addr] = &c.Account{
			Balance: balance,
			Nonce:   nonce,
		}
	}
	return &EthState{
		Client:   client,
		Accounts: accounts,
	}, nil
}

// AddAllUTXOs fetches all the UTXOs referenced by [addresses] that were sent
// from [sourceChainID] to [destinationChainID] from the [client]. It then uses
// [codec] to parse the returned UTXOs and it adds them into [utxos]. If [ctx]
// expires, then the returned error will be immediately reported.
func AddAllUTXOs(
	ctx context.Context,
	utxos walletcommon.UTXOs,
	client UTXOClient,
	codec codec.Manager,
	sourceChainID ids.ID,
	destinationChainID ids.ID,
	addrs []ids.ShortID,
) error {
	var (
		sourceChainIDStr = sourceChainID.String()
		startAddr        ids.ShortID
		startUTXO        ids.ID
	)
	for {
		utxosBytes, endAddr, endUTXO, err := client.GetAtomicUTXOs(
			ctx,
			addrs,
			sourceChainIDStr,
			fetchLimit,
			startAddr,
			startUTXO,
		)
		if err != nil {
			return err
		}

		for _, utxoBytes := range utxosBytes {
			var utxo lux.UTXO
			_, err := codec.Unmarshal(utxoBytes, &utxo)
			if err != nil {
				return err
			}

			if err := utxos.AddUTXO(ctx, sourceChainID, destinationChainID, &utxo); err != nil {
				return err
			}
		}

		if len(utxosBytes) < fetchLimit {
			break
		}

		// Update the vars to query the next page of UTXOs.
		startAddr = endAddr
		startUTXO = endUTXO
	}
	return nil
}
