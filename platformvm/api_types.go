// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"strconv"
	"time"

	apitypes "github.com/luxfi/api/types"
	"github.com/luxfi/formatting"
	"github.com/luxfi/ids"
	"github.com/luxfi/proto/p/status"
	"github.com/luxfi/utxo"
	validators "github.com/luxfi/validators"
	"github.com/luxfi/vm/components/gas"
)

// Uint16 is a uint16 that marshals as a JSON string for compatibility with
// the historical luxd JSON-RPC numeric wire (every numeric over the wire is
// a string). luxfi/api/types ships Uint64/Uint32/Float64 but not Uint16;
// this is a local 1:1 mirror sitting at the same external boundary.
type Uint16 uint16

const jsonNull = "null"

func (u Uint16) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strconv.FormatUint(uint64(u), 10) + `"`), nil
}

func (u *Uint16) UnmarshalJSON(b []byte) error {
	str := string(b)
	if str == jsonNull {
		return nil
	}
	if len(str) >= 2 {
		if lastIndex := len(str) - 1; str[0] == '"' && str[lastIndex] == '"' {
			str = str[1:lastIndex]
		}
	}
	val, err := strconv.ParseUint(str, 10, 16)
	*u = Uint16(val)
	return err
}

// GetBalanceRequest is the request for GetBalance
type GetBalanceRequest struct {
	Addresses []string `json:"addresses"`
}

// GetBalanceResponse is the response from GetBalance
type GetBalanceResponse struct {
	Balance             apitypes.Uint64            `json:"balance"`
	Unlocked            apitypes.Uint64            `json:"unlocked"`
	LockedStakeable     apitypes.Uint64            `json:"lockedStakeable"`
	LockedNotStakeable  apitypes.Uint64            `json:"lockedNotStakeable"`
	Balances            map[ids.ID]apitypes.Uint64 `json:"balances"`
	Unlockeds           map[ids.ID]apitypes.Uint64 `json:"unlockeds"`
	LockedStakeables    map[ids.ID]apitypes.Uint64 `json:"lockedStakeables"`
	LockedNotStakeables map[ids.ID]apitypes.Uint64 `json:"lockedNotStakeables"`
	UTXOIDs             []*utxo.UTXOID             `json:"utxoIDs"`
}

// GetNetArgs is the request for GetNet
type GetNetArgs struct {
	ChainID ids.ID `json:"chainID"`
}

// GetNetResponse is the response from GetNet
type GetNetResponse struct {
	IsPermissioned        bool            `json:"isPermissioned"`
	ControlKeys           []string        `json:"controlKeys"`
	Threshold             apitypes.Uint32 `json:"threshold"`
	Locktime              apitypes.Uint64 `json:"locktime"`
	NetTransformationTxID ids.ID          `json:"netTransformationTxID"`
	ConversionID          ids.ID          `json:"conversionID"`
	ManagerChainID        ids.ID          `json:"managerChainID"`
	ManagerAddress        []byte          `json:"managerAddress"`
}

// GetNetsArgs is the request for GetNets
type GetNetsArgs struct {
	IDs []ids.ID `json:"ids"`
}

// APINet represents a net in the API
type APINet struct {
	ID          ids.ID          `json:"id"`
	ControlKeys []string        `json:"controlKeys"`
	Threshold   apitypes.Uint32 `json:"threshold"`
}

// GetNetsResponse is the response from GetNets
type GetNetsResponse struct {
	Nets []APINet `json:"nets"`
}

// GetStakingAssetIDArgs is the request for GetStakingAssetID
type GetStakingAssetIDArgs struct {
	ChainID ids.ID `json:"chainID"`
}

// GetStakingAssetIDResponse is the response from GetStakingAssetID
type GetStakingAssetIDResponse struct {
	AssetID ids.ID `json:"assetID"`
}

// GetCurrentValidatorsArgs is the request for GetCurrentValidators
type GetCurrentValidatorsArgs struct {
	ChainID ids.ID       `json:"chainID"`
	NodeIDs []ids.NodeID `json:"nodeIDs"`
}

// GetCurrentValidatorsReply is the response from GetCurrentValidators
type GetCurrentValidatorsReply struct {
	Validators []interface{} `json:"validators"`
}

// GetL1ValidatorArgs is the request for GetL1Validator
type GetL1ValidatorArgs struct {
	ValidationID ids.ID `json:"validationID"`
}

// L1ValidatorOwner represents an owner in an L1 validator response
type L1ValidatorOwner struct {
	Locktime  apitypes.Uint64 `json:"locktime"`
	Threshold apitypes.Uint32 `json:"threshold"`
	Addresses []string        `json:"addresses"`
}

// GetL1ValidatorReply is the response from GetL1Validator
type GetL1ValidatorReply struct {
	ChainID               ids.ID           `json:"chainID"`
	NodeID                ids.NodeID       `json:"nodeID"`
	PublicKey             *[]byte          `json:"publicKey,omitempty"`
	RemainingBalanceOwner L1ValidatorOwner `json:"remainingBalanceOwner"`
	DeactivationOwner     L1ValidatorOwner `json:"deactivationOwner"`
	StartTime             apitypes.Uint64  `json:"startTime"`
	Weight                apitypes.Uint64  `json:"weight"`
	MinNonce              *apitypes.Uint64 `json:"minNonce,omitempty"`
	Balance               *apitypes.Uint64 `json:"balance,omitempty"`
	Height                apitypes.Uint64  `json:"height"`
}

