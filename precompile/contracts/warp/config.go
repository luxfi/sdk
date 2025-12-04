// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"

	consensuscontext "github.com/luxfi/consensus/context"
	validators "github.com/luxfi/consensus/validator"
	evmconsensus "github.com/luxfi/evm/consensus"
	"github.com/luxfi/evm/precompile/precompileconfig"
	"github.com/luxfi/evm/predicate"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/common/math"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/node/utils/constants"
	luxwarp "github.com/luxfi/warp"
	"github.com/luxfi/warp/payload"
)

const (
	WarpDefaultQuorumNumerator uint64 = 67
	WarpQuorumNumeratorMinimum uint64 = 33
	WarpQuorumDenominator      uint64 = 100
)

var (
	_ precompileconfig.Config     = (*Config)(nil)
	_ precompileconfig.Predicater = (*Config)(nil)
	_ precompileconfig.Accepter   = (*Config)(nil)
)

var (
	errOverflowSignersGasCost     = errors.New("overflow calculating warp signers gas cost")
	errInvalidPredicateBytes      = errors.New("cannot unpack predicate bytes")
	errInvalidWarpMsg             = errors.New("cannot unpack warp message")
	errCannotParseWarpMsg         = errors.New("cannot parse warp message")
	errInvalidWarpMsgPayload      = errors.New("cannot unpack warp message payload")
	errInvalidAddressedPayload    = errors.New("cannot unpack addressed payload")
	errInvalidBlockHashPayload    = errors.New("cannot unpack block hash payload")
	errCannotGetNumSigners        = errors.New("cannot fetch num signers from warp message")
	errWarpCannotBeActivated      = errors.New("warp cannot be activated before Durango")
	errFailedVerification         = errors.New("cannot verify warp signature")
	errCannotRetrieveValidatorSet = errors.New("cannot retrieve validator set")
)

// Config implements the precompileconfig.Config interface and
// adds specific configuration for Warp.
type Config struct {
	precompileconfig.Upgrade
	QuorumNumerator              uint64 `json:"quorumNumerator"`
	RequirePrimaryNetworkSigners bool   `json:"requirePrimaryNetworkSigners"`
}

// NewConfig returns a config for a network upgrade at [blockTimestamp] that enables
// Warp with the given quorum numerator.
func NewConfig(blockTimestamp *uint64, quorumNumerator uint64, requirePrimaryNetworkSigners bool) *Config {
	return &Config{
		Upgrade:                      precompileconfig.Upgrade{BlockTimestamp: blockTimestamp},
		QuorumNumerator:              quorumNumerator,
		RequirePrimaryNetworkSigners: requirePrimaryNetworkSigners,
	}
}

// NewDefaultConfig returns a config for a network upgrade at [blockTimestamp] that enables
// Warp with the default quorum numerator (0 denotes using the default).
func NewDefaultConfig(blockTimestamp *uint64) *Config {
	return NewConfig(blockTimestamp, 0, false)
}

// NewDisableConfig returns config for a network upgrade at [blockTimestamp]
// that disables Warp.
func NewDisableConfig(blockTimestamp *uint64) *Config {
	return &Config{
		Upgrade: precompileconfig.Upgrade{
			BlockTimestamp: blockTimestamp,
			Disable:        true,
		},
	}
}

// Key returns the key for the Warp precompileconfig.
// This should be the same key as used in the precompile module.
func (*Config) Key() string { return ConfigKey }

