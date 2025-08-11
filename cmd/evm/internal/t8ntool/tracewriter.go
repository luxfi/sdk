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
	"log"
	"reflect"

	"github.com/luxfi/geth/core/tracing"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/evm/eth/tracers"
)

// traceWriter wraps a tracer with file writing capabilities.
// When the transaction ends, the tracer result is written to the file.
type traceWriter struct {
	inner  *tracing.Hooks
	tracer tracers.Tracer
	f      io.WriteCloser
}

// newTraceWriter creates a new trace writer that will output to the given file
func newTraceWriter(hooks *tracing.Hooks, tracer tracers.Tracer, f io.WriteCloser) *traceWriter {
	tw := &traceWriter{
		inner:  hooks,
		tracer: tracer,
		f:      f,
	}
	
	// Wrap the TxEnd hook to write output when transaction completes
	originalTxEnd := hooks.OnTxEnd
	hooks.OnTxEnd = func(receipt *types.Receipt, err error) {
		if originalTxEnd != nil {
			originalTxEnd(receipt, err)
		}
		tw.writeResult()
	}
	
	return tw
}

// writeResult writes the tracer result to the file and closes it
func (t *traceWriter) writeResult() {
	defer t.f.Close()

	// Check if tracer was set using reflection
	if !reflect.ValueOf(t.tracer).IsNil() {
		result, err := t.tracer.GetResult()
		if err != nil {
			log.Printf("Error in tracer: %v", err)
			return
		}
		err = json.NewEncoder(t.f).Encode(result)
		if err != nil {
			log.Printf("Error writing tracer output: %v", err)
			return
		}
	}
}

// Hooks returns the wrapped tracing hooks
func (t *traceWriter) Hooks() *tracing.Hooks {
	return t.inner
}
