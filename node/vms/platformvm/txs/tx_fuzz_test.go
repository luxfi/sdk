// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"bytes"
	"testing"
	"time"

	"github.com/luxfi/sdk/node/codec/linearcodec"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/utils/wrappers"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/components/verify"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

// FuzzTransactionParsing tests transaction parsing with random data
func FuzzTransactionParsing(f *testing.F) {
	// Seed corpus with various transaction-like structures
	f.Add([]byte{})
	f.Add([]byte{0x00, 0x00, 0x00, 0x00}) // Codec version
	f.Add([]byte{0x00, 0x00, 0x00, 0x01}) // Type ID
	
	// Add a more structured transaction-like data
	txData := make([]byte, 100)
	copy(txData[:4], []byte{0x00, 0x00, 0x00, 0x00}) // Version
	copy(txData[4:8], []byte{0x00, 0x00, 0x00, 0x0c}) // Type ID
	f.Add(txData)
	
	// Add data with IDs
	testID := ids.GenerateTestID()
	withID := append([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, testID[:]...)
	f.Add(withID)
	
	// Parser is not available in this package, using Codec instead
	codec := Codec
	
	f.Fuzz(func(t *testing.T, data []byte) {
		// Try to parse as transaction
		var tx Tx
		_, err := codec.Unmarshal(data, &tx)
		if err != nil {
			// Expected for invalid transaction data
			return
		}
		
		// If parsing succeeded, test that methods don't panic
		_ = tx.ID()
		_ = tx.Bytes()
		unsigned := tx.Unsigned
		if unsigned != nil {
			_ = unsigned.Visit(&visitor{})
		}
		
		// Try to serialize back
		bytes := tx.Bytes()
		if len(bytes) == 0 {
			t.Error("Parsed transaction has empty bytes")
		}
		
		// Parse again and verify consistency
		var tx2 Tx
		_, err = codec.Unmarshal(bytes, &tx2)
		if err != nil {
			t.Errorf("Failed to re-parse serialized transaction: %v", err)
			return
		}
		
		if tx.ID() != tx2.ID() {
			t.Errorf("Transaction ID changed after re-parsing")
		}
	})
}

// FuzzBaseTx tests BaseTx parsing and serialization
func FuzzBaseTx(f *testing.F) {
	// Seed corpus
	f.Add(uint64(1), uint32(1), []byte{})
	f.Add(uint64(1000000), uint32(42), bytes.Repeat([]byte{0xff}, 32))
	testID := ids.GenerateTestID()
	f.Add(uint64(0), uint32(0), testID[:])
	
	c := linearcodec.NewDefault()
	
	f.Fuzz(func(t *testing.T, networkID uint64, blockchainID uint32, assetData []byte) {
		// Create asset ID
		var assetID ids.ID
		if len(assetData) >= 32 {
			copy(assetID[:], assetData[:32])
		}
		
		// Create a base transaction
		baseTx := &BaseTx{
			BaseTx: lux.BaseTx{
				NetworkID:    uint32(networkID & 0xFFFFFFFF), // Limit to uint32
				BlockchainID: ids.GenerateTestID(),
				Outs: []*lux.TransferableOutput{
					{
						Asset: lux.Asset{ID: assetID},
						Out: &secp256k1fx.TransferOutput{
							Amt: 1000,
							OutputOwners: secp256k1fx.OutputOwners{
								Threshold: 1,
								Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
							},
						},
					},
				},
				Ins: []*lux.TransferableInput{},
			},
		}
		
		// Try to serialize
		p := wrappers.Packer{MaxSize: 1024 * 1024}
		err := c.MarshalInto(baseTx, &p)
		if err != nil {
			// Some combinations might be invalid
			return
		}
		
		// Try to deserialize
		var parsed BaseTx
		p2 := wrappers.Packer{Bytes: p.Bytes, MaxSize: 1024 * 1024}
		err = c.UnmarshalFrom(&p2, &parsed)
		if err != nil {
			t.Errorf("Failed to unmarshal BaseTx: %v", err)
			return
		}
		
		// Verify fields match
		if parsed.NetworkID != baseTx.NetworkID {
			t.Errorf("NetworkID mismatch: got %v, want %v", parsed.NetworkID, baseTx.NetworkID)
		}
		
		if parsed.BlockchainID != baseTx.BlockchainID {
			t.Errorf("BlockchainID mismatch")
		}
	})
}

// FuzzCreateChainTx tests CreateChainTx parsing
func FuzzCreateChainTx(f *testing.F) {
	// Seed corpus
	f.Add([]byte("chainName"), []byte{}, []byte("vmID"))
	f.Add([]byte("test"), bytes.Repeat([]byte{0x01}, 100), []byte("customvm"))
	f.Add([]byte{}, []byte{}, []byte{})
	
	c := linearcodec.NewDefault()
	
	f.Fuzz(func(t *testing.T, chainName []byte, genesisData []byte, vmIDData []byte) {
		// Limit sizes
		if len(chainName) > 128 {
			chainName = chainName[:128]
		}
		if len(genesisData) > 10000 {
			genesisData = genesisData[:10000]
		}
		
		// Create VM ID
		var vmID ids.ID
		if len(vmIDData) >= 32 {
			copy(vmID[:], vmIDData[:32])
		} else {
			vmID = ids.GenerateTestID()
		}
		
		// Create transaction
		tx := &CreateChainTx{
			BaseTx: BaseTx{
				BaseTx: lux.BaseTx{
					NetworkID:    1,
					BlockchainID: ids.GenerateTestID(),
					Outs:         []*lux.TransferableOutput{},
					Ins:          []*lux.TransferableInput{},
				},
			},
			ChainID:       ids.GenerateTestID(),
			BlockchainName:   string(chainName),
			VMID:        vmID,
			FxIDs:       []ids.ID{},
			GenesisData: genesisData,
			ChainAuth:     &secp256k1fx.Input{},
		}
		
		// Try to serialize
		p := wrappers.Packer{MaxSize: 10 * 1024 * 1024}
		err := c.MarshalInto(tx, &p)
		if err != nil {
			// Some combinations might be invalid
			return
		}
		
		// Try to deserialize
		var parsed CreateChainTx
		p2 := wrappers.Packer{Bytes: p.Bytes, MaxSize: 10 * 1024 * 1024}
		err = c.UnmarshalFrom(&p2, &parsed)
		if err != nil {
			t.Errorf("Failed to unmarshal CreateChainTx: %v", err)
			return
		}
		
		// Verify key fields
		if parsed.BlockchainName != tx.BlockchainName {
			t.Errorf("ChainName mismatch: got %q, want %q", parsed.BlockchainName, tx.BlockchainName)
		}
		
		if parsed.VMID != tx.VMID {
			t.Errorf("VMID mismatch")
		}
		
		if !bytes.Equal(parsed.GenesisData, tx.GenesisData) {
			t.Errorf("GenesisData mismatch")
		}
	})
}

// FuzzAddValidatorTx tests validator transaction parsing
func FuzzAddValidatorTx(f *testing.F) {
	// Seed corpus
	f.Add(uint64(1), uint64(100), uint64(1000), uint32(100000))
	f.Add(uint64(0), uint64(0), uint64(0), uint32(0))
	f.Add(uint64(time.Now().Unix()), uint64(time.Now().Add(time.Hour).Unix()), uint64(1000000), uint32(20000))
	
	c := linearcodec.NewDefault()
	
	f.Fuzz(func(t *testing.T, startTime, endTime, weight uint64, shares uint32) {
		// Create validator transaction
		tx := &AddValidatorTx{
			BaseTx: BaseTx{
				BaseTx: lux.BaseTx{
					NetworkID:    1,
					BlockchainID: ids.GenerateTestID(),
					Outs:         []*lux.TransferableOutput{},
					Ins:          []*lux.TransferableInput{},
				},
			},
			Validator: Validator{
				NodeID: ids.GenerateTestNodeID(),
				Start:  startTime,
				End:    endTime,
				Wght:   weight,
			},
			StakeOuts: []*lux.TransferableOutput{
				{
					Asset: lux.Asset{ID: ids.GenerateTestID()},
					Out: &secp256k1fx.TransferOutput{
						Amt: weight,
						OutputOwners: secp256k1fx.OutputOwners{
							Threshold: 1,
							Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
						},
					},
				},
			},
			RewardsOwner: &secp256k1fx.OutputOwners{
				Threshold: 1,
				Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
			},
			DelegationShares: shares,
		}
		
		// Try to serialize
		p := wrappers.Packer{MaxSize: 1024 * 1024}
		err := c.MarshalInto(tx, &p)
		if err != nil {
			return
		}
		
		// Try to deserialize
		var parsed AddValidatorTx
		p2 := wrappers.Packer{Bytes: p.Bytes, MaxSize: 1024 * 1024}
		err = c.UnmarshalFrom(&p2, &parsed)
		if err != nil {
			t.Errorf("Failed to unmarshal AddValidatorTx: %v", err)
			return
		}
		
		// Verify fields
		if parsed.Start != tx.Start {
			t.Errorf("Start time mismatch: got %v, want %v", parsed.Start, tx.Start)
		}
		
		if parsed.End != tx.End {
			t.Errorf("End time mismatch: got %v, want %v", parsed.End, tx.End)
		}
		
		if parsed.Wght != tx.Wght {
			t.Errorf("Weight mismatch: got %v, want %v", parsed.Wght, tx.Wght)
		}
		
		if parsed.DelegationShares != tx.DelegationShares {
			t.Errorf("DelegationShares mismatch: got %v, want %v", parsed.DelegationShares, tx.DelegationShares)
		}
	})
}

// FuzzImportExportTx tests import/export transaction parsing
func FuzzImportExportTx(f *testing.F) {
	// Seed corpus
	f.Add([]byte{}, []byte{})
	testID1 := ids.GenerateTestID()
	testID2 := ids.GenerateTestID()
	f.Add(testID1[:], testID2[:])
	f.Add(bytes.Repeat([]byte{0xff}, 32), bytes.Repeat([]byte{0xaa}, 32))
	
	c := linearcodec.NewDefault()
	
	f.Fuzz(func(t *testing.T, sourceChainData, destChainData []byte) {
		// Create chain IDs
		var sourceChain, destChain ids.ID
		if len(sourceChainData) >= 32 {
			copy(sourceChain[:], sourceChainData[:32])
		} else {
			sourceChain = ids.GenerateTestID()
		}
		
		if len(destChainData) >= 32 {
			copy(destChain[:], destChainData[:32])
		} else {
			destChain = ids.GenerateTestID()
		}
		
		// Create ImportTx
		importTx := &ImportTx{
			BaseTx: BaseTx{
				BaseTx: lux.BaseTx{
					NetworkID:    1,
					BlockchainID: destChain,
					Outs:         []*lux.TransferableOutput{},
					Ins:          []*lux.TransferableInput{},
				},
			},
			SourceChain: sourceChain,
			ImportedInputs: []*lux.TransferableInput{
				{
					UTXOID: lux.UTXOID{
						TxID:        ids.GenerateTestID(),
						OutputIndex: 0,
					},
					Asset: lux.Asset{ID: ids.GenerateTestID()},
					In: &secp256k1fx.TransferInput{
						Amt: 1000,
						Input: secp256k1fx.Input{
							SigIndices: []uint32{0},
						},
					},
				},
			},
		}
		
		// Try to serialize ImportTx
		p := wrappers.Packer{MaxSize: 1024 * 1024}
		err := c.MarshalInto(importTx, &p)
		if err != nil {
			return
		}
		
		// Try to deserialize
		var parsedImport ImportTx
		p2 := wrappers.Packer{Bytes: p.Bytes, MaxSize: 1024 * 1024}
		err = c.UnmarshalFrom(&p2, &parsedImport)
		if err != nil {
			t.Errorf("Failed to unmarshal ImportTx: %v", err)
			return
		}
		
		// Verify fields
		if parsedImport.SourceChain != importTx.SourceChain {
			t.Errorf("SourceChain mismatch")
		}
		
		// Create ExportTx
		exportTx := &ExportTx{
			BaseTx: BaseTx{
				BaseTx: lux.BaseTx{
					NetworkID:    1,
					BlockchainID: sourceChain,
					Outs:         []*lux.TransferableOutput{},
					Ins:          []*lux.TransferableInput{},
				},
			},
			DestinationChain: destChain,
			ExportedOutputs: []*lux.TransferableOutput{
				{
					Asset: lux.Asset{ID: ids.GenerateTestID()},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1000,
						OutputOwners: secp256k1fx.OutputOwners{
							Threshold: 1,
							Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
						},
					},
				},
			},
		}
		
		// Try to serialize ExportTx
		p3 := wrappers.Packer{MaxSize: 1024 * 1024}
		err = c.MarshalInto(exportTx, &p3)
		if err != nil {
			return
		}
		
		// Try to deserialize
		var parsedExport ExportTx
		p4 := wrappers.Packer{Bytes: p3.Bytes, MaxSize: 1024 * 1024}
		err = c.UnmarshalFrom(&p4, &parsedExport)
		if err != nil {
			t.Errorf("Failed to unmarshal ExportTx: %v", err)
			return
		}
		
		// Verify fields
		if parsedExport.DestinationChain != exportTx.DestinationChain {
			t.Errorf("DestinationChain mismatch")
		}
	})
}

