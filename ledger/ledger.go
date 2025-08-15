// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package ledger

import (
	"fmt"

	"github.com/luxfi/sdk/network"
	"github.com/luxfi/sdk/utils"

	"github.com/luxfi/ids"
	luxledger "github.com/luxfi/ledger-lux-go"
	"github.com/luxfi/node/version"
	"github.com/luxfi/node/vms/platformvm"
)

const (
	maxIndexToSearch           = 1000
	maxIndexToSearchForBalance = 100
)

type LedgerDevice struct {
	device *luxledger.LedgerLux
}

func New() (*LedgerDevice, error) {
	// Open connection to Ledger device
	luxDevice, err := luxledger.FindLedgerLuxApp()
	if err != nil {
		return nil, fmt.Errorf("failed to find Ledger device: %w", err)
	}

	return &LedgerDevice{
		device: luxDevice,
	}, nil
}

// Version returns the version of the ledger device
func (dev *LedgerDevice) Version() (v *version.Semantic, err error) {
	if dev.device != nil {
		info, err := dev.device.GetVersion()
		if err != nil {
			return nil, err
		}
		// Convert ledger version info to semantic version
		return &version.Semantic{
			Major: int(info.Major),
			Minor: int(info.Minor),
			Patch: int(info.Patch),
		}, nil
	}
	return nil, fmt.Errorf("device not connected")
}

// Address returns the address at the given index
func (dev *LedgerDevice) Address(hrp string, index uint32) (ids.ShortID, error) {
	path := fmt.Sprintf("m/44'/9000'/0'/0/%d", index)
	resp, err := dev.device.GetPubKey(path, false, hrp, "P")
	if err != nil {
		return ids.ShortEmpty, err
	}
	
	// Parse address to ID
	addrID, err := ids.ShortFromString(resp.Address)
	if err != nil {
		return ids.ShortEmpty, fmt.Errorf("failed to parse address: %w", err)
	}
	
	return addrID, nil
}

// Addresses returns addresses for the given indices
func (dev *LedgerDevice) Addresses(indices []uint32) ([]ids.ShortID, error) {
	addresses := make([]ids.ShortID, len(indices))
	for i, index := range indices {
		// Use default hrp "lux" for platform chain
		addr, err := dev.Address("lux", index)
		if err != nil {
			return nil, err
		}
		addresses[i] = addr
	}
	return addresses, nil
}

func (dev *LedgerDevice) FindAddresses(addresses []string, maxIndex uint32) (map[string]uint32, error) {
	// for all ledger indices to search for, find if the ledger address belongs to the input
	// addresses and, if so, add an index association to indexMap.
	// breaks the loop if all addresses were found
	if maxIndex == 0 {
		maxIndex = maxIndexToSearch
	}
	indices := map[string]uint32{}
	for index := uint32(0); index < maxIndex; index++ {
		// Get the address from ledger at this index
		path := fmt.Sprintf("m/44'/9000'/0'/0/%d", index)
		resp, err := dev.device.GetPubKey(path, false, "lux", "P")
		if err != nil {
			return nil, err
		}

		// Check if this address matches any of our target addresses
		for i, targetAddr := range addresses {
			if resp.Address == targetAddr {
				indices[addresses[i]] = index
			}
		}

		if len(indices) == len(addresses) {
			break
		}
	}
	return indices, nil
}

// FindFunds searches for a set of indices that pay a given amount
func (dev *LedgerDevice) FindFunds(
	network network.Network,
	amount uint64,
	maxIndex uint32,
) ([]uint32, error) {
	// Use the first node's endpoint
	endpoint := ""
	if len(network.Nodes) > 0 && network.Nodes[0] != nil {
		endpoint = network.Nodes[0].Endpoint
	}
	pClient := platformvm.NewClient(endpoint)
	totalBalance := uint64(0)
	indices := []uint32{}
	if maxIndex == 0 {
		maxIndex = maxIndexToSearchForBalance
	}
	for index := uint32(0); index < maxIndex; index++ {
		// Get the address from ledger at this index
		path := fmt.Sprintf("m/44'/9000'/0'/0/%d", index)
		resp, err := dev.device.GetPubKey(path, false, "lux", "P")
		if err != nil {
			return []uint32{}, err
		}

		// Parse address to ID for balance check
		addrID, err := ids.ShortFromString(resp.Address)
		if err != nil {
			return nil, fmt.Errorf("failed to parse address: %w", err)
		}

		ctx, cancel := utils.GetAPIContext()
		balanceResp, err := pClient.GetBalance(ctx, []ids.ShortID{addrID})
		cancel()
		if err != nil {
			return nil, err
		}
		if balanceResp.Balance > 0 {
			totalBalance += uint64(balanceResp.Balance)
			indices = append(indices, index)
		}
		if totalBalance >= amount {
			break
		}
	}
	if totalBalance < amount {
		return nil, fmt.Errorf("not enough funds on ledger")
	}
	return indices, nil
}

// GetAddresses returns Lux addresses for the given indices
func (dev *LedgerDevice) GetAddresses(indices []uint32, hrp string, chainID string) ([]string, error) {
	addresses := make([]string, len(indices))
	for i, index := range indices {
		path := fmt.Sprintf("m/44'/9000'/0'/0/%d", index)
		resp, err := dev.device.GetPubKey(path, false, hrp, chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to get address at index %d: %w", index, err)
		}
		addresses[i] = resp.Address
	}
	return addresses, nil
}

// Disconnect closes the connection to the ledger device
func (dev *LedgerDevice) Disconnect() error {
	if dev.device != nil {
		return dev.device.Close()
	}
	return nil
}

// SignHash signs a hash with the ledger device for multiple indices
func (dev *LedgerDevice) SignHash(hash []byte, indices []uint32) ([][]byte, error) {
	return dev.Sign(hash, indices)
}

// Sign signs a transaction with the ledger device for multiple indices
func (dev *LedgerDevice) Sign(hash []byte, indices []uint32) ([][]byte, error) {
	signatures := make([][]byte, len(indices))
	for i, index := range indices {
		path := fmt.Sprintf("m/44'/9000'/0'/0/%d", index)
		// For Lux ledger, we need signing paths and change paths
		signingPaths := []string{path}
		changePaths := []string{} // No change paths for simple signature

		resp, err := dev.device.Sign(path, signingPaths, hash, changePaths)
		if err != nil {
			return nil, fmt.Errorf("failed to sign with ledger at index %d: %w", index, err)
		}

		// Extract the signature from response map
		if len(resp.Signature) == 0 {
			return nil, fmt.Errorf("no signature returned from ledger for index %d", index)
		}

		// Get the signature for the requested path
		sig, ok := resp.Signature[path]
		if !ok {
			return nil, fmt.Errorf("signature not found for path %s", path)
		}
		signatures[i] = sig
	}

	return signatures, nil
}

// Close closes the connection to the ledger device
func (dev *LedgerDevice) Close() error {
	if dev.device != nil {
		return dev.device.Close()
	}
	return nil
}
