// (c) 2019-2024, Lux Industries, Inc.
// All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"github.com/luxfi/node/utils/logging"
	"github.com/luxfi/node/vms"
	plugin "github.com/luxfi/evm/plugin/evm"
)

var _ vms.Factory = (*Factory)(nil)

// Factory is a factory for creating EVM VMs
type Factory struct{}

// New creates a new VM instance
func (f *Factory) New(log logging.Logger) (interface{}, error) {
	return &plugin.VM{}, nil
}