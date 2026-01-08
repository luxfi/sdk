// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/luxfi/ids"
	validators "github.com/luxfi/consensus/validator"
	"github.com/luxfi/metric"
	"luxfi/cache/lru"
	"github.com/luxfi/upgrade"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/math"
	"github.com/luxfi/math/set"
)

var (
	_ utils.Sortable[*Validator] = (*Validator)(nil)

	ErrUnknownValidator = errors.New("unknown validator")
	ErrWeightOverflow   = errors.New("weight overflowed")
)

// ValidatorState defines the functions that must be implemented to get
// the canonical validator set for warp message validation.
type ValidatorState interface {
	GetValidatorSet(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*ValidatorData, error)
}

// ValidatorData contains the data for a single validator
type ValidatorData struct {
	NodeID           ids.NodeID
	PublicKey        []byte // BLS public key (classical)
	CoronaPubKey   []byte // Corona public key (post-quantum)
	Weight           uint64
}

type CanonicalValidatorSet struct {
	// Validators slice in canonical ordering of the validators that has public key
	Validators []*Validator
	// The total weight of all the validators, including the ones that doesn't have a public key
	TotalWeight uint64
}

type Validator struct {
	PublicKey        *bls.PublicKey
	PublicKeyBytes   []byte
	CoronaPubKey   []byte // Post-quantum Corona public key
	Weight           uint64
	NodeIDs          []ids.NodeID
}

func (v *Validator) Compare(o *Validator) int {
	return bytes.Compare(v.PublicKeyBytes, o.PublicKeyBytes)
}

// GetCanonicalValidatorSetFromSubsubnetID returns the CanonicalValidatorSet of [subsubnetID] at
// [pChcainHeight]. The returned CanonicalValidatorSet includes the validator set in a canonical ordering
// and the total weight.
func GetCanonicalValidatorSetFromSubsubnetID(
	ctx context.Context,
	pChainState ValidatorState,
	pChainHeight uint64,
	subsubnetID ids.ID,
) (CanonicalValidatorSet, error) {
	// Get the validator set at the given height.
	vdrSet, err := pChainState.GetValidatorSet(ctx, pChainHeight, subsubnetID)
	if err != nil {
		return CanonicalValidatorSet{}, err
	}

	// Convert the validator set into the canonical ordering.
	return FlattenValidatorSet(vdrSet)
}

// FlattenValidatorSet converts the provided [vdrSet] into a canonical ordering.
// Also returns the total weight of the validator set.
func FlattenValidatorSet(vdrSet map[ids.NodeID]*ValidatorData) (CanonicalValidatorSet, error) {
	var (
		// Map public keys to validators to handle duplicates
		pkToValidator = make(map[string]*Validator)
		totalWeight   uint64
		err           error
	)
	for _, vdr := range vdrSet {
		totalWeight, err = math.Add(totalWeight, vdr.Weight)
		if err != nil {
			return CanonicalValidatorSet{}, fmt.Errorf("%w: %w", ErrWeightOverflow, err)
		}

		// Skip validators without public keys
		if len(vdr.PublicKey) == 0 {
			continue
		}

		// Convert []byte to *bls.PublicKey
		// Validator's public key is stored in uncompressed format (96 bytes)
		blsPK := bls.PublicKeyFromValidUncompressedBytes(vdr.PublicKey)
		if blsPK == nil {
			continue // Skip invalid public keys
		}

		// Use uncompressed bytes as the canonical key representation
		pkBytes := bls.PublicKeyToUncompressedBytes(blsPK)
		pkKey := string(pkBytes)

		// Check if we already have a validator with this public key
		if existingVdr, exists := pkToValidator[pkKey]; exists {
			// Merge validators with duplicate public keys
			existingVdr.Weight, err = math.Add(existingVdr.Weight, vdr.Weight)
			if err != nil {
				return CanonicalValidatorSet{}, fmt.Errorf("%w: %w", ErrWeightOverflow, err)
			}
			existingVdr.NodeIDs = append(existingVdr.NodeIDs, vdr.NodeID)
			// Note: For RT keys, first one wins on merge (should be same anyway)
		} else {
			// Create new validator with both BLS and Corona public keys
			newVdr := &Validator{
				PublicKey:      blsPK,
				PublicKeyBytes: pkBytes,
				CoronaPubKey: vdr.CoronaPubKey, // Post-quantum key
				Weight:         vdr.Weight,
				NodeIDs:        []ids.NodeID{vdr.NodeID},
			}
			pkToValidator[pkKey] = newVdr
		}
	}

	// Sort validators by public key
	vdrList := slices.Collect(maps.Values(pkToValidator))
	utils.Sort(vdrList)
	return CanonicalValidatorSet{Validators: vdrList, TotalWeight: totalWeight}, nil
}

// FilterValidators returns the validators in [vdrs] whose bit is set to 1 in
// [indices].
//
// Returns an error if [indices] references an unknown validator.
func FilterValidators(
	indices set.Bits,
	vdrs []*Validator,
) ([]*Validator, error) {
	// Verify that all alleged signers exist
	if indices.BitLen() > len(vdrs) {
		return nil, fmt.Errorf(
			"%w: NumIndices (%d) >= NumFilteredValidators (%d)",
			ErrUnknownValidator,
			indices.BitLen()-1, // -1 to convert from length to index
			len(vdrs),
		)
	}

	filteredVdrs := make([]*Validator, 0, len(vdrs))
	for i, vdr := range vdrs {
		if !indices.Contains(i) {
			continue
		}

		filteredVdrs = append(filteredVdrs, vdr)
	}
	return filteredVdrs, nil
}