// FuzzTransactionSignatures tests transaction signature handling
func FuzzTransactionSignatures(f *testing.F) {
	// Seed corpus
	f.Add([]byte{}, []byte{})
	f.Add(bytes.Repeat([]byte{0x01}, 65), bytes.Repeat([]byte{0x02}, 32))
	
	// Parser is not available in this package, using Codec instead
	codec := Codec
	
	f.Fuzz(func(t *testing.T, sigData []byte, txData []byte) {
		// Create a basic transaction
		baseTx := &Tx{
			Unsigned: &BaseTx{
				BaseTx: lux.BaseTx{
					NetworkID:    1,
					BlockchainID: ids.GenerateTestID(),
					Outs:         []*lux.TransferableOutput{},
					Ins:          []*lux.TransferableInput{},
				},
			},
			Creds: []verify.Verifiable{},
		}
		
		// Add credentials based on signature data
		if len(sigData) >= 65 {
			cred := secp256k1fx.Credential{
				Sigs: [][secp256k1.SignatureLen]byte{},
			}
			
			// Add signatures (65 bytes each)
			for i := 0; i+65 <= len(sigData); i += 65 {
				var sig [65]byte
				copy(sig[:], sigData[i:i+65])
				cred.Sigs = append(cred.Sigs, sig)
				
				if len(cred.Sigs) >= 10 { // Limit number of signatures
					break
				}
			}
			
			baseTx.Creds = append(baseTx.Creds, &cred)
		}
		
		// Initialize the transaction
		if err := baseTx.Initialize(codec); err != nil {
			// Some combinations might be invalid
			return
		}
		
		// Get transaction bytes
		bytes := baseTx.Bytes()
		
		// Try to parse back
		var parsed Tx
		_, err := codec.Unmarshal(bytes, &parsed)
		if err != nil {
			// Should not fail for a transaction we created
			t.Errorf("Failed to parse transaction we created: %v", err)
			return
		}

		// Initialize the parsed transaction to compute its ID
		if err := parsed.Initialize(codec); err != nil {
			// Should not fail for a valid parsed transaction
			t.Errorf("Failed to initialize parsed transaction: %v", err)
			return
		}

		// Verify ID matches
		if baseTx.ID() != parsed.ID() {
			t.Errorf("Transaction ID mismatch after parsing")
		}
	})
}