// APIBlockchain represents a blockchain in the API response.
// NetID is the ID of the network that validates this blockchain.
type APIBlockchain struct {
	ID    ids.ID `json:"id"`
	Name  string `json:"name"`
	NetID ids.ID `json:"netID"`
	VMID  ids.ID `json:"vmID"`
}

// GetBlockchainsReply is the response from GetBlockchains
type GetBlockchainsReply struct {
	Blockchains []APIBlockchain `json:"blockchains"`
}

// GetTxStatusArgs is the request for GetTxStatus
type GetTxStatusArgs struct {
	TxID ids.ID `json:"txID"`
}

// GetTxStatusResponse is the response from GetTxStatus
type GetTxStatusResponse struct {
	Status status.Status `json:"status"`
	Reason string        `json:"reason,omitempty"`
}

// GetCurrentSupplyArgs is the request for GetCurrentSupply
type GetCurrentSupplyArgs struct {
	ChainID ids.ID `json:"chainID"`
}

// GetCurrentSupplyReply is the response from GetCurrentSupply
type GetCurrentSupplyReply struct {
	Supply apitypes.Uint64 `json:"supply"`
	Height apitypes.Uint64 `json:"height"`
}

// SampleValidatorsArgs is the request for SampleValidators
type SampleValidatorsArgs struct {
	ChainID ids.ID `json:"chainID"`
	Size    Uint16 `json:"size"`
}

// SampleValidatorsReply is the response from SampleValidators
type SampleValidatorsReply struct {
	Validators []ids.NodeID `json:"validators"`
}

// GetBlockchainStatusArgs is the request for GetBlockchainStatus
type GetBlockchainStatusArgs struct {
	BlockchainID string `json:"blockchainID"`
}

// GetBlockchainStatusReply is the response from GetBlockchainStatus
type GetBlockchainStatusReply struct {
	Status status.BlockchainStatus `json:"status"`
}

// ValidatedByArgs is the request for ValidatedBy
type ValidatedByArgs struct {
	BlockchainID ids.ID `json:"blockchainID"`
}

// ValidatedByResponse is the response from ValidatedBy
type ValidatedByResponse struct {
	ChainID ids.ID `json:"chainID"`
}

// ValidatesArgs is the request for Validates
type ValidatesArgs struct {
	ChainID ids.ID `json:"chainID"`
}

// ValidatesResponse is the response from Validates
type ValidatesResponse struct {
	BlockchainIDs []ids.ID `json:"blockchainIDs"`
}

// GetBlockchainsResponse is the response from GetBlockchains
type GetBlockchainsResponse struct {
	Blockchains []APIBlockchain `json:"blockchains"`
}

// GetStakeArgs is the request for GetStake
type GetStakeArgs struct {
	JSONAddresses  interface{}         `json:"addresses"`
	ValidatorsOnly bool                `json:"validatorsOnly"`
	Encoding       formatting.Encoding `json:"encoding"`
}

// GetStakeReply is the response from GetStake
type GetStakeReply struct {
	Stakeds  map[ids.ID]apitypes.Uint64 `json:"stakeds"`
	Outputs  []string                   `json:"stakedOutputs"`
	Encoding formatting.Encoding        `json:"encoding"`
}

// GetMinStakeArgs is the request for GetMinStake
type GetMinStakeArgs struct {
	ChainID ids.ID `json:"chainID"`
}

// GetMinStakeReply is the response from GetMinStake
type GetMinStakeReply struct {
	MinValidatorStake apitypes.Uint64 `json:"minValidatorStake"`
	MinDelegatorStake apitypes.Uint64 `json:"minDelegatorStake"`
}

// GetTotalStakeArgs is the request for GetTotalStake
type GetTotalStakeArgs struct {
	ChainID ids.ID `json:"chainID"`
}

// GetTotalStakeReply is the response from GetTotalStake
type GetTotalStakeReply struct {
	Stake  apitypes.Uint64 `json:"stake,omitempty"`
	Weight apitypes.Uint64 `json:"weight,omitempty"`
}

// GetRewardUTXOsReply is the response from GetRewardUTXOs
type GetRewardUTXOsReply struct {
	UTXOs    []string            `json:"utxos"`
	Encoding formatting.Encoding `json:"encoding"`
}

// GetTimestampReply is the response from GetTimestamp
type GetTimestampReply struct {
	Timestamp time.Time `json:"timestamp"`
}

// GetValidatorsAtArgs is the request for GetValidatorsAt
type GetValidatorsAtArgs struct {
	Height  interface{} `json:"height"`
	ChainID ids.ID      `json:"chainID"`
}

// GetValidatorsAtReply is the response from GetValidatorsAt
type GetValidatorsAtReply struct {
	Validators map[ids.NodeID]*validators.GetValidatorOutput `json:"validators"`
}

// GetFeeConfigReply is the response from GetFeeConfig
type GetFeeConfigReply struct {
	GasConfig interface{} `json:"gasConfig"`
	GasPrice  interface{} `json:"gasPrice"`
	Timestamp time.Time   `json:"timestamp"`
}

// GetFeeStateReply is the response from GetFeeState
type GetFeeStateReply struct {
	gas.State
	Price gas.Price `json:"price"`
	Time  time.Time `json:"timestamp"`
}

// GetValidatorFeeStateReply is the response from GetValidatorFeeState
type GetValidatorFeeStateReply struct {
	Excess gas.Gas   `json:"excess"`
	Price  gas.Price `json:"price"`
	Time   time.Time `json:"timestamp"`
}
