// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package server

import (
	"net/http"
	"sync"

	"github.com/luxfi/metric"
)

type labeledCounter struct {
	counters map[string]metric.Counter
	mu       sync.RWMutex
}

func newLabeledCounter(baseName, help string) *labeledCounter {
	return &labeledCounter{
		counters: make(map[string]metric.Counter),
	}
}

func (lc *labeledCounter) WithLabelValues(method, endpoint string) metric.Counter {
	key := method + "_" + endpoint
	lc.mu.RLock()
	counter, exists := lc.counters[key]
	lc.mu.RUnlock()

	if !exists {
		lc.mu.Lock()
		defer lc.mu.Unlock()
		// Double-check after acquiring write lock
		if counter, exists = lc.counters[key]; !exists {
			counter = metric.NewCounter("api_requests_total_"+key, "Total number of API requests for "+method+" "+endpoint)
			lc.counters[key] = counter
		}
	}

	return counter
}

type labeledHistogram struct {
	histograms map[string]metric.Histogram
	mu         sync.RWMutex
}

func newLabeledHistogram(baseName, help string) *labeledHistogram {
	return &labeledHistogram{
		histograms: make(map[string]metric.Histogram),
	}
}

func (lh *labeledHistogram) WithLabelValues(method, endpoint string) metric.Histogram {
	key := method + "_" + endpoint
	lh.mu.RLock()
	histogram, exists := lh.histograms[key]
	lh.mu.RUnlock()

	if !exists {
		lh.mu.Lock()
		defer lh.mu.Unlock()
		// Double-check after acquiring write lock
		if histogram, exists = lh.histograms[key]; !exists {
			histogram = metric.NewHistogram("api_request_duration_seconds_"+key, "API request duration in seconds for "+method+" "+endpoint, metric.DefBuckets)
			lh.histograms[key] = histogram
		}
	}

	return histogram
}

type serverMetrics struct {
	requests metric.Counter
	duration metric.Histogram
	inflight metric.Gauge
	// For labeled metrics
	labeledRequests *labeledCounter
	labeledDuration *labeledHistogram
}

func newMetrics(reg *metric.MetricsRegistry) (*serverMetrics, error) {
	m := &serverMetrics{
		requests:        metric.NewCounter("api_requests_total", "Total number of API requests"),
		duration:        metric.NewHistogram("api_request_duration_seconds", "API request duration in seconds", metric.DefBuckets),
		inflight:        metric.NewGauge("api_requests_inflight", "Number of inflight API requests"),
		labeledRequests: newLabeledCounter("api_requests_total", "Total number of API requests"),
		labeledDuration: newLabeledHistogram("api_request_duration_seconds", "API request duration in seconds"),
	}

	return m, nil
}

func (m *serverMetrics) wrapHandler(chainName string, handler http.Handler) http.Handler {
	// Instrument handler with metrics
	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.labeledRequests.WithLabelValues(r.Method, chainName).Inc()
		m.inflight.Inc()
		defer m.inflight.Dec()

		timer := m.labeledDuration.WithLabelValues(r.Method, chainName)
		defer func(start float64) {
			timer.Observe(float64(start))
		}(float64(0)) // TODO: implement proper timing

		handler.ServeHTTP(w, r)
	})
	return handler
}