// visitor implements the Visitor interface for testing
type visitor struct{}

func (v *visitor) AddDelegatorTx(*AddDelegatorTx) error { return nil }
func (v *visitor) AddChainValidatorTx(*AddChainValidatorTx) error { return nil }
func (v *visitor) AddPermissionlessDelegatorTx(*AddPermissionlessDelegatorTx) error { return nil }
func (v *visitor) AddPermissionlessValidatorTx(*AddPermissionlessValidatorTx) error { return nil }
func (v *visitor) AddValidatorTx(*AddValidatorTx) error { return nil }
func (v *visitor) AdvanceTimeTx(*AdvanceTimeTx) error { return nil }
func (v *visitor) BaseTx(*BaseTx) error { return nil }
func (v *visitor) CreateChainTx(*CreateChainTx) error { return nil }
func (v *visitor) CreateSubnetTx(*CreateSubnetTx) error { return nil }
func (v *visitor) ExportTx(*ExportTx) error { return nil }
func (v *visitor) ImportTx(*ImportTx) error { return nil }
func (v *visitor) RemoveChainValidatorTx(*RemoveChainValidatorTx) error { return nil }
func (v *visitor) RewardValidatorTx(*RewardValidatorTx) error { return nil }
func (v *visitor) TransferChainOwnershipTx(*TransferChainOwnershipTx) error { return nil }
func (v *visitor) TransformChainTx(*TransformChainTx) error { return nil }
func (v *visitor) ConvertChainToL1Tx(*ConvertChainToL1Tx) error { return nil }
func (v *visitor) RegisterL1ValidatorTx(*RegisterL1ValidatorTx) error { return nil }
func (v *visitor) SetL1ValidatorWeightTx(*SetL1ValidatorWeightTx) error { return nil }
func (v *visitor) DisableL1ValidatorTx(*DisableL1ValidatorTx) error { return nil }
func (v *visitor) IncreaseL1ValidatorBalanceTx(*IncreaseL1ValidatorBalanceTx) error { return nil }