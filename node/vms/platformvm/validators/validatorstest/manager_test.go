// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validatorstest

import (
	"testing"

	vmvalidators "github.com/luxfi/sdk/node/vms/platformvm/validators"
)

// TestManagerImplementsInterface verifies that the test manager
// correctly implements the Manager interface
func TestManagerImplementsInterface(t *testing.T) {
	// This test will fail at compile time if manager doesn't implement Manager
	var _ vmvalidators.Manager = manager{}
	
	// Verify the singleton Manager also implements the interface
	var _ vmvalidators.Manager = Manager
}