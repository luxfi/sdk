// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package contract

import (
	"fmt"
	"math/big"

	"github.com/luxfi/sdk/application"
	"github.com/luxfi/sdk/key"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/crypto"
	"github.com/luxfi/evm/precompile/contracts/nativeminter"
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
	genesis, err := utils.ByteSliceToSubnetEvmGenesis(genesisData)
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
		if alloc.Balance == nil {
			continue
		}
		// address is already a string
		found, keyName, addressStr, privKey, err := SearchForManagedKey(app, network, address, false)
		if err != nil {
			return "", "", "", err
		}
		if found && alloc.Balance.Cmp(maxBalance) > 0 {
			maxBalance = alloc.Balance
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
	if !utils.ByteSliceIsSubnetEvmGenesis(genesisData) {
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
	_, err := GetBlockchainID(app, network, chainSpec)
	if err != nil {
		return nil, err
	}
	// GetBlockchainTx is not implemented, return error for now
	// TODO: Implement GetBlockchainTx to retrieve genesis data from network
	return nil, fmt.Errorf("GetBlockchainTx not yet implemented")
}

func sumGenesisSupply(
	genesisData []byte,
) (*big.Int, error) {
	sum := new(big.Int)
	genesis, err := utils.ByteSliceToSubnetEvmGenesis(genesisData)
	if err != nil {
		return sum, err
	}
	for _, allocation := range genesis.Alloc {
		sum.Add(sum, allocation.Balance)
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
	if !utils.ByteSliceIsSubnetEvmGenesis(genesisData) {
		return nil, fmt.Errorf("genesis supply calculation is only supported on EVM based vms")
	}
	return sumGenesisSupply(genesisData)
}

func getGenesisNativeMinterAdmin(
	app *application.Lux,
	network models.Network,
	genesisData []byte,
) (bool, bool, string, string, string, error) {
	_, err := utils.ByteSliceToSubnetEvmGenesis(genesisData)
	if err != nil {
		return false, false, "", "", "", err
	}
	// TODO: Fix GenesisPrecompiles access - it's not in params.ChainConfig
	// Need to use extras.ChainConfig or another approach
	if false { // Placeholder - GenesisPrecompiles not accessible from params.ChainConfig
		var allowListCfg *nativeminter.Config
		_ = allowListCfg
		if len(allowListCfg.AllowListConfig.AdminAddresses) == 0 {
			return false, false, "", "", "", nil
		}
		for _, admin := range allowListCfg.AllowListConfig.AdminAddresses {
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
		return true, false, "", allowListCfg.AllowListConfig.AdminAddresses[0].Hex(), "", nil
	}
	return false, false, "", "", "", nil
}

func getGenesisNativeMinterManager(
	app *application.Lux,
	network models.Network,
	genesisData []byte,
) (bool, bool, string, string, string, error) {
	_, err := utils.ByteSliceToSubnetEvmGenesis(genesisData)
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
	if !utils.ByteSliceIsSubnetEvmGenesis(genesisData) {
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
	if !utils.ByteSliceIsSubnetEvmGenesis(genesisData) {
		return false, false, "", "", "", fmt.Errorf("genesis native minter manager query is only supported on EVM based vms")
	}
	return getGenesisNativeMinterManager(app, network, genesisData)
}

func ContractAddressIsInGenesisData(
	genesisData []byte,
	contractAddress crypto.Address,
) (bool, error) {
	genesis, err := utils.ByteSliceToSubnetEvmGenesis(genesisData)
	if err != nil {
		return false, err
	}
	// Convert contractAddress to string for comparison
	contractAddrStr := fmt.Sprintf("0x%x", contractAddress.Bytes())
	for address, allocation := range genesis.Alloc {
		if address == contractAddrStr {
			return len(allocation.Code) > 0, nil
		}
	}
	return false, nil
}
