// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package executor

import (
	"context"
	"reflect"

	consContext "github.com/luxfi/consensus/context"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/sdk/node/codec"
	"github.com/luxfi/sdk/node/vms/components/verify"
	"github.com/luxfi/sdk/node/vms/exchangevm/config"
	"github.com/luxfi/sdk/node/vms/exchangevm/fxs"
)

type Backend struct {
	Ctx           context.Context
	LuxCtx        *consContext.Context // Lux consensus context
	Config        *config.Config
	Fxs           []*fxs.ParsedFx
	TypeToFxIndex map[reflect.Type]int
	Codec         codec.Manager
	// Note: FeeAssetID may be different than ctx.XAssetID if this XVM is
	// running in a subnet.
	FeeAssetID   ids.ID
	Bootstrapped bool

	// Chain IDs for cross-chain operations
	XChainID ids.ID
	CChainID ids.ID

	// Logger for this backend
	Log log.Logger

	// SharedMemory provides cross-chain atomic operations
	SharedMemory SharedMemory
}

// SharedMemory interface for cross-chain operations
type SharedMemory interface {
	Get(peerChainID ids.ID, keys [][]byte) ([][]byte, error)
	Apply(requests map[ids.ID]interface{}, batch ...interface{}) error
}

// ToChainContext creates a verify.ChainContext from this backend
func (b *Backend) ToChainContext() *verify.ChainContext {
	return &verify.ChainContext{
		ChainID:        b.LuxCtx.ChainID,
		NetID:          b.LuxCtx.ChainID,
		ValidatorState: &validatorStateAdapter{vs: b.LuxCtx.ValidatorState.(consContext.ValidatorState)},
	}
}

// validatorStateAdapter adapts consensusctx.ValidatorState to verify.ValidatorState
type validatorStateAdapter struct {
	vs consContext.ValidatorState
}

func (v *validatorStateAdapter) GetNetID(ctx context.Context, chainID ids.ID) (ids.ID, error) {
	return v.vs.GetNetworkID(chainID)
}
