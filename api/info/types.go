// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package info

import (
	"net/netip"

	"github.com/luxfi/codec/jsonrpc"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/p2p/peer"
)

// ProofOfPossession is a JSON-friendly representation of a BLS PoP.
type ProofOfPossession struct {
	PublicKey         string `json:"publicKey"`
	ProofOfPossession string `json:"proofOfPossession"`
}

// GetNodeVersionReply are the results from calling GetNodeVersion.
type GetNodeVersionReply struct {
	Version            string            `json:"version"`
	DatabaseVersion    string            `json:"databaseVersion"`
	RPCProtocolVersion json.Uint32       `json:"rpcProtocolVersion"`
	GitCommit          string            `json:"gitCommit"`
	VMVersions         map[string]string `json:"vmVersions"`
}

// GetNodeIDReply are the results from calling GetNodeID.
type GetNodeIDReply struct {
	NodeID  ids.NodeID         `json:"nodeID"`
	NodePOP *ProofOfPossession `json:"nodePOP"`
}

// GetNetworkIDReply are the results from calling GetNetworkID.
type GetNetworkIDReply struct {
	NetworkID json.Uint32 `json:"networkID"`
}

// GetNodeIPReply are the results from calling GetNodeIP.
type GetNodeIPReply struct {
	IP netip.AddrPort `json:"ip"`
}

// GetNetworkNameReply is the result from calling GetNetworkName.
type GetNetworkNameReply struct {
	NetworkName string `json:"networkName"`
}

// GetBlockchainIDArgs are the arguments for calling GetBlockchainID.
type GetBlockchainIDArgs struct {
	Alias string `json:"alias"`
}

// GetBlockchainIDReply are the results from calling GetBlockchainID.
type GetBlockchainIDReply struct {
	BlockchainID ids.ID `json:"blockchainID"`
}

// PeersArgs are the arguments for calling Peers.
type PeersArgs struct {
	NodeIDs []ids.NodeID `json:"nodeIDs"`
}

type Peer struct {
	peer.Info

	Benched []string `json:"benched"`
}

// PeersReply are the results from calling Peers.
type PeersReply struct {
	// Number of elements in [Peers]
	NumPeers json.Uint64 `json:"numPeers"`
	// Each element is a peer
	Peers []Peer `json:"peers"`
}

// IsBootstrappedArgs are the arguments for calling IsBootstrapped.
type IsBootstrappedArgs struct {
	// Alias of the chain
	// Can also be the string representation of the chain's ID
	Chain string `json:"chain"`
}

// IsBootstrappedResponse are the results from calling IsBootstrapped.
type IsBootstrappedResponse struct {
	// True iff the chain exists and is done bootstrapping
	IsBootstrapped bool `json:"isBootstrapped"`
}

// UptimeResponse are the results from calling Uptime.
type UptimeResponse struct {
	// RewardingStakePercentage shows what percent of network stake thinks we're
	// above the uptime requirement.
	RewardingStakePercentage json.Float64 `json:"rewardingStakePercentage"`

	// WeightedAveragePercentage is the average perceived uptime of this node,
	// weighted by stake.
	// Note that this is different from RewardingStakePercentage, which shows
	// the percent of the network stake that thinks this node is above the
	// uptime requirement. WeightedAveragePercentage is weighted by uptime.
	// i.e If uptime requirement is 85 and a peer reports 40 percent it will be
	// counted (40*weight) in WeightedAveragePercentage but not in
	// RewardingStakePercentage since 40 < 85
	WeightedAveragePercentage json.Float64 `json:"weightedAveragePercentage"`
}

type LP struct {
	SupportWeight json.Uint64         `json:"supportWeight"`
	Supporters    set.Set[ids.NodeID] `json:"supporters"`
	ObjectWeight  json.Uint64         `json:"objectWeight"`
	Objectors     set.Set[ids.NodeID] `json:"objectors"`
	AbstainWeight json.Uint64         `json:"abstainWeight"`
}

type LPsReply struct {
	LPs map[uint32]*LP `json:"lps"`
}

func (a *LPsReply) GetLP(lpNum uint32) *LP {
	lp, ok := a.LPs[lpNum]
	if !ok {
		lp = &LP{}
		a.LPs[lpNum] = lp
	}
	return lp
}

type GetTxFeeResponse struct {
	TxFee                         json.Uint64 `json:"txFee"`
	CreateAssetTxFee              json.Uint64 `json:"createAssetTxFee"`
	CreateNetTxFee                json.Uint64 `json:"createNetTxFee"`
	TransformChainTxFee           json.Uint64 `json:"transformNetTxFee"`
	CreateBlockchainTxFee         json.Uint64 `json:"createBlockchainTxFee"`
	AddPrimaryNetworkValidatorFee json.Uint64 `json:"addPrimaryNetworkValidatorFee"`
	AddPrimaryNetworkDelegatorFee json.Uint64 `json:"addPrimaryNetworkDelegatorFee"`
	AddNetValidatorFee            json.Uint64 `json:"addNetValidatorFee"`
	AddNetDelegatorFee            json.Uint64 `json:"addNetDelegatorFee"`
}

// GetVMsReply contains the response metadata for GetVMs.
type GetVMsReply struct {
	VMs map[ids.ID][]string `json:"vms"`
	Fxs map[ids.ID]string   `json:"fxs"`
}
