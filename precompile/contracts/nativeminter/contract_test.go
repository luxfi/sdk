// (c) 2021-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package nativeminter

import (
	"math/big"
	"testing"

	"github.com/luxdefi/evm/core/state"
	"github.com/luxdefi/evm/precompile/allowlist"
	"github.com/luxdefi/evm/precompile/contract"
<<<<<<< HEAD
<<<<<<< HEAD
	"github.com/luxdefi/evm/precompile/precompileconfig"
=======
>>>>>>> fd08c47 (Update import path)
=======
	"github.com/luxdefi/evm/precompile/precompileconfig"
>>>>>>> d5328b4 (Sync upstream)
	"github.com/luxdefi/evm/precompile/testutils"
	"github.com/luxdefi/evm/vmerrs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	tests = map[string]testutils.PrecompileTest{
		"calling mintNativeCoin from NoRole should fail": {
			Caller:     allowlist.TestNoRoleAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestNoRoleAddr, common.Big1)
				require.NoError(t, err)

				return input
			},
			SuppliedGas: MintGasCost,
			ReadOnly:    false,
			ExpectedErr: ErrCannotMint.Error(),
		},
		"calling mintNativeCoin from Enabled should succeed": {
			Caller:     allowlist.TestEnabledAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestEnabledAddr, common.Big1)
				require.NoError(t, err)

				return input
			},
<<<<<<< HEAD
			SuppliedGas: MintGasCost + NativeCoinMintedEventGasCost,
			ReadOnly:    false,
			ExpectedRes: []byte{},
			AfterHook: func(t testing.TB, stateDB contract.StateDB) {
				require.Equal(t, common.Big1, stateDB.GetBalance(allowlist.TestEnabledAddr), "expected minted funds")

				logsTopics, logsData := stateDB.GetLogData()
				assertNativeCoinMintedEvent(t, logsTopics, logsData, allowlist.TestEnabledAddr, allowlist.TestEnabledAddr, common.Big1)
=======
			SuppliedGas: MintGasCost,
			ReadOnly:    false,
			ExpectedRes: []byte{},
			AfterHook: func(t testing.TB, state contract.StateDB) {
				require.Equal(t, common.Big1, state.GetBalance(allowlist.TestEnabledAddr), "expected minted funds")
>>>>>>> d5328b4 (Sync upstream)
			},
		},
		"initial mint funds": {
			Caller:     allowlist.TestEnabledAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			Config: &Config{
				InitialMint: map[common.Address]*math.HexOrDecimal256{
					allowlist.TestEnabledAddr: math.NewHexOrDecimal256(2),
				},
			},
<<<<<<< HEAD
			AfterHook: func(t testing.TB, stateDB contract.StateDB) {
				require.Equal(t, common.Big2, stateDB.GetBalance(allowlist.TestEnabledAddr), "expected minted funds")
=======
			AfterHook: func(t testing.TB, state contract.StateDB) {
				require.Equal(t, common.Big2, state.GetBalance(allowlist.TestEnabledAddr), "expected minted funds")
>>>>>>> d5328b4 (Sync upstream)
			},
		},
		"calling mintNativeCoin from Manager should succeed": {
			Caller:     allowlist.TestManagerAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestEnabledAddr, common.Big1)
				require.NoError(t, err)

				return input
			},
<<<<<<< HEAD
			SuppliedGas: MintGasCost + NativeCoinMintedEventGasCost,
			ReadOnly:    false,
			ExpectedRes: []byte{},
			AfterHook: func(t testing.TB, stateDB contract.StateDB) {
				require.Equal(t, common.Big1, stateDB.GetBalance(allowlist.TestEnabledAddr), "expected minted funds")

				logsTopics, logsData := stateDB.GetLogData()
				assertNativeCoinMintedEvent(t, logsTopics, logsData, allowlist.TestManagerAddr, allowlist.TestEnabledAddr, common.Big1)
			},
		},
		"mint funds from admin address": {
			Caller:     allowlist.TestAdminAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestAdminAddr, common.Big1)
=======
			SuppliedGas: MintGasCost,
			ReadOnly:    false,
			ExpectedRes: []byte{},
			AfterHook: func(t testing.TB, state contract.StateDB) {
				require.Equal(t, common.Big1, state.GetBalance(allowlist.TestEnabledAddr), "expected minted funds")
			},
		},
		"calling mintNativeCoin from Admin should succeed": {
			Caller:     allowlist.TestAdminAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestAdminAddr, common.Big1)
				require.NoError(t, err)

				return input
			},
			SuppliedGas: MintGasCost,
			ReadOnly:    false,
			ExpectedRes: []byte{},
			AfterHook: func(t testing.TB, state contract.StateDB) {
				require.Equal(t, common.Big1, state.GetBalance(allowlist.TestAdminAddr), "expected minted funds")
			},
		},
		"mint max big funds": {
			Caller:     allowlist.TestAdminAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestAdminAddr, math.MaxBig256)
