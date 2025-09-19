package main

import (
	"fmt"
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/rlp"
)

func main() {
	// Create the same header as in the test
	header := &types.Header{
		ParentHash:       common.Hash{1},
		UncleHash:        common.Hash{2},
		Coinbase:         common.Address{3},
		Root:             common.Hash{4},
		TxHash:           common.Hash{5},
		ReceiptHash:      common.Hash{6},
		Bloom:            types.Bloom{7},
		Difficulty:       big.NewInt(8),
		Number:           big.NewInt(9),
		GasLimit:         10,
		GasUsed:          11,
		Time:             12,
		Extra:            []byte{13},
		MixDigest:        common.Hash{14},
		Nonce:            types.BlockNonce{15},
		BaseFee:          big.NewInt(16),
		WithdrawalsHash:  &common.Hash{17},
		BlobGasUsed:      ptrTo(uint64(18)),
		ExcessBlobGas:    ptrTo(uint64(19)),
		ParentBeaconRoot: &common.Hash{20},
	}

	fmt.Printf("Hash with WithdrawalsHash: %s\n", header.Hash().Hex())

	// Now set WithdrawalsHash to nil like the test expects
	header.WithdrawalsHash = nil
	fmt.Printf("Hash with WithdrawalsHash=nil: %s\n", header.Hash().Hex())

	// Also clear the other newer fields
	header.BlobGasUsed = nil
	header.ExcessBlobGas = nil
	header.ParentBeaconRoot = nil
	header.RequestsHash = nil
	fmt.Printf("Hash with all newer fields=nil: %s\n", header.Hash().Hex())

	// Test RLP encoding
	rlpBytes, _ := rlp.EncodeToBytes(header)
	fmt.Printf("RLP with all newer fields=nil: %x\n", rlpBytes)
}

func ptrTo[T any](x T) *T { return &x }
