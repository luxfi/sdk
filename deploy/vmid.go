// Copyright (C) 2020-2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

package deploy

import (
	"github.com/luxfi/ids"
	"github.com/luxfi/node/utils/hashing"
)

// Well-known VM IDs
var (
	// SubnetEVMID is the VM ID for SubnetEVM/LuxEVM
	SubnetEVMID = ids.ID{'s', 'r', 'E', 'X', 'i', 'W', 'a', 'H', 'u', 'h', 'N', 'y', 'G', 'w', 'P', 'U', 'i', '4', '4', '4', 'T', 'u', '4', '7', 'Z', 'E', 'D', 'w', 'x', 'T', 'W', 'r'}

	// XSVMID is the example cross-subnet VM
	XSVMID ids.ID
)

func init() {
	// SubnetEVM VM ID: srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy
	subnetEVMID, err := ids.FromString("srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy")
	if err == nil {
		SubnetEVMID = subnetEVMID
	}

	// XSVM ID from constants
	xsvmID, err := ids.FromString("2YTjjpBFnRTdGNg7EvbvJXj3P6JeYPcFUPyQBLZK8GRqrMpPHT")
	if err == nil {
		XSVMID = xsvmID
	}
}

// VMID computes the VM ID from a chain name using the same algorithm as netrunner.
// This ensures consistent VM IDs across the ecosystem.
func VMID(chainName string) (ids.ID, error) {
	// Use SHA256 of the chain name to generate a deterministic VM ID
	hash := hashing.ComputeHash256([]byte(chainName))
	return ids.ToID(hash)
}

// MustVMID is like VMID but panics on error.
// Useful for initialization of package-level variables.
func MustVMID(chainName string) ids.ID {
	id, err := VMID(chainName)
	if err != nil {
		panic(err)
	}
	return id
}
