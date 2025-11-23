// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"fmt"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/rlp"

	"github.com/luxfi/node/chainmigrate"
)

// Exporter implements chainmigrate.ChainExporter for EVM
type Exporter struct {
	vm     *VM
	config chainmigrate.ExporterConfig
}

// NewExporter creates a new EVM chain exporter
func NewExporter(vm *VM) *Exporter {
	return &Exporter{
		vm: vm,
	}
}

// Init initializes the exporter with configuration
func (e *Exporter) Init(config chainmigrate.ExporterConfig) error {
	e.config = config
	return nil
}

// GetChainInfo returns metadata about the EVM chain
func (e *Exporter) GetChainInfo() (*chainmigrate.ChainInfo, error) {
	currentBlock := e.vm.blockChain.CurrentBlock()

	return &chainmigrate.ChainInfo{
		ChainType:       chainmigrate.ChainTypeSubnetEVM,
		NetworkID:       uint64(e.vm.chainCtx.NetworkID),
		ChainID:         e.vm.chainConfig.ChainID,
		GenesisHash:     e.vm.genesisHash,
		CurrentHeight:   currentBlock.Number.Uint64(),
		TotalDifficulty: currentBlock.Difficulty, // Use block difficulty
		StateRoot:       currentBlock.Root,
		VMVersion:       Version,
		DatabaseType:    "pebbledb", // EVM uses PebbleDB
		IsPruned:        false,      // TODO: Check pruning config
		ArchiveMode:     true,       // TODO: Check archive mode
		HasWarpMessages: true,       // EVM supports Warp
	}, nil
}

// ExportBlocks exports blocks in a range (start to end inclusive)
func (e *Exporter) ExportBlocks(ctx context.Context, start, end uint64) (<-chan *chainmigrate.BlockData, <-chan error) {
	blockCh := make(chan *chainmigrate.BlockData, 100)
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
			block := e.vm.blockChain.GetBlockByNumber(height)
			if block == nil {
				errCh <- fmt.Errorf("block %d not found", height)
				return
			}

			// Get receipts
			receipts := e.vm.blockChain.GetReceiptsByHash(block.Hash())

			// Convert to BlockData
			blockData, err := e.convertBlock(block, receipts)
			if err != nil {
				errCh <- err
				return
			}

			blockCh <- blockData
		}
	}()

	return blockCh, errCh
}

// ExportState exports state at a specific block height
func (e *Exporter) ExportState(ctx context.Context, blockNumber uint64) (<-chan *chainmigrate.StateAccount, <-chan error) {
	accountCh := make(chan *chainmigrate.StateAccount, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(accountCh)
		defer close(errCh)

		// TODO: Implement state export
		// This requires iterating the state trie at the given block height
		errCh <- fmt.Errorf("state export not yet implemented")
	}()

	return accountCh, errCh
}

// ExportAccount exports a specific account at a block height
func (e *Exporter) ExportAccount(ctx context.Context, address common.Address, blockNumber uint64) (*chainmigrate.StateAccount, error) {
	// Get block to get state root
	block := e.vm.blockChain.GetBlockByNumber(blockNumber)
	if block == nil {
		return nil, fmt.Errorf("block %d not found", blockNumber)
	}

	// Get state at block
	state, err := e.vm.blockChain.StateAt(block.Root())
	if err != nil {
		return nil, err
	}

	// Get account data
	balance := state.GetBalance(address)
	return &chainmigrate.StateAccount{
		Address:     address,
		Nonce:       state.GetNonce(address),
		Balance:     balance.ToBig(), // Convert uint256 to *big.Int
		CodeHash:    common.BytesToHash(state.GetCodeHash(address).Bytes()),
		StorageRoot: state.GetStorageRoot(address),
		Code:        state.GetCode(address),
		Storage:     make(map[common.Hash]common.Hash), // TODO: Export storage slots
	}, nil
}

