// Copyright (C) 2019-2024, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package params

import "math/big"

// Genesis-specific gas costs for precompiled contracts
const (
	// AssetBalanceApricot is the gas cost for querying native asset balance
	AssetBalanceApricot uint64 = 2474

	// AssetCallApricot is the gas cost for calling native asset transfer
	AssetCallApricot uint64 = 9000
)

// CallCreateDepth is the maximum depth of call/create stack.
const CallCreateDepth = 1024

// MaxCodeSize is the maximum bytecode to permit for a contract
const MaxCodeSize = 24576

// CreateDataGas is the gas cost per byte of data for contract creation
const CreateDataGas uint64 = 200

// CallNewAccountGas is the gas cost for calling a new account
const CallNewAccountGas uint64 = 25000

// TxGasContractCreation is the gas cost for contract creation
const TxGasContractCreation uint64 = 53000

// TxGas is the per-transaction gas cost
const TxGas uint64 = 21000

// TxDataNonZeroGasFrontier is the gas cost per non-zero byte of transaction data
const TxDataNonZeroGasFrontier uint64 = 68

// TxDataNonZeroGasEIP2028 is the gas cost per non-zero byte after EIP-2028
const TxDataNonZeroGasEIP2028 uint64 = 16

// TxDataZeroGas is the gas cost per zero byte of transaction data
const TxDataZeroGas uint64 = 4

// InitCodeWordGas is the gas cost per word of init code
const InitCodeWordGas uint64 = 2

// TxAccessListAddressGas is the gas cost per address in the access list
const TxAccessListAddressGas uint64 = 2400

// TxAccessListStorageKeyGas is the gas cost per storage key in the access list
const TxAccessListStorageKeyGas uint64 = 1900

// GasLimitBoundDivisor is the bound divisor of the gas limit, used in update calculations
const GasLimitBoundDivisor uint64 = 1024

// MinGasLimit is the minimum the gas limit may ever be
const MinGasLimit uint64 = 5000

// MaxInitCodeSize is the maximum size of init code (2 * MaxCodeSize)
const MaxInitCodeSize = 2 * MaxCodeSize

// GenesisGasLimit is the gas limit of the Genesis block
const GenesisGasLimit uint64 = 8000000

// GenesisDifficulty is the difficulty of the Genesis block
var GenesisDifficulty = big.NewInt(131072)
