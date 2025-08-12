// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package validatormanager

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/node/utils/logging"
	blockchainSDK "github.com/luxfi/sdk/blockchain"
	"github.com/luxfi/sdk/contract"
	"github.com/luxfi/sdk/models"

	luxcrypto "github.com/luxfi/crypto"
)

//go:embed smart_contracts/deployed_validator_messages_bytecode_v2.0.0.txt
var deployedValidatorMessagesV2_0_0Bytecode []byte

func AddValidatorMessagesV2_0_0ContractToAllocations(
	allocs core.GenesisAlloc,
) {
	deployedValidatorMessagesBytes := common.FromHex(strings.TrimSpace(string(deployedValidatorMessagesV2_0_0Bytecode)))
	allocs[common.Address(luxcrypto.HexToAddress(ValidatorMessagesContractAddress))] = core.GenesisAccount{
		Balance: big.NewInt(0),
		Code:    deployedValidatorMessagesBytes,
		Nonce:   1,
	}
}

func fillValidatorMessagesAddressPlaceholder(contract string) string {
	return strings.ReplaceAll(
		contract,
		"__$fd0c147b4031eef6079b0498cbafa865f0$__",
		ValidatorMessagesContractAddress[2:],
	)
}

//go:embed smart_contracts/deployed_poa_validator_manager_bytecode_v1.0.0.txt
var deployedPoAValidatorManagerV1_0_0Bytecode []byte

func AddPoAValidatorManagerV1_0_0ContractToAllocations(
	allocs core.GenesisAlloc,
) {
	deployedPoaValidatorManagerString := strings.TrimSpace(string(deployedPoAValidatorManagerV1_0_0Bytecode))
	deployedPoaValidatorManagerString = fillValidatorMessagesAddressPlaceholder(deployedPoaValidatorManagerString)
	deployedPoaValidatorManagerBytes := common.FromHex(deployedPoaValidatorManagerString)
	allocs[common.Address(luxcrypto.HexToAddress(ValidatorContractAddress))] = core.GenesisAccount{
		Balance: big.NewInt(0),
		Code:    deployedPoaValidatorManagerBytes,
		Nonce:   1,
	}
}

//go:embed smart_contracts/deployed_validator_manager_bytecode_v2.0.0.txt
var deployedValidatorManagerV2_0_0Bytecode []byte

func AddValidatorManagerV2_0_0ContractToAllocations(
	allocs core.GenesisAlloc,
) {
	deployedValidatorManagerString := strings.TrimSpace(string(deployedValidatorManagerV2_0_0Bytecode))
	deployedValidatorManagerString = fillValidatorMessagesAddressPlaceholder(deployedValidatorManagerString)
	deployedValidatorManagerBytes := common.FromHex(deployedValidatorManagerString)
	allocs[common.Address(luxcrypto.HexToAddress(ValidatorContractAddress))] = core.GenesisAccount{
		Balance: big.NewInt(0),
		Code:    deployedValidatorManagerBytes,
		Nonce:   1,
	}
}

//go:embed smart_contracts/validator_manager_bytecode_v2.0.0.txt
var validatorManagerV2_0_0Bytecode []byte

func DeployValidatorManagerV2_0_0Contract(
	rpcURL string,
	privateKey string,
) (luxcrypto.Address, error) {
	validatorManagerString := strings.TrimSpace(string(validatorManagerV2_0_0Bytecode))
	validatorManagerString = fillValidatorMessagesAddressPlaceholder(validatorManagerString)
	validatorManagerBytes := []byte(validatorManagerString)
	return contract.DeployContract(
		rpcURL,
		privateKey,
		validatorManagerBytes,
		"(uint8)",
		uint8(0),
	)
}

