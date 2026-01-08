// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"time"

	"github.com/luxfi/database"
	"github.com/luxfi/ids"
	"github.com/luxfi/const"
	"github.com/luxfi/math"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
)

type addValidatorRules struct {
	assetID           ids.ID
	minValidatorStake uint64
	maxValidatorStake uint64
	minStakeDuration  time.Duration
	maxStakeDuration  time.Duration
	minDelegationFee  uint32
}

func getValidatorRules(
	backend *Backend,
	chainState state.Chain,
	netID ids.ID,
) (*addValidatorRules, error) {
	if netID == constants.PrimaryNetworkID {
		return &addValidatorRules{
			assetID:           backend.Ctx.XAssetID,
			minValidatorStake: backend.Config.MinValidatorStake,
			maxValidatorStake: backend.Config.MaxValidatorStake,
			minStakeDuration:  backend.Config.MinStakeDuration,
			maxStakeDuration:  backend.Config.MaxStakeDuration,
			minDelegationFee:  backend.Config.MinDelegationFee,
		}, nil
	}

	transformNet, err := GetTransformChainTx(chainState, netID)
	if err != nil {
		return nil, err
	}

	return &addValidatorRules{
		assetID:           transformNet.AssetID,
		minValidatorStake: transformNet.MinValidatorStake,
		maxValidatorStake: transformNet.MaxValidatorStake,
		minStakeDuration:  time.Duration(transformNet.MinStakeDuration) * time.Second,
		maxStakeDuration:  time.Duration(transformNet.MaxStakeDuration) * time.Second,
		minDelegationFee:  transformNet.MinDelegationFee,
	}, nil
}

type addDelegatorRules struct {
	assetID                  ids.ID
	minDelegatorStake        uint64
	maxValidatorStake        uint64
	minStakeDuration         time.Duration
	maxStakeDuration         time.Duration
	maxValidatorWeightFactor byte
}

func getDelegatorRules(
	backend *Backend,
	chainState state.Chain,
	netID ids.ID,
) (*addDelegatorRules, error) {
	if netID == constants.PrimaryNetworkID {
		return &addDelegatorRules{
			assetID:                  backend.Ctx.XAssetID,
			minDelegatorStake:        backend.Config.MinDelegatorStake,
			maxValidatorStake:        backend.Config.MaxValidatorStake,
			minStakeDuration:         backend.Config.MinStakeDuration,
			maxStakeDuration:         backend.Config.MaxStakeDuration,
			maxValidatorWeightFactor: MaxValidatorWeightFactor,
		}, nil
	}

	transformNet, err := GetTransformChainTx(chainState, netID)
	if err != nil {
		return nil, err
	}

	return &addDelegatorRules{
		assetID:                  transformNet.AssetID,
		minDelegatorStake:        transformNet.MinDelegatorStake,
		maxValidatorStake:        transformNet.MaxValidatorStake,
		minStakeDuration:         time.Duration(transformNet.MinStakeDuration) * time.Second,
		maxStakeDuration:         time.Duration(transformNet.MaxStakeDuration) * time.Second,
		maxValidatorWeightFactor: transformNet.MaxValidatorWeightFactor,
	}, nil
}

// GetValidator returns information about the given validator, which may be a
// current validator or pending validator.
func GetValidator(state state.Chain, netID ids.ID, nodeID ids.NodeID) (*state.Staker, error) {
	validator, err := state.GetCurrentValidator(netID, nodeID)
	if err == nil {
		// This node is currently validating the subnet.
		return validator, nil
	}
	if err != database.ErrNotFound {
		// Unexpected error occurred.
		return nil, err
	}
	return state.GetPendingValidator(netID, nodeID)
}

