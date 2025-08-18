// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
//
// This file is a derived work, based on the go-ethereum library whose original
// notices appear below.
//
// It is distributed under a license compatible with the licensing terms of the
// original code from which it is derived.
//
// Much love to the original authors for their work.
// **********
// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package tracers

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/core/vm"
	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/eth/tracers/logger"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/evm/plugin/evm/customrawdb"
	"github.com/luxfi/evm/tests"
)

func BenchmarkPrestateTracer(b *testing.B) {
	for _, scheme := range []string{rawdb.HashScheme, customrawdb.FirewoodScheme} {
		b.Run(scheme, func(b *testing.B) {
			benchmarkTransactionTrace(b, scheme)
		})
	}
}

func benchmarkTransactionTrace(b *testing.B, scheme string) {
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	from := func() common.Address {
		cryptoAddr := crypto.PubkeyToAddress(key.PublicKey)
		var commonAddr common.Address
		copy(commonAddr[:], cryptoAddr[:])
		return commonAddr
	}()
	gas := uint64(1000000) // 1M gas
	to := common.HexToAddress("0x00000000000000000000000000000000deadbeef")
	signer := types.LatestSignerForChainID(big.NewInt(1337))
	tx, err := types.SignNewTx(key, signer,
		&types.LegacyTx{
			Nonce:    1,
			GasPrice: big.NewInt(500),
			Gas:      gas,
			To:       &to,
		})
	if err != nil {
		b.Fatal(err)
	}
	txContext := vm.TxContext{
		Origin:   from,
		GasPrice: tx.GasPrice(),
	}
	context := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    common.Address{},
		BlockNumber: new(big.Int).SetUint64(uint64(5)),
		Time:        5,
		Difficulty:  big.NewInt(0xffffffff),
		GasLimit:    gas,
		BaseFee:     big.NewInt(8),
	}
	alloc := types.GenesisAlloc{}
	// The code pushes 'deadbeef' into memory, then the other params, and calls CREATE2, then returns
	// the address
	loop := []byte{
		byte(vm.JUMPDEST), //  [ count ]
		byte(vm.PUSH1), 0, // jumpdestination
		byte(vm.JUMP),
	}
	alloc[common.HexToAddress("0x00000000000000000000000000000000deadbeef")] = types.Account{
		Nonce:   1,
		Code:    loop,
		Balance: big.NewInt(1),
	}
	alloc[from] = types.Account{
		Nonce:   1,
		Code:    []byte{},
		Balance: big.NewInt(500000000000000),
	}
	state := tests.MakePreState(rawdb.NewMemoryDatabase(), alloc, false, scheme)
	defer state.Close()

	// Create the tracer, the EVM environment and run it
	tracer := logger.NewStructLogger(&logger.Config{
		// Debug field no longer exists in Config
		//DisableStorage: true,
		//EnableMemory: false,
		//EnableReturnData: false,
	})
	// vm.NewEVM signature changed - no longer takes txContext separately
	// Tracer field now expects *tracing.Hooks, use tracer.Hooks()
	evm := vm.NewEVM(context, state.StateDB, params.TestChainConfig, vm.Config{Tracer: tracer.Hooks()})
	evm.SetTxContext(txContext)
	msg, err := core.TransactionToMessage(tx, signer, context.BaseFee)
	if err != nil {
		b.Fatalf("failed to prepare transaction for tracing: %v", err)
	}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		snap := state.StateDB.Snapshot()
		st := core.NewStateTransition(evm, msg, new(core.GasPool).AddGas(tx.Gas()))
		_, err = st.TransitionDb()
		if err != nil {
			b.Fatal(err)
		}
		state.StateDB.RevertToSnapshot(snap)
		// StructLogger no longer has StructLogs() method, use GetResult() instead
		result, _ := tracer.GetResult()
		var execResult logger.ExecutionResult
		if err := json.Unmarshal(result, &execResult); err == nil {
			if have, want := len(execResult.StructLogs), 244752; have != want {
				b.Fatalf("trace wrong, want %d steps, have %d", want, have)
			}
		}
		// Reset method no longer exists, create a new tracer instead
		tracer = logger.NewStructLogger(&logger.Config{})
	}
}

// GetMemoryCopyPadded returns offset + size as a new slice.
// It zero-pads the slice if it extends beyond memory bounds.
func GetMemoryCopyPadded(mem *vm.Memory, offset, size int64) ([]byte, error) {
	if offset < 0 || size < 0 {
		return nil, errors.New("offset or size must not be negative")
	}
	m := mem.Data()
	length := int64(len(m))
	if offset+size < length { // slice fully inside memory
		return m[offset : offset+size], nil
	}
	const memoryPadLimit = 1024 * 1024
	paddingNeeded := offset + size - length
	if paddingNeeded > memoryPadLimit {
		return nil, fmt.Errorf("reached limit for padding memory slice: %d", paddingNeeded)
	}
	cpy := make([]byte, size)
	if overlap := length - offset; overlap > 0 {
		copy(cpy, m[offset:])
	}
	return cpy, nil
}

func TestMemCopying(t *testing.T) {
	for i, tc := range []struct {
		memsize  int64
		offset   int64
		size     int64
		wantErr  string
		wantSize int
	}{
		{0, 0, 100, "", 100},    // Should pad up to 100
		{0, 100, 0, "", 0},      // No need to pad (0 size)
		{100, 50, 100, "", 100}, // Should pad 100-150
		{100, 50, 5, "", 5},     // Wanted range fully within memory
		{100, -50, 0, "offset or size must not be negative", 0},                        // Error
		{0, 1, 1024*1024 + 1, "reached limit for padding memory slice: 1048578", 0},    // Error
		{10, 0, 1024*1024 + 100, "reached limit for padding memory slice: 1048666", 0}, // Error

	} {
		mem := vm.NewMemory()
		mem.Resize(uint64(tc.memsize))
		cpy, err := GetMemoryCopyPadded(mem, tc.offset, tc.size)
		if want := tc.wantErr; want != "" {
			if err == nil {
				t.Fatalf("test %d: want '%v' have no error", i, want)
			}
			if have := err.Error(); want != have {
				t.Fatalf("test %d: want '%v' have '%v'", i, want, have)
			}
			continue
		}
		if err != nil {
			t.Fatalf("test %d: unexpected error: %v", i, err)
		}
		if want, have := tc.wantSize, len(cpy); have != want {
			t.Fatalf("test %d: want %v have %v", i, want, have)
		}
	}
}
