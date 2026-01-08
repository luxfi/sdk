// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import "time"

// MainnetParameters contains mainnet consensus parameters
var MainnetParameters = struct {
	K                     int
	AlphaPreference       int
	AlphaConfidence       int
	Beta                  int
	ConcurrentPolls       int
	OptimalProcessing     int
	MaxOutstandingItems   int
	MaxItemProcessingTime time.Duration
}{
	K:                     20,
	AlphaPreference:       15,
	AlphaConfidence:       15,
	Beta:                  20,
	ConcurrentPolls:       4,
	OptimalProcessing:     10,
	MaxOutstandingItems:   1024,
	MaxItemProcessingTime: 2 * time.Minute,
}

// RouterHealthConfig contains configuration for router health checks
type RouterHealthConfig struct {
	MaxTimeSinceMsgReceived time.Duration
	MaxTimeSinceMsgSent     time.Duration
	MaxPortionSendQueueFull float64
	MinConnectedPeers       uint
	ReadTimeout             time.Duration
	WriteTimeout            time.Duration
	MaxSendFailRate         float64
	MaxDropRate             float64
	MaxOutstandingRequests  int
	MaxOutstandingDuration  time.Duration
	MaxRunTimeRequests      time.Duration
	MaxDropRateHalflife     time.Duration
}

// BenchlistConfig contains configuration for benchlisting
type BenchlistConfig struct {
	Deprecated         bool
	Duration           time.Duration
	MinFailingDuration time.Duration
	Threshold          int
	MaxPortion         float64
	FailThreshold      int
}

// PrismParameters contains prism protocol parameters
type PrismParameters struct {
	NumParents          int
	NumNodes            int
	AlphaPreference     int
	AlphaConfidence     int
	K                   int
	MaxOutstandingItems int
}
