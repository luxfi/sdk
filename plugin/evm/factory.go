// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/node/utils/logging"
	"github.com/luxfi/node/vms"
)

var (
	// ID this VM should be referenced by
	IDStr = "subnetevm"
	ID    = ids.ID{'s', 'u', 'b', 'n', 'e', 't', 'e', 'v', 'm'}

	_ vms.Factory = (*Factory)(nil)
)

type Factory struct{}

func (*Factory) New(logging.Logger) (interface{}, error) {
	return &VM{}, nil
}

// ConsensusFactory wraps Factory to work with consensus package expectations
type ConsensusFactory struct {
	*Factory
}

// New creates a new VM instance with luxfi/log.Logger
func (f *ConsensusFactory) New(logger luxlog.Logger) (interface{}, error) {
	// Create a VM without initializing the logger
	// The logger will be set during Initialize
	return &VM{}, nil
}