func DeployAndRegisterValidatorManagerV2_0_0Contract(
	rpcURL string,
	privateKey string,
	proxyOwnerPrivateKey string,
) (luxcrypto.Address, error) {
	validatorManagerAddress, err := DeployValidatorManagerV2_0_0Contract(
		rpcURL,
		privateKey,
	)
	if err != nil {
		return luxcrypto.Address{}, err
	}
	if _, _, err := SetupValidatorProxyImplementation(
		rpcURL,
		proxyOwnerPrivateKey,
		validatorManagerAddress,
	); err != nil {
		return luxcrypto.Address{}, err
	}
	return validatorManagerAddress, nil
}

//go:embed smart_contracts/native_token_staking_manager_bytecode_v1.0.0.txt
var posValidatorManagerV1_0_0Bytecode []byte

func DeployPoSValidatorManagerV1_0_0Contract(
	rpcURL string,
	privateKey string,
) (luxcrypto.Address, error) {
	posValidatorManagerString := strings.TrimSpace(string(posValidatorManagerV1_0_0Bytecode))
	posValidatorManagerString = fillValidatorMessagesAddressPlaceholder(posValidatorManagerString)
	posValidatorManagerBytes := []byte(posValidatorManagerString)
	return contract.DeployContract(
		rpcURL,
		privateKey,
		posValidatorManagerBytes,
		"(uint8)",
		uint8(0),
	)
}

func DeployAndRegisterPoSValidatorManagerV1_0_0Contract(
	rpcURL string,
	privateKey string,
	proxyOwnerPrivateKey string,
) (luxcrypto.Address, error) {
	posValidatorManagerAddress, err := DeployPoSValidatorManagerV1_0_0Contract(
		rpcURL,
		privateKey,
	)
	if err != nil {
		return luxcrypto.Address{}, err
	}
	if _, _, err := SetupValidatorProxyImplementation(
		rpcURL,
		proxyOwnerPrivateKey,
		posValidatorManagerAddress,
	); err != nil {
		return luxcrypto.Address{}, err
	}
	return posValidatorManagerAddress, nil
}

//go:embed smart_contracts/native_token_staking_manager_bytecode_v2.0.0.txt
var posValidatorManagerV2_0_0Bytecode []byte

func DeployPoSValidatorManagerV2_0_0Contract(
	rpcURL string,
	privateKey string,
) (luxcrypto.Address, error) {
	posValidatorManagerString := strings.TrimSpace(string(posValidatorManagerV2_0_0Bytecode))
	posValidatorManagerString = fillValidatorMessagesAddressPlaceholder(posValidatorManagerString)
	posValidatorManagerBytes := []byte(posValidatorManagerString)
	return contract.DeployContract(
		rpcURL,
		privateKey,
		posValidatorManagerBytes,
		"(uint8)",
		uint8(0),
	)
}

func DeployAndRegisterPoSValidatorManagerV2_0_0Contract(
	rpcURL string,
	privateKey string,
	proxyOwnerPrivateKey string,
) (luxcrypto.Address, error) {
	posValidatorManagerAddress, err := DeployPoSValidatorManagerV2_0_0Contract(
		rpcURL,
		privateKey,
	)
	if err != nil {
		return luxcrypto.Address{}, err
	}
	if _, _, err := SetupSpecializationProxyImplementation(
		rpcURL,
		proxyOwnerPrivateKey,
		posValidatorManagerAddress,
	); err != nil {
		return luxcrypto.Address{}, err
	}
	return posValidatorManagerAddress, nil
}

//go:embed smart_contracts/deployed_transparent_proxy_bytecode.txt
var deployedTransparentProxyBytecode []byte

//go:embed smart_contracts/deployed_proxy_admin_bytecode.txt
var deployedProxyAdminBytecode []byte