>>>>>>> d5328b4 (Sync upstream)
				require.NoError(t, err)

				return input
			},
<<<<<<< HEAD
			SuppliedGas: MintGasCost + NativeCoinMintedEventGasCost,
			ReadOnly:    false,
			ExpectedRes: []byte{},
			AfterHook: func(t testing.TB, stateDB contract.StateDB) {
				require.Equal(t, common.Big1, stateDB.GetBalance(allowlist.TestAdminAddr), "expected minted funds")

				logsTopics, logsData := stateDB.GetLogData()
				assertNativeCoinMintedEvent(t, logsTopics, logsData, allowlist.TestAdminAddr, allowlist.TestAdminAddr, common.Big1)
			},
		},
		"mint max big funds": {
			Caller:     allowlist.TestAdminAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestAdminAddr, math.MaxBig256)
=======
			SuppliedGas: MintGasCost,
			ReadOnly:    false,
			ExpectedRes: []byte{},
			AfterHook: func(t testing.TB, state contract.StateDB) {
				require.Equal(t, math.MaxBig256, state.GetBalance(allowlist.TestAdminAddr), "expected minted funds")
			},
		},
		"readOnly mint with noRole fails": {
			Caller:     allowlist.TestNoRoleAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestAdminAddr, common.Big1)
				require.NoError(t, err)

				return input
			},
			SuppliedGas: MintGasCost,
			ReadOnly:    true,
			ExpectedErr: vmerrs.ErrWriteProtection.Error(),
		},
		"readOnly mint with allow role fails": {
			Caller:     allowlist.TestEnabledAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestEnabledAddr, common.Big1)
>>>>>>> d5328b4 (Sync upstream)
				require.NoError(t, err)

				return input
			},
<<<<<<< HEAD
			SuppliedGas: MintGasCost + NativeCoinMintedEventGasCost,
			ReadOnly:    false,
			ExpectedRes: []byte{},
			AfterHook: func(t testing.TB, stateDB contract.StateDB) {
				require.Equal(t, math.MaxBig256, stateDB.GetBalance(allowlist.TestAdminAddr), "expected minted funds")

				logsTopics, logsData := stateDB.GetLogData()
				assertNativeCoinMintedEvent(t, logsTopics, logsData, allowlist.TestAdminAddr, allowlist.TestAdminAddr, math.MaxBig256)
			},
		},
		"readOnly mint with noRole fails": {
			Caller:     allowlist.TestNoRoleAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestAdminAddr, common.Big1)
=======
			SuppliedGas: MintGasCost,
			ReadOnly:    true,
			ExpectedErr: vmerrs.ErrWriteProtection.Error(),
		},
		"readOnly mint with admin role fails": {
			Caller:     allowlist.TestAdminAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestAdminAddr, common.Big1)
				require.NoError(t, err)

				return input
			},
			SuppliedGas: MintGasCost,
			ReadOnly:    true,
			ExpectedErr: vmerrs.ErrWriteProtection.Error(),
		},
		"insufficient gas mint from admin": {
			Caller:     allowlist.TestAdminAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestEnabledAddr, common.Big1)
>>>>>>> d5328b4 (Sync upstream)
				require.NoError(t, err)

				return input
			},
