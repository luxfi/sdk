// Copyright (C) 2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func retry[T any](
	fn func() (T, error),
	maxAttempts int,
	retryInterval time.Duration,
) (T, error) {
	const defaultRetryInterval = 2 * time.Second
	if retryInterval == 0 {
		retryInterval = defaultRetryInterval
	}
	var (
		result T
		cumErr error
	)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		var err error
		result, err = fn()
		if err == nil {
			return result, nil
		}
		cumErr = errors.Join(cumErr, err)
		time.Sleep(retryInterval)
	}
	return result, fmt.Errorf(
		"maximum retry attempts %d reached: cumulated err = %w",
		maxAttempts,
		cumErr,
	)
}

func retryWithContextGen[T any](
	ctxGen func() (context.Context, context.CancelFunc),
	fn func(context.Context) (T, error),
	maxAttempts int,
	retryInterval time.Duration,
) (T, error) {
	return retry(func() (T, error) {
		ctx, cancel := ctxGen()
		defer cancel()
		return fn(ctx)
	}, maxAttempts, retryInterval)
}
