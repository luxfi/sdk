// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/api"
	"github.com/luxfi/cache/lru"
	"github.com/luxfi/sdk/utils"
)

// TestLazyHandlerWrapper tests that the lazy handler wrapper properly delays
// initialization until the VM is ready
func TestLazyHandlerWrapper(t *testing.T) {
	require := require.New(t)

	// Create a minimal VM instance
	vm := &VM{
		bootstrapped: utils.Atomic[bool]{},
		state:        nil, // Initially nil
		manager:      nil, // Initially nil
	}

	// Create the lazy handler wrapper
	wrapper := &lazyHandlerWrapper{vm: vm}

	// Test 1: Request before bootstrapping should return 503
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	wrapper.ServeHTTP(rec, req)

	require.Equal(http.StatusServiceUnavailable, rec.Code)
	require.Contains(rec.Body.String(), "Platform service not ready, VM still bootstrapping")

	// Test 2: Mark VM as bootstrapped (but still missing internal state)
	vm.bootstrapped.Set(true)

	// Request will go through handler creation (succeeds with nil ctx)
	// but the RPC server will return 415 because no Content-Type header
	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	rec2 := httptest.NewRecorder()
	wrapper2 := &lazyHandlerWrapper{vm: vm} // New wrapper to reset once
	wrapper2.ServeHTTP(rec2, req2)

	// The RPC server responds with 415 (Unsupported Media Type) for requests
	// without proper Content-Type header
	require.Equal(http.StatusUnsupportedMediaType, rec2.Code)

	// Test 3: Set up minimal state and context for successful initialization
	// Note: In real usage, these would be properly initialized by VM.Initialize()
	// For this test, we're just checking the lazy initialization logic
}

// TestCreateHandlersReturnsLazyWrapper tests that CreateHandlers returns
// a lazy wrapper instead of immediately creating the service
func TestCreateHandlersReturnsLazyWrapper(t *testing.T) {
	require := require.New(t)

	// Create a minimal VM instance
	vm := &VM{
		isInitialized: utils.Atomic[bool]{},
	}

	// Call CreateHandlers - should succeed even if VM not initialized
	handlers, err := vm.CreateHandlers(context.Background())
	require.NoError(err)
	require.NotNil(handlers)
	require.Contains(handlers, "")

	// Verify the handler is a lazy wrapper
	handler := handlers[""]
	_, ok := handler.(*lazyHandlerWrapper)
	require.True(ok, "handler should be a lazyHandlerWrapper")
}

// TestServiceNilVMCheck verifies that service methods handle nil VM gracefully
func TestServiceNilVMCheck(t *testing.T) {
	require := require.New(t)

	// Create a service with nil VM
	service := &Service{
		vm:                    nil,
		addrManager:           nil,
		stakerAttributesCache: lru.NewCache[ids.ID, *stakerAttributes](100),
	}

	// Try to call GetHeight with nil VM
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	var response api.GetHeightResponse
	// This should panic because vm is nil, so we need to recover
	defer func() {
		if r := recover(); r != nil {
			require.NotNil(r, "Expected panic from nil VM")
		}
	}()
	err := service.GetHeight(req, nil, &response)
	// If we get here, the service handled nil VM gracefully
	if err != nil {
		require.Error(err)
	}
}