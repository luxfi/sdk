package memorydb

import "github.com/ethereum/go-ethereum/ethdb/memorydb"

// Re-export memorydb functions
var New = memorydb.New
var NewWithCap = memorydb.NewWithCap
