// (c) 2019-2024, Lux Industries, Inc.
// All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
	
	"github.com/luxfi/node/api/health"
	"github.com/luxfi/node/consensus"
	"github.com/luxfi/node/consensus/engine/core"
	"github.com/luxfi/node/consensus/engine/linear/block"
	"github.com/luxfi/node/consensus/linear"
	"github.com/luxfi/node/database"
	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/utils/hashing"
	"github.com/luxfi/node/version"
	"go.uber.org/zap"
)

// Use Version from version.go

var (
	_ core.VM                     = (*SimpleVM)(nil)
	_ core.AppHandler            = (*SimpleVM)(nil)
	_ health.Checker             = (*SimpleVM)(nil)
	_ block.ChainVM              = (*SimpleVM)(nil)
	
	// errNotImplemented declared in vm.go
)

// SimpleVM implements a minimal Ethereum Virtual Machine for the Lux network.
type SimpleVM struct {
	vmLock       sync.RWMutex
	ctx          *consensus.Context
	db           database.Database
	genesisBlock *SimpleBlock
	lastAccepted *SimpleBlock
	appSender    core.AppSender
	chainID      int64
}

// SimpleBlock implementation
type SimpleBlock struct {
	id       ids.ID
	parentID ids.ID
	height   uint64
	bytes    []byte
}

func (b *SimpleBlock) ID() ids.ID                 { return b.id }
func (b *SimpleBlock) Parent() ids.ID             { return b.parentID }
func (b *SimpleBlock) Bytes() []byte              { return b.bytes }
func (b *SimpleBlock) Height() uint64             { return b.height }
func (b *SimpleBlock) Timestamp() time.Time       { return time.Unix(0, 0) }
func (b *SimpleBlock) Verify(context.Context) error { return nil }
func (b *SimpleBlock) Accept(context.Context) error { return nil }
func (b *SimpleBlock) Reject(context.Context) error { return nil }

// Initialize implements the core.VM interface
func (vm *SimpleVM) Initialize(
	ctx context.Context,
	chainCtx *consensus.Context,
	db database.Database,
	genesisBytes []byte,
	upgradeBytes []byte,
	configBytes []byte,
	fxs []*core.Fx,
	appSender core.AppSender,
) error {
	vm.ctx = chainCtx
	vm.db = db
	vm.appSender = appSender
	
	// Try to parse chain ID from genesis
	vm.chainID = 1 // default
	var genesis map[string]interface{}
	if err := json.Unmarshal(genesisBytes, &genesis); err == nil {
		if config, ok := genesis["config"].(map[string]interface{}); ok {
			if chainID, ok := config["chainId"].(float64); ok {
				vm.chainID = int64(chainID)
			}
		}
	}
	
	// Create genesis block
	vm.genesisBlock = &SimpleBlock{
		id:       ids.GenerateTestID(),
		parentID: ids.Empty,
		height:   0,
		bytes:    genesisBytes,
	}
	vm.lastAccepted = vm.genesisBlock
	
	chainCtx.Log.Info("initialized Simple EVM",
		zap.Int64("chainID", vm.chainID),
	)
	
	return nil
}

// SetState implements the core.VM interface
func (vm *SimpleVM) SetState(ctx context.Context, state consensus.State) error {
	return nil
}

// Shutdown implements the core.VM interface
func (vm *SimpleVM) Shutdown(ctx context.Context) error {
	vm.vmLock.Lock()
	defer vm.vmLock.Unlock()
	
	return nil
}

// Version implements the core.VM interface
func (vm *SimpleVM) Version(context.Context) (string, error) {
	return Version, nil
}

// CreateHandlers implements the core.VM interface
func (vm *SimpleVM) CreateHandlers(context.Context) (map[string]http.Handler, error) {
	vm.vmLock.RLock()
	defer vm.vmLock.RUnlock()
	
	handlers := make(map[string]http.Handler)
	
	// Simple RPC handler
	handlers["/rpc"] = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := fmt.Sprintf(`{"jsonrpc":"2.0","result":{"message":"Simple EVM running","chainId":%d},"id":1}`, vm.chainID)
		w.Write([]byte(response))
	})
	
	return handlers, nil
}

