// Copyright (C) 2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"context"
	"os"
	"os/signal"
	"slices"
	"time"

	"github.com/luxfi/sdk/constants"
)

// Unique returns a new slice containing only the unique elements from the input slice.
func Unique[T comparable](arr []T) []T {
	visited := map[T]bool{}
	unique := []T{}
	for _, e := range arr {
		if !visited[e] {
			unique = append(unique, e)
			visited[e] = true
		}
	}
	return unique
}

func Belongs[T comparable](input []T, elem T) bool {
	return slices.Contains(input, elem)
}

func Map[T, U any](input []T, f func(T) U) []U {
	output := make([]U, 0, len(input))
	for _, e := range input {
		output = append(output, f(e))
	}
	return output
}

func Uint32Sort(arr []uint32) {
	slices.Sort(arr)
}

// GetAPIContext returns a context for API requests.
func GetAPIContext() (context.Context, context.CancelFunc) {
	return GetTimedContext(constants.APIRequestTimeout)
}

// GetAPILargeContext returns a context for API requests with large timeout.
func GetAPILargeContext() (context.Context, context.CancelFunc) {
	return GetTimedContext(constants.APIRequestLargeTimeout)
}

// GetTimedContext returns a context with the given timeout.
func GetTimedContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	parent, sigCancel := signal.NotifyContext(context.Background(), os.Interrupt)
	ctx, timeCancel := context.WithTimeout(parent, timeout)
	return ctx, func() {
		sigCancel()
		timeCancel()
	}
}

// PointersSlice converts a slice of values to a slice of pointers
func PointersSlice[T any](values []T) []*T {
	pointers := make([]*T, len(values))
	for i := range values {
		pointers[i] = &values[i]
	}
	return pointers
}
