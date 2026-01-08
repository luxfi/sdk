// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package info

import (
	"errors"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/gorilla/rpc/v2"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/chains"
	"github.com/luxfi/sdk/node/network"
	// "github.com/luxfi/consensus/networking/benchlist" // Unused
	validators "github.com/luxfi/consensus/validator"
	"github.com/luxfi/const"
	"github.com/luxfi/log"
	"github.com/luxfi/sdk/node/vms"
	"github.com/luxfi/sdk/node/vms/nftfx"
	"github.com/luxfi/sdk/node/vms/platformvm/signer"
	"github.com/luxfi/sdk/node/vms/propertyfx"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/sdk/utils/formatting"
	"github.com/luxfi/sdk/utils/json"
	"github.com/luxfi/sdk/utils/units"
	"github.com/luxfi/upgrade"
	"github.com/luxfi/version"
)

var (
	errNoChainProvided = errors.New("argument 'chain' not given")

	mainnetGetTxFeeResponse = GetTxFeeResponse{
		CreateNetTxFee:                json.Uint64(1 * units.Lux),
		TransformChainTxFee:           json.Uint64(10 * units.Lux),
		CreateBlockchainTxFee:         json.Uint64(1 * units.Lux),
		AddPrimaryNetworkValidatorFee: json.Uint64(0),
		AddPrimaryNetworkDelegatorFee: json.Uint64(0),
		AddNetValidatorFee:            json.Uint64(units.MilliLux),
		AddNetDelegatorFee:            json.Uint64(units.MilliLux),
	}
	fujiGetTxFeeResponse = GetTxFeeResponse{
		CreateNetTxFee:                json.Uint64(100 * units.MilliLux),
		TransformChainTxFee:           json.Uint64(1 * units.Lux),
		CreateBlockchainTxFee:         json.Uint64(100 * units.MilliLux),
		AddPrimaryNetworkValidatorFee: json.Uint64(0),
		AddPrimaryNetworkDelegatorFee: json.Uint64(0),
		AddNetValidatorFee:            json.Uint64(units.MilliLux),
		AddNetDelegatorFee:            json.Uint64(units.MilliLux),
	}
	defaultGetTxFeeResponse = GetTxFeeResponse{
		CreateNetTxFee:                json.Uint64(100 * units.MilliLux),
		TransformChainTxFee:           json.Uint64(100 * units.MilliLux),
		CreateBlockchainTxFee:         json.Uint64(100 * units.MilliLux),
		AddPrimaryNetworkValidatorFee: json.Uint64(0),
		AddPrimaryNetworkDelegatorFee: json.Uint64(0),
		AddNetValidatorFee:            json.Uint64(units.MilliLux),
		AddNetDelegatorFee:            json.Uint64(units.MilliLux),
	}
)

// Info is the API service for unprivileged info on a node
type Info struct {
	Parameters
	log          log.Logger
	validators   validators.Manager
	myIP         *utils.Atomic[netip.AddrPort]
	networking   network.Network
	chainManager chains.Manager
	vmManager    vms.Manager
	// benchlist    benchlist.Manager // benchlist package doesn't exist
}

type Parameters struct {
	Version   *version.Application
	NodeID    ids.NodeID
	NodePOP   *signer.ProofOfPossession
	NetworkID uint32
	VMManager vms.Manager
	Upgrades  upgrade.Config

	TxFee            uint64
	CreateAssetTxFee uint64
}

func NewService(
	parameters Parameters,
	log log.Logger,
	validators validators.Manager,
	chainManager chains.Manager,
	vmManager vms.Manager,
	myIP *utils.Atomic[netip.AddrPort],
	network network.Network,
	// benchlist benchlist.Manager, // benchlist package doesn't exist
) (http.Handler, error) {
	server := rpc.NewServer()
	codec := json.NewCodec()
	server.RegisterCodec(codec, "application/json")
	server.RegisterCodec(codec, "application/json;charset=UTF-8")
	return server, server.RegisterService(
		&Info{
			Parameters:   parameters,
			log:          log,
			validators:   validators,
			chainManager: chainManager,
			vmManager:    vmManager,
			myIP:         myIP,
			networking:   network,
			// benchlist:    benchlist, // benchlist removed
		},
		"info",
	)
}

