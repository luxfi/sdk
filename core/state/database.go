// (c) 2019-2025, Hanzo Industries, Inc.
//
// This file is a derived work, based on the go-ethereum library whose original
// notices appear below.
//
// It is distributed under a license compatible with the licensing terms of the
// original code from which it is derived.
//
// Much love to the original authors for their work.
// **********
// Copyright 2017 The go-ethereum Authors
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

package state

import (
	"github.com/ethereum/go-ethereum/ethdb"
	ethstate "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/triedb"
)

type (
	Database = ethstate.Database
	Trie     = ethstate.Trie
)

func NewDatabase(db ethdb.Database) ethstate.Database {
	triedb := triedb.NewDatabase(db, &triedb.Config{
		Preimages: true,
	})
	return ethstate.NewDatabase(triedb, nil)
}

func NewDatabaseWithConfig(db ethdb.Database, config *triedb.Config) ethstate.Database {
	triedb := triedb.NewDatabase(db, config)
	return ethstate.NewDatabase(triedb, nil)
}

func NewDatabaseWithNodeDB(db ethdb.Database, triedb *triedb.Database) ethstate.Database {
	return ethstate.NewDatabase(triedb, nil)
}
