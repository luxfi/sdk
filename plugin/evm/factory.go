// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
)

var (
	// ID this VM should be referenced by
	IDStr = "subnetevm"
	ID    = interfaces.ID{'s', 'u', 'b', 'n', 'e', 't', 'e', 'v', 'm'}

	_ interfaces.Factory = &Factory{}
)

type Factory struct{}

func (*Factory) New(logging.Logger) (interface{}, error) {
	return &VM{}, nil
}