// SumWeight returns the total weight of the provided validators.
func SumWeight(vdrs []*Validator) (uint64, error) {
	var (
		weight uint64
		err    error
	)
	for _, vdr := range vdrs {
		weight, err = math.Add(weight, vdr.Weight)
		if err != nil {
			return 0, fmt.Errorf("%w: %w", ErrWeightOverflow, err)
		}
	}
	return weight, nil
}

// AggregatePublicKeys returns the public key of the provided validators.
//
// Invariant: All of the public keys in [vdrs] are valid.
func AggregatePublicKeys(vdrs []*Validator) (*bls.PublicKey, error) {
	pks := make([]*bls.PublicKey, len(vdrs))
	for i, vdr := range vdrs {
		pks[i] = vdr.PublicKey
	}
	return bls.AggregatePublicKeys(pks)
}

// validatorStateAdapter adapts validators.State to ValidatorState
type validatorStateAdapter struct {
	state validators.State
}

func (v *validatorStateAdapter) GetValidatorSet(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*ValidatorData, error) {
	validatorSet, err := v.state.GetValidatorSet(ctx, height, subnetID)
	if err != nil {
		return nil, err
	}

	result := make(map[ids.NodeID]*ValidatorData, len(validatorSet))
	for nodeID, validator := range validatorSet {
		result[nodeID] = &ValidatorData{
			NodeID:         validator.NodeID,
			PublicKey:      validator.PublicKey,
			CoronaPubKey: validator.CoronaPubKey, // Post-quantum key
			Weight:         validator.Weight,
		}
	}
	return result, nil
}

// GetCanonicalValidatorSetFromChainID returns the canonical validator set given a validators.State, pChain height and a sourceChainID.
func GetCanonicalValidatorSetFromChainID(ctx context.Context,
	pChainState validators.State,
	pChainHeight uint64,
	sourceChainID ids.ID,
) (CanonicalValidatorSet, error) {
	// Adapt validators.State to ValidatorState
	adapter := &validatorStateAdapter{
		state: pChainState,
	}
	// In the new architecture, use sourceChainID as the subnet ID
	// This assumes a 1:1 mapping between chains and subnets
	return GetCanonicalValidatorSetFromSubsubnetID(ctx, adapter, pChainHeight, sourceChainID)
}

// cacheKey combines height and subnetID for cache lookups
type cacheKey struct {
	height   uint64
	subnetID ids.ID
}

// CachedValidatorState wraps ValidatorState with an LRU cache
type CachedValidatorState struct {
	state         ValidatorState
	upgradeConfig *upgrade.Config
	networkID     uint32
	cache         *lru.Cache[cacheKey, map[ids.NodeID]*ValidatorData]
	metrics       *cacheMetrics
}

type cacheMetrics struct {
	hits   metric.Counter
	misses metric.Counter
}

// NewCachedValidatorState creates a new cached validator state with Granite upgrade awareness
func NewCachedValidatorState(
	state ValidatorState,
	upgradeConfig *upgrade.Config,
	networkID uint32,
	registerer metric.Registerer,
) (*CachedValidatorState, error) {
	metrics := &cacheMetrics{
		hits: metric.NewCounter(
			"warp_validator_cache_hits",
			"number of validator set cache hits",
		),
		misses: metric.NewCounter(
			"warp_validator_cache_misses",
			"number of validator set cache misses",
		),
	}

	if err := registerer.Register(metrics.hits); err != nil {
		return nil, fmt.Errorf("failed to register cache hits metric: %w", err)
	}
	if err := registerer.Register(metrics.misses); err != nil {
		return nil, fmt.Errorf("failed to register cache misses metric: %w", err)
	}

	return &CachedValidatorState{
		state:         state,
		upgradeConfig: upgradeConfig,
		networkID:     networkID,
		cache:         lru.NewCache[cacheKey, map[ids.NodeID]*ValidatorData](8),
		metrics:       metrics,
	}, nil
}

// GetValidatorSet implements ValidatorState with caching for post-Granite queries
func (c *CachedValidatorState) GetValidatorSet(
	ctx context.Context,
	height uint64,
	subnetID ids.ID,
) (map[ids.NodeID]*ValidatorData, error) {
	// Check if Granite is activated - we only cache post-Granite
	// Use current time as approximation since we don't have block timestamp
	if c.upgradeConfig != nil && c.upgradeConfig.IsGraniteActivated(time.Now()) {
		key := cacheKey{height: height, subnetID: subnetID}
		if cached, ok := c.cache.Get(key); ok {
			c.metrics.hits.Inc()
			return cached, nil
		}
		c.metrics.misses.Inc()
	}

	// Cache miss or pre-Granite - fetch from underlying state
	vdrSet, err := c.state.GetValidatorSet(ctx, height, subnetID)
	if err != nil {
		return nil, err
	}

	// Cache the result if Granite is active
	if c.upgradeConfig != nil && c.upgradeConfig.IsGraniteActivated(time.Now()) {
		key := cacheKey{height: height, subnetID: subnetID}
		c.cache.Put(key, vdrSet)
	}

	return vdrSet, nil
}
