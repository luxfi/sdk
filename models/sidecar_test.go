// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetVMID_imported(t *testing.T) {
	assert := require.New(t)
	testVMID := "abcd"
	sc := Sidecar{
		ImportedFromLPM: true,
		ImportedVMID:    testVMID,
	}

	vmid, err := sc.GetVMID()
	assert.NoError(err)
	assert.Equal(testVMID, vmid)
}

func TestGetVMID_derived(t *testing.T) {
	assert := require.New(t)
	testVMName := "subnet"
	testVMID := "test-vm-id"
	sc := Sidecar{
		ImportedFromLPM: false,
		Name:            testVMName,
		VMID:            testVMID,
	}

	vmid, err := sc.GetVMID()
	assert.NoError(err)
	assert.Equal(testVMID, vmid)
}
