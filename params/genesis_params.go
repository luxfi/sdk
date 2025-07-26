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

// MaxGasLimit is the maximum gas limit
const MaxGasLimit uint64 = 0x7fffffffffffffff

// MaxInitCodeSize is the maximum size of init code (2 * MaxCodeSize)
const MaxInitCodeSize = 2 * MaxCodeSize

// GenesisGasLimit is the gas limit of the Genesis block
const GenesisGasLimit uint64 = 8000000

// ExpByteGas is the gas cost per byte of the EXP instruction
const ExpByteGas uint64 = 10

// SloadGas is the gas cost for SLOAD
const SloadGas uint64 = 50

// CallValueTransferGas is the gas cost for a value transfer
const CallValueTransferGas uint64 = 9000

// QuadCoeffDiv is the divisor for the quadratic part of the memory cost equation
const QuadCoeffDiv uint64 = 512

// LogDataGas is the gas cost per byte of log data
const LogDataGas uint64 = 8

// CallStipend is the free gas given for a CALL operation
const CallStipend uint64 = 2300

// Keccak256Gas is the base gas for Keccak256
const Keccak256Gas uint64 = 30

// Keccak256WordGas is the gas per word for Keccak256
const Keccak256WordGas uint64 = 6

// SstoreSetGas is the gas cost for setting a storage value
const SstoreSetGas uint64 = 20000

// SstoreResetGas is the gas cost for resetting a storage value
const SstoreResetGas uint64 = 5000

// SstoreClearGas is the gas cost for clearing a storage value
const SstoreClearGas uint64 = 5000

// SstoreRefundGas is the refund for clearing a storage value
const SstoreRefundGas uint64 = 15000

// NetSstoreNoopGas is the gas cost for a no-op sstore
const NetSstoreNoopGas uint64 = 200

// NetSstoreInitGas is the gas cost for initializing a storage value
const NetSstoreInitGas uint64 = 20000

// NetSstoreCleanGas is the gas cost for a clean sstore
const NetSstoreCleanGas uint64 = 5000

// NetSstoreDirtyGas is the gas cost for a dirty sstore
const NetSstoreDirtyGas uint64 = 200

// NetSstoreClearRefund is the refund for clearing a storage value
const NetSstoreClearRefund uint64 = 15000

// NetSstoreResetRefund is the refund for resetting a storage value
const NetSstoreResetRefund uint64 = 4800

// NetSstoreResetClearRefund is the refund for reset then clear
const NetSstoreResetClearRefund uint64 = 19800

// SstoreSentryGasEIP2200 is the minimum gas for sstore
const SstoreSentryGasEIP2200 uint64 = 2300

// SstoreSetGasEIP2200 is the gas cost for setting storage
const SstoreSetGasEIP2200 uint64 = 20000

// SstoreResetGasEIP2200 is the gas cost for resetting storage
const SstoreResetGasEIP2200 uint64 = 5000

// SstoreClearsScheduleRefundEIP2200 is the refund for clearing storage
const SstoreClearsScheduleRefundEIP2200 uint64 = 15000

// ColdAccountAccessCostEIP2929 is the gas cost for cold account access
const ColdAccountAccessCostEIP2929 = uint64(2600)

// ColdSloadCostEIP2929 is the gas cost for cold sload
const ColdSloadCostEIP2929 = uint64(2100)

// WarmStorageReadCostEIP2929 is the gas cost for warm storage read
const WarmStorageReadCostEIP2929 = uint64(100)

// SstoreClearsScheduleRefundEIP3529 is the refund for clearing storage after EIP-3529
const SstoreClearsScheduleRefundEIP3529 uint64 = 4800

// JumpdestGas is the gas cost for JUMPDEST
const JumpdestGas uint64 = 1

// EpochDuration is the duration of an epoch
const EpochDuration uint64 = 30000

// CreateGas is the gas cost for CREATE
const CreateGas uint64 = 32000

// Create2Gas is the gas cost for CREATE2
const Create2Gas uint64 = 32000

// SelfdestructRefundGas is the refund for selfdestruct
const SelfdestructRefundGas uint64 = 24000

// MemoryGas is the gas cost per word of memory
const MemoryGas uint64 = 3

// TxDataNonZeroGas is the gas per non-zero byte of data
const TxDataNonZeroGas uint64 = 68

// CallGas is the gas cost for CALL
const CallGas uint64 = 40

// ExpGas is the base gas for EXP
const ExpGas uint64 = 10

// LogGas is the gas cost per LOG operation
const LogGas uint64 = 375

// CopyGas is the gas cost per word for copying
const CopyGas uint64 = 3

// StackLimit is the maximum stack depth
const StackLimit uint64 = 1024

// TierStepGas is the gas cost for tier operations
const TierStepGas uint64 = 0

// LogTopicGas is the gas cost per topic
const LogTopicGas uint64 = 375

// CallGasFrontier is the gas cost for CALL in Frontier
const CallGasFrontier uint64 = 40

// CallGasEIP150 is the gas cost for CALL after EIP-150
const CallGasEIP150 uint64 = 700

// BalanceGasFrontier is the gas cost for BALANCE in Frontier
const BalanceGasFrontier uint64 = 20

// BalanceGasEIP150 is the gas cost for BALANCE after EIP-150
const BalanceGasEIP150 uint64 = 400

// ExtcodeSizeGasFrontier is the gas cost for EXTCODESIZE in Frontier
const ExtcodeSizeGasFrontier uint64 = 20

// ExtcodeSizeGasEIP150 is the gas cost for EXTCODESIZE after EIP-150
const ExtcodeSizeGasEIP150 uint64 = 700

// GenesisDifficulty is the difficulty of the Genesis block
var GenesisDifficulty = big.NewInt(131072)
