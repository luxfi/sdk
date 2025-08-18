// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"github.com/luxfi/consensus/core"
	nodecore "github.com/luxfi/node/consensus/engine/core"
)

// AdaptAppError converts from consensus AppError to node AppError
func AdaptAppError(err *core.AppError) *nodecore.AppError {
	if err == nil {
		return nil
	}
	return &nodecore.AppError{
		Code:    err.Code,
		Message: err.Message,
	}
}

// AdaptNodeAppError converts from node AppError to consensus AppError
func AdaptNodeAppError(err *nodecore.AppError) *core.AppError {
	if err == nil {
		return nil
	}
	return &core.AppError{
		Code:    err.Code,
		Message: err.Message,
	}
}