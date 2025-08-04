// Copyright (C) 2023-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"github.com/luxfi/log"
	"github.com/luxfi/vms"

	"github.com/luxfi/sdk/examples/tokenvm/controller"
)

var _ vms.Factory = &Factory{}

type Factory struct{}

func (*Factory) New(logging.Logger) (interface{}, error) {
	return controller.New(), nil
}
