// Copyright (C) 2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.
package contract

import (
	"fmt"
	"math/big"

	"github.com/luxfi/crypto"
	"github.com/luxfi/evm/precompile/contracts/nativeminter"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/application"
	"github.com/luxfi/sdk/key"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/utils"
)

// returns information for the blockchain default allocation key
// if found, returns
// key name, address, private key
func GetDefaultBlockchainAirdropKeyInfo(
	app *application.Lux,
	blockchainName string,
) (string, string, string, error) {
	keyName := utils.GetDefaultBlockchainAirdropKeyName(blockchainName)
	keyPath := app.GetKeyPath(keyName)
	if utils.FileExists(keyPath) {
		k, err := key.LoadSoft(models.NewLocalNetwork().ID(), keyPath)
		if err != nil {
			return "", "", "", err
		}
		return keyName, k.C(), k.PrivKeyHex(), nil
	}
	return "", "", "", nil
}

// from a given genesis, look for known private keys inside it, giving
// preference to the ones expected to be default
// it searches for:
// 1) default CLI allocation key for blockchains
// 2) ewoq
// 3) all other stored keys managed by CLI
// returns address + private key when found
func GetBlockchainAirdropKeyInfo(
	app *application.Lux,
	network models.Network,
	blockchainName string,
	genesisData []byte,
) (string, string, string, error) {
	genesis, err := utils.ByteSliceToEVMGenesis(genesisData)
	if err != nil {
		return "", "", "", err
	}
	if blockchainName != "" {
		airdropKeyName, airdropAddress, airdropPrivKey, err := GetDefaultBlockchainAirdropKeyInfo(app, blockchainName)
		if err != nil {
			return "", "", "", err
		}
		for address := range genesis.Alloc {
			if address == airdropAddress {
				return airdropKeyName, airdropAddress, airdropPrivKey, nil
			}
		}
	}
	// Try to load ewoq key
	ewoqPath := app.GetKeyPath("ewoq")
	if utils.FileExists(ewoqPath) {
		ewoq, err := key.LoadSoft(network.ID(), ewoqPath)
		if err == nil {
			for address := range genesis.Alloc {
				if address == ewoq.C() {
					return "ewoq", ewoq.C(), ewoq.PrivKeyHex(), nil
				}
			}
		}
	}
	maxBalance := big.NewInt(0)
	maxBalanceKeyName := ""
	maxBalanceAddr := ""
	maxBalancePrivKey := ""
	for address, alloc := range genesis.Alloc {
		balance := alloc.GetBalance()
		if balance == nil || balance.Cmp(big.NewInt(0)) == 0 {
			continue
		}
		// address is already a string
		found, keyName, addressStr, privKey, err := SearchForManagedKey(app, network, address, false)
		if err != nil {
			return "", "", "", err
		}
		if found && balance.Cmp(maxBalance) > 0 {
			maxBalance = balance
			maxBalanceKeyName = keyName
			maxBalanceAddr = addressStr
			maxBalancePrivKey = privKey
		}
	}
	return maxBalanceKeyName, maxBalanceAddr, maxBalancePrivKey, nil
}

func SearchForManagedKey(
	app *application.Lux,
	network models.Network,
	address string,
	includeEwoq bool,
) (bool, string, string, string, error) {
	keyNames, err := utils.GetKeyNames(app.GetKeyDir(), includeEwoq)
	if err != nil {
		return false, "", "", "", err
	}
	for _, keyName := range keyNames {
		keyPath := app.GetKeyPath(keyName)
		if k, err := key.LoadSoft(network.ID(), keyPath); err != nil {
			return false, "", "", "", err
		} else if address == k.C() {
			return true, keyName, k.C(), k.PrivKeyHex(), nil
		}
	}
	return false, "", "", "", nil
}

// get the deployed blockchain genesis, and then look for known
// private keys inside it
// returns address + private key when found
func GetEVMSubnetPrefundedKey(
	app *application.Lux,
	network models.Network,
	chainSpec ChainSpec,
) (string, string, error) {
	genesisData, err := GetBlockchainGenesis(
		app,
		network,
		chainSpec,
	)
	if err != nil {
		return "", "", err
	}
	if !utils.ByteSliceIsEVMGenesis(genesisData) {
		return "", "", fmt.Errorf("search for prefunded key is only supported on EVM based vms")
	}
	_, genesisAddress, genesisPrivateKey, err := GetBlockchainAirdropKeyInfo(
		app,
		network,
		chainSpec.BlockchainName,
		genesisData,
	)
	if err != nil {
		return "", "", err
	}
	return genesisAddress, genesisPrivateKey, nil
}

// get the deployed blockchain genesis
func GetBlockchainGenesis(
	app *application.Lux,
	network models.Network,
	chainSpec ChainSpec,
) ([]byte, error) {
	blockchainID, err := GetBlockchainID(app, network, chainSpec)
	if err != nil {
		return nil, err
	}
	// Retrieve the blockchain creation transaction to get the genesis data
	createChainTxIntf, err := utils.GetBlockchainTx(network.Endpoint(), blockchainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get blockchain tx: %w", err)
	}
	createChainTx, ok := createChainTxIntf.(*txs.CreateChainTx)
	if !ok {
		return nil, fmt.Errorf("expected CreateChainTx, got %T", createChainTxIntf)
	}
	return createChainTx.GenesisData, nil
}

func sumGenesisSupply(
	genesisData []byte,
) (*big.Int, error) {
	sum := new(big.Int)
	genesis, err := utils.ByteSliceToEVMGenesis(genesisData)
	if err != nil {
		return sum, err
	}
	for _, allocation := range genesis.Alloc {
		sum.Add(sum, allocation.GetBalance())
	}
	return sum, nil
}

