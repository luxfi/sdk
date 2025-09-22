// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package core

import (
	"math/big"

	"github.com/luxfi/crypto"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
)

// Common test private keys and addresses used across the test suite
var (
	// TestKey1 is a well-known private key for testing
	TestKey1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	// TestAddr1 is the address corresponding to TestKey1
	TestAddr1 = crypto.PubkeyToAddress(TestKey1.PublicKey)

	// TestKey2 is another well-known private key for testing
	TestKey2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	// TestAddr2 is the address corresponding to TestKey2
	TestAddr2 = crypto.PubkeyToAddress(TestKey2.PublicKey)

	// TestKey3 is another well-known private key for testing
	TestKey3, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	// TestAddr3 is the address corresponding to TestKey3
	TestAddr3 = crypto.PubkeyToAddress(TestKey3.PublicKey)
)

// TestGenesisAlloc returns a genesis allocation with funded test accounts
func TestGenesisAlloc() GenesisAlloc {
	balance := new(big.Int).Mul(big.NewInt(1000000), big.NewInt(params.Ether))
	return GenesisAlloc{
		common.Address(TestAddr1): {Balance: balance},
		common.Address(TestAddr2): {Balance: balance},
		common.Address(TestAddr3): {Balance: balance},
	}
}

// TestGenesisWithBalance returns a genesis allocation with specified balance for test accounts
func TestGenesisWithBalance(balance *big.Int) GenesisAlloc {
	return GenesisAlloc{
		common.Address(TestAddr1): {Balance: balance},
		common.Address(TestAddr2): {Balance: balance},
		common.Address(TestAddr3): {Balance: balance},
	}
}

// FundTestAddresses adds test addresses to an existing genesis allocation
func FundTestAddresses(alloc GenesisAlloc) GenesisAlloc {
	if alloc == nil {
		alloc = make(GenesisAlloc)
	}

	balance := new(big.Int).Mul(big.NewInt(1000000), big.NewInt(params.Ether))

	// Only add if not already present
	if _, ok := alloc[common.Address(TestAddr1)]; !ok {
		alloc[common.Address(TestAddr1)] = types.Account{Balance: balance}
	}
	if _, ok := alloc[common.Address(TestAddr2)]; !ok {
		alloc[common.Address(TestAddr2)] = types.Account{Balance: balance}
	}
	if _, ok := alloc[common.Address(TestAddr3)]; !ok {
		alloc[common.Address(TestAddr3)] = types.Account{Balance: balance}
	}

	return alloc
}