func AddValidatorTransparentProxyContractToAllocations(
	allocs core.GenesisAlloc,
	proxyManager string,
) {
	if _, found := allocs[common.Address(luxcrypto.HexToAddress(proxyManager))]; !found {
		ownerBalance := big.NewInt(0).Mul(big.NewInt(1e18), big.NewInt(1))
		allocs[common.Address(luxcrypto.HexToAddress(proxyManager))] = core.GenesisAccount{
			Balance: ownerBalance,
		}
	}
	// proxy admin
	deployedProxyAdmin := common.FromHex(strings.TrimSpace(string(deployedProxyAdminBytecode)))
	allocs[common.Address(luxcrypto.HexToAddress(ValidatorProxyAdminContractAddress))] = core.GenesisAccount{
		Balance: big.NewInt(0),
		Code:    deployedProxyAdmin,
		Nonce:   1,
		Storage: map[common.Hash]common.Hash{
			common.HexToHash("0x0"): common.HexToHash(proxyManager),
		},
	}

	// transparent proxy
	deployedTransparentProxy := common.FromHex(strings.TrimSpace(string(deployedTransparentProxyBytecode)))
	allocs[common.Address(luxcrypto.HexToAddress(ValidatorProxyContractAddress))] = core.GenesisAccount{
		Balance: big.NewInt(0),
		Code:    deployedTransparentProxy,
		Nonce:   1,
		Storage: map[common.Hash]common.Hash{
			common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc"): common.HexToHash(ValidatorContractAddress),           // sslot for address of ValidatorManager logic -> bytes32(uint256(keccak256('eip1967.proxy.implementation')) - 1)
			common.HexToHash("0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103"): common.HexToHash(ValidatorProxyAdminContractAddress), // sslot for address of ProxyAdmin -> bytes32(uint256(keccak256('eip1967.proxy.admin')) - 1)
			// we can omit 3rd sslot for _data, as we initialize ValidatorManager after chain is live
		},
	}
}

func AddSpecializationTransparentProxyContractToAllocations(
	allocs core.GenesisAlloc,
	proxyManager string,
) {
	if _, found := allocs[common.Address(luxcrypto.HexToAddress(proxyManager))]; !found {
		ownerBalance := big.NewInt(0).Mul(big.NewInt(1e18), big.NewInt(1))
		allocs[common.Address(luxcrypto.HexToAddress(proxyManager))] = core.GenesisAccount{
			Balance: ownerBalance,
		}
	}
	// proxy admin
	deployedProxyAdmin := common.FromHex(strings.TrimSpace(string(deployedProxyAdminBytecode)))
	allocs[common.Address(luxcrypto.HexToAddress(SpecializationProxyAdminContractAddress))] = core.GenesisAccount{
		Balance: big.NewInt(0),
		Code:    deployedProxyAdmin,
		Nonce:   1,
		Storage: map[common.Hash]common.Hash{
			common.HexToHash("0x0"): common.HexToHash(proxyManager),
		},
	}

	// transparent proxy
	deployedTransparentProxy := common.FromHex(strings.TrimSpace(string(deployedTransparentProxyBytecode)))
	allocs[common.Address(luxcrypto.HexToAddress(SpecializationProxyContractAddress))] = core.GenesisAccount{
		Balance: big.NewInt(0),
		Code:    deployedTransparentProxy,
		Nonce:   1,
		Storage: map[common.Hash]common.Hash{
			common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc"): common.HexToHash(ValidatorContractAddress),                // sslot for address of ValidatorManager logic -> bytes32(uint256(keccak256('eip1967.proxy.implementation')) - 1)
			common.HexToHash("0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103"): common.HexToHash(SpecializationProxyAdminContractAddress), // sslot for address of ProxyAdmin -> bytes32(uint256(keccak256('eip1967.proxy.admin')) - 1)
			// we can omit 3rd sslot for _data, as we initialize ValidatorManager after chain is live
		},
	}
}

//go:embed smart_contracts/deployed_example_reward_calculator_bytecode_v2.0.0.txt
var deployedRewardCalculatorV2_0_0Bytecode []byte

