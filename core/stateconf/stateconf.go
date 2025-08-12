// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package stateconf

// SnapshotUpdateOption is a placeholder for snapshot update options
// This is implemented as an empty interface for now, but can be expanded
// to carry payloads as needed.
type SnapshotUpdateOption interface{}

// TrieDBUpdateOption is a placeholder for trie database update options
// This is implemented as an empty interface for now, but can be expanded
// to carry payloads as needed.
type TrieDBUpdateOption interface{}

// snapshotUpdatePayload represents a snapshot update with payload
type snapshotUpdatePayload struct {
	payload interface{}
}

// WithSnapshotUpdatePayload returns a SnapshotUpdateOption carrying an arbitrary payload
func WithSnapshotUpdatePayload(p interface{}) SnapshotUpdateOption {
	return &snapshotUpdatePayload{payload: p}
}

// ExtractSnapshotUpdatePayload extracts the payload from snapshot update options
func ExtractSnapshotUpdatePayload(opts ...SnapshotUpdateOption) interface{} {
	for _, opt := range opts {
		if p, ok := opt.(*snapshotUpdatePayload); ok {
			return p.payload
		}
	}
	return nil
}

// trieDBUpdatePayload represents a trie DB update with block hashes
type trieDBUpdatePayload struct {
	parentBlockHash  interface{} // Using interface{} to avoid importing common.Hash
	currentBlockHash interface{}
}

// WithTrieDBUpdatePayload returns a TrieDBUpdateOption carrying two block hashes
func WithTrieDBUpdatePayload(parent interface{}, current interface{}) TrieDBUpdateOption {
	return &trieDBUpdatePayload{
		parentBlockHash:  parent,
		currentBlockHash: current,
	}
}

// ExtractTrieDBUpdatePayload extracts the payload from trie DB update options
func ExtractTrieDBUpdatePayload(opts ...TrieDBUpdateOption) (interface{}, interface{}, bool) {
	for _, opt := range opts {
		if p, ok := opt.(*trieDBUpdatePayload); ok {
			return p.parentBlockHash, p.currentBlockHash, true
		}
	}
	return nil, nil, false
}