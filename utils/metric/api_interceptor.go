// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utilmetric

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/rpc/v2"
	metric "github.com/luxfi/metric"
)

type APIInterceptor interface {
	InterceptRequest(i *rpc.RequestInfo) *http.Request
	AfterRequest(i *rpc.RequestInfo)
}

type contextKey int

const requestTimestampKey contextKey = iota

type apiInterceptor struct {
	requestDurationCount metric.CounterVec
	requestDurationSum   metric.GaugeVec
	requestErrors        metric.CounterVec
}

func NewAPIInterceptor(registry metric.Registry) (APIInterceptor, error) {
	metricsInstance := metric.NewWithRegistry("api_interceptor", registry)

	requestDurationCount := metricsInstance.NewCounterVec(
		"request_duration_count",
		"Number of times this type of request was made",
		[]string{"method"},
	)
	requestDurationSum := metricsInstance.NewGaugeVec(
		"request_duration_sum",
		"Amount of time in nanoseconds that has been spent handling this type of request",
		[]string{"method"},
	)
	requestErrors := metricsInstance.NewCounterVec(
		"request_error_count",
		"Number of request errors",
		[]string{"method"},
	)

	return &apiInterceptor{
		requestDurationCount: requestDurationCount,
		requestDurationSum:   requestDurationSum,
		requestErrors:        requestErrors,
	}, nil
}

func (*apiInterceptor) InterceptRequest(i *rpc.RequestInfo) *http.Request {
	ctx := i.Request.Context()
	ctx = context.WithValue(ctx, requestTimestampKey, time.Now())
	return i.Request.WithContext(ctx)
}

func (apr *apiInterceptor) AfterRequest(i *rpc.RequestInfo) {
	timestampIntf := i.Request.Context().Value(requestTimestampKey)
	timestamp, ok := timestampIntf.(time.Time)
	if !ok {
		return
	}

	durationMetricCount := apr.requestDurationCount.With(metric.Labels{
		"method": i.Method,
	})
	durationMetricCount.Inc()

	duration := time.Since(timestamp)
	durationMetricSum := apr.requestDurationSum.With(metric.Labels{
		"method": i.Method,
	})
	durationMetricSum.Add(float64(duration))

	if i.Error != nil {
		errMetric := apr.requestErrors.With(metric.Labels{
			"method": i.Method,
		})
		errMetric.Inc()
	}
}