// Verify tries to verify Config and returns an error accordingly.
func (c *Config) Verify(chainConfig precompileconfig.ChainConfig) error {
	if c.Timestamp() != nil {
		// If Warp attempts to activate before Durango, fail verification
		timestamp := *c.Timestamp()
		if !chainConfig.IsDurango(timestamp) {
			return errWarpCannotBeActivated
		}
	}

	if c.QuorumNumerator > WarpQuorumDenominator {
		return fmt.Errorf("cannot specify quorum numerator (%d) > quorum denominator (%d)", c.QuorumNumerator, WarpQuorumDenominator)
	}
	// If a non-default quorum numerator is specified and it is less than the minimum, return an error
	if c.QuorumNumerator != 0 && c.QuorumNumerator < WarpQuorumNumeratorMinimum {
		return fmt.Errorf("cannot specify quorum numerator (%d) < min quorum numerator (%d)", c.QuorumNumerator, WarpQuorumNumeratorMinimum)
	}
	return nil
}

// Equal returns true if [s] is a [*Config] and it has been configured identical to [c].
func (c *Config) Equal(s precompileconfig.Config) bool {
	// typecast before comparison
	other, ok := (s).(*Config)
	if !ok {
		return false
	}
	equals := c.Upgrade.Equal(&other.Upgrade)
	return equals && c.QuorumNumerator == other.QuorumNumerator
}

func (c *Config) Accept(acceptCtx *precompileconfig.AcceptContext, blockHash common.Hash, blockNumber uint64, txHash common.Hash, logIndex int, topics []common.Hash, logData []byte) error {
	unsignedMessage, err := UnpackSendWarpEventDataToMessage(logData)
	if err != nil {
		return fmt.Errorf("failed to parse warp log data into unsigned message (TxHash: %s, LogIndex: %d): %w", txHash, logIndex, err)
	}
	log.Debug(
		"Accepted warp unsigned message",
		"blockHash", blockHash,
		"blockNumber", blockNumber,
		"txHash", txHash,
		"logIndex", logIndex,
		"logData", common.Bytes2Hex(logData),
		"warpMessageID", unsignedMessage.ID(),
	)
	if err := acceptCtx.Warp.AddMessage(unsignedMessage); err != nil {
		return fmt.Errorf("failed to add warp message during accept (TxHash: %s, LogIndex: %d): %w", txHash, logIndex, err)
	}
	return nil
}

// PredicateGas returns the amount of gas necessary to verify the predicate
// PredicateGas charges for:
// 1. Base cost of the message
// 2. Size of the message
// 3. Number of signers
// 4. TODO: Lookup of the validator set
//
// If the payload of the warp message fails parsing, return a non-nil error invalidating the transaction.
func (c *Config) PredicateGas(predicateBytes []byte) (uint64, error) {
	totalGas := GasCostPerSignatureVerification
	bytesGasCost, overflow := math.SafeMul(GasCostPerWarpMessageBytes, uint64(len(predicateBytes)))
	if overflow {
		return 0, fmt.Errorf("overflow calculating gas cost for warp message bytes of size %d", len(predicateBytes))
	}
	totalGas, overflow = math.SafeAdd(totalGas, bytesGasCost)
	if overflow {
		return 0, fmt.Errorf("overflow adding bytes gas cost of size %d", len(predicateBytes))
	}

	unpackedPredicateBytes, err := predicate.UnpackPredicate(predicateBytes)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", errInvalidPredicateBytes, err)
	}
	warpMessage, err := luxwarp.ParseMessage(unpackedPredicateBytes)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", errInvalidWarpMsg, err)
	}
	_, err = payload.ParsePayload(warpMessage.UnsignedMessage.Payload)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", errInvalidWarpMsgPayload, err)
	}

	// Type assert to BitSetSignature to get number of signers
	bitSetSig, ok := warpMessage.Signature.(*luxwarp.BitSetSignature)
	if !ok {
		return 0, fmt.Errorf("%w: signature is not a BitSetSignature", errCannotGetNumSigners)
	}
	numSigners := uint64(bitSetSig.Signers.Len())
	if numSigners == 0 {
		return 0, fmt.Errorf("%w: no signers in bit set", errCannotGetNumSigners)
	}
	signerGas, overflow := math.SafeMul(uint64(numSigners), GasCostPerWarpSigner)
	if overflow {
		return 0, errOverflowSignersGasCost
	}
	totalGas, overflow = math.SafeAdd(totalGas, signerGas)
	if overflow {
		return 0, fmt.Errorf("overflow adding signer gas (PrevTotal: %d, VerificationGas: %d)", totalGas, signerGas)
	}

	return totalGas, nil
}

