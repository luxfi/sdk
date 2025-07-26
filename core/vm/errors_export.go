// (c) 2019-2024, Lux Industries, Inc.
// All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import "github.com/luxfi/evm/vmerrs"

// Re-export commonly used VM errors for external packages
var (
	ErrOutOfGas          = vmerrs.ErrOutOfGas
	ErrWriteProtection   = vmerrs.ErrWriteProtection
	ErrExecutionReverted = vmerrs.ErrExecutionReverted
)