// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package consensus

import (
	"context"

	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
)

// Context keys for consensus-related values
type contextKey string

const (
	chainIDKey     contextKey = "chainID"
	networkIDKey   contextKey = "networkID"
	subnetIDKey    contextKey = "subnetID"
	warpSignerKey  contextKey = "warpSigner"
)

// GetChainID retrieves the chain ID from the context
func GetChainID(ctx context.Context) ids.ID {
	if v := ctx.Value(chainIDKey); v != nil {
		if chainID, ok := v.(ids.ID); ok {
			return chainID
		}
	}
	// Return empty ID if not found
	return ids.Empty
}

// GetNetworkID retrieves the network ID from the context
func GetNetworkID(ctx context.Context) uint32 {
	if v := ctx.Value(networkIDKey); v != nil {
		if networkID, ok := v.(uint32); ok {
			return networkID
		}
	}
	// Return default network ID
	return 1
}

// GetSubnetID retrieves the subnet ID from the context
func GetSubnetID(ctx context.Context) ids.ID {
	if v := ctx.Value(subnetIDKey); v != nil {
		if subnetID, ok := v.(ids.ID); ok {
			return subnetID
		}
	}
	// Return empty ID if not found
	return ids.Empty
}

// WarpSigner is an interface for signing warp messages
type WarpSigner interface {
	PublicKey() *bls.PublicKey
	Sign(msg []byte) (*bls.Signature, error)
	SignProofOfPossession(msg []byte) (*bls.Signature, error)
}

// GetWarpSigner retrieves the warp signer from the context
func GetWarpSigner(ctx context.Context) WarpSigner {
	if v := ctx.Value(warpSignerKey); v != nil {
		if signer, ok := v.(WarpSigner); ok {
			return signer
		}
	}
	// Return nil if not found
	return nil
}

// WithChainID adds a chain ID to the context
func WithChainID(ctx context.Context, chainID ids.ID) context.Context {
	return context.WithValue(ctx, chainIDKey, chainID)
}

// WithNetworkID adds a network ID to the context
func WithNetworkID(ctx context.Context, networkID uint32) context.Context {
	return context.WithValue(ctx, networkIDKey, networkID)
}

// WithSubnetID adds a subnet ID to the context
func WithSubnetID(ctx context.Context, subnetID ids.ID) context.Context {
	return context.WithValue(ctx, subnetIDKey, subnetID)
}

// WithWarpSigner adds a warp signer to the context
func WithWarpSigner(ctx context.Context, signer WarpSigner) context.Context {
	return context.WithValue(ctx, warpSignerKey, signer)
}

// WithValidatorState adds a validator state to the context
func WithValidatorState(ctx context.Context, state interface{}) context.Context {
	return context.WithValue(ctx, "validatorState", state)
}