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
// Copyright 2020 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package t8ntool

import (
	"encoding/json"
	"io"
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/tracing"
	"github.com/luxfi/geth/core/vm"
	"github.com/luxfi/log"
	"github.com/luxfi/evm/eth/tracers"
)

// traceWriter is a tracing.Hooks which also holds an inner tracer.
// When the TxEnd event happens, the inner tracer result is written to the file, and
// the file is closed.
type traceWriter struct {
	inner *tracing.Hooks
	f     io.WriteCloser
}

// TODO: Fix traceWriter to match new tracing.Hooks interface
// Compile-time interface check
// var _ = (*tracing.Hooks)((*traceWriter)(nil))

func (t *traceWriter) CaptureTxEnd(restGas uint64) {
	// TODO: Fix - CaptureTxEnd doesn't exist in new API
	// t.inner.CaptureTxEnd(restGas)
	defer t.f.Close()

	// TODO: Fix tracer interface check
	// if tracer, ok := t.inner.(tracers.Tracer); ok {
	//	result, err := tracer.GetResult()
	//	if err != nil {
	//		log.Warn("Error in tracer", "err", err)
	//		return
	//	}
	//	err = json.NewEncoder(t.f).Encode(result)
	//	if err != nil {
	//		log.Warn("Error writing tracer output", "err", err)
	//		return
	//	}
	// }
}

// func (t *traceWriter) CaptureTxStart(gasLimit uint64) { t.inner.CaptureTxStart(gasLimit) }
// func (t *traceWriter) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
//	t.inner.CaptureStart(env, from, to, create, input, gas, value)
// }

// func (t *traceWriter) CaptureEnd(output []byte, gasUsed uint64, err error) {
//	t.inner.CaptureEnd(output, gasUsed, err)
// }

// TODO: Fix these methods to match new tracing.Hooks interface
// func (t *traceWriter) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
//	t.inner.CaptureEnter(typ, from, to, input, gas, value)
// }

// func (t *traceWriter) CaptureExit(output []byte, gasUsed uint64, err error) {
//	t.inner.CaptureExit(output, gasUsed, err)
// }

// func (t *traceWriter) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
//	t.inner.CaptureState(pc, op, gas, cost, scope, rData, depth, err)
// }
// func (t *traceWriter) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
//	t.inner.CaptureFault(pc, op, gas, cost, scope, depth, err)
// }
