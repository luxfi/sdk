// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
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
	ethstate "github.com/luxfi/geth/core/state"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/triedb"
	// "github.com/luxfi/evm/triedb/firewood"
)

type (
	Database = ethstate.Database
	Trie     = ethstate.Trie
)

func NewDatabase(db ethdb.Database) Database {
	// TODO: NewDatabase now requires triedb and snapshot tree parameters
	// Using nil for now, may need to be updated based on usage
	trieDB := triedb.NewDatabase(db, nil)
	return ethstate.NewDatabase(trieDB, nil)
}

func NewDatabaseWithConfig(db ethdb.Database, config *triedb.Config) Database {
	// TODO: NewDatabaseWithConfig seems to be removed, using NewDatabase instead
	trieDB := triedb.NewDatabase(db, config)
	return ethstate.NewDatabase(trieDB, nil)
}

func NewDatabaseWithNodeDB(db ethdb.Database, tdb *triedb.Database) Database {
	// TODO: NewDatabaseWithNodeDB seems to be removed, using NewDatabase instead
	return ethstate.NewDatabase(tdb, nil)
}

// func wrapIfFirewood(db Database) Database {
// 	fw, ok := db.TrieDB().Backend().(*firewood.Database)
// 	if !ok {
// 		return db
// 	}
// 	return &firewoodAccessorDb{
// 		Database: db,
// 		fw:       fw,
// 	}
// }
