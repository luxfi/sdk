// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package health

import "time"

// Result describes a single health check outcome.
type Result struct {
	// Details of the HealthCheck.
	Details interface{} `json:"message,omitempty"`

	// Error is the string representation of the error returned by the failing
	// HealthCheck. The value is nil if the check passed.
	Error *string `json:"error,omitempty"`

	// Timestamp of the last HealthCheck.
	Timestamp time.Time `json:"timestamp,omitempty"`

	// Duration is the amount of time this HealthCheck last took to evaluate.
	Duration time.Duration `json:"duration"`

	// ContiguousFailures the HealthCheck has returned.
	ContiguousFailures int64 `json:"contiguousFailures,omitempty"`

	// TimeOfFirstFailure of the HealthCheck,
	TimeOfFirstFailure *time.Time `json:"timeOfFirstFailure,omitempty"`
}

// APIReply is the response for Readiness, Health, and Liveness.
type APIReply struct {
	Checks  map[string]Result `json:"checks"`
	Healthy bool              `json:"healthy"`
}

// APIArgs is the arguments for Readiness, Health, and Liveness.
type APIArgs struct {
	Tags []string `json:"tags"`
}
