// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/vms"
)

var (
	// ID this VM should be referenced by
	IDStr = "subnetevm"
	ID    = ids.ID{'s', 'u', 'b', 'n', 'e', 't', 'e', 'v', 'm'}

	_ vms.Factory = (*Factory)(nil)
)

type Factory struct{}

func (*Factory) New(log.Logger) (interface{}, error) {
	return &VM{}, nil
}
