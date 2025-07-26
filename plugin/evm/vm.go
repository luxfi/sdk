// (c) 2019-2024, Lux Industries, Inc.
// All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	
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
	
	"github.com/luxfi/evm/consensus/dummy"
	evmcore "github.com/luxfi/evm/core"
	"github.com/luxfi/evm/core/rawdb"
	"github.com/luxfi/evm/core/txpool"
	"github.com/luxfi/evm/core/vm"
	"github.com/luxfi/evm/eth"
	"github.com/luxfi/evm/eth/ethconfig"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/evm/utils"
)

// Use Version from version.go

var (
	_ core.VM                     = (*VM)(nil)
	_ core.AppHandler            = (*VM)(nil)
	_ health.Checker             = (*VM)(nil)
	_ block.ChainVM              = (*VM)(nil)
	
	errNotImplemented = errors.New("not implemented")
)

// VM implements the Ethereum Virtual Machine for the Lux network.
type VM struct {
	vmLock       sync.RWMutex
	ctx          *consensus.Context
	db           database.Database
	chaindb      ethdb.Database
	genesisBlock *Block
	lastAccepted *Block
	appSender    core.AppSender
	
	// Core Ethereum components
	eth        *eth.Ethereum
	txPool     *txpool.TxPool
	blockChain *evmcore.BlockChain
	
	// Configuration
	chainConfig *params.ChainConfig
	ethConfig   ethconfig.Config
}

// Simple block implementation
type Block struct {
	id       ids.ID
	parentID ids.ID
	height   uint64
	bytes    []byte
}

func (b *Block) ID() ids.ID                 { return b.id }
func (b *Block) Parent() ids.ID             { return b.parentID }
func (b *Block) Bytes() []byte              { return b.bytes }
func (b *Block) Height() uint64             { return b.height }
func (b *Block) Timestamp() time.Time       { return time.Unix(0, 0) }
func (b *Block) Verify(context.Context) error { return nil }
func (b *Block) Accept(context.Context) error { return nil }
func (b *Block) Reject(context.Context) error { return nil }

// Initialize implements the core.VM interface
func (evm *VM) Initialize(
	ctx context.Context,
	chainCtx *consensus.Context,
	db database.Database,
	genesisBytes []byte,
	upgradeBytes []byte,
	configBytes []byte,
	fxs []*core.Fx,
	appSender core.AppSender,
) error {
	evm.ctx = chainCtx
	evm.db = db
	evm.appSender = appSender
	
	// Parse genesis
	var genesisConfig evmcore.Genesis
	if err := json.Unmarshal(genesisBytes, &genesisConfig); err != nil {
		// If parsing fails, create a default genesis
		chainCtx.Log.Warn("failed to parse genesis bytes, using default", zap.Error(err))
		genesisConfig = evmcore.Genesis{
			Config:     params.EVMDefaultChainConfig,
			Nonce:      0,
			Timestamp:  0,
			ExtraData:  nil,
			GasLimit:   params.GenesisGasLimit,
			Difficulty: common.Big0,
			Mixhash:    common.Hash{},
			Coinbase:   common.Address{},
			Alloc:      make(evmcore.GenesisAlloc),
		}
	}
	
	// Set chain config
	evm.chainConfig = genesisConfig.Config
	if evm.chainConfig == nil {
		evm.chainConfig = params.EVMDefaultChainConfig
	}
	
	// Create in-memory database for now
	evm.chaindb = rawdb.NewMemoryDatabase()
	
	// Initialize Ethereum config
	evm.ethConfig = ethconfig.NewDefaultConfig()
	evm.ethConfig.Genesis = &genesisConfig
	evm.ethConfig.NetworkId = uint64(evm.chainConfig.ChainID.Int64())
	
	
	// For now, skip full Ethereum backend initialization
	// TODO: Properly initialize Ethereum backend with all required parameters
	// This requires setting up:
	// - PushGossiper for transaction gossip
	// - Settings for various configurations  
	// - Consensus engine (dummy for subnet)
	// - Clock for time management
	
	// Initialize a basic blockchain directly
	// Create a simple clock
	clock := utils.NewMockableClock()
	engine := dummy.NewDummyEngine(dummy.Mode{}, clock)
	cacheConfig := &evmcore.CacheConfig{
		TrieCleanLimit: 256,
		TrieDirtyLimit: 256,
		SnapshotLimit:  0, // Disable snapshots for now
	}
	vmConfig := vm.Config{}
	
	var err error
	evm.blockChain, err = evmcore.NewBlockChain(
		evm.chaindb,
		cacheConfig,
		&genesisConfig,
		engine,
		vmConfig,
		common.Hash{}, // lastAcceptedHash - empty for new chain
		false,         // skipChainConfigCheckCompatible
	)
	if err != nil {
		return fmt.Errorf("failed to create blockchain: %w", err)
	}
	
	// Create genesis block wrapper
	genesisHeader := evm.blockChain.Genesis().Header()
	evm.genesisBlock = &Block{
		id:       ids.ID(genesisHeader.Hash()),
		parentID: ids.Empty,
		height:   0,
		bytes:    genesisBytes,
	}
	evm.lastAccepted = evm.genesisBlock
	
	chainCtx.Log.Info("initialized EVM",
		zap.Any("chainID", evm.chainConfig.ChainID),
		zap.Uint64("networkID", evm.ethConfig.NetworkId),
		zap.String("genesisHash", genesisHeader.Hash().Hex()),
	)
	
	return nil
}

// SetState implements the core.VM interface
func (vm *VM) SetState(ctx context.Context, state consensus.State) error {
	return nil
}

