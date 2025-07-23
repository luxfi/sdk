// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package interfaces

import (
	"errors"
	"io"

	"github.com/luxfi/geth/ethdb"
)

// Database errors
var (
	ErrNotFound = errors.New("not found")
	ErrClosed   = errors.New("database closed")
)

// Database wraps all database operations for node compatibility
type Database interface {
	KeyValueReader
	KeyValueWriter
	KeyValueDeleter
	Batcher
	Iteratee
	Compacter
	io.Closer
	HealthChecker
}

// HealthChecker provides health check capability
type HealthChecker interface {
	HealthCheck() (interface{}, error)
}

// Batch represents a batch operation
type Batch interface {
	ethdb.Batch
}

// Iterator represents an iterator
type Iterator interface {
	ethdb.Iterator
}

// NewPrefixDB creates a new prefix database
func NewPrefixDB(prefix []byte, db Database) Database {
	// This is a placeholder - in real implementation, this would wrap the database
	// For now, return the same database
	return db
}

// NewVersionDB creates a new version database
func NewVersionDB(db Database) VersionDB {
	// This is a placeholder - in real implementation, this would create a versioned database
	// For now, return nil
	return nil
}

// KeyValueReader wraps the Has and Get method of a backing data store.
type KeyValueReader interface {
	// Has retrieves if a key is present in the key-value data store.
	Has(key []byte) (bool, error)

	// Get retrieves the given key if it's present in the key-value data store.
	Get(key []byte) ([]byte, error)
}

// KeyValueWriter wraps the Put method of a backing data store.
type KeyValueWriter interface {
	// Put inserts the given value into the key-value data store.
	Put(key []byte, value []byte) error
}

// KeyValueDeleter wraps the Delete method of a backing data store.
type KeyValueDeleter interface {
	// Delete removes the key from the key-value data store.
	Delete(key []byte) error
}

// Batcher wraps batch operations
type Batcher interface {
	// NewBatch creates a new batch
	NewBatch() ethdb.Batch
}

// Iteratee wraps iteration operations
type Iteratee interface {
	// NewIterator creates a new iterator
	NewIterator(prefix []byte, start []byte) ethdb.Iterator
}

// Compacter wraps compaction operations
type Compacter interface {
	// Compact compacts the underlying DB
	Compact(start []byte, limit []byte) error
}

// PrefixDB wraps a database with a key prefix
type PrefixDB interface {
	Database
}

// VersionDB provides versioned database operations
type VersionDB interface {
	Database
	// Commit commits the current database state
	Commit() error
	// Abort aborts the current database operations
	Abort()
}