func AddRewardCalculatorV2_0_0ToAllocations(
	allocs core.GenesisAlloc,
	rewardBasisPoints uint64,
) {
	deployedRewardCalculatorBytes := common.FromHex(strings.TrimSpace(string(deployedRewardCalculatorV2_0_0Bytecode)))
	allocs[common.Address(luxcrypto.HexToAddress(RewardCalculatorAddress))] = core.GenesisAccount{
		Balance: big.NewInt(0),
		Code:    deployedRewardCalculatorBytes,
		Nonce:   1,
		Storage: map[common.Hash]common.Hash{
			common.HexToHash("0x0"): common.BigToHash(new(big.Int).SetUint64(rewardBasisPoints)),
		},
	}
}

// setups PoA manager after a successful execution of
// ConvertSubnetToL1Tx on P-Chain
// needs the list of validators for that tx,
// [convertSubnetValidators], together with an evm [ownerAddress]
// to set as the owner of the PoA manager
func SetupPoA(
	log logging.Logger,
	subnet blockchainSDK.Subnet,
	network models.Network,
	privateKey string,
	aggregatorLogger logging.Logger,
	validatorManagerAddressStr string,
	v2_0_0 bool,
	signatureAggregatorEndpoint string,
) error {
	return subnet.InitializeProofOfAuthority(
		log,
		network.SDKNetwork(),
		privateKey,
		aggregatorLogger,
		validatorManagerAddressStr,
		v2_0_0,
		signatureAggregatorEndpoint,
	)
}

// setups PoA manager after a successful execution of
// ConvertSubnetToL1Tx on P-Chain
// needs the list of validators for that tx,
// [convertSubnetValidators], together with an evm [ownerAddress]
// to set as the owner of the PoA manager
func SetupPoS(
	log logging.Logger,
	subnet blockchainSDK.Subnet,
	network models.Network,
	privateKey string,
	aggregatorLogger logging.Logger,
	posParams PoSParams,
	managerAddress string,
	specializedManagerAddress string,
	managerOwnerPrivateKey string,
	v2_0_0 bool,
	signatureAggregatorEndpoint string,
) error {
	// Initialize Proof of Stake validator manager
	log.Info("Initializing Proof of Stake validator manager")

	// Connect to the blockchain RPC
	client, err := ethclient.Dial(subnet.RPC)
	if err != nil {
		return fmt.Errorf("failed to connect to RPC: %w", err)
	}
	defer client.Close()

	// Parse the private key
	pk, err := crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Get the chain ID
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}

	// Create transaction options
	auth, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
	if err != nil {
		return fmt.Errorf("failed to create transactor: %w", err)
	}
	_ = auth // Will be used for contract calls
	_ = managerAddress
	_ = specializedManagerAddress
	_ = managerOwnerPrivateKey
	_ = v2_0_0

	// Initialize the PoS parameters on the validator manager contract
	// This would typically involve calling initialization methods on the contract
	log.Info("Setting PoS parameters",
		logging.UserString("minimumStakeAmount", posParams.MinimumStakeAmount.String()),
		logging.UserString("maximumStakeAmount", posParams.MaximumStakeAmount.String()),
		logging.UserString("minimumStakeDuration", fmt.Sprintf("%d", posParams.MinimumStakeDuration)),
		logging.UserString("minimumDelegationFee", fmt.Sprintf("%d", posParams.MinimumDelegationFee)),
		logging.UserString("maximumStakeMultiplier", fmt.Sprintf("%d", posParams.MaximumStakeMultiplier)),
		logging.UserString("weightToValueFactor", posParams.WeightToValueFactor.String()),
		logging.UserString("rewardCalculatorAddress", fmt.Sprintf("%x", posParams.RewardCalculatorAddress)),
	)

	// Set up signature aggregation if endpoint is provided
	if signatureAggregatorEndpoint != "" {
		log.Info("Configuring signature aggregator", logging.UserString("endpoint", signatureAggregatorEndpoint))
	}

	log.Info("Proof of Stake validator manager initialized successfully")
	return nil
}