// GetNodeVersion returns the version this node is running
func (i *Info) GetNodeVersion(_ *http.Request, _ *struct{}, reply *GetNodeVersionReply) error {
	i.log.Debug("API called",
		log.String("service", "info"),
		log.String("method", "getNodeVersion"),
	)

	vmVersions, err := i.vmManager.Versions()
	if err != nil {
		return err
	}

	reply.Version = i.Version.String()
	reply.DatabaseVersion = version.CurrentDatabase.String()
	reply.RPCProtocolVersion = json.Uint32(version.RPCChainVMProtocol)
	reply.GitCommit = version.GitCommit
	reply.VMVersions = vmVersions
	return nil
}

// GetNodeID returns the node ID of this node
func (i *Info) GetNodeID(_ *http.Request, _ *struct{}, reply *GetNodeIDReply) error {
	i.log.Debug("API called",
		log.String("service", "info"),
		log.String("method", "getNodeID"),
	)

	reply.NodeID = i.NodeID
	if i.NodePOP != nil {
		publicKey, err := formatting.Encode(formatting.HexNC, i.NodePOP.PublicKey[:])
		if err != nil {
			return err
		}
		pop, err := formatting.Encode(formatting.HexNC, i.NodePOP.ProofOfPossession[:])
		if err != nil {
			return err
		}
		reply.NodePOP = &ProofOfPossession{
			PublicKey:         publicKey,
			ProofOfPossession: pop,
		}
	}
	return nil
}

// GetNodeIP returns the IP of this node
func (i *Info) GetNodeIP(_ *http.Request, _ *struct{}, reply *GetNodeIPReply) error {
	i.log.Debug("API called",
		log.String("service", "info"),
		log.String("method", "getNodeIP"),
	)

	reply.IP = i.myIP.Get()
	return nil
}

// GetNetworkID returns the network ID this node is running on
func (i *Info) GetNetworkID(_ *http.Request, _ *struct{}, reply *GetNetworkIDReply) error {
	i.log.Debug("API called",
		log.String("service", "info"),
		log.String("method", "getNetworkID"),
	)

	reply.NetworkID = json.Uint32(i.NetworkID)
	return nil
}

// GetNetworkName returns the network name this node is running on
func (i *Info) GetNetworkName(_ *http.Request, _ *struct{}, reply *GetNetworkNameReply) error {
	i.log.Debug("API called",
		log.String("service", "info"),
		log.String("method", "getNetworkName"),
	)

	reply.NetworkName = constants.NetworkName(i.NetworkID)
	return nil
}

// GetBlockchainID returns the blockchain ID that resolves the alias that was supplied
func (i *Info) GetBlockchainID(_ *http.Request, args *GetBlockchainIDArgs, reply *GetBlockchainIDReply) error {
	i.log.Debug("API called",
		log.String("service", "info"),
		log.String("method", "getBlockchainID"),
	)

	bID, err := i.chainManager.Lookup(args.Alias)
	reply.BlockchainID = bID
	return err
}

// Peers returns the list of current validators
func (i *Info) Peers(_ *http.Request, args *PeersArgs, reply *PeersReply) error {
	i.log.Debug("API called",
		log.String("service", "info"),
		log.String("method", "peers"),
	)

	peers := i.networking.PeerInfo(args.NodeIDs)
	peerInfo := make([]Peer, len(peers))
	for index, peer := range peers {
		// benchlist removed
		// benchedIDs := i.benchlist.GetBenched(peer.ID)
		benchedAliases := make([]string, 0) // Empty list since benchlist is removed
		// for idx, id := range benchedIDs {
		// 	alias, err := i.chainManager.PrimaryAlias(id)
		// 	if err != nil {
		// 		return fmt.Errorf("failed to get primary alias for chain ID %s: %w", id, err)
		// 	}
		// 	benchedAliases[idx] = alias
		// }
		peerInfo[index] = Peer{
			Info:    peer,
			Benched: benchedAliases,
		}
	}

	reply.Peers = peerInfo
	reply.NumPeers = json.Uint64(len(reply.Peers))
	return nil
}

