// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"github.com/luxfi/evm/interfaces"
)

func init() {
	// Register the EVM plugin
	interfaces.RegisterPlugin("evm", &vmFactory{})
}

type vmFactory struct{}

func (f *vmFactory) New() (interface{}, error) {
	return &VM{}, nil
}