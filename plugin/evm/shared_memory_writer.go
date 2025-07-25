// (c) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"github.com/luxfi/evm/precompile/precompileconfig"
	"github.com/luxfi/node/vms/platformvm/warp"
)

// sharedMemoryWriter implements precompileconfig.WarpMessageWriter
type sharedMemoryWriter struct {
	messages []*warp.UnsignedMessage
}

// NewSharedMemoryWriter creates a new shared memory writer
func NewSharedMemoryWriter() precompileconfig.WarpMessageWriter {
	return &sharedMemoryWriter{
		messages: make([]*warp.UnsignedMessage, 0),
	}
}

// AddMessage implements precompileconfig.WarpMessageWriter
func (s *sharedMemoryWriter) AddMessage(msg *warp.UnsignedMessage) error {
	s.messages = append(s.messages, msg)
	return nil
}

// GetMessages returns all messages that were written
func (s *sharedMemoryWriter) GetMessages() []*warp.UnsignedMessage {
	return s.messages
}