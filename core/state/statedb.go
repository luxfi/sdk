// (c) 2019-2020, Hanzo Industries, Inc.
//
// This file is a derived work, based on the go-ethereum library whose original
// notices appear below.
//
// It is distributed under a license compatible with the licensing terms of the
// original code from which it is derived.
//
// Much love to the original authors for their work.
// **********
// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package state provides a caching layer atop the Ethereum state trie.
package state

import (
	ethstate "github.com/luxfi/geth/core/state"
	"github.com/luxfi/geth/core/state/snapshot"
	"github.com/luxfi/geth/common"
)

// StateDB wraps go-ethereum's StateDB with minimal extensions
type StateDB struct {
	*ethstate.StateDB
	thash common.Hash
}

// New creates a new state from a given trie.
func New(root common.Hash, db Database, snaps *snapshot.Tree) (*StateDB, error) {
	ethStateDB, err := ethstate.New(root, db, snaps)
	if err != nil {
		return nil, err
	}
	return &StateDB{StateDB: ethStateDB}, nil
}

// GetTxHash returns the current transaction hash
func (s *StateDB) GetTxHash() common.Hash {
	return s.thash
}

// SetTxContext sets the current transaction hash and index
func (s *StateDB) SetTxContext(thash common.Hash, ti int) {
	s.thash = thash
	s.StateDB.SetTxContext(thash, ti)
}

