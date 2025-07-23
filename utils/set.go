// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"github.com/luxfi/evm/interfaces"
)

// Set implements interfaces.Set
type Set[T comparable] struct {
	items map[T]struct{}
}

// NewSet creates a new set with optional initial capacity
func NewSet[T comparable](capacity ...int) interfaces.GenericSet[T] {
	cap := 0
	if len(capacity) > 0 {
		cap = capacity[0]
	}
	return &Set[T]{
		items: make(map[T]struct{}, cap),
	}
}

// Add adds items to the set
func (s *Set[T]) Add(items ...T) {
	for _, item := range items {
		s.items[item] = struct{}{}
	}
}

// Contains checks if an item is in the set
func (s *Set[T]) Contains(item T) bool {
	_, ok := s.items[item]
	return ok
}

// Remove removes an item from the set
func (s *Set[T]) Remove(item T) {
	delete(s.items, item)
}

// Clear clears all items
func (s *Set[T]) Clear() {
	s.items = make(map[T]struct{})
}

// Len returns the number of items
func (s *Set[T]) Len() int {
	return len(s.items)
}

// List returns all items as a slice
func (s *Set[T]) List() []T {
	result := make([]T, 0, len(s.items))
	for item := range s.items {
		result = append(result, item)
	}
	return result
}