// (c) 2023, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package iface

import (
	"bytes"
	"context"
	"errors"
	"github.com/luxfi/geth/common"
	"golang.org/x/exp/maps"
)

var (
	ErrUnknownValidator = errors.New("unknown validator")
	ErrWeightOverflow   = errors.New("weight overflowed")
)

// Context defines the block context for warp validation
type Context struct {
	// PChainHeight is the height that this block will use to verify it's state.
	PChainHeight uint64
}

// WarpValidator represents a validator with BLS public key and weight
type WarpValidator struct {
	PublicKey      *BLSPublicKey
	PublicKeyBytes []byte
	Weight         uint64
	NodeIDs        []common.Hash
}

func (v *WarpValidator) Compare(o *WarpValidator) int {
	return bytes.Compare(v.PublicKeyBytes, o.PublicKeyBytes)
}

// CanonicalValidatorSet represents the canonical set of validators
type CanonicalValidatorSet struct {
	// Validators slice in canonical ordering of the validators that has public key
	Validators []*WarpValidator
	// The total weight of all the validators, including the ones that doesn't have a public key
	TotalWeight uint64
}

// ValidatorOutput represents a validator in the validator set
type ValidatorOutput struct {
	NodeID    common.Hash
	PublicKey *BLSPublicKey
	Weight    uint64
}

// GetCanonicalValidatorSetFromChainID returns the canonical validator set given a ValidatorState, pChain height and a sourceChainID.
func GetCanonicalValidatorSetFromChainID(
	ctx context.Context,
	pChainState ValidatorState,
	pChainHeight uint64,
	sourceChainID common.Hash,
) (CanonicalValidatorSet, error) {
	subnetID, err := pChainState.GetSubnetID(ctx, sourceChainID)
	if err != nil {
		return CanonicalValidatorSet{}, err
	}

	return GetCanonicalValidatorSetFromSubnetID(ctx, pChainState, pChainHeight, subnetID)
}

// GetCanonicalValidatorSetFromSubnetID returns the CanonicalValidatorSet of subnetID at pChainHeight.
func GetCanonicalValidatorSetFromSubnetID(
	ctx context.Context,
	pChainState ValidatorState,
	pChainHeight uint64,
	subnetID common.Hash,
) (CanonicalValidatorSet, error) {
	// Get the validator set at the given height.
	vdrSet, err := pChainState.GetValidatorSet(ctx, pChainHeight, subnetID)
	if err != nil {
		return CanonicalValidatorSet{}, err
	}

	// Convert the validator set into the canonical ordering.
	return FlattenValidatorSet(vdrSet)
}

// FlattenValidatorSet converts the provided vdrSet into a canonical ordering.
func FlattenValidatorSet(vdrSet map[common.Hash]*ValidatorOutput) (CanonicalValidatorSet, error) {
	var (
		vdrs        = make(map[string]*WarpValidator, len(vdrSet))
		totalWeight uint64
	)
	for nodeID, vdr := range vdrSet {
		var overflow bool
		totalWeight, overflow = SafeAdd(totalWeight, vdr.Weight)
		if overflow {
			return CanonicalValidatorSet{}, ErrWeightOverflow
		}

		if vdr.PublicKey == nil {
			continue
		}

		pkBytes := vdr.PublicKey.UncompressedBytes()
		uniqueVdr, ok := vdrs[string(pkBytes)]
		if !ok {
			uniqueVdr = &WarpValidator{
				PublicKey:      vdr.PublicKey,
				PublicKeyBytes: pkBytes,
			}
			vdrs[string(pkBytes)] = uniqueVdr
		}

		uniqueVdr.Weight += vdr.Weight // Impossible to overflow here
		uniqueVdr.NodeIDs = append(uniqueVdr.NodeIDs, nodeID)
	}

	// Sort validators by public key
	vdrList := maps.Values(vdrs)
	SortValidators(vdrList)
	return CanonicalValidatorSet{Validators: vdrList, TotalWeight: totalWeight}, nil
}

// SortValidators sorts validators by their public key bytes
func SortValidators(vdrs []*WarpValidator) {
	// Simple insertion sort for now
	for i := 1; i < len(vdrs); i++ {
		j := i
		for j > 0 && vdrs[j-1].Compare(vdrs[j]) > 0 {
			vdrs[j-1], vdrs[j] = vdrs[j], vdrs[j-1]
			j--
		}
	}
}

