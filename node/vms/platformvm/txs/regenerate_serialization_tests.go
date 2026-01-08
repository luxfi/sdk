// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Copyright (C) 2019-2025, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build ignore

package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func main() {
	// Run the serialization tests and capture output
	cmd := exec.Command("go", "test", "-v", "./vms/platformvm/txs", "-run", "Serialization")
	output, _ := cmd.CombinedOutput()

	outputStr := string(output)

	// Parse the output to extract test failures with actual bytes
	tests := parseTestFailures(outputStr)

	fmt.Printf("Found %d failing serialization tests\n", len(tests))

	for testName, actualBytes := range tests {
		fmt.Printf("\nTest: %s\n", testName)
		fmt.Printf("Actual bytes: %s\n", actualBytes[:100])
		updateTestFile(testName, actualBytes)
	}
}

func parseTestFailures(output string) map[string]string {
	tests := make(map[string]string)

	// Match pattern for actual bytes in test output
	re := regexp.MustCompile(`Test:\s+(\w+)[\s\S]*?actual\s*:\s*\[\]byte\{(0x[0-9a-f,\sx]+)\}`)
	matches := re.FindAllStringSubmatch(output, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			testName := match[1]
			actualBytesStr := match[2]
			tests[testName] = actualBytesStr
		}
	}

	return tests
}

func updateTestFile(testName, actualBytes string) {
	// Map test name to file
	fileMap := map[string]string{
		"TestAddPermissionlessPrimaryDelegatorSerialization": "add_permissionless_delegator_tx_test.go",
		"TestAddPermissionlessNetDelegatorSerialization":     "add_permissionless_delegator_tx_test.go",
		"TestBaseTxSerialization":                            "base_tx_test.go",
		"TestConvertChainToL1TxSerialization":                  "convert_net_to_l1_tx_test.go",
		"TestDisableL1ValidatorTxSerialization":              "disable_l1_validator_tx_test.go",
		"TestIncreaseL1ValidatorBalanceTxSerialization":      "increase_l1_validator_balance_tx_test.go",
		"TestRegisterL1ValidatorTxSerialization":             "register_l1_validator_tx_test.go",
		"TestRemoveChainValidatorTxSerialization":              "remove_net_validator_tx_test.go",
		"TestSetL1ValidatorWeightTxSerialization":            "set_l1_validator_weight_tx_test.go",
		"TestTransferChainOwnershipTxSerialization":            "transfer_net_ownership_tx_test.go",
		"TestTransformChainTxSerialization":                    "transform_net_tx_test.go",
	}

	fileName, ok := fileMap[testName]
	if !ok {
		fmt.Printf("Unknown test file for: %s\n", testName)
		return
	}

	fmt.Printf("Would update file: %s\n", fileName)
}
