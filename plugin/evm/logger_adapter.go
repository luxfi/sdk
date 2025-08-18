// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"io"
	
	"github.com/luxfi/consensus"
)

// loggerWriter wraps a consensus.Logger to implement io.Writer
type loggerWriter struct {
	logger interface{}
}

// Write implements io.Writer
func (w *loggerWriter) Write(p []byte) (n int, err error) {
	// Since the logger is an interface{} from GetLogger, we can't call methods on it
	// This is a placeholder that just returns success
	// In practice, the logger will be set differently during initialization
	return len(p), nil
}

// newLoggerWriter creates a new loggerWriter
func newLoggerWriter(logger interface{}) io.Writer {
	return &loggerWriter{logger: logger}
}