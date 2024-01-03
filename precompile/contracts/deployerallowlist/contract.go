// (c) 2021-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package deployerallowlist

import (
	"github.com/luxdefi/evm/precompile/allowlist"
	"github.com/luxdefi/evm/precompile/contract"
	"github.com/ethereum/go-ethereum/common"
)

// Singleton StatefulPrecompiledContract for W/R access to the contract deployer allow list.
var ContractDeployerAllowListPrecompile contract.StatefulPrecompiledContract = allowlist.CreateAllowListPrecompile(ContractAddress)

// GetContractDeployerAllowListStatus returns the role of [address] for the contract deployer
// allow list.
func GetContractDeployerAllowListStatus(stateDB contract.StateDB, address common.Address) allowlist.Role {
	return allowlist.GetAllowListStatus(stateDB, ContractAddress, address)
}

// SetContractDeployerAllowListStatus sets the permissions of [address] to [role] for the
// contract deployer allow list.
// assumes [role] has already been verified as valid.
func SetContractDeployerAllowListStatus(stateDB contract.StateDB, address common.Address, role allowlist.Role) {
	allowlist.SetAllowListRole(stateDB, ContractAddress, address, role)
}
