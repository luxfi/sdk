// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"fmt"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/rlp"
)

// MigrateAPI provides RPC methods for blockchain export/import
// Used by lux-cli for RPC-based migration control
type MigrateAPI struct {
	vm *VM
}

// NewMigrateAPI creates a new migrate API instance
func NewMigrateAPI(vm *VM) *MigrateAPI {
	return &MigrateAPI{vm: vm}
}

// BlockData represents a block with all its data for export
type BlockData struct {
	Number       uint64                `json:"number"`
	Hash         common.Hash           `json:"hash"`
	ParentHash   common.Hash           `json:"parentHash"`
	Header       string                `json:"header"`       // RLP-encoded header (hex)
	Body         string                `json:"body"`         // RLP-encoded body (hex)
	Receipts     string                `json:"receipts"`     // RLP-encoded receipts (hex)
	Transactions []*types.Transaction  `json:"transactions"` // Full transactions
}

// ChainInfo provides metadata about the blockchain
type ChainInfo struct {
	ChainID       uint64      `json:"chainId"`
	NetworkID     uint32      `json:"networkId"`
	GenesisHash   common.Hash `json:"genesisHash"`
	CurrentHeight uint64      `json:"currentHeight"`
	VMVersion     string      `json:"vmVersion"`
}

// GetChainInfo returns metadata about the blockchain
func (api *MigrateAPI) GetChainInfo() (*ChainInfo, error) {
	currentBlock := api.vm.blockChain.CurrentBlock()

	return &ChainInfo{
		ChainID:       api.vm.chainConfig.ChainID.Uint64(),
		NetworkID:     uint32(api.vm.chainCtx.NetworkID),
		GenesisHash:   api.vm.genesisHash,
		CurrentHeight: currentBlock.Number.Uint64(),
		VMVersion:     Version,
	}, nil
}

// StreamBlocks streams blocks in a range via RPC
// Returns blocks one at a time for efficient memory usage
func (api *MigrateAPI) StreamBlocks(ctx context.Context, start, end uint64) (chan *BlockData, chan error) {
	blockCh := make(chan *BlockData, 10)
	errCh := make(chan error, 1)

	go func() {
		defer close(blockCh)
		defer close(errCh)

		for height := start; height <= end; height++ {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			// Get block by number
			block := api.vm.blockChain.GetBlockByNumber(height)
			if block == nil {
				errCh <- fmt.Errorf("block %d not found", height)
				return
			}

			// Get receipts
			receipts := api.vm.blockChain.GetReceiptsByHash(block.Hash())

			// Encode header, body, receipts
			headerRLP, err := rlp.EncodeToBytes(block.Header())
			if err != nil {
				errCh <- fmt.Errorf("failed to encode header: %w", err)
				return
			}

			bodyRLP, err := rlp.EncodeToBytes(block.Body())
			if err != nil {
				errCh <- fmt.Errorf("failed to encode body: %w", err)
				return
			}

			receiptsRLP, err := rlp.EncodeToBytes(receipts)
			if err != nil {
				errCh <- fmt.Errorf("failed to encode receipts: %w", err)
				return
			}

			// Create block data
			blockData := &BlockData{
				Number:       height,
				Hash:         block.Hash(),
				ParentHash:   block.ParentHash(),
				Header:       common.Bytes2Hex(headerRLP),
				Body:         common.Bytes2Hex(bodyRLP),
				Receipts:     common.Bytes2Hex(receiptsRLP),
				Transactions: block.Transactions(),
			}

			blockCh <- blockData
		}
	}()

	return blockCh, errCh
}

// GetBlocks returns blocks in a range via RPC (batch mode)
// For smaller ranges, returns all blocks at once
func (api *MigrateAPI) GetBlocks(start, end uint64, limit int) ([]*BlockData, error) {
	if limit == 0 {
		limit = 100 // Default limit
	}

	if end-start+1 > uint64(limit) {
		return nil, fmt.Errorf("range too large, max %d blocks", limit)
	}

	blocks := make([]*BlockData, 0, end-start+1)
	for height := start; height <= end; height++ {
		block := api.vm.blockChain.GetBlockByNumber(height)
		if block == nil {
			return nil, fmt.Errorf("block %d not found", height)
		}

		receipts := api.vm.blockChain.GetReceiptsByHash(block.Hash())

		// Encode header, body, receipts
		headerRLP, err := rlp.EncodeToBytes(block.Header())
		if err != nil {
			return nil, fmt.Errorf("failed to encode header: %w", err)
		}

		bodyRLP, err := rlp.EncodeToBytes(block.Body())
		if err != nil {
			return nil, fmt.Errorf("failed to encode body: %w", err)
		}

		receiptsRLP, err := rlp.EncodeToBytes(receipts)
		if err != nil {
			return nil, fmt.Errorf("failed to encode receipts: %w", err)
		}

		blockData := &BlockData{
			Number:       height,
			Hash:         block.Hash(),
			ParentHash:   block.ParentHash(),
			Header:       common.Bytes2Hex(headerRLP),
			Body:         common.Bytes2Hex(bodyRLP),
			Receipts:     common.Bytes2Hex(receiptsRLP),
			Transactions: block.Transactions(),
		}

		blocks = append(blocks, blockData)
	}

	return blocks, nil
}

// ImportBlocks imports blocks via RPC
// Accepts RLP-encoded blocks and inserts them into the chain
func (api *MigrateAPI) ImportBlocks(blocks []*BlockData) (int, error) {
	imported := 0
	for _, blockData := range blocks {
		// Decode header
		headerBytes := common.Hex2Bytes(blockData.Header)
		var header types.Header
		if err := rlp.DecodeBytes(headerBytes, &header); err != nil {
			return imported, fmt.Errorf("failed to decode header for block %d: %w", blockData.Number, err)
		}

		// Decode body
		bodyBytes := common.Hex2Bytes(blockData.Body)
		var body types.Body
		if err := rlp.DecodeBytes(bodyBytes, &body); err != nil {
			return imported, fmt.Errorf("failed to decode body for block %d: %w", blockData.Number, err)
		}

		// Reconstruct block
		block := types.NewBlockWithHeader(&header).WithBody(body)

		// Skip genesis
		if block.NumberU64() == 0 {
			continue
		}

		// Check if block already exists
		if api.vm.blockChain.HasBlock(block.Hash(), block.NumberU64()) {
			continue
		}

		// Insert block
		if _, err := api.vm.blockChain.InsertChain([]*types.Block{block}); err != nil {
			return imported, fmt.Errorf("failed to import block %d: %w", blockData.Number, err)
		}

		imported++
	}

	return imported, nil
}
