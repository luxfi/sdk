// Copyright (C) 2019-2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package c

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/luxfi/crypto/hash"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/ids"
	lux "github.com/luxfi/utxo"
	"github.com/luxfi/utxo/secp256k1fx"
	"github.com/luxfi/vm/components/verify"
)

// LP-185 / Wave 2C of the codec rip (#101). The C-chain atomic-tx flow is
// dead code in the SDK runtime today (see `wallet/primary/wallet.go:170-180`,
// where c-chain wallet wiring is commented out pending an EVM client
// implementation). This file used to carry a residual `codec.Manager`
// scaffold from upstream luxfi/codec; that scaffold is now removed and the
// few marshal points left in the package go through the deterministic
// `marshalAtomicTxBytes` helper below.
//
// Once luxfi/node ships a `vms/evm/wire` ZAP-native API for atomic txs
// (NewImportTx / NewExportTx / WrapImportTx / WrapExportTx), the helper
// here can be deleted and Sign can route directly through ZAP buffer
// accessors — the buffer IS the wire, no marshal pass.

// Tx represents a transaction on the C-Chain
type Tx struct {
	ID               ids.ID
	UnsignedAtomicTx UnsignedAtomicTx
	Creds            []verify.Verifiable
	unsignedBytes    []byte
	signedBytes      []byte
}

// SignedBytes returns the signed bytes of the transaction
func (tx *Tx) SignedBytes() []byte {
	return tx.signedBytes
}

// UnsignedImportTx is an unsigned import transaction
type UnsignedImportTx struct {
	BaseTx
	SourceChain    ids.ID
	ImportedInputs []*lux.TransferableInput
	Outs           []*EVMOutput
}

// InputUTXOs implements UnsignedAtomicTx
func (tx *UnsignedImportTx) InputUTXOs() []ids.ID {
	utxos := make([]ids.ID, len(tx.ImportedInputs))
	for i, input := range tx.ImportedInputs {
		utxos[i] = input.InputID()
	}
	return utxos
}

// GasUsed returns the gas used by this transaction
func (tx *UnsignedImportTx) GasUsed(fixedFee bool) (uint64, error) {
	return 100000, nil // placeholder gas value
}

// EVMOutput represents an output on the C-chain
type EVMOutput struct {
	Address common.Address
	Amount  uint64
}

// EVMInput represents an input on the C-chain
type EVMInput struct {
	Address common.Address
	Amount  uint64
	AssetID ids.ID
	Nonce   uint64
}

// Compare implements utils.Sortable
func (e *EVMInput) Compare(other *EVMInput) int {
	addrCmp := e.Address.Cmp(other.Address)
	if addrCmp != 0 {
		return addrCmp
	}

	if e.Amount < other.Amount {
		return -1
	}
	if e.Amount > other.Amount {
		return 1
	}

	return e.AssetID.Compare(other.AssetID)
}

// UnsignedExportTx is an unsigned export transaction
type UnsignedExportTx struct {
	BaseTx
	DestinationChain ids.ID
	ExportedOutputs  []*lux.TransferableOutput
	Ins              []*EVMInput
}

// ID returns the transaction ID
func (tx *UnsignedExportTx) ID() ids.ID {
	return ids.Empty
}

// InputUTXOs implements UnsignedAtomicTx
func (tx *UnsignedExportTx) InputUTXOs() []ids.ID {
	// Export transactions don't consume UTXOs directly
	return nil
}

// GasUsed returns the gas used by this transaction
func (tx *UnsignedExportTx) GasUsed(fixedFee bool) (uint64, error) {
	return 100000, nil // placeholder gas value
}

// UnsignedAtomicTx is the interface for unsigned atomic transactions
type UnsignedAtomicTx interface {
	InputUTXOs() []ids.ID
}

// BaseTx contains common transaction fields
type BaseTx struct {
	NetworkID    uint32
	BlockchainID ids.ID
	Outs         []*lux.TransferableOutput
	Ins          []*lux.TransferableInput
	Memo         []byte
}

// Client represents the C-chain client interface
type Client interface {
	IssueTx(tx *Tx) (ids.ID, error)
	GetAtomicTxStatus(txID ids.ID) (Status, error)
}

