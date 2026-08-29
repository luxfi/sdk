// Copyright (C) 2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

// Package multisig provides multi-signature transaction support.
package multisig

import (
	"context"
	"fmt"

	"github.com/luxfi/sdk/platformvm"

	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/sdk/network"
	"github.com/luxfi/utxo/secp256k1fx"
	"github.com/luxfi/vm/components/verify"

	"github.com/luxfi/ids"
	"github.com/luxfi/proto/p/txs"
)

type TxKind int64

var ErrUndefinedTx = fmt.Errorf("tx is undefined")

const (
	Undefined TxKind = iota
	PChainRemoveChainValidatorTx
	PChainAddChainValidatorTx
	PChainCreateChainTx
	PChainTransformChainTx
	PChainAddPermissionlessValidatorTx
	PChainTransferChainOwnershipTx
)

type Multisig struct {
	PChainTx    *txs.Tx
	controlKeys []ids.ShortID
	threshold   uint32
}

func New(pChainTx *txs.Tx) *Multisig {
	ms := Multisig{
		PChainTx: pChainTx,
	}
	return &ms
}

func (ms *Multisig) String() string {
	if ms.PChainTx != nil {
		return ms.PChainTx.ID().String()
	}
	return ""
}

func (ms *Multisig) Undefined() bool {
	return ms.PChainTx == nil
}

func (ms *Multisig) ToBytes() ([]byte, error) {
	if ms.Undefined() {
		return nil, ErrUndefinedTx
	}
	// The signed tx already holds its ZAP wire bytes.
	return ms.PChainTx.Bytes(), nil
}

func (ms *Multisig) FromBytes(txBytes []byte) error {
	// Parse binds the tx from its own wire bytes, cached bytes and ID
	// included, so there is nothing left to initialize.
	tx, err := txs.Parse(txBytes)
	if err != nil {
		return fmt.Errorf("error parsing signed tx: %w", err)
	}
	ms.PChainTx = tx
	return nil
}

func (ms *Multisig) IsReadyToCommit() (bool, error) {
	if ms.Undefined() {
		return false, ErrUndefinedTx
	}
	unsignedTx := ms.PChainTx.Unsigned
	switch unsignedTx.(type) {
	case *txs.CreateChainTx:
		return true, nil
	default:
	}
	_, remainingSigners, err := ms.GetRemainingAuthSigners()
	if err != nil {
		return false, err
	}
	return len(remainingSigners) == 0, nil
}

// GetRemainingAuthSigners gets chain auth addresses that have not signed a given tx
//   - get the string slice of auth signers for the tx (GetAuthSigners)
//   - verifies that all creds in tx.Creds, except the last one, are fully signed
//     (a cred is fully signed if all the signatures in cred.Sigs are non-empty)
//   - computes remaining signers by iterating the last cred in tx.Creds, associated to chain auth signing
//   - for each sig in cred.Sig: if sig is empty, then add the associated auth signer address (obtained from
//     authSigners by using the index) to the remaining signers list
//
// if the tx is fully signed, returns empty slice
func (ms *Multisig) GetRemainingAuthSigners() ([]ids.ShortID, []ids.ShortID, error) {
	if ms.Undefined() {
		return nil, nil, ErrUndefinedTx
	}
	authSigners, err := ms.GetAuthSigners()
	if err != nil {
		return nil, nil, err
	}
	emptySig := [secp256k1.SignatureLen]byte{}
	numCreds := len(ms.PChainTx.Creds)
	// we should have at least 1 cred for output owners and 1 cred for chain auth
	if numCreds < 2 {
		return nil, nil, fmt.Errorf("expected tx.Creds of len 2, got %d. doesn't seem to be a multisig tx with chain auth requirements", numCreds)
	}
	// signatures for output owners should be filled (all creds except last one)
	for credIndex := range ms.PChainTx.Creds[:numCreds-1] {
		cred, ok := ms.PChainTx.Creds[credIndex].(*secp256k1fx.Credential)
		if !ok {
			return nil, nil, fmt.Errorf("expected cred to be of type *secp256k1fx.Credential, got %T", ms.PChainTx.Creds[credIndex])
		}
		for i, sig := range cred.Sigs {
			if sig == emptySig {
				return nil, nil, fmt.Errorf("expected funding sig %d of cred %d to be filled", i, credIndex)
			}
		}
	}
	// signatures for chain auth (last cred)
	cred, ok := ms.PChainTx.Creds[numCreds-1].(*secp256k1fx.Credential)
	if !ok {
		return nil, nil, fmt.Errorf("expected cred to be of type *secp256k1fx.Credential, got %T", ms.PChainTx.Creds[1])
	}
	if len(cred.Sigs) != len(authSigners) {
		return nil, nil, fmt.Errorf("expected number of cred's signatures %d to equal number of auth signers %d",
			len(cred.Sigs),
			len(authSigners),
		)
	}
	remainingSigners := []ids.ShortID{}
	for i, sig := range cred.Sigs {
		if sig == emptySig {
			remainingSigners = append(remainingSigners, authSigners[i])
		}
	}
	return authSigners, remainingSigners, nil
}

