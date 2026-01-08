// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package timer

import (
	"encoding/binary"
	"time"
)

// ProgressFromHash returns the progress out of MaxUint64 assuming [b] is a key
// in a uniformly distributed sequence that is being iterated lexicographically.
func ProgressFromHash(b []byte) uint64 {
	// binary.BigEndian.Uint64 will panic if the input length is less than 8, so
	// pad 0s as needed.
	var progress [8]byte
	copy(progress[:], b)
	return binary.BigEndian.Uint64(progress[:])
}

// EstimateETA attempts to estimate the remaining time for a job to finish given
// the [startTime] and it's current progress.
func EstimateETA(startTime time.Time, progress, end uint64) time.Duration {
	timeSpent := time.Since(startTime)

	percentExecuted := float64(progress) / float64(end)
	estimatedTotalDuration := time.Duration(float64(timeSpent) / percentExecuted)
	eta := estimatedTotalDuration - timeSpent
	return eta.Round(time.Second)
}

// EtaTracker provides exponentially weighted moving average ETA estimates
type EtaTracker struct {
	startTime      time.Time
	samples        int
	alpha          float64
	sampleCount    int
	lastProgress   uint64
	lastSampleTime time.Time
}

// NewEtaTracker creates a new ETA tracker with the given number of samples and alpha
func NewEtaTracker(samples int, alpha float64) *EtaTracker {
	return &EtaTracker{
		startTime: time.Now(),
		samples:   samples,
		alpha:     alpha,
	}
}

// Update updates the ETA tracker with new progress
func (e *EtaTracker) Update(progress, total uint64) {
	// Simple implementation - just track the time
}

// AddSample adds a sample to the ETA tracker and returns ETA pointer and progress percentage
func (e *EtaTracker) AddSample(progress, total uint64, sampleTime time.Time) (*time.Duration, float64) {
	if total == 0 {
		return nil, 0
	}

	// Initialize start time on first sample
	if e.sampleCount == 0 {
		e.startTime = sampleTime
		e.lastProgress = progress
		e.lastSampleTime = sampleTime
	}

	// Update sample count
	e.sampleCount++

	// Check if we have enough samples
	if e.sampleCount < e.samples {
		e.lastProgress = progress
		e.lastSampleTime = sampleTime
		return nil, 0
	}

	// Check if time went backwards
	if sampleTime.Before(e.lastSampleTime) {
		return nil, 0
	}

	// Check if progress didn't advance
	if progress <= e.lastProgress {
		return nil, 0
	}

	// If we're at or past the end, return 0 ETA immediately
	if progress >= total {
		e.lastProgress = progress
		e.lastSampleTime = sampleTime
		eta := time.Duration(0)
		return &eta, 100.0
	}

	// Calculate ETA based on recent rate (since last sample), not overall rate
	// This allows us to detect acceleration/deceleration
	timeSinceLastSample := sampleTime.Sub(e.lastSampleTime)
	progressSinceLastSample := progress - e.lastProgress

	if timeSinceLastSample > 0 && progressSinceLastSample > 0 {
		// Calculate rate: progress per unit time
		rate := float64(progressSinceLastSample) / timeSinceLastSample.Seconds()
		remaining := total - progress
		etaSeconds := float64(remaining) / rate
		eta := time.Duration(etaSeconds * float64(time.Second)).Round(time.Second)

		e.lastProgress = progress
		e.lastSampleTime = sampleTime

		progressPercent := (float64(progress) / float64(total)) * 100

		return &eta, progressPercent
	}

	e.lastProgress = progress
	e.lastSampleTime = sampleTime
	return nil, 0
}

// ETA returns the estimated time remaining
func (e *EtaTracker) ETA(progress, total uint64) time.Duration {
	return EstimateETA(e.startTime, progress, total)
}