// Status represents the status of a transaction
type Status string

// Constants
const (
	codecVersion uint16 = 0
	EVMOutputGas        = 100 // Implementation note
	EVMInputGas         = 100 // Implementation note

	// Transaction status
	Accepted = "Accepted"
)

// marshalAtomicTxBytes returns a deterministic byte representation of any
// supported atomic-tx shape. Layout:
//
//	[u16 version=0][kind byte][big-endian struct fields...]
//
// kind = 0x01 for *UnsignedImportTx, 0x02 for *UnsignedExportTx, 0x03 for
// signed *Tx (UnsignedAtomicTx kind byte + creds), and 0x00 for an empty /
// nil pointer (defensive — never expected). The encoding is purely an
// implementation detail of this dead-path package; it exists only so
// `tx.Sign` can produce a stable hash for tests. No external peer reads
// these bytes.
func marshalAtomicTxBytes(v interface{}) ([]byte, error) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, codecVersion)
	switch tx := v.(type) {
	case *UnsignedAtomicTx:
		if tx == nil || *tx == nil {
			buf = append(buf, 0x00)
			return buf, nil
		}
		return marshalAtomicTxBytes(*tx)
	case *UnsignedImportTx:
		buf = append(buf, 0x01)
		buf = appendBaseTx(buf, &tx.BaseTx)
		buf = append(buf, tx.SourceChain[:]...)
		buf = appendUint32(buf, uint32(len(tx.ImportedInputs)))
		for _, in := range tx.ImportedInputs {
			id := in.InputID()
			buf = append(buf, id[:]...)
			buf = appendUint64(buf, in.In.Amount())
		}
		buf = appendUint32(buf, uint32(len(tx.Outs)))
		for _, o := range tx.Outs {
			buf = append(buf, o.Address.Bytes()...)
			buf = appendUint64(buf, o.Amount)
		}
		return buf, nil
	case *UnsignedExportTx:
		buf = append(buf, 0x02)
		buf = appendBaseTx(buf, &tx.BaseTx)
		buf = append(buf, tx.DestinationChain[:]...)
		buf = appendUint32(buf, uint32(len(tx.ExportedOutputs)))
		for _, o := range tx.ExportedOutputs {
			id := o.AssetID()
			buf = append(buf, id[:]...)
			buf = appendUint64(buf, o.Out.Amount())
		}
		buf = appendUint32(buf, uint32(len(tx.Ins)))
		for _, in := range tx.Ins {
			buf = append(buf, in.Address.Bytes()...)
			buf = appendUint64(buf, in.Amount)
			buf = append(buf, in.AssetID[:]...)
			buf = appendUint64(buf, in.Nonce)
		}
		return buf, nil
	case *Tx:
		buf = append(buf, 0x03)
		uBytes, err := marshalAtomicTxBytes(tx.UnsignedAtomicTx)
		if err != nil {
			return nil, err
		}
		buf = appendUint32(buf, uint32(len(uBytes)))
		buf = append(buf, uBytes...)
		buf = appendUint32(buf, uint32(len(tx.Creds)))
		for _, c := range tx.Creds {
			cred, ok := c.(*secp256k1fx.Credential)
			if !ok {
				return nil, fmt.Errorf("c.marshalAtomicTxBytes: unsupported credential type %T", c)
			}
			buf = appendUint32(buf, uint32(len(cred.Sigs)))
			for _, sig := range cred.Sigs {
				buf = append(buf, sig[:]...)
			}
		}
		return buf, nil
	default:
		return nil, fmt.Errorf("c.marshalAtomicTxBytes: unsupported type %T", v)
	}
}

func appendBaseTx(buf []byte, b *BaseTx) []byte {
	buf = appendUint32(buf, b.NetworkID)
	buf = append(buf, b.BlockchainID[:]...)
	buf = appendUint32(buf, uint32(len(b.Outs)))
	for _, o := range b.Outs {
		id := o.AssetID()
		buf = append(buf, id[:]...)
		buf = appendUint64(buf, o.Out.Amount())
	}
	buf = appendUint32(buf, uint32(len(b.Ins)))
	for _, in := range b.Ins {
		id := in.InputID()
		buf = append(buf, id[:]...)
		buf = appendUint64(buf, in.In.Amount())
	}
	buf = appendUint32(buf, uint32(len(b.Memo)))
	buf = append(buf, b.Memo...)
	return buf
}

