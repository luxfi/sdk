package memorydb

import "github.com/luxfi/geth/ethdb/memorydb"

// Re-export memorydb functions
var New = memorydb.New
var NewWithCap = memorydb.NewWithCap
