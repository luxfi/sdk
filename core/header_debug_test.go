package core

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/rlp"
	"github.com/stretchr/testify/require"
)

func TestHeaderEncoding(t *testing.T) {
	require := require.New(t)

	// Create a simple header
	header := &types.Header{
		ParentHash:  common.Hash{1},
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.Address{},
		Root:        common.Hash{2},
		TxHash:      types.EmptyTxsHash,
		ReceiptHash: types.EmptyReceiptsHash,
		Bloom:       types.Bloom{},
		Difficulty:  big.NewInt(1),
		Number:      big.NewInt(0),
		GasLimit:    1000000,
		GasUsed:     0,
		Time:        1000,
		Extra:       []byte{},
		MixDigest:   common.Hash{},
		Nonce:       types.BlockNonce{},
		BaseFee:     big.NewInt(875000000),
	}

	// Encode it manually
	buf := new(bytes.Buffer)
	err := rlp.Encode(buf, header)
	require.NoError(err, "Failed to encode header")

	encodedBytes := buf.Bytes()
	t.Logf("Encoded header size: %d bytes", len(encodedBytes))
	t.Logf("First 32 bytes: %x", encodedBytes[:32])

	// Try to decode it back
	var decoded types.Header
	err = rlp.DecodeBytes(encodedBytes, &decoded)
	require.NoError(err, "Failed to decode header")

	// Verify fields match
	require.Equal(header.Number.Uint64(), decoded.Number.Uint64(), "Number mismatch")
	require.Equal(header.BaseFee.Uint64(), decoded.BaseFee.Uint64(), "BaseFee mismatch")

	// Now test with database
	db := rawdb.NewMemoryDatabase()

	// Compute hash
	hash := header.Hash()
	t.Logf("Header hash: %s", hash.Hex())

	// Write using rawdb
	rawdb.WriteHeader(db, header)

	// Check what was written
	it := db.NewIterator(nil, nil)
	defer it.Release()

	count := 0
	for it.Next() {
		key := it.Key()
		value := it.Value()
		t.Logf("DB Key[%d]: %x (len=%d)", count, key, len(key))
		t.Logf("DB Value[%d]: len=%d", count, len(value))

		// If this looks like our header key, try to decode the value
		if len(key) == 41 && key[0] == 'h' {
			var storedHeader types.Header
			err := rlp.DecodeBytes(value, &storedHeader)
			if err != nil {
				t.Logf("  Failed to decode as header: %v", err)
			} else {
				t.Logf("  Successfully decoded as header with number=%d", storedHeader.Number.Uint64())
			}
		}
		count++
	}

	// Try to read it back
	readHeader := rawdb.ReadHeader(db, hash, 0)
	require.NotNil(readHeader, "Should be able to read header back")
}
