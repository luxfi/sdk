package snapshot

import (
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/ethdb"
)

// snapshotBlockHashKey = snapshotPrefix + blockHash
var snapshotBlockHashKey = []byte("SnapshotBlockHash")

// WriteSnapshotBlockHash writes the block hash of a snapshot.
func WriteSnapshotBlockHash(db ethdb.KeyValueWriter, blockHash common.Hash) {
	if err := db.Put(snapshotBlockHashKey, blockHash.Bytes()); err != nil {
		panic(err)
	}
}

// ReadSnapshotBlockHash reads the block hash of a snapshot.
func ReadSnapshotBlockHash(db ethdb.KeyValueReader) common.Hash {
	data, _ := db.Get(snapshotBlockHashKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// DeleteSnapshotBlockHash deletes the block hash of a snapshot.
func DeleteSnapshotBlockHash(db ethdb.KeyValueWriter) {
	if err := db.Delete(snapshotBlockHashKey); err != nil {
		panic(err)
	}
}