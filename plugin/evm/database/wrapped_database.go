// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package database

import (
	"errors"

	"github.com/luxfi/database"
	"github.com/luxfi/geth/ethdb"
)

var (
	_ ethdb.KeyValueStore = (*ethDbWrapper)(nil)

	ErrSnapshotNotSupported = errors.New("snapshot is not supported")
)

// ethDbWrapper implements ethdb.Database
type ethDbWrapper struct{ database.Database }

func WrapDatabase(db database.Database) ethdb.KeyValueStore { return ethDbWrapper{db} }

// Stat implements ethdb.Database
func (db ethDbWrapper) Stat() (string, error) { return "", errors.New("stat not supported") }

// NewBatch implements ethdb.Database
func (db ethDbWrapper) NewBatch() ethdb.Batch { return &wrappedBatch{db.Database.NewBatch()} }

// NewBatchWithSize implements ethdb.Database
// TODO: propagate size through luxd Database interface
func (db ethDbWrapper) NewBatchWithSize(size int) ethdb.Batch {
	return &wrappedBatch{db.Database.NewBatch()}
}

func (db ethDbWrapper) NewSnapshot() (interface{}, error) {
	return nil, ErrSnapshotNotSupported
}

// DeleteRange implements ethdb.KeyValueRangeDeleter
func (db ethDbWrapper) DeleteRange(start, end []byte) error {
	return errors.New("DeleteRange not supported")
}

// SyncKeyValue implements ethdb.KeyValueStore
func (db ethDbWrapper) SyncKeyValue() error {
	return nil
}

// NewIterator implements ethdb.Database
//
// Note: This method assumes that the prefix is NOT part of the start, so there's
// no need for the caller to prepend the prefix to the start.
func (db ethDbWrapper) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	// luxd's database implementation assumes that the prefix is part of the
	// start, so it is added here (if it is provided).
	if len(prefix) > 0 {
		newStart := make([]byte, len(prefix)+len(start))
		copy(newStart, prefix)
		copy(newStart[len(prefix):], start)
		start = newStart
	}
	return db.NewIteratorWithStartAndPrefix(start, prefix)
}

// NewIteratorWithStart implements ethdb.Database
func (db ethDbWrapper) NewIteratorWithStart(start []byte) ethdb.Iterator {
	return db.Database.NewIteratorWithStart(start)
}

// wrappedBatch implements ethdb.wrappedBatch
type wrappedBatch struct{ batch database.Batch }

// ValueSize implements ethdb.Batch
func (batch wrappedBatch) ValueSize() int { return batch.batch.Size() }

// Replay implements ethdb.Batch
func (batch wrappedBatch) Replay(w ethdb.KeyValueWriter) error { return batch.batch.Replay(w) }

// DeleteRange implements ethdb.KeyValueRangeDeleter for the batch
func (batch wrappedBatch) DeleteRange(start, end []byte) error {
	return errors.New("DeleteRange not supported in batch")
}

// Implement missing methods from database.Batch
func (batch wrappedBatch) Put(key []byte, value []byte) error {
	return batch.batch.Put(key, value)
}

func (batch wrappedBatch) Delete(key []byte) error {
	return batch.batch.Delete(key)
}

func (batch wrappedBatch) Write() error {
	return batch.batch.Write()
}

func (batch wrappedBatch) Reset() {
	batch.batch.Reset()
}
