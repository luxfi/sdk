// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package precompile

import (
	"os"
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"

	// Import the solidity package, so that ginkgo maps out the tests declared within the package
	"github.com/luxfi/evm/tests/precompile/solidity"
)

func TestE2E(t *testing.T) {
	// Skip if LUXD_PATH is not set - this is an E2E test that requires
	// a full node environment with the luxd binary available
	if os.Getenv("LUXD_PATH") == "" && os.Getenv("E2E_TEST") == "" {
		t.Skip("Skipping E2E test: LUXD_PATH environment variable not set")
	}
	if basePath := os.Getenv("TEST_SOURCE_ROOT"); basePath != "" {
		os.Chdir(basePath)
	}
	solidity.RegisterAsyncTests()
	ginkgo.RunSpecs(t, "evm precompile ginkgo test suite")
}