// ValidatorOutputGetter is an optional interface that can be implemented
// by validator states to provide full validator output including public keys
type ValidatorOutputGetter interface {
	GetValidatorSetWithOutput(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error)
}

// VerifyPredicate returns whether the predicate described by [predicateBytes] passes verification.
func (c *Config) VerifyPredicate(predicateContext *precompileconfig.PredicateContext, predicateBytes []byte) error {
	unpackedPredicateBytes, err := predicate.UnpackPredicate(predicateBytes)
	if err != nil {
		return fmt.Errorf("%w: %w", errInvalidPredicateBytes, err)
	}

	// Note: PredicateGas should be called before VerifyPredicate, so we should never reach an error case here.
	warpMsg, err := luxwarp.ParseMessage(unpackedPredicateBytes)
	if err != nil {
		return fmt.Errorf("%w: %w", errCannotParseWarpMsg, err)
	}

	quorumNumerator := WarpDefaultQuorumNumerator
	if c.QuorumNumerator != 0 {
		quorumNumerator = c.QuorumNumerator
	}

	// Get ValidatorState from context
	validatorState := consensuscontext.GetValidatorState(predicateContext.ConsensusCtx)
	if validatorState == nil {
		return fmt.Errorf("validator state not found in context")
	}

	// Get the source chain ID from the warp message
	sourceChainID := warpMsg.UnsignedMessage.SourceChainID

	// Get subnet ID from validator state for the source chain
	var chainIDFixed ids.ID
	copy(chainIDFixed[:], sourceChainID)
	sourceSubnetID, err := validatorState.GetSubnetID(chainIDFixed)
	if err != nil {
		return fmt.Errorf("failed to get subnet ID for source chain: %w", err)
	}

	// Get the receiving subnet ID (the subnet this VM is running on)
	// Use the EVM consensus package's GetSubnetID which matches how vm.ctx is set up
	receivingSubnetID := evmconsensus.GetSubnetID(predicateContext.ConsensusCtx)

	// Determine which subnet's validators to use
	// The logic is:
	// 1. For subnet sources (sourceSubnetID != Empty): Use source subnet's validators
	// 2. For P-Chain sources: Use receiving subnet's validators (P-Chain exempt)
	// 3. For other primary network sources (e.g., C-Chain):
	//    - With RequirePrimaryNetworkSigners=true: Use primary network's validators
	//    - With RequirePrimaryNetworkSigners=false: Use receiving subnet's validators
	var requestedSubnetID ids.ID
	if sourceSubnetID != ids.Empty && sourceSubnetID != constants.PrimaryNetworkID {
		// Source is from a subnet - use that subnet's validators
		requestedSubnetID = sourceSubnetID
	} else if chainIDFixed == constants.PlatformChainID {
		// P-Chain source - always use receiving subnet's validators
		requestedSubnetID = receivingSubnetID
	} else if c.RequirePrimaryNetworkSigners {
		// Other primary network source with RequirePrimaryNetworkSigners
		// Use primary network validators
		requestedSubnetID = constants.PrimaryNetworkID
	} else {
		// Other primary network source without RequirePrimaryNetworkSigners
		// Use receiving subnet's validators
		requestedSubnetID = receivingSubnetID
	}

	pChainHeight := predicateContext.ProposerVMBlockCtx.PChainHeight

	// Build warp validators - try to get full validator output with public keys
	var allValidators []*luxwarp.Validator
	var totalWeight uint64

	// Check if the validator state supports getting full output with public keys
	if outputGetter, ok := validatorState.(ValidatorOutputGetter); ok {
		// Use the full validator output which includes public keys
		vdrOutputs, err := outputGetter.GetValidatorSetWithOutput(predicateContext.ConsensusCtx, pChainHeight, requestedSubnetID)
		if err != nil {
			return fmt.Errorf("%w: %w", errCannotRetrieveValidatorSet, err)
		}

		allValidators = make([]*luxwarp.Validator, 0, len(vdrOutputs))
		for nodeID, output := range vdrOutputs {
			totalWeight += output.Weight

			vdr := &luxwarp.Validator{
				NodeID: nodeID[:],
				Weight: output.Weight,
			}

			// Parse public key if available
			if len(output.PublicKey) > 0 {
				pk, pkErr := luxwarp.ParsePublicKey(output.PublicKey)
				if pkErr == nil {
					vdr.PublicKey = pk
					vdr.PublicKeyBytes = output.PublicKey
				} else {
					log.Debug("failed to parse validator public key", "nodeID", nodeID, "err", pkErr)
				}
			}

			allValidators = append(allValidators, vdr)
		}
	} else {
		// Fallback: get weights only (signature verification will fail without public keys)
		vdrWeights, err := validatorState.GetValidatorSet(pChainHeight, requestedSubnetID)
		if err != nil {
			return fmt.Errorf("%w: %w", errCannotRetrieveValidatorSet, err)
		}

		allValidators = make([]*luxwarp.Validator, 0, len(vdrWeights))
		for nodeID, weight := range vdrWeights {
			totalWeight += weight
			vdr := &luxwarp.Validator{
				NodeID: nodeID[:],
				Weight: weight,
			}
			allValidators = append(allValidators, vdr)
		}
	}

	// Aggregate validators by public key - this handles the case where multiple
	// validators share the same public key. We sum their weights and keep one
	// representative validator per unique public key.
	pkeyToWeight := make(map[string]uint64)
	pkeyToValidator := make(map[string]*luxwarp.Validator)

	for _, vdr := range allValidators {
		if len(vdr.PublicKeyBytes) == 0 {
			continue // Skip validators without public keys
		}
		pkStr := string(vdr.PublicKeyBytes)
		pkeyToWeight[pkStr] += vdr.Weight
		if _, exists := pkeyToValidator[pkStr]; !exists {
			pkeyToValidator[pkStr] = vdr
		}
	}

	// Build canonical validator set with aggregated weights
	canonicalValidators := make([]*luxwarp.Validator, 0, len(pkeyToValidator))
	for pkStr, vdr := range pkeyToValidator {
		// Create a copy with aggregated weight
		aggregatedVdr := &luxwarp.Validator{
			NodeID:         vdr.NodeID,
			Weight:         pkeyToWeight[pkStr],
			PublicKey:      vdr.PublicKey,
			PublicKeyBytes: vdr.PublicKeyBytes,
		}
		canonicalValidators = append(canonicalValidators, aggregatedVdr)
	}

	// Sort validators by public key bytes for canonical ordering
	// This matches the order used when creating signatures
	sort.Slice(canonicalValidators, func(i, j int) bool {
		return bytes.Compare(canonicalValidators[i].PublicKeyBytes, canonicalValidators[j].PublicKeyBytes) < 0
	})

	// Get signature
	bitSetSig, ok := warpMsg.Signature.(*luxwarp.BitSetSignature)
	if !ok {
		return fmt.Errorf("%w: signature is not a BitSetSignature", errCannotGetNumSigners)
	}

	// Calculate signed weight using canonical validators (with aggregated weights)
	signedWeight, err := bitSetSig.GetSignedWeight(canonicalValidators)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedVerification, err)
	}

	// Verify quorum
	err = luxwarp.VerifyWeight(signedWeight, totalWeight, quorumNumerator, WarpQuorumDenominator)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedVerification, err)
	}

	// Verify the BLS signature using canonical validators
	err = bitSetSig.Verify(warpMsg.UnsignedMessage.Bytes(), canonicalValidators)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedVerification, err)
	}

	return nil
}
