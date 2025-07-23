// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"sync"

	"github.com/luxfi/evm/interfaces"
)

// LRUCache implements an LRU cache using a simple map with a fixed size
type LRUCache[K comparable, V any] struct {
	cache    map[K]V
	keys     []K
	capacity int
	mu       sync.RWMutex
}

// NewLRUCache creates a new LRU cache with the specified capacity
func NewLRUCache[K comparable, V any](capacity int) interfaces.Cacher[K, V] {
	if capacity <= 0 {
		capacity = 1
	}
	return &LRUCache[K, V]{
		cache:    make(map[K]V, capacity),
		keys:     make([]K, 0, capacity),
		capacity: capacity,
	}
}

// Put adds a value to the cache
func (c *LRUCache[K, V]) Put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if key already exists
	if _, exists := c.cache[key]; exists {
		// Update existing value
		c.cache[key] = value
		// Move key to end (most recently used)
		c.moveToEnd(key)
		return
	}

	// If cache is full, evict oldest
	if len(c.cache) >= c.capacity {
		oldestKey := c.keys[0]
		delete(c.cache, oldestKey)
		c.keys = c.keys[1:]
	}

	// Add new entry
	c.cache[key] = value
	c.keys = append(c.keys, key)
}

// Get retrieves a value from the cache
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, exists := c.cache[key]
	if exists {
		// Move to end (most recently used)
		c.moveToEnd(key)
	}
	return value, exists
}

// Evict removes a key from the cache
func (c *LRUCache[K, V]) Evict(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.cache[key]; !exists {
		return
	}

	delete(c.cache, key)
	// Remove from keys list
	for i, k := range c.keys {
		if k == key {
			c.keys = append(c.keys[:i], c.keys[i+1:]...)
			break
		}
	}
}

// Flush clears the cache
func (c *LRUCache[K, V]) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[K]V, c.capacity)
	c.keys = c.keys[:0]
}

// Len returns the number of items in the cache
func (c *LRUCache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// moveToEnd moves a key to the end of the LRU list (most recently used)
func (c *LRUCache[K, V]) moveToEnd(key K) {
	for i, k := range c.keys {
		if k == key {
			// Remove from current position
			c.keys = append(c.keys[:i], c.keys[i+1:]...)
			// Add to end
			c.keys = append(c.keys, key)
			break
		}
	}
}

// EmptyCache is a cache that never stores anything
type EmptyCache[K comparable, V any] struct{}

// Put does nothing
func (e *EmptyCache[K, V]) Put(key K, value V) {}

// Get always returns false
func (e *EmptyCache[K, V]) Get(key K) (V, bool) {
	var zero V
	return zero, false
}

// Evict does nothing
func (e *EmptyCache[K, V]) Evict(key K) {}

// Flush does nothing
func (e *EmptyCache[K, V]) Flush() {}

// Len always returns 0
func (e *EmptyCache[K, V]) Len() int {
	return 0
}