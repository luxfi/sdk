// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"time"

	"github.com/luxfi/constants"
)

func apiContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

func apiLargeContext() (context.Context, context.CancelFunc) {
	return apiContext(constants.APIRequestLargeTimeout)
}