// GetAuthSigners gets all chain auth addresses that are required to sign a given tx
//   - get chain control keys as string slice using P-Chain API (GetOwners)
//   - get chain auth indices from the tx, field tx.UnsignedTx.ChainAuth
//   - creates the string slice of required chain auth addresses by applying
//     the indices to the control keys slice
func (ms *Multisig) GetAuthSigners() ([]ids.ShortID, error) {
	if ms.Undefined() {
		return nil, ErrUndefinedTx
	}
	controlKeys, _, err := ms.GetNetOwners()
	if err != nil {
		return nil, err
	}
	unsignedTx := ms.PChainTx.Unsigned
	var netAuth verify.Verifiable
	switch unsignedTx := unsignedTx.(type) {
	case *txs.RemoveChainValidatorTx:
		netAuth = unsignedTx.ChainAuth
	case *txs.AddChainValidatorTx:
		netAuth = unsignedTx.ChainAuth
	case *txs.CreateChainTx:
		netAuth = unsignedTx.ChainAuth
	case *txs.TransformChainTx:
		netAuth = unsignedTx.ChainAuth
	case *txs.TransferChainOwnershipTx:
		netAuth = unsignedTx.ChainAuth
	default:
		return nil, fmt.Errorf("unexpected unsigned tx type %T", unsignedTx)
	}
	netInput, ok := netAuth.(*secp256k1fx.Input)
	if !ok {
		return nil, fmt.Errorf("expected netAuth of type *secp256k1fx.Input, got %T", netAuth)
	}
	authSigners := []ids.ShortID{}
	for _, sigIndex := range netInput.SigIndices {
		if sigIndex >= uint32(len(controlKeys)) {
			return nil, fmt.Errorf("signer index %d exceeds number of control keys", sigIndex)
		}
		authSigners = append(authSigners, controlKeys[sigIndex])
	}
	return authSigners, nil
}