// Shutdown implements the core.VM interface
func (vm *VM) Shutdown(ctx context.Context) error {
	vm.vmLock.Lock()
	defer vm.vmLock.Unlock()
	
	if vm.eth != nil {
		vm.eth.Stop()
	}
	
	if vm.blockChain != nil {
		vm.blockChain.Stop()
	}
	
	if vm.chaindb != nil {
		vm.chaindb.Close()
	}
	
	return nil
}

// Version implements the core.VM interface
func (vm *VM) Version(context.Context) (string, error) {
	return Version, nil
}

// CreateHandlers implements the core.VM interface
func (vm *VM) CreateHandlers(context.Context) (map[string]http.Handler, error) {
	vm.vmLock.RLock()
	defer vm.vmLock.RUnlock()
	
	handlers := make(map[string]http.Handler)
	
	// For now, return a simple handler that indicates EVM is running
	// TODO: Integrate full Ethereum RPC server
	handlers["/rpc"] = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chainID := int64(1) // default
		if vm.chainConfig != nil && vm.chainConfig.ChainID != nil {
			chainID = vm.chainConfig.ChainID.Int64()
		}
		response := fmt.Sprintf(`{"jsonrpc":"2.0","result":{"message":"EVM running","chainId":%d},"id":1}`, chainID)
		w.Write([]byte(response))
	})
	
	return handlers, nil
}

// NewHTTPHandler implements the core.VM interface
func (vm *VM) NewHTTPHandler(ctx context.Context) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("C-Chain HTTP handler not implemented"))
	}), nil
}

// WaitForEvent implements the core.VM interface
func (vm *VM) WaitForEvent(ctx context.Context) (core.Message, error) {
	<-ctx.Done()
	return core.Message(0), ctx.Err()
}

// HealthCheck implements the health.Checker interface
func (vm *VM) HealthCheck(context.Context) (interface{}, error) {
	return map[string]string{"status": "ok"}, nil
}

// Connected implements the validators.Connector interface
func (vm *VM) Connected(ctx context.Context, nodeID ids.NodeID, nodeVersion *version.Application) error {
	return nil
}

// Disconnected implements the validators.Connector interface
func (vm *VM) Disconnected(ctx context.Context, nodeID ids.NodeID) error {
	return nil
}

// CrossChainAppRequest implements the core.AppHandler interface
func (vm *VM) CrossChainAppRequest(ctx context.Context, chainID ids.ID, requestID uint32, deadline time.Time, request []byte) error {
	return nil
}

// CrossChainAppRequestFailed implements the core.AppHandler interface
func (vm *VM) CrossChainAppRequestFailed(ctx context.Context, chainID ids.ID, requestID uint32, appErr *core.AppError) error {
	return nil
}

// CrossChainAppResponse implements the core.AppHandler interface
func (vm *VM) CrossChainAppResponse(ctx context.Context, chainID ids.ID, requestID uint32, response []byte) error {
	return nil
}

// AppRequest implements the core.AppHandler interface
func (vm *VM) AppRequest(ctx context.Context, nodeID ids.NodeID, requestID uint32, deadline time.Time, request []byte) error {
	return nil
}

// AppRequestFailed implements the core.AppHandler interface
func (vm *VM) AppRequestFailed(ctx context.Context, nodeID ids.NodeID, requestID uint32, appErr *core.AppError) error {
	return nil
}

// AppResponse implements the core.AppHandler interface
func (vm *VM) AppResponse(ctx context.Context, nodeID ids.NodeID, requestID uint32, response []byte) error {
	return nil
}

// AppGossip implements the core.AppHandler interface
func (vm *VM) AppGossip(ctx context.Context, nodeID ids.NodeID, msg []byte) error {
	return nil
}

// BuildBlock implements the block.ChainVM interface
func (vm *VM) BuildBlock(ctx context.Context) (linear.Block, error) {
	// For now, just return an error - no new blocks
	return nil, errNotImplemented
}

// SetPreference implements the block.ChainVM interface
func (vm *VM) SetPreference(ctx context.Context, blkID ids.ID) error {
	return nil
}

// LastAccepted implements the block.ChainVM interface
func (vm *VM) LastAccepted(context.Context) (ids.ID, error) {
	if vm.lastAccepted == nil {
		return ids.Empty, errNotImplemented
	}
	return vm.lastAccepted.ID(), nil
}

// GetBlockIDAtHeight implements the block.ChainVM interface
func (vm *VM) GetBlockIDAtHeight(ctx context.Context, height uint64) (ids.ID, error) {
	if height == 0 && vm.genesisBlock != nil {
		return vm.genesisBlock.ID(), nil
	}
	return ids.Empty, database.ErrNotFound
}

// GetBlock implements the block.ChainVM interface
func (vm *VM) GetBlock(ctx context.Context, blkID ids.ID) (linear.Block, error) {
	if vm.genesisBlock != nil && blkID == vm.genesisBlock.ID() {
		return vm.genesisBlock, nil
	}
	if vm.lastAccepted != nil && blkID == vm.lastAccepted.ID() {
		return vm.lastAccepted, nil
	}
	return nil, database.ErrNotFound
}

// ParseBlock implements the block.ChainVM interface
func (vm *VM) ParseBlock(ctx context.Context, blockBytes []byte) (linear.Block, error) {
	// For now, create a simple block
	blk := &Block{
		id:       hashing.ComputeHash256Array(blockBytes),
		parentID: ids.Empty,
		height:   0,
		bytes:    blockBytes,
	}
	return blk, nil
}