// ExportConfig exports chain configuration
func (e *Exporter) ExportConfig() (*chainmigrate.ChainConfig, error) {
	genesisBlock := e.vm.blockChain.GetBlockByNumber(0)
	if genesisBlock == nil {
		return nil, fmt.Errorf("genesis block not found")
	}

	return &chainmigrate.ChainConfig{
		NetworkID:           uint64(e.vm.chainCtx.NetworkID),
		ChainID:             e.vm.chainConfig.ChainID,
		HomesteadBlock:      e.vm.chainConfig.HomesteadBlock,
		EIP150Block:         e.vm.chainConfig.EIP150Block,
		EIP155Block:         e.vm.chainConfig.EIP155Block,
		EIP158Block:         e.vm.chainConfig.EIP158Block,
		ByzantiumBlock:      e.vm.chainConfig.ByzantiumBlock,
		ConstantinopleBlock: e.vm.chainConfig.ConstantinopleBlock,
		PetersburgBlock:     e.vm.chainConfig.PetersburgBlock,
		IstanbulBlock:       e.vm.chainConfig.IstanbulBlock,
		BerlinBlock:         e.vm.chainConfig.BerlinBlock,
		LondonBlock:         e.vm.chainConfig.LondonBlock,
		IsCoreth:            false,
		HasNetID:            true,
		NetID:               e.vm.chainCtx.NetID.String(),
		Precompiles:         make(map[common.Address]string), // TODO: Export precompile config
	}, nil
}

// VerifyExport verifies export integrity at a block height
func (e *Exporter) VerifyExport(blockNumber uint64) error {
	// Verify block exists
	block := e.vm.blockChain.GetBlockByNumber(blockNumber)
	if block == nil {
		return fmt.Errorf("block %d not found", blockNumber)
	}
	return nil
}

// Close closes the exporter
func (e *Exporter) Close() error {
	return nil
}

// convertBlock converts geth Block to chainmigrate.BlockData
func (e *Exporter) convertBlock(block *types.Block, receipts types.Receipts) (*chainmigrate.BlockData, error) {
	header := block.Header()

	// Convert transactions
	txs := make([]*chainmigrate.Transaction, 0, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		v, r, s := tx.RawSignatureValues()

		var receipt *chainmigrate.TransactionReceipt
		if i < len(receipts) {
			var contractAddr *common.Address
			if receipts[i].ContractAddress != (common.Address{}) {
				addr := receipts[i].ContractAddress
				contractAddr = &addr
			}
			receipt = &chainmigrate.TransactionReceipt{
				Status:            receipts[i].Status,
				CumulativeGasUsed: receipts[i].CumulativeGasUsed,
				Bloom:             receipts[i].Bloom,
				Logs:              receipts[i].Logs,
				TransactionHash:   tx.Hash(),
				ContractAddress:   contractAddr,
				GasUsed:           receipts[i].GasUsed,
			}
		}

		txs = append(txs, &chainmigrate.Transaction{
			Hash:        tx.Hash(),
			Nonce:       tx.Nonce(),
			To:          tx.To(),
			Value:       tx.Value(),
			Gas:         tx.Gas(),
			GasPrice:    tx.GasPrice(),
			Data:        tx.Data(),
			V:           v,
			R:           r,
			S:           s,
			GasTipCap:   tx.GasTipCap(),
			GasFeeCap:   tx.GasFeeCap(),
			AccessList:  tx.AccessList(),
			Receipt:     receipt,
		})
	}

	// Encode header and body
	headerRLP, err := rlp.EncodeToBytes(header)
	if err != nil {
		return nil, err
	}

	bodyRLP, err := rlp.EncodeToBytes(block.Body())
	if err != nil {
		return nil, err
	}

	// Encode receipts
	receiptsRLP, err := rlp.EncodeToBytes(receipts)
	if err != nil {
		return nil, err
	}

	return &chainmigrate.BlockData{
		Number:              header.Number.Uint64(),
		Hash:                block.Hash(),
		ParentHash:          header.ParentHash,
		Timestamp:           header.Time,
		StateRoot:           header.Root,
		ReceiptsRoot:        header.ReceiptHash,
		TransactionsRoot:    header.TxHash,
		GasLimit:            header.GasLimit,
		GasUsed:             header.GasUsed,
		Difficulty:          header.Difficulty,
		TotalDifficulty:     header.Difficulty, // Use block's own difficulty
		Coinbase:            header.Coinbase,
		Nonce:               header.Nonce,
		MixHash:             header.MixDigest,
		ExtraData:           header.Extra,
		BaseFee:             header.BaseFee,
		Header:              headerRLP,
		Body:                bodyRLP,
		Receipts:            receiptsRLP,
		Transactions:        txs,
		UncleHeaders:        [][]byte{}, // EVM doesn't have uncles
		Extensions:          make(map[string]interface{}),
	}, nil
}