func (*Multisig) GetSpendSigners() ([]ids.ShortID, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (ms *Multisig) GetTxKind() (TxKind, error) {
	if ms.Undefined() {
		return Undefined, ErrUndefinedTx
	}
	unsignedTx := ms.PChainTx.Unsigned
	switch unsignedTx := unsignedTx.(type) {
	case *txs.RemoveChainValidatorTx:
		return PChainRemoveChainValidatorTx, nil
	case *txs.AddChainValidatorTx:
		return PChainAddChainValidatorTx, nil
	case *txs.CreateChainTx:
		return PChainCreateChainTx, nil
	case *txs.TransformChainTx:
		return PChainTransformChainTx, nil
	case *txs.AddPermissionlessValidatorTx:
		return PChainAddPermissionlessValidatorTx, nil
	case *txs.TransferChainOwnershipTx:
		return PChainTransferChainOwnershipTx, nil
	default:
		return Undefined, fmt.Errorf("unexpected unsigned tx type %T", unsignedTx)
	}
}

// GetNetworkID gets network id associated to tx.
func (ms *Multisig) GetNetworkID() (uint32, error) {
	if ms.Undefined() {
		return 0, ErrUndefinedTx
	}
	unsignedTx := ms.PChainTx.Unsigned
	var networkID uint32
	switch unsignedTx := unsignedTx.(type) {
	case *txs.RemoveChainValidatorTx:
		networkID = unsignedTx.NetworkID
	case *txs.AddChainValidatorTx:
		networkID = unsignedTx.NetworkID
	case *txs.CreateChainTx:
		networkID = unsignedTx.NetworkID
	case *txs.TransformChainTx:
		networkID = unsignedTx.NetworkID
	case *txs.AddPermissionlessValidatorTx:
		networkID = unsignedTx.NetworkID
	case *txs.TransferChainOwnershipTx:
		networkID = unsignedTx.NetworkID
	default:
		return 0, fmt.Errorf("unexpected unsigned tx type %T", unsignedTx)
	}
	return networkID, nil
}

// GetNetwork gets network model associated to tx.
func (ms *Multisig) GetNetwork() (network.LegacyNetwork, error) {
	if ms.Undefined() {
		return network.UndefinedNetwork, ErrUndefinedTx
	}
	networkID, err := ms.GetNetworkID()
	if err != nil {
		return network.UndefinedNetwork, err
	}
	newNetwork := network.NetworkFromNetworkID(networkID)
	if newNetwork.Kind == network.Undefined {
		return network.UndefinedNetwork, fmt.Errorf("undefined network model for tx")
	}
	return newNetwork, nil
}

// GetBlockchainID is deprecated - the transaction types in luxfi/node/vms/platformvm/txs
// do not have BlockchainID fields. These transactions work with net IDs.
// Use GetChainID() instead for net-related operations.
//
// TODO: Remove this function or implement it correctly if blockchain IDs are needed
// func (ms *Multisig) GetBlockchainID() (ids.ID, error) {
// 	return ids.Empty, fmt.Errorf("GetBlockchainID is not implemented - transaction types do not have BlockchainID fields")
// }

// GetChainID gets net id associated to tx
func (ms *Multisig) GetChainID() (ids.ID, error) {
	if ms.Undefined() {
		return ids.Empty, ErrUndefinedTx
	}
	unsignedTx := ms.PChainTx.Unsigned
	var chainID ids.ID
	switch unsignedTx := unsignedTx.(type) {
	case *txs.RemoveChainValidatorTx:
		chainID = unsignedTx.Chain
	case *txs.AddChainValidatorTx:
		chainID = unsignedTx.Chain
	case *txs.CreateChainTx:
		chainID = unsignedTx.ValidateNetworkID
	case *txs.TransformChainTx:
		chainID = unsignedTx.Chain
	case *txs.AddPermissionlessValidatorTx:
		chainID = unsignedTx.Chain
	case *txs.TransferChainOwnershipTx:
		chainID = unsignedTx.Chain
	default:
		return ids.Empty, fmt.Errorf("unexpected unsigned tx type %T", unsignedTx)
	}
	return chainID, nil
}

func (ms *Multisig) GetNetOwners() ([]ids.ShortID, uint32, error) {
	if ms.Undefined() {
		return nil, 0, ErrUndefinedTx
	}
	if ms.controlKeys == nil {
		chainID, err := ms.GetChainID()
		if err != nil {
			return nil, 0, err
		}

		network, err := ms.GetNetwork()
		if err != nil {
			return nil, 0, err
		}
		controlKeys, threshold, err := GetOwners(network, chainID)
		if err != nil {
			return nil, 0, err
		}
		ms.controlKeys = controlKeys
		ms.threshold = threshold
	}
	return ms.controlKeys, ms.threshold, nil
}

func GetOwners(network network.LegacyNetwork, chainID ids.ID) ([]ids.ShortID, uint32, error) {
	pClient := platformvm.NewClient(network.Endpoint)
	ctx := context.Background()
	netResponse, err := pClient.GetNet(ctx, chainID)
	if err != nil {
		return nil, 0, fmt.Errorf("net tx %s query error: %w", chainID, err)
	}
	controlKeys := netResponse.ControlKeys
	threshold := netResponse.Threshold
	return controlKeys, threshold, nil
}

func (ms *Multisig) GetWrappedPChainTx() (*txs.Tx, error) {
	if ms.Undefined() {
		return nil, ErrUndefinedTx
	}
	return ms.PChainTx, nil
}
