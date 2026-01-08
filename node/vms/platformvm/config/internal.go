// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"time"

	"github.com/luxfi/sdk/node/chains"
	"github.com/luxfi/ids"
	"github.com/luxfi/consensus/validator/uptime"
	validators "github.com/luxfi/consensus/validator"
	"github.com/luxfi/upgrade"
	"github.com/luxfi/const"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/vms/components/gas"
	"github.com/luxfi/sdk/node/vms/platformvm/reward"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/platformvm/validators/fee"
)

// Internal contains all of the parameters for the PlatformVM that are
// internally set by the node.
type Internal struct {
	// The node's chain manager
	Chains chains.Manager

	// Node's validator set maps chainID -> validators of the subnet
	//
	// Invariant: The primary network's validator set should have been added to
	//            the manager before calling VM.Initialize.
	// Invariant: The primary network's validator set should be empty before
	//            calling VM.Initialize.
	Validators validators.Manager

	// Dynamic fees are active after Etna
	DynamicFeeConfig gas.Config

	// LP-77 validator fees are active after Etna
	ValidatorFeeConfig fee.Config

	// Provides access to the uptime manager as a thread safe data structure
	UptimeLockedCalculator uptime.LockedCalculator

	// True if the node is being run with staking enabled
	SybilProtectionEnabled bool

	// If true, only the P-chain will be instantiated on the primary network.
	PartialSyncPrimaryNetwork bool

	// Set of chains that this node is validating
	TrackedChains set.Set[ids.ID]

	// If true, track all chains automatically (useful for dev/test networks)
	TrackAllChains bool

	// The minimum amount of tokens one must bond to be a validator
	MinValidatorStake uint64

	// The maximum amount of tokens that can be bonded on a validator
	MaxValidatorStake uint64

	// Minimum stake, in nLUX, that can be delegated on the primary network
	MinDelegatorStake uint64

	// Minimum fee that can be charged for delegation
	MinDelegationFee uint32

	// UptimePercentage is the minimum uptime required to be rewarded for staking
	UptimePercentage float64

	// Minimum amount of time to allow a staker to stake
	MinStakeDuration time.Duration

	// Maximum amount of time to allow a staker to stake
	MaxStakeDuration time.Duration

	// Config for the minting function
	RewardConfig reward.Config

	// All network upgrade timestamps
	UpgradeConfig upgrade.Config

	// UseCurrentHeight forces [GetMinimumHeight] to return the current height
	// of the P-Chain instead of the oldest block in the [recentlyAccepted]
	// window.
	//
	// This config is particularly useful for triggering proposervm activation
	// on recently created subnets (without this, users need to wait for
	// [recentlyAcceptedWindowTTL] to pass for activation to occur).
	UseCurrentHeight bool
}

// Create the blockchain described in [tx], but only if this node is a member of
// the chain that validates the blockchain
func (c *Internal) CreateChain(blockchainID ids.ID, tx *txs.CreateChainTx) {
	if c.SybilProtectionEnabled && // Sybil protection is enabled, so nodes might not validate all blockchains
		constants.PrimaryNetworkID != tx.ChainID && // All nodes must validate the primary network
		!c.TrackAllChains && // Not tracking all chains automatically
		!c.TrackedChains.Contains(tx.ChainID) { // This node doesn't validate this blockchain
		return
	}

	chainParams := chains.ChainParameters{
		ID:          blockchainID,
		ChainID:     tx.ChainID,
		GenesisData: tx.GenesisData,
		VMID:        tx.VMID,
		FxIDs:       tx.FxIDs,
		Name:        tx.BlockchainName,
	}

	c.Chains.QueueChainCreation(chainParams)
}