// NewHTTPHandler implements the core.VM interface
func (vm *SimpleVM) NewHTTPHandler(ctx context.Context) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Simple EVM HTTP handler not implemented"))
	}), nil
}

// WaitForEvent implements the core.VM interface
func (vm *SimpleVM) WaitForEvent(ctx context.Context) (core.Message, error) {
	<-ctx.Done()
	return core.Message(0), ctx.Err()
}

// HealthCheck implements the health.Checker interface
func (vm *SimpleVM) HealthCheck(context.Context) (interface{}, error) {
	return map[string]string{"status": "ok"}, nil
}

// Connected implements the validators.Connector interface
func (vm *SimpleVM) Connected(ctx context.Context, nodeID ids.NodeID, nodeVersion *version.Application) error {
	return nil
}

// Disconnected implements the validators.Connector interface
func (vm *SimpleVM) Disconnected(ctx context.Context, nodeID ids.NodeID) error {
	return nil
}

// CrossChainAppRequest implements the core.AppHandler interface
func (vm *SimpleVM) CrossChainAppRequest(ctx context.Context, chainID ids.ID, requestID uint32, deadline time.Time, request []byte) error {
	return nil
}

// CrossChainAppRequestFailed implements the core.AppHandler interface
func (vm *SimpleVM) CrossChainAppRequestFailed(ctx context.Context, chainID ids.ID, requestID uint32, appErr *core.AppError) error {
	return nil
}

// CrossChainAppResponse implements the core.AppHandler interface
func (vm *SimpleVM) CrossChainAppResponse(ctx context.Context, chainID ids.ID, requestID uint32, response []byte) error {
	return nil
}

// AppRequest implements the core.AppHandler interface
func (vm *SimpleVM) AppRequest(ctx context.Context, nodeID ids.NodeID, requestID uint32, deadline time.Time, request []byte) error {
	return nil
}

// AppRequestFailed implements the core.AppHandler interface
func (vm *SimpleVM) AppRequestFailed(ctx context.Context, nodeID ids.NodeID, requestID uint32, appErr *core.AppError) error {
	return nil
}

// AppResponse implements the core.AppHandler interface
func (vm *SimpleVM) AppResponse(ctx context.Context, nodeID ids.NodeID, requestID uint32, response []byte) error {
	return nil
}

// AppGossip implements the core.AppHandler interface
func (vm *SimpleVM) AppGossip(ctx context.Context, nodeID ids.NodeID, msg []byte) error {
	return nil
}

// BuildBlock implements the block.ChainVM interface
func (vm *SimpleVM) BuildBlock(ctx context.Context) (linear.Block, error) {
	// For now, just return an error - no new blocks
	return nil, errNotImplemented
}

// SetPreference implements the block.ChainVM interface
func (vm *SimpleVM) SetPreference(ctx context.Context, blkID ids.ID) error {
	return nil
}

// LastAccepted implements the block.ChainVM interface
func (vm *SimpleVM) LastAccepted(context.Context) (ids.ID, error) {
	if vm.lastAccepted == nil {
		return ids.Empty, errNotImplemented
	}
	return vm.lastAccepted.ID(), nil
}

// GetBlockIDAtHeight implements the block.ChainVM interface
func (vm *SimpleVM) GetBlockIDAtHeight(ctx context.Context, height uint64) (ids.ID, error) {
	if height == 0 && vm.genesisBlock != nil {
		return vm.genesisBlock.ID(), nil
	}
	return ids.Empty, database.ErrNotFound
}

// GetBlock implements the block.ChainVM interface
func (vm *SimpleVM) GetBlock(ctx context.Context, blkID ids.ID) (linear.Block, error) {
	if vm.genesisBlock != nil && blkID == vm.genesisBlock.ID() {
		return vm.genesisBlock, nil
	}
	if vm.lastAccepted != nil && blkID == vm.lastAccepted.ID() {
		return vm.lastAccepted, nil
	}
	return nil, database.ErrNotFound
}

// ParseBlock implements the block.ChainVM interface
func (vm *SimpleVM) ParseBlock(ctx context.Context, blockBytes []byte) (linear.Block, error) {
	// For now, create a simple block
	blk := &SimpleBlock{
		id:       hashing.ComputeHash256Array(blockBytes),
		parentID: ids.Empty,
		height:   0,
		bytes:    blockBytes,
	}
	return blk, nil
}