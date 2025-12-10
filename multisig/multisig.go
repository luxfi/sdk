// Copyright (C) 2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.
package multisig

import (
	"context"
	"fmt"

	"github.com/luxfi/node/vms/platformvm"

	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/node/vms/components/verify"
	"github.com/luxfi/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/network"

	"github.com/luxfi/ids"
	"github.com/luxfi/node/vms/platformvm/txs"
)

type TxKind int64

var ErrUndefinedTx = fmt.Errorf("tx is undefined")

const (
	Undefined TxKind = iota
	PChainRemoveNetValidatorTx
	PChainAddNetValidatorTx
	PChainCreateChainTx
	PChainTransformNetTx
	PChainAddPermissionlessValidatorTx
	PChainTransferNetOwnershipTx
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
	txBytes, err := txs.Codec.Marshal(txs.CodecVersion, ms.PChainTx)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal signed tx: %w", err)
	}
	return txBytes, nil
}

func (ms *Multisig) FromBytes(txBytes []byte) error {
	var tx txs.Tx
	if _, err := txs.Codec.Unmarshal(txBytes, &tx); err != nil {
		return fmt.Errorf("error unmarshaling signed tx: %w", err)
	}
	if err := tx.Initialize(txs.Codec); err != nil {
		return fmt.Errorf("error initializing signed tx: %w", err)
	}
	ms.PChainTx = &tx
	return nil
}

func (ms *Multisig) IsReadyToCommit() (bool, error) {
	if ms.Undefined() {
		return false, ErrUndefinedTx
	}
	unsignedTx := ms.PChainTx.Unsigned
	switch unsignedTx.(type) {
	case *txs.CreateNetTx:
		return true, nil
	default:
	}
	_, remainingSigners, err := ms.GetRemainingAuthSigners()
	if err != nil {
		return false, err
	}
	return len(remainingSigners) == 0, nil
}

// GetRemainingAuthSigners gets subnet auth addresses that have not signed a given tx
//   - get the string slice of auth signers for the tx (GetAuthSigners)
//   - verifies that all creds in tx.Creds, except the last one, are fully signed
//     (a cred is fully signed if all the signatures in cred.Sigs are non-empty)
//   - computes remaining signers by iterating the last cred in tx.Creds, associated to subnet auth signing
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
	// we should have at least 1 cred for output owners and 1 cred for subnet auth
	if numCreds < 2 {
		return nil, nil, fmt.Errorf("expected tx.Creds of len 2, got %d. doesn't seem to be a multisig tx with subnet auth requirements", numCreds)
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
	// signatures for subnet auth (last cred)
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

// GetAuthSigners gets all subnet auth addresses that are required to sign a given tx
//   - get subnet control keys as string slice using P-Chain API (GetOwners)
//   - get subnet auth indices from the tx, field tx.UnsignedTx.SubnetAuth
//   - creates the string slice of required subnet auth addresses by applying
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
	case *txs.RemoveNetValidatorTx:
		netAuth = unsignedTx.NetAuth
	case *txs.AddNetValidatorTx:
		netAuth = unsignedTx.NetAuth
	case *txs.CreateChainTx:
		netAuth = unsignedTx.NetAuth
	case *txs.TransformNetTx:
		netAuth = unsignedTx.NetAuth
	case *txs.TransferNetOwnershipTx:
		netAuth = unsignedTx.NetAuth
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
	case *txs.RemoveNetValidatorTx:
		return PChainRemoveNetValidatorTx, nil
	case *txs.AddNetValidatorTx:
		return PChainAddNetValidatorTx, nil
	case *txs.CreateChainTx:
		return PChainCreateChainTx, nil
	case *txs.TransformNetTx:
		return PChainTransformNetTx, nil
	case *txs.AddPermissionlessValidatorTx:
		return PChainAddPermissionlessValidatorTx, nil
	case *txs.TransferNetOwnershipTx:
		return PChainTransferNetOwnershipTx, nil
	default:
		return Undefined, fmt.Errorf("unexpected unsigned tx type %T", unsignedTx)
	}
}

// get network id associated to tx
func (ms *Multisig) GetNetworkID() (uint32, error) {
	if ms.Undefined() {
		return 0, ErrUndefinedTx
	}
	unsignedTx := ms.PChainTx.Unsigned
	var networkID uint32
	switch unsignedTx := unsignedTx.(type) {
	case *txs.RemoveNetValidatorTx:
		networkID = unsignedTx.NetworkID
	case *txs.AddNetValidatorTx:
		networkID = unsignedTx.NetworkID
	case *txs.CreateChainTx:
		networkID = unsignedTx.NetworkID
	case *txs.TransformNetTx:
		networkID = unsignedTx.NetworkID
	case *txs.AddPermissionlessValidatorTx:
		networkID = unsignedTx.NetworkID
	case *txs.TransferNetOwnershipTx:
		networkID = unsignedTx.NetworkID
	default:
		return 0, fmt.Errorf("unexpected unsigned tx type %T", unsignedTx)
	}
	return networkID, nil
}

// get network model associated to tx
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
// Use GetNetID() instead for net-related operations.
//
// TODO: Remove this function or implement it correctly if blockchain IDs are needed
// func (ms *Multisig) GetBlockchainID() (ids.ID, error) {
// 	return ids.Empty, fmt.Errorf("GetBlockchainID is not implemented - transaction types do not have BlockchainID fields")
// }

// GetNetID gets net id associated to tx
func (ms *Multisig) GetNetID() (ids.ID, error) {
	if ms.Undefined() {
		return ids.Empty, ErrUndefinedTx
	}
	unsignedTx := ms.PChainTx.Unsigned
	var netID ids.ID
	switch unsignedTx := unsignedTx.(type) {
	case *txs.RemoveNetValidatorTx:
		netID = unsignedTx.Net
	case *txs.AddNetValidatorTx:
		netID = unsignedTx.NetValidator.Net
	case *txs.CreateChainTx:
		netID = unsignedTx.NetID
	case *txs.TransformNetTx:
		netID = unsignedTx.Net
	case *txs.AddPermissionlessValidatorTx:
		netID = unsignedTx.Net
	case *txs.TransferNetOwnershipTx:
		netID = unsignedTx.Net
	default:
		return ids.Empty, fmt.Errorf("unexpected unsigned tx type %T", unsignedTx)
	}
	return netID, nil
}

func (ms *Multisig) GetNetOwners() ([]ids.ShortID, uint32, error) {
	if ms.Undefined() {
		return nil, 0, ErrUndefinedTx
	}
	if ms.controlKeys == nil {
		netID, err := ms.GetNetID()
		if err != nil {
			return nil, 0, err
		}

		network, err := ms.GetNetwork()
		if err != nil {
			return nil, 0, err
		}
		controlKeys, threshold, err := GetOwners(network, netID)
		if err != nil {
			return nil, 0, err
		}
		ms.controlKeys = controlKeys
		ms.threshold = threshold
	}
	return ms.controlKeys, ms.threshold, nil
}

func GetOwners(network network.LegacyNetwork, netID ids.ID) ([]ids.ShortID, uint32, error) {
	pClient := platformvm.NewClient(network.Endpoint)
	ctx := context.Background()
	netResponse, err := pClient.GetNet(ctx, netID)
	if err != nil {
		return nil, 0, fmt.Errorf("net tx %s query error: %w", netID, err)
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
