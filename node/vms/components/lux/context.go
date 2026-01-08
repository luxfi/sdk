// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package lux

import "context"

// ContextInitializable can be initialized with a context
type ContextInitializable interface {
	InitCtx(context.Context)
}
