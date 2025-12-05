// Copyright (C) 2020-2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

package deploy

import (
	"testing"

	"github.com/luxfi/ids"
	"github.com/stretchr/testify/require"
)

func TestEWOQKeychain(t *testing.T) {
	kc := EWOQKeychain()
	require.NotNil(t, kc)

	// Should have exactly one address
	addrs := kc.Addresses()
	require.Equal(t, 1, addrs.Len())
}

func TestKeychainFromPrivateKeyBytes(t *testing.T) {
	kc, err := KeychainFromPrivateKeyBytes(EWOQKeyBytes)
	require.NoError(t, err)
	require.NotNil(t, kc)

	// Verify it matches the EWOQ keychain
	ewoqKc := EWOQKeychain()
	require.Equal(t, ewoqKc.Addresses(), kc.Addresses())
}

func TestKeychainFromPrivateKeyBytes_Invalid(t *testing.T) {
	// Too short
	_, err := KeychainFromPrivateKeyBytes([]byte{0x01, 0x02})
	require.Error(t, err)
}

func TestNewDeployer_NoEndpoint(t *testing.T) {
	_, err := New(
		WithKeychain(EWOQKeychain()),
	)
	require.ErrorIs(t, err, ErrNoEndpoint)
}

func TestNewDeployer_NoKeychain(t *testing.T) {
	_, err := New(
		WithEndpoint("http://localhost:9650"),
	)
	require.ErrorIs(t, err, ErrNoKeychain)
}

func TestNewDeployer_Valid(t *testing.T) {
	d, err := New(
		WithEndpoint("http://localhost:9650"),
		WithKeychain(EWOQKeychain()),
	)
	require.NoError(t, err)
	require.NotNil(t, d)
	require.Equal(t, "http://localhost:9650", d.Endpoint())
}

func TestSubnetEVMID(t *testing.T) {
	// Verify the SubnetEVM VM ID
	expectedID, err := ids.FromString("srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy")
	require.NoError(t, err)
	require.Equal(t, expectedID, SubnetEVMID)
}

func TestVMID(t *testing.T) {
	// Test that VMID is deterministic
	id1, err := VMID("test-chain")
	require.NoError(t, err)

	id2, err := VMID("test-chain")
	require.NoError(t, err)

	require.Equal(t, id1, id2)

	// Different names should produce different IDs
	id3, err := VMID("other-chain")
	require.NoError(t, err)
	require.NotEqual(t, id1, id3)
}
