// Package set provides a generic set implementation
package set

// Set is a generic set type
type Set[T comparable] map[T]struct{}

// New creates a new set
func New[T comparable]() Set[T] {
	return make(Set[T])
}

// Add adds an element to the set
func (s Set[T]) Add(v T) {
	s[v] = struct{}{}
}

// Contains checks if an element is in the set
func (s Set[T]) Contains(v T) bool {
	_, ok := s[v]
	return ok
}

// Remove removes an element from the set
func (s Set[T]) Remove(v T) {
	delete(s, v)
}

// Size returns the number of elements in the set
func (s Set[T]) Size() int {
	return len(s)
}