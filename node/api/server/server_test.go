// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	consensuscontext "github.com/luxfi/consensus/context"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
)

func TestRejectMiddleware(t *testing.T) {
	require := require.New(t)

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	// Create a consensus context
	ctx := &consensuscontext.Context{
		NetworkID: 1,
		ChainID:   ids.Empty,
		NodeID:    ids.EmptyNodeID,
		Log:       log.NoLog{},
	}

	// rejectMiddleware currently just returns the handler
	// TODO: When state checking is implemented, add more comprehensive tests
	middleware := rejectMiddleware(testHandler, ctx)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(http.StatusTeapot, w.Code)
}

func TestHTTPHeaderRouteIsCanonical(t *testing.T) {
	wantHeaderKey := http.CanonicalHeaderKey(HTTPHeaderRoute)
	require.Equal(t, wantHeaderKey, HTTPHeaderRoute)
}
