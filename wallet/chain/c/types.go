// Copyright (C) 2019-2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package c

import (
	"fmt"
	"math/big"

	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/codec"
	"github.com/luxfi/node/codec/linearcodec"
	"github.com/luxfi/node/utils/hashing"
	"github.com/luxfi/node/vms/components/lux"
	"github.com/luxfi/node/vms/components/verify"
	"github.com/luxfi/node/vms/secp256k1fx"
)

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
	codecVersion = 0
	EVMOutputGas = 100 // Implementation note
	EVMInputGas  = 100 // Implementation note

	// Transaction status
	Accepted = "Accepted"
)

// Variables
var (
	// Codec is the codec used for serialization
	Codec codec.Manager
)

func init() {
	// Initialize codec
	Codec = codec.NewDefaultManager()
	lcodec := linearcodec.NewDefault()
	Codec.RegisterCodec(codecVersion, lcodec)
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

// Sign signs the transaction with the provided private keys
func (tx *Tx) Sign(codec codec.Manager, signers [][]*secp256k1.PrivateKey) error {
	// Serialize the unsigned transaction
	unsignedBytes, err := codec.Marshal(codecVersion, &tx.UnsignedAtomicTx)
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
			sig, err := signer.SignHash(hashing.ComputeHash256(unsignedBytes))
			if err != nil {
				return fmt.Errorf("failed to sign tx at input %d, signer %d: %w", i, j, err)
			}
			copy(cred.Sigs[j][:], sig)
		}

		tx.Creds[i] = cred
	}

	// Serialize the signed transaction
	signedBytes, err := codec.Marshal(codecVersion, tx)
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
	tx.ID = ids.ID(hashing.ComputeHash256(unsignedBytes))

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
