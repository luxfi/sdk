// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package pool

import (
	"bytes"
	"sync"
)

// ObjectPool provides a generic object pool for reducing allocations
type ObjectPool[T any] struct {
	pool  sync.Pool
	new   func() T
	reset func(T)
}

// NewObjectPool creates a new object pool
func NewObjectPool[T any](new func() T, reset func(T)) *ObjectPool[T] {
	return &ObjectPool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return new()
			},
		},
		new:   new,
		reset: reset,
	}
}

// Get retrieves an object from the pool
func (p *ObjectPool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put returns an object to the pool after resetting it
func (p *ObjectPool[T]) Put(obj T) {
	if p.reset != nil {
		p.reset(obj)
	}
	p.pool.Put(obj)
}

// ByteSlicePool is a pool for byte slices of a specific size
type ByteSlicePool struct {
	pool sync.Pool
	size int
}

// NewByteSlicePool creates a pool for byte slices
func NewByteSlicePool(size int) *ByteSlicePool {
	return &ByteSlicePool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
		size: size,
	}
}

// Get retrieves a byte slice from the pool
func (p *ByteSlicePool) Get() []byte {
	return p.pool.Get().([]byte)
}

// Put returns a byte slice to the pool
func (p *ByteSlicePool) Put(b []byte) {
	if cap(b) >= p.size {
		p.pool.Put(b[:p.size])
	}
}

// BufferPool is a pool for bytes.Buffer objects
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a new buffer pool
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (p *BufferPool) Get() *bytes.Buffer {
	buf := p.pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// Put returns a buffer to the pool
func (p *BufferPool) Put(buf *bytes.Buffer) {
	if buf.Cap() > 1024*1024 { // Don't pool buffers larger than 1MB
		return
	}
	buf.Reset()
	p.pool.Put(buf)
}

// MapPool provides pooling for maps
type MapPool[K comparable, V any] struct {
	pool sync.Pool
}

// NewMapPool creates a new map pool
func NewMapPool[K comparable, V any]() *MapPool[K, V] {
	return &MapPool[K, V]{
		pool: sync.Pool{
			New: func() interface{} {
				return make(map[K]V)
			},
		},
	}
}

// Get retrieves a map from the pool
func (p *MapPool[K, V]) Get() map[K]V {
	return p.pool.Get().(map[K]V)
}

// Put returns a map to the pool after clearing it
func (p *MapPool[K, V]) Put(m map[K]V) {
	// Clear the map
	for k := range m {
		delete(m, k)
	}
	p.pool.Put(m)
}

// SlicePool provides pooling for slices
type SlicePool[T any] struct {
	pool sync.Pool
}

// NewSlicePool creates a new slice pool
func NewSlicePool[T any](initialCap int) *SlicePool[T] {
	return &SlicePool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]T, 0, initialCap)
			},
		},
	}
}

// Get retrieves a slice from the pool
func (p *SlicePool[T]) Get() []T {
	return p.pool.Get().([]T)
}

// Put returns a slice to the pool after clearing it
func (p *SlicePool[T]) Put(s []T) {
	p.pool.Put(s[:0])
}