func GetEVMSubnetGenesisSupply(
	app *application.Lux,
	network models.Network,
	chainSpec ChainSpec,
) (*big.Int, error) {
	genesisData, err := GetBlockchainGenesis(
		app,
		network,
		chainSpec,
	)
	if err != nil {
		return nil, err
	}
	if !utils.ByteSliceIsEVMGenesis(genesisData) {
		return nil, fmt.Errorf("genesis supply calculation is only supported on EVM based vms")
	}
	return sumGenesisSupply(genesisData)
}

func getGenesisNativeMinterAdmin(
	app *application.Lux,
	network models.Network,
	genesisData []byte,
) (bool, bool, string, string, string, error) {
	genesis, err := utils.ByteSliceToEVMGenesis(genesisData)
	if err != nil {
		return false, false, "", "", "", err
	}

	// Check if native minter is configured in the genesis
	// Parse the genesis config to check for native minter precompile
	if genesis.Config != nil {
		// Check for native minter in the genesis allocations
		// The native minter precompile is at a specific address
		nativeMinterAddress := common.HexToAddress("0x0200000000000000000000000000000000000001")

		if alloc, exists := genesis.Alloc[nativeMinterAddress.Hex()]; exists && len(alloc.GetCode()) > 0 {
			// Native minter is configured
			// Look for admin addresses in the genesis state
			for addr := range genesis.Alloc {
				// Check if this address has admin permissions
				// Admin addresses typically have a non-zero balance or specific code
				if addr != nativeMinterAddress.Hex() {
					adminStr := addr
					found, keyName, addressStr, privKey, err := SearchForManagedKey(app, network, adminStr, true)
					if err != nil {
						return false, false, "", "", "", err
					}
					if found {
						return true, true, keyName, addressStr, privKey, nil
					}
				}
			}
			// Native minter exists but no managed key found
			if len(genesis.Alloc) > 1 {
				for addr := range genesis.Alloc {
					if addr != nativeMinterAddress.Hex() {
						return true, false, "", addr, "", nil
					}
				}
			}
		}
	}
	return false, false, "", "", "", nil
}

func getGenesisNativeMinterManager(
	app *application.Lux,
	network models.Network,
	genesisData []byte,
) (bool, bool, string, string, string, error) {
	_, err := utils.ByteSliceToEVMGenesis(genesisData)
	if err != nil {
		return false, false, "", "", "", err
	}
	// TODO: Fix GenesisPrecompiles access - it's not in params.ChainConfig
	// Need to use extras.ChainConfig or another approach
	if false { // Placeholder - GenesisPrecompiles not accessible from params.ChainConfig
		var allowListCfg *nativeminter.Config
		_ = allowListCfg
		if len(allowListCfg.AllowListConfig.ManagerAddresses) == 0 {
			return false, false, "", "", "", nil
		}
		for _, admin := range allowListCfg.AllowListConfig.ManagerAddresses {
			// Convert address to string
			adminStr := fmt.Sprintf("0x%x", admin.Bytes())
			found, keyName, addressStr, privKey, err := SearchForManagedKey(app, network, adminStr, true)
			if err != nil {
				return false, false, "", "", "", err
			}
			if found {
				return true, true, keyName, addressStr, privKey, nil
			}
		}
		return true, false, "", allowListCfg.AllowListConfig.ManagerAddresses[0].Hex(), "", nil
	}
	return false, false, "", "", "", nil
}

func GetEVMSubnetGenesisNativeMinterAdmin(
	app *application.Lux,
	network models.Network,
	chainSpec ChainSpec,
) (bool, bool, string, string, string, error) {
	genesisData, err := GetBlockchainGenesis(
		app,
		network,
		chainSpec,
	)
	if err != nil {
		return false, false, "", "", "", err
	}
	if !utils.ByteSliceIsEVMGenesis(genesisData) {
		return false, false, "", "", "", fmt.Errorf("genesis native minter admin query is only supported on EVM based vms")
	}
	return getGenesisNativeMinterAdmin(app, network, genesisData)
}

func GetEVMSubnetGenesisNativeMinterManager(
	app *application.Lux,
	network models.Network,
	chainSpec ChainSpec,
) (bool, bool, string, string, string, error) {
	genesisData, err := GetBlockchainGenesis(
		app,
		network,
		chainSpec,
	)
	if err != nil {
		return false, false, "", "", "", err
	}
	if !utils.ByteSliceIsEVMGenesis(genesisData) {
		return false, false, "", "", "", fmt.Errorf("genesis native minter manager query is only supported on EVM based vms")
	}
	return getGenesisNativeMinterManager(app, network, genesisData)
}

func ContractAddressIsInGenesisData(
	genesisData []byte,
	contractAddress crypto.Address,
) (bool, error) {
	genesis, err := utils.ByteSliceToEVMGenesis(genesisData)
	if err != nil {
		return false, err
	}
	// Check if it's a valid EVM genesis
	if genesis.Config == nil {
		return false, fmt.Errorf("invalid EVM genesis: missing config")
	}
	// Convert contractAddress to string for comparison
	// Try both with and without 0x prefix
	contractAddrStr := fmt.Sprintf("%x", contractAddress.Bytes())
	contractAddrStrWithPrefix := fmt.Sprintf("0x%x", contractAddress.Bytes())
	for address, allocation := range genesis.Alloc {
		if address == contractAddrStr || address == contractAddrStrWithPrefix {
			code := allocation.GetCode()
			return len(code) > 0, nil
		}
	}
	return false, nil
}