func appendUint32(buf []byte, v uint32) []byte {
	var tmp [4]byte
	binary.BigEndian.PutUint32(tmp[:], v)
	return append(buf, tmp[:]...)
}

func appendUint64(buf []byte, v uint64) []byte {
	var tmp [8]byte
	binary.BigEndian.PutUint64(tmp[:], v)
	return append(buf, tmp[:]...)
}

// CalculateDynamicFee calculates the dynamic fee based on EIP-1559
// Fee = gasUsed * (baseFee + priorityFee)
func CalculateDynamicFee(gasUsed uint64, baseFee *big.Int) (uint64, error) {
	// Calculate base fee component
	baseFeeComponent := new(big.Int).Mul(baseFee, new(big.Int).SetUint64(gasUsed))

	// Add priority fee (tip) - using a minimum priority fee of 1 Gwei
	priorityFeePerGas := new(big.Int).SetUint64(1_000_000_000) // 1 Gwei in Wei
	priorityFeeComponent := new(big.Int).Mul(priorityFeePerGas, new(big.Int).SetUint64(gasUsed))

	// Total fee = base fee + priority fee
	totalFee := new(big.Int).Add(baseFeeComponent, priorityFeeComponent)

	// Check if fee fits in uint64
	if !totalFee.IsUint64() {
		return 0, errInsufficientFunds
	}
	return totalFee.Uint64(), nil
}

// Sign signs the transaction with the provided private keys.
//
// The wire format is whatever `marshalAtomicTxBytes` produces — see that
// helper's doc. No codec.Manager is needed; the buffer IS the wire.
func (tx *Tx) Sign(signers [][]*secp256k1.PrivateKey) error {
	// Serialize the unsigned transaction
	unsignedBytes, err := marshalAtomicTxBytes(&tx.UnsignedAtomicTx)
	if err != nil {
		return fmt.Errorf("failed to marshal unsigned tx: %w", err)
	}

	// Create signature placeholder for each input
	tx.Creds = make([]verify.Verifiable, len(signers))

	// Sign each input with the corresponding signers
	for i, inputSigners := range signers {
		cred := &secp256k1fx.Credential{
			Sigs: make([][secp256k1.SignatureLen]byte, len(inputSigners)),
		}

		// Generate signature for each signer
		for j, signer := range inputSigners {
			sig, err := signer.SignHash(hash.ComputeHash256(unsignedBytes))
			if err != nil {
				return fmt.Errorf("failed to sign tx at input %d, signer %d: %w", i, j, err)
			}
			copy(cred.Sigs[j][:], sig)
		}

		tx.Creds[i] = cred
	}

	// Serialize the signed transaction
	signedBytes, err := marshalAtomicTxBytes(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal signed tx: %w", err)
	}

	// Initialize the transaction with the serialized bytes
	tx.Initialize(unsignedBytes, signedBytes)
	return nil
}

// Initialize initializes the transaction with computed ID and caches bytes
func (tx *Tx) Initialize(unsignedBytes, signedBytes []byte) {
	// Calculate transaction ID from unsigned bytes (standard for atomic txs)
	tx.ID = ids.ID(hash.ComputeHash256(unsignedBytes))

	// Cache the unsigned and signed bytes for future use
	tx.unsignedBytes = unsignedBytes
	tx.signedBytes = signedBytes

	// Also initialize the underlying unsigned transaction if it has an Initialize method
	if initializable, ok := tx.UnsignedAtomicTx.(interface{ Initialize([]byte) }); ok {
		initializable.Initialize(unsignedBytes)
	}
}

// UnsignedAtomicTx field for Tx
type UnsignedAtomicTxWrapper struct {
	UnsignedAtomicTx UnsignedAtomicTx
}

// errInsufficientFunds is defined in builder.go
