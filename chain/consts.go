// Copyright (C) 2023-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"time"

	"github.com/luxdefi/node/utils/units"
)

const (
	FutureBound        = 10 * time.Second
	MaxWarpMessageSize = 256 * units.KiB
)
