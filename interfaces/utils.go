// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package interfaces

import (
	"context"
	"time"
)

// Logger provides logging functionality
type Logger interface {
	// Fatal logs a fatal error and exits
	Fatal(msg string, keyVals ...interface{})
	
	// Error logs an error
	Error(msg string, keyVals ...interface{})
	
	// Warn logs a warning
	Warn(msg string, keyVals ...interface{})
	
	// Info logs an info message
	Info(msg string, keyVals ...interface{})
	
	// Debug logs a debug message
	Debug(msg string, keyVals ...interface{})
	
	// Trace logs a trace message
	Trace(msg string, keyVals ...interface{})
	
	// With returns a logger with additional context
	With(keyVals ...interface{}) Logger
}

// Constants provides access to network constants
type Constants interface {
	// NetworkID returns the network ID
	NetworkID() uint32
	
	// NetworkName returns the network name
	NetworkName() string
}

// Units provides unit conversions
const (
	// Storage units
	Byte = 1
	KiB  = 1024 * Byte
	MiB  = 1024 * KiB
	GiB  = 1024 * MiB
	
	// Time units
	Nanosecond  = time.Nanosecond
	Microsecond = time.Microsecond
	Millisecond = time.Millisecond
	Second      = time.Second
	Minute      = time.Minute
	Hour        = time.Hour
)

// Timer provides timing functionality
type MockableTimer interface {
	// Time returns the current time
	Time() time.Time
	
	// Set sets the current time (for testing)
	Set(time.Time)
	
	// Advance advances the time by duration (for testing)
	Advance(time.Duration)
}

// Wrappers provides error wrapping utilities
type Wrappers interface {
	// Errs wraps multiple errors
	Errs(errs ...error) error
}

// Profiler provides profiling functionality
type Profiler interface {
	// StartCPUProfiler starts CPU profiling
	StartCPUProfiler() error
	
	// StopCPUProfiler stops CPU profiling
	StopCPUProfiler() error
	
	// MemoryProfile captures a memory profile
	MemoryProfile() error
}

// Permission utilities
type FilePermissions uint32

const (
	// ReadOnly file permission
	ReadOnly FilePermissions = 0o444
	// ReadWrite file permission
	ReadWrite FilePermissions = 0o644
	// ReadWriteExecute file permission
	ReadWriteExecute FilePermissions = 0o755
)

// JSON utilities for serialization
type JSON interface {
	// Marshal converts a Go value to JSON
	Marshal(v interface{}) ([]byte, error)
	
	// Unmarshal parses JSON data
	Unmarshal(data []byte, v interface{}) error
}

// Cache provides caching functionality
type Cache interface {
	// Get retrieves a value from cache
	Get(key interface{}) (interface{}, bool)
	
	// Put stores a value in cache
	Put(key interface{}, value interface{})
	
	// Evict removes a value from cache
	Evict(key interface{})
	
	// Flush clears the cache
	Flush()
}

// LRU provides LRU cache functionality
type LRU interface {
	Cache
	// Len returns the number of items in cache
	Len() int
}

// BoundedBuffer provides a bounded buffer
type BoundedBuffer interface {
	// Put adds an item to the buffer
	Put(ctx context.Context, item interface{}) error
	
	// Get retrieves an item from the buffer
	Get(ctx context.Context) (interface{}, error)
	
	// Len returns the number of items in the buffer
	Len() int
	
	// Close closes the buffer
	Close()
}

// Bits provides a bit set interface
type Bits interface {
	// Add adds a bit to the set
	Add(i int)
	
	// Contains checks if a bit is in the set
	Contains(i int) bool
	
	// Remove removes a bit from the set
	Remove(i int)
	
	// Clear clears all bits
	Clear()
	
	// Len returns the number of bits set
	Len() int
	
	// Bytes returns the byte representation
	Bytes() []byte
}

// GenericSet provides a generic set interface
type GenericSet[T comparable] interface {
	// Add adds items to the set
	Add(items ...T)
	
	// Contains checks if an item is in the set
	Contains(item T) bool
	
	// Remove removes an item from the set
	Remove(item T)
	
	// Clear clears all items
	Clear()
	
	// Len returns the number of items
	Len() int
	
	// List returns all items as a slice
	List() []T
}

// Cacher provides a generic cache interface
type Cacher[K comparable, V any] interface {
	// Put adds a value to the cache
	Put(key K, value V)
	
	// Get retrieves a value from the cache
	Get(key K) (V, bool)
	
	// Evict removes a value from the cache
	Evict(key K)
	
	// Flush clears the cache
	Flush()
	
	// Len returns the number of items in the cache
	Len() int
}

// Formatting provides encoding/decoding utilities
type Formatting interface {
	// Encode encodes bytes to string
	Encode(encoding Encoding, bytes []byte) (string, error)
	
	// Decode decodes string to bytes
	Decode(encoding Encoding, str string) ([]byte, error)
}

// Encoding represents an encoding type
type Encoding uint8

const (
	// CB58 encoding
	CB58 Encoding = iota
	// Hex encoding
	Hex
	// JSONEncoding for JSON format
	JSONEncoding
)