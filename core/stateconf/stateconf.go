// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package stateconf

// TrieDBUpdateOption is a functional option for trie database updates
type TrieDBUpdateOption func()

// SnapshotUpdateOption is a functional option for snapshot updates
type SnapshotUpdateOption func()