// overDelegated returns true if [validator] will be overdelegated when adding [delegator].
//
// A [validator] would become overdelegated if:
// - the maximum total weight on [validator] exceeds [weightLimit]
func overDelegated(
	state state.Chain,
	validator *state.Staker,
	weightLimit uint64,
	delegatorWeight uint64,
	delegatorStartTime time.Time,
	delegatorEndTime time.Time,
) (bool, error) {
	maxWeight, err := GetMaxWeight(state, validator, delegatorStartTime, delegatorEndTime)
	if err != nil {
		return true, err
	}
	newMaxWeight, err := math.Add(maxWeight, delegatorWeight)
	if err != nil {
		return true, err
	}
	return newMaxWeight > weightLimit, nil
}

// GetMaxWeight returns the maximum total weight of the [validator], including
// its own weight, between [startTime] and [endTime].
// The weight changes are applied in the order they will be applied as chain
// time advances.
// Invariant:
// - [validator.StartTime] <= [startTime] < [endTime] <= [validator.EndTime]
func GetMaxWeight(
	chainState state.Chain,
	validator *state.Staker,
	startTime time.Time,
	endTime time.Time,
) (uint64, error) {
	currentDelegatorIterator, err := chainState.GetCurrentDelegatorIterator(validator.ChainID, validator.NodeID)
	if err != nil {
		return 0, err
	}

	//       stored in the validator state.
	//
	// Calculate the current total weight on this validator, including the
	// weight of the actual validator and the sum of the weights of all of the
	// currently active delegators.
	currentWeight := validator.Weight
	for currentDelegatorIterator.Next() {
		currentDelegator := currentDelegatorIterator.Value()

		currentWeight, err = math.Add(currentWeight, currentDelegator.Weight)
		if err != nil {
			currentDelegatorIterator.Release()
			return 0, err
		}
	}
	currentDelegatorIterator.Release()

	currentDelegatorIterator, err = chainState.GetCurrentDelegatorIterator(validator.ChainID, validator.NodeID)
	if err != nil {
		return 0, err
	}
	pendingDelegatorIterator, err := chainState.GetPendingDelegatorIterator(validator.ChainID, validator.NodeID)
	if err != nil {
		currentDelegatorIterator.Release()
		return 0, err
	}
	delegatorChangesIterator := state.NewStakerDiffIterator(currentDelegatorIterator, pendingDelegatorIterator)
	defer delegatorChangesIterator.Release()

	// Iterate over the future stake weight changes and calculate the maximum
	// total weight on the validator, only including the points in the time
	// range [startTime, endTime].
	var currentMax uint64
	for delegatorChangesIterator.Next() {
		delegator, isAdded := delegatorChangesIterator.Value()
		// [delegator.NextTime] > [endTime]
		if delegator.NextTime.After(endTime) {
			// This delegation change (and all following changes) occurs after
			// [endTime]. Since we're calculating the max amount staked in
			// [startTime, endTime], we can stop.
			break
		}

		// [delegator.NextTime] >= [startTime]
		if !delegator.NextTime.Before(startTime) {
			// We have advanced time to be at the inside of the delegation
			// window. Make sure that the max weight is updated accordingly.
			currentMax = max(currentMax, currentWeight)
		}

		var op func(uint64, uint64) (uint64, error)
		if isAdded {
			op = math.Add
		} else {
			op = math.Sub
		}
		currentWeight, err = op(currentWeight, delegator.Weight)
		if err != nil {
			return 0, err
		}
	}
	// Because we assume [startTime] < [endTime], we have advanced time to
	// be at the end of the delegation window. Make sure that the max weight is
	// updated accordingly.
	return max(currentMax, currentWeight), nil
}

func GetTransformChainTx(chain state.Chain, netID ids.ID) (*txs.TransformChainTx, error) {
	transformNetIntf, err := chain.GetNetTransformation(netID)
	if err != nil {
		return nil, err
	}

	transformNet, ok := transformNetIntf.Unsigned.(*txs.TransformChainTx)
	if !ok {
		return nil, ErrIsNotTransformChainTx
	}

	return transformNet, nil
}