// IsBootstrapped returns nil and sets [reply.IsBootstrapped] == true iff [args.Chain] exists and is done bootstrapping
// Returns an error if the chain doesn't exist
func (i *Info) IsBootstrapped(_ *http.Request, args *IsBootstrappedArgs, reply *IsBootstrappedResponse) error {
	i.log.Debug("API called",
		log.String("service", "info"),
		log.String("method", "isBootstrapped"),
		log.String("chain", args.Chain),
	)

	if args.Chain == "" {
		return errNoChainProvided
	}
	chainID, err := i.chainManager.Lookup(args.Chain)
	if err != nil {
		return fmt.Errorf("there is no chain with alias/ID '%s'", args.Chain)
	}
	reply.IsBootstrapped = i.chainManager.IsBootstrapped(chainID)
	return nil
}

// Upgrades returns the upgrade schedule this node is running.
func (i *Info) Upgrades(_ *http.Request, _ *struct{}, reply *upgrade.Config) error {
	i.log.Debug("API called",
		log.String("service", "info"),
		log.String("method", "upgrades"),
	)

	*reply = i.Parameters.Upgrades
	return nil
}

func (i *Info) Uptime(_ *http.Request, _ *struct{}, reply *UptimeResponse) error {
	i.log.Debug("API called",
		log.String("service", "info"),
		log.String("method", "uptime"),
	)

	result, err := i.networking.NodeUptime()
	if err != nil {
		return fmt.Errorf("couldn't get node uptime: %w", err)
	}
	reply.WeightedAveragePercentage = json.Float64(result.WeightedAveragePercentage)
	reply.RewardingStakePercentage = json.Float64(result.RewardingStakePercentage)
	return nil
}

func (i *Info) Lps(_ *http.Request, _ *struct{}, reply *LPsReply) error {
	i.log.Debug("API called",
		log.String("service", "info"),
		log.String("method", "lps"),
	)

	reply.LPs = make(map[uint32]*LP, constants.CurrentLPs.Len())
	peers := i.networking.PeerInfo(nil)
	for _, peer := range peers {
		w := i.validators.GetWeight(constants.PrimaryNetworkID, peer.ID)
		weight := json.Uint64(w)
		if weight == 0 {
			continue
		}

		// SupportedLPs and ObjectedLPs not available on peer.Info type
		// for lpNum := range peer.SupportedLPs {
		// 	lp := reply.getLP(lpNum)
		// 	lp.Supporters.Add(peer.ID)
		// 	lp.SupportWeight += weight
		// }
		// for lpNum := range peer.ObjectedLPs {
		// 	lp := reply.getLP(lpNum)
		// 	lp.Objectors.Add(peer.ID)
		// 	lp.ObjectWeight += weight
		// }
		_ = peer // Silence unused variable warning
	}

	totalWeight, err := i.validators.TotalWeight(constants.PrimaryNetworkID)
	if err != nil {
		return err
	}
	for lpNum := range constants.CurrentLPs {
		lp := reply.GetLP(lpNum)
		lp.AbstainWeight = json.Uint64(totalWeight) - lp.SupportWeight - lp.ObjectWeight
	}
	return nil
}

// GetTxFee returns the transaction fee in nLUX.
func (i *Info) GetTxFee(_ *http.Request, _ *struct{}, reply *GetTxFeeResponse) error {
	i.log.Warn("deprecated API called",
		log.String("service", "info"),
		log.String("method", "getTxFee"),
	)

	switch i.NetworkID {
	case constants.MainnetID:
		*reply = mainnetGetTxFeeResponse
	// case constants.FujiID: // FujiID not available in constants package
	// 	*reply = fujiGetTxFeeResponse
	default:
		*reply = defaultGetTxFeeResponse
	}
	reply.TxFee = json.Uint64(i.TxFee)
	reply.CreateAssetTxFee = json.Uint64(i.CreateAssetTxFee)
	return nil
}

// GetVMs lists the virtual machines installed on the node
func (i *Info) GetVMs(_ *http.Request, _ *struct{}, reply *GetVMsReply) error {
	i.log.Debug("API called",
		log.String("service", "info"),
		log.String("method", "getVMs"),
	)

	// Fetch the VMs registered on this node.
	vmIDs, err := i.VMManager.ListFactories()
	if err != nil {
		return err
	}

	reply.VMs, err = ids.GetRelevantAliases(i.VMManager, vmIDs)
	reply.Fxs = map[ids.ID]string{
		secp256k1fx.ID: secp256k1fx.Name,
		nftfx.ID:       nftfx.Name,
		propertyfx.ID:  propertyfx.Name,
	}
	return err
}
