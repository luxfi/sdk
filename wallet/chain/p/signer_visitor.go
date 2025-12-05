// Copyright (C) 2019-2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package p

import (
	"errors"
	"fmt"

	stdcontext "context"

	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/database"
	"github.com/luxfi/ids"
	"github.com/luxfi/ledger-lux-go/keychain"
	"github.com/luxfi/node/utils/constants"
	"github.com/luxfi/node/utils/hashing"
	"github.com/luxfi/node/vms/components/lux"
	"github.com/luxfi/node/vms/components/verify"
	"github.com/luxfi/node/vms/platformvm/stakeable"
	"github.com/luxfi/node/vms/platformvm/txs"
	"github.com/luxfi/node/vms/secp256k1fx"
)

var (
	_ txs.Visitor = (*signerVisitor)(nil)

	errUnsupportedTxType     = errors.New("unsupported tx type")
	errUnknownInputType      = errors.New("unknown input type")
	errUnknownCredentialType = errors.New("unknown credential type")
	errUnknownOutputType     = errors.New("unknown output type")
	errUnknownNetAuthType = errors.New("unknown net auth type")
	errInvalidUTXOSigIndex   = errors.New("invalid UTXO signature index")

	emptySig [secp256k1.SignatureLen]byte
)

// signerVisitor handles signing transactions for the signer
type signerVisitor struct {
	kc      keychain.Keychain
	backend SignerBackend
	ctx     stdcontext.Context
	tx      *txs.Tx
}

func (*signerVisitor) AdvanceTimeTx(*txs.AdvanceTimeTx) error {
	return errUnsupportedTxType
}

func (*signerVisitor) RewardValidatorTx(*txs.RewardValidatorTx) error {
	return errUnsupportedTxType
}

func (s *signerVisitor) BaseTx(tx *txs.BaseTx) error {
	txSigners, err := s.getSigners(constants.PlatformChainID, tx.Ins)
	if err != nil {
		return err
	}
	return sign(s.tx, false, txSigners)
}

func (s *signerVisitor) AddValidatorTx(tx *txs.AddValidatorTx) error {
	txSigners, err := s.getSigners(constants.PlatformChainID, tx.Ins)
	if err != nil {
		return err
	}
	return sign(s.tx, false, txSigners)
}

func (s *signerVisitor) AddNetValidatorTx(tx *txs.AddNetValidatorTx) error {
	txSigners, err := s.getSigners(constants.PlatformChainID, tx.Ins)
	if err != nil {
		return err
	}
	subnetAuthSigners, err := s.getNetSigners(tx.NetValidator.Net, tx.NetAuth)
	if err != nil {
		return err
	}
	txSigners = append(txSigners, subnetAuthSigners)
	return sign(s.tx, false, txSigners)
}

func (s *signerVisitor) AddDelegatorTx(tx *txs.AddDelegatorTx) error {
	txSigners, err := s.getSigners(constants.PlatformChainID, tx.Ins)
	if err != nil {
		return err
	}
	return sign(s.tx, false, txSigners)
}

func (s *signerVisitor) CreateChainTx(tx *txs.CreateChainTx) error {
	txSigners, err := s.getSigners(constants.PlatformChainID, tx.Ins)
	if err != nil {
		return err
	}
	subnetAuthSigners, err := s.getNetSigners(tx.NetID, tx.NetAuth)
	if err != nil {
		return err
	}
	txSigners = append(txSigners, subnetAuthSigners)
	return sign(s.tx, false, txSigners)
}

func (s *signerVisitor) CreateNetTx(tx *txs.CreateNetTx) error {
	txSigners, err := s.getSigners(constants.PlatformChainID, tx.Ins)
	if err != nil {
		return err
	}
	return sign(s.tx, false, txSigners)
}

func (s *signerVisitor) ImportTx(tx *txs.ImportTx) error {
	txSigners, err := s.getSigners(constants.PlatformChainID, tx.Ins)
	if err != nil {
		return err
	}
	txImportSigners, err := s.getSigners(tx.SourceChain, tx.ImportedInputs)
	if err != nil {
		return err
	}
	txSigners = append(txSigners, txImportSigners...)
	return sign(s.tx, false, txSigners)
}

func (s *signerVisitor) ExportTx(tx *txs.ExportTx) error {
	txSigners, err := s.getSigners(constants.PlatformChainID, tx.Ins)
	if err != nil {
		return err
	}
	return sign(s.tx, false, txSigners)
}

func (s *signerVisitor) RemoveNetValidatorTx(tx *txs.RemoveNetValidatorTx) error {
	txSigners, err := s.getSigners(constants.PlatformChainID, tx.Ins)
	if err != nil {
		return err
	}
	subnetAuthSigners, err := s.getNetSigners(tx.Net, tx.NetAuth)
	if err != nil {
		return err
	}
	txSigners = append(txSigners, subnetAuthSigners)
	return sign(s.tx, true, txSigners)
}