<<<<<<< HEAD
			SuppliedGas: MintGasCost,
			ReadOnly:    true,
			ExpectedErr: vmerrs.ErrWriteProtection.Error(),
		},
		"readOnly mint with allow role fails": {
			Caller:     allowlist.TestEnabledAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestEnabledAddr, common.Big1)
				require.NoError(t, err)

				return input
			},
			SuppliedGas: MintGasCost,
			ReadOnly:    true,
			ExpectedErr: vmerrs.ErrWriteProtection.Error(),
		},
		"readOnly mint with admin role fails": {
			Caller:     allowlist.TestAdminAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestAdminAddr, common.Big1)
				require.NoError(t, err)

				return input
			},
			SuppliedGas: MintGasCost,
			ReadOnly:    true,
			ExpectedErr: vmerrs.ErrWriteProtection.Error(),
		},
		"insufficient gas mint from admin": {
			Caller:     allowlist.TestAdminAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestEnabledAddr, common.Big1)
				require.NoError(t, err)

				return input
			},
			SuppliedGas: MintGasCost + NativeCoinMintedEventGasCost - 1,
			ReadOnly:    false,
			ExpectedErr: vmerrs.ErrOutOfGas.Error(),
		},
		"mint doesn't log pre-DUpgrade": {
			Caller:     allowlist.TestEnabledAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			ChainConfigFn: func(ctrl *gomock.Controller) precompileconfig.ChainConfig {
				config := precompileconfig.NewMockChainConfig(ctrl)
				config.EXPECT().IsDUpgrade(gomock.Any()).Return(false).AnyTimes()
				return config
			},
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestEnabledAddr, common.Big1)
				require.NoError(t, err)
				return input
			},
			SuppliedGas: MintGasCost,
			ReadOnly:    false,
			ExpectedRes: []byte{},
			AfterHook: func(t testing.TB, stateDB contract.StateDB) {
				// Check no logs are stored in state
				logsTopics, logsData := stateDB.GetLogData()
				require.Len(t, logsTopics, 0)
				require.Len(t, logsData, 0)
			},
		},
		"mint with extra padded bytes should fail pre-DUpgrade": {
			Caller:     allowlist.TestEnabledAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			ChainConfigFn: func(ctrl *gomock.Controller) precompileconfig.ChainConfig {
				config := precompileconfig.NewMockChainConfig(ctrl)
=======
			SuppliedGas: MintGasCost - 1,
			ReadOnly:    false,
			ExpectedErr: vmerrs.ErrOutOfGas.Error(),
		},
		"mint with extra padded bytes should fail before DUpgrade": {
			Caller:     allowlist.TestEnabledAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
			ChainConfigFn: func(t testing.TB) precompileconfig.ChainConfig {
				config := precompileconfig.NewMockChainConfig(gomock.NewController(t))
>>>>>>> d5328b4 (Sync upstream)
				config.EXPECT().IsDUpgrade(gomock.Any()).Return(false).AnyTimes()
				return config
			},
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestEnabledAddr, common.Big1)
				require.NoError(t, err)

				// Add extra bytes to the end of the input
				input = append(input, make([]byte, 32)...)

				return input
			},
			SuppliedGas: MintGasCost,
			ReadOnly:    false,
			ExpectedErr: ErrInvalidLen.Error(),
		},
		"mint with extra padded bytes should succeed with DUpgrade": {
			Caller:     allowlist.TestEnabledAddr,
			BeforeHook: allowlist.SetDefaultRoles(Module.Address),
<<<<<<< HEAD
			ChainConfigFn: func(ctrl *gomock.Controller) precompileconfig.ChainConfig {
				config := precompileconfig.NewMockChainConfig(ctrl)
=======
			ChainConfigFn: func(t testing.TB) precompileconfig.ChainConfig {
				config := precompileconfig.NewMockChainConfig(gomock.NewController(t))
>>>>>>> d5328b4 (Sync upstream)
				config.EXPECT().IsDUpgrade(gomock.Any()).Return(true).AnyTimes()
				return config
			},
			InputFn: func(t testing.TB) []byte {
				input, err := PackMintNativeCoin(allowlist.TestEnabledAddr, common.Big1)
				require.NoError(t, err)

				// Add extra bytes to the end of the input
				input = append(input, make([]byte, 32)...)

				return input
			},
			ExpectedRes: []byte{},
<<<<<<< HEAD
			SuppliedGas: MintGasCost + NativeCoinMintedEventGasCost,
			ReadOnly:    false,
			AfterHook: func(t testing.TB, state contract.StateDB) {
				require.Equal(t, common.Big1, state.GetBalance(allowlist.TestEnabledAddr), "expected minted funds")

				logsTopics, logsData := state.GetLogData()
				assertNativeCoinMintedEvent(t, logsTopics, logsData, allowlist.TestEnabledAddr, allowlist.TestEnabledAddr, common.Big1)
=======
			SuppliedGas: MintGasCost,
			ReadOnly:    false,
			AfterHook: func(t testing.TB, state contract.StateDB) {
				require.Equal(t, common.Big1, state.GetBalance(allowlist.TestEnabledAddr), "expected minted funds")
>>>>>>> d5328b4 (Sync upstream)
			},
		},
	}
)

func TestContractNativeMinterRun(t *testing.T) {
	allowlist.RunPrecompileWithAllowListTests(t, Module, state.NewTestStateDB, tests)
}

func BenchmarkContractNativeMinter(b *testing.B) {
	allowlist.BenchPrecompileWithAllowList(b, Module, state.NewTestStateDB, tests)
}

func assertNativeCoinMintedEvent(t testing.TB,
	logsTopics [][]common.Hash,
	logsData [][]byte,
	expectedSender common.Address,
	expectedRecipient common.Address,
	expectedAmount *big.Int) {
	require.Len(t, logsTopics, 1)
	require.Len(t, logsData, 1)
	topics := logsTopics[0]
	require.Len(t, topics, 3)
	require.Equal(t, NativeMinterABI.Events["NativeCoinMinted"].ID, topics[0])
	require.Equal(t, expectedSender.Hash(), topics[1])
	require.Equal(t, expectedRecipient.Hash(), topics[2])
	require.NotEmpty(t, logsData[0])
	amount, err := UnpackNativeCoinMintedEventData(logsData[0])
	require.NoError(t, err)
	require.True(t, expectedAmount.Cmp(amount) == 0, "expected", expectedAmount, "got", amount)
}
