// Copyright (C) 2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package blockchain

import (
	"sync"
	"time"
)

// Metrics tracks blockchain performance metrics
type Metrics struct {
	mu sync.RWMutex
	
	// Block metrics
	BlocksProduced   uint64
	LastBlockTime    time.Time
	AverageBlockTime time.Duration
	
	// Transaction metrics
	TxProcessed      uint64
	TxFailed         uint64
	TPS              float64
	
	// Network metrics
	PeersConnected   int
	NetworkLatency   time.Duration
	
	// Resource metrics
	CPUUsage         float64
	MemoryUsage      uint64
	DiskUsage        uint64
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{}
}

// RecordBlock records a new block
func (m *Metrics) RecordBlock(blockTime time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.BlocksProduced++
	if !m.LastBlockTime.IsZero() {
		interval := blockTime.Sub(m.LastBlockTime)
		if m.AverageBlockTime == 0 {
			m.AverageBlockTime = interval
		} else {
			// Exponential moving average
			m.AverageBlockTime = time.Duration(float64(m.AverageBlockTime)*0.9 + float64(interval)*0.1)
		}
	}
	m.LastBlockTime = blockTime
}

// RecordTransaction records a transaction
func (m *Metrics) RecordTransaction(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if success {
		m.TxProcessed++
	} else {
		m.TxFailed++
	}
}

// UpdateTPS updates transactions per second
func (m *Metrics) UpdateTPS(tps float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.TPS = tps
}

// UpdateNetwork updates network metrics
func (m *Metrics) UpdateNetwork(peers int, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.PeersConnected = peers
	m.NetworkLatency = latency
}

// UpdateResources updates resource usage metrics
func (m *Metrics) UpdateResources(cpu float64, memory, disk uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.CPUUsage = cpu
	m.MemoryUsage = memory
	m.DiskUsage = disk
}

// GetSnapshot returns a snapshot of current metrics
func (m *Metrics) GetSnapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return map[string]interface{}{
		"blocks": map[string]interface{}{
			"produced":         m.BlocksProduced,
			"lastBlockTime":    m.LastBlockTime,
			"averageBlockTime": m.AverageBlockTime.String(),
		},
		"transactions": map[string]interface{}{
			"processed": m.TxProcessed,
			"failed":    m.TxFailed,
			"tps":       m.TPS,
		},
		"network": map[string]interface{}{
			"peers":    m.PeersConnected,
			"latency":  m.NetworkLatency.String(),
		},
		"resources": map[string]interface{}{
			"cpuUsage":    m.CPUUsage,
			"memoryUsage": m.MemoryUsage,
			"diskUsage":   m.DiskUsage,
		},
	}
}