func (s *signerVisitor) TransferNetOwnershipTx(tx *txs.TransferNetOwnershipTx) error {
	txSigners, err := s.getSigners(constants.PlatformChainID, tx.Ins)
	if err != nil {
		return err
	}
	subnetAuthSigners, err := s.getNetSigners(tx.Net, tx.NetAuth)
	if err != nil {
		return err
	}
	txSigners = append(txSigners, subnetAuthSigners)
	return sign(s.tx, true, txSigners)
}

func (s *signerVisitor) TransformNetTx(tx *txs.TransformNetTx) error {
	txSigners, err := s.getSigners(constants.PlatformChainID, tx.Ins)
	if err != nil {
		return err
	}
	subnetAuthSigners, err := s.getNetSigners(tx.Net, tx.NetAuth)
	if err != nil {
		return err
	}
	txSigners = append(txSigners, subnetAuthSigners)
	return sign(s.tx, true, txSigners)
}

func (s *signerVisitor) AddPermissionlessValidatorTx(tx *txs.AddPermissionlessValidatorTx) error {
	txSigners, err := s.getSigners(constants.PlatformChainID, tx.Ins)
	if err != nil {
		return err
	}
	return sign(s.tx, true, txSigners)
}

func (s *signerVisitor) AddPermissionlessDelegatorTx(tx *txs.AddPermissionlessDelegatorTx) error {
	txSigners, err := s.getSigners(constants.PlatformChainID, tx.Ins)
	if err != nil {
		return err
	}
	return sign(s.tx, true, txSigners)
}

func (s *signerVisitor) getSigners(sourceChainID ids.ID, ins []*lux.TransferableInput) ([][]keychain.Signer, error) {
	txSigners := make([][]keychain.Signer, len(ins))
	for credIndex, transferInput := range ins {
		inIntf := transferInput.In
		if stakeableIn, ok := inIntf.(*stakeable.LockIn); ok {
			inIntf = stakeableIn.TransferableIn
		}

		input, ok := inIntf.(*secp256k1fx.TransferInput)
		if !ok {
			return nil, errUnknownInputType
		}

		inputSigners := make([]keychain.Signer, len(input.SigIndices))
		txSigners[credIndex] = inputSigners

		utxoID := transferInput.InputID()
		utxo, err := s.backend.GetUTXO(s.ctx, sourceChainID, utxoID)
		if err == database.ErrNotFound {
			// If we don't have access to the UTXO, then we can't sign this
			// transaction. However, we can attempt to partially sign it.
			continue
		}
		if err != nil {
			return nil, err
		}

		outIntf := utxo.Out
		if stakeableOut, ok := outIntf.(*stakeable.LockOut); ok {
			outIntf = stakeableOut.TransferableOut
		}

		out, ok := outIntf.(*secp256k1fx.TransferOutput)
		if !ok {
			return nil, errUnknownOutputType
		}

		for sigIndex, addrIndex := range input.SigIndices {
			if addrIndex >= uint32(len(out.Addrs)) {
				return nil, errInvalidUTXOSigIndex
			}

			addr := out.Addrs[addrIndex]
			key, ok := s.kc.Get(addr)
			if !ok {
				// If we don't have access to the key, then we can't sign this
				// transaction. However, we can attempt to partially sign it.
				continue
			}
			inputSigners[sigIndex] = key
		}
	}
	return txSigners, nil
}

func (s *signerVisitor) getNetSigners(netID ids.ID, subnetAuth verify.Verifiable) ([]keychain.Signer, error) {
	subnetInput, ok := subnetAuth.(*secp256k1fx.Input)
	if !ok {
		return nil, errUnknownNetAuthType
	}

	subnetTx, err := s.backend.GetTx(s.ctx, netID)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch net %q: %w",
			netID,
			err,
		)
	}
	subnet, ok := subnetTx.Unsigned.(*txs.CreateNetTx)
	if !ok {
		return nil, errWrongTxType
	}

	owner, ok := subnet.Owner.(*secp256k1fx.OutputOwners)
	if !ok {
		return nil, errUnknownOwnerType
	}

	authSigners := make([]keychain.Signer, len(subnetInput.SigIndices))
	for sigIndex, addrIndex := range subnetInput.SigIndices {
		if addrIndex >= uint32(len(owner.Addrs)) {
			return nil, errInvalidUTXOSigIndex
		}

		addr := owner.Addrs[addrIndex]
		key, ok := s.kc.Get(addr)
		if !ok {
			// If we don't have access to the key, then we can't sign this
			// transaction. However, we can attempt to partially sign it.
			continue
		}
		authSigners[sigIndex] = key
	}
	return authSigners, nil
}

func sign(tx *txs.Tx, signHash bool, txSigners [][]keychain.Signer) error {
	unsignedBytes, err := txs.Codec.Marshal(txs.Version, &tx.Unsigned)
	if err != nil {
		return fmt.Errorf("couldn't marshal unsigned tx: %w", err)
	}
	unsignedHash := hashing.ComputeHash256(unsignedBytes)

	if expectedLen := len(txSigners); expectedLen != len(tx.Creds) {
		tx.Creds = make([]verify.Verifiable, expectedLen)
	}

	sigCache := make(map[ids.ShortID][secp256k1.SignatureLen]byte)
	for credIndex, inputSigners := range txSigners {
		credIntf := tx.Creds[credIndex]
		if credIntf == nil {
			credIntf = &secp256k1fx.Credential{}
			tx.Creds[credIndex] = credIntf
		}

		cred, ok := credIntf.(*secp256k1fx.Credential)
		if !ok {
			return errUnknownCredentialType
		}
		if expectedLen := len(inputSigners); expectedLen != len(cred.Sigs) {
			cred.Sigs = make([][secp256k1.SignatureLen]byte, expectedLen)
		}

		for sigIndex, signer := range inputSigners {
			if signer == nil {
				// If we don't have access to the key, then we can't sign this
				// transaction. However, we can attempt to partially sign it.
				continue
			}
			addr := signer.Address()
			if sig := cred.Sigs[sigIndex]; sig != emptySig {
				// If this signature has already been populated, we can just
				// copy the needed signature for the future.
				sigCache[addr] = sig
				continue
			}

			if sig, exists := sigCache[addr]; exists {
				// If this key has already produced a signature, we can just
				// copy the previous signature.
				cred.Sigs[sigIndex] = sig
				continue
			}

			var sig []byte
			if signHash {
				sig, err = signer.SignHash(unsignedHash)
			} else {
				sig, err = signer.Sign(unsignedBytes)
			}
			if err != nil {
				return fmt.Errorf("problem signing tx: %w", err)
			}
			copy(cred.Sigs[sigIndex][:], sig)
			sigCache[addr] = cred.Sigs[sigIndex]
		}
	}

	signedBytes, err := txs.Codec.Marshal(txs.Version, tx)
	if err != nil {
		return fmt.Errorf("couldn't marshal tx: %w", err)
	}
	tx.SetBytes(unsignedBytes, signedBytes)
	return nil
}

// DisableL1ValidatorTx signs a DisableL1ValidatorTx
func (s *signerVisitor) DisableL1ValidatorTx(tx *txs.DisableL1ValidatorTx) error {
	txSigners, err := s.getSigners(constants.PrimaryNetworkID, tx.Ins)
	if err != nil {
		return err
	}
	return sign(s.tx, false, txSigners)
}

// IncreaseL1ValidatorBalanceTx signs an IncreaseL1ValidatorBalanceTx
func (s *signerVisitor) IncreaseL1ValidatorBalanceTx(tx *txs.IncreaseL1ValidatorBalanceTx) error {
	txSigners, err := s.getSigners(constants.PrimaryNetworkID, tx.Ins)
	if err != nil {
		return err
	}
	return sign(s.tx, false, txSigners)
}

// RegisterL1ValidatorTx signs a RegisterL1ValidatorTx
func (s *signerVisitor) RegisterL1ValidatorTx(tx *txs.RegisterL1ValidatorTx) error {
	txSigners, err := s.getSigners(constants.PrimaryNetworkID, tx.Ins)
	if err != nil {
		return err
	}
	return sign(s.tx, false, txSigners)
}

// SetL1ValidatorWeightTx signs a SetL1ValidatorWeightTx
func (s *signerVisitor) SetL1ValidatorWeightTx(tx *txs.SetL1ValidatorWeightTx) error {
	txSigners, err := s.getSigners(constants.PrimaryNetworkID, tx.Ins)
	if err != nil {
		return err
	}
	return sign(s.tx, false, txSigners)
}

// ConvertNetToL1Tx signs a ConvertNetToL1Tx
func (s *signerVisitor) ConvertNetToL1Tx(tx *txs.ConvertNetToL1Tx) error {
	txSigners, err := s.getSigners(constants.PrimaryNetworkID, tx.Ins)
	if err != nil {
		return err
	}
	subnetAuthSigners, err := s.getNetSigners(tx.Net, tx.NetAuth)
	if err != nil {
		return err
	}
	txSigners = append(txSigners, subnetAuthSigners)
	return sign(s.tx, false, txSigners)
}
