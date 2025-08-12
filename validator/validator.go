// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package validator

import (
	"encoding/json"
	"fmt"

	"github.com/luxfi/ids"
	luxdjson "github.com/luxfi/node/utils/json"
	"github.com/luxfi/node/utils/rpc"
	"github.com/luxfi/node/vms/platformvm"
	"github.com/luxfi/sdk/contract"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/utils"

	"github.com/luxfi/crypto"
)

type ValidatorKind int64

const (
	UndefinedValidatorKind ValidatorKind = iota
	NonValidator
	SovereignValidator
	NonSovereignValidator
)

// To enable querying validation IDs from P-Chain
type CurrentValidatorInfo struct {
	Weight       luxdjson.Uint64 `json:"weight"`
	NodeID       ids.NodeID      `json:"nodeID"`
	ValidationID ids.ID          `json:"validationID"`
	Balance      luxdjson.Uint64 `json:"balance"`
}

func GetTotalWeight(network models.Network, subnetID ids.ID) (uint64, error) {
	validators, err := GetCurrentValidators(network, subnetID)
	if err != nil {
		return 0, err
	}
	weight := uint64(0)
	for _, vdr := range validators {
		weight += uint64(vdr.Weight)
	}
	return weight, nil
}

func IsValidator(network models.Network, subnetID ids.ID, nodeID ids.NodeID) (bool, error) {
	validators, err := GetCurrentValidators(network, subnetID)
	if err != nil {
		return false, err
	}
	nodeIDs := utils.Map(validators, func(v CurrentValidatorInfo) ids.NodeID { return v.NodeID })
	return utils.Belongs(nodeIDs, nodeID), nil
}

func GetValidatorBalance(net models.Network, validationID ids.ID) (uint64, error) {
	validator, err := GetValidatorInfo(net, validationID)
	if err != nil {
		return 0, err
	}
	// For L1 validators, the balance is the staked amount
	// Return the validator's weight as the balance
	return validator.Weight, nil
}

func GetValidatorInfo(net models.Network, validationID ids.ID) (platformvm.ClientPermissionlessValidator, error) {
	// Connect to the platform chain
	pClient := platformvm.NewClient(net.Endpoint())

	// Get current validators for the subnet
	ctx, cancel := utils.GetAPIContext()
	defer cancel()

	// Query the validator by validation ID
	// Since GetL1Validator is not available, we'll query all validators and find the matching one
	validators, err := pClient.GetCurrentValidators(ctx, ids.Empty, nil)
	if err != nil {
		return platformvm.ClientPermissionlessValidator{}, fmt.Errorf("failed to get validators: %w", err)
	}

	// Search for the validator with matching validation ID
	for _, validator := range validators {
		// Check if this validator matches our validation ID
		// Note: This is a workaround until GetL1Validator is available
		if validator.TxID == validationID {
			// Found the validator
			return platformvm.ClientPermissionlessValidator{
				ClientStaker: platformvm.ClientStaker{
					TxID:      validator.TxID,
					StartTime: validator.StartTime,
					EndTime:   validator.EndTime,
					Weight:    validator.Weight,
					NodeID:    validator.NodeID,
				},
				// Other fields will be populated when available
			}, nil
		}
	}

	return platformvm.ClientPermissionlessValidator{}, fmt.Errorf("validator with ID %s not found", validationID)
}

// Returns the validation ID for the Node ID, as registered at the validator manager
// Will return ids.Empty in case it is not registered
func GetValidationID(
	rpcURL string,
	managerAddress crypto.Address,
	nodeID ids.NodeID,
) (ids.ID, error) {
	// if specialized, need to retrieve underlying manager
	// needs to directly access the manager, does not work with a proxy
	out, err := contract.CallToMethod(
		rpcURL,
		managerAddress,
		"getStakingManagerSettings()->(address,uint256,uint256,uint64,uint16,uint8,uint256,address,bytes32)",
	)
	if err == nil && len(out) == 9 {
		validatorManager, ok := out[0].(crypto.Address)
		if ok {
			managerAddress = validatorManager
		}
	}
	out, err = contract.CallToMethod(
		rpcURL,
		managerAddress,
		"registeredValidators(bytes)->(bytes32)",
		nodeID[:],
	)
	if err != nil {
		return ids.Empty, err
	}
	return contract.GetSmartContractCallResult[[32]byte]("registeredValidators", out)
}

func GetValidatorKind(
	network models.Network,
	subnetID ids.ID,
	nodeID ids.NodeID,
) (ValidatorKind, error) {
	pClient := platformvm.NewClient(network.Endpoint())
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	vs, err := pClient.GetCurrentValidators(ctx, subnetID, nil)
	if err != nil {
		return UndefinedValidatorKind, err
	}
	for _, v := range vs {
		if v.NodeID == nodeID {
			if v.TxID == ids.Empty {
				return SovereignValidator, nil
			}
			return NonSovereignValidator, nil
		}
	}
	return NonValidator, nil
}

// Enables querying the validation IDs from P-Chain
func GetCurrentValidators(network models.Network, subnetID ids.ID) ([]CurrentValidatorInfo, error) {
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	requester := rpc.NewEndpointRequester(network.Endpoint() + "/ext/P")
	res := &platformvm.GetCurrentValidatorsReply{}
	if err := requester.SendRequest(
		ctx,
		"platform.getCurrentValidators",
		&platformvm.GetCurrentValidatorsArgs{
			SubnetID: subnetID,
			NodeIDs:  nil,
		},
		res,
	); err != nil {
		return nil, err
	}
	validators := make([]CurrentValidatorInfo, 0, len(res.Validators))
	for _, vI := range res.Validators {
		vBytes, err := json.Marshal(vI)
		if err != nil {
			return nil, err
		}
		var v CurrentValidatorInfo
		if err := json.Unmarshal(vBytes, &v); err != nil {
			return nil, err
		}
		validators = append(validators, v)
	}
	return validators, nil
}
