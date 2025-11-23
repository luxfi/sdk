package main

import (
	"fmt"
	"os"

	"github.com/luxfi/database/factory"
	"github.com/luxfi/log"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	dbPath := "/Users/z/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb"

	// Create logger
	logger := log.New("info")

	// Create registry
	registerer := prometheus.NewRegistry()

	// Test readonly access
	fmt.Printf("Testing readonly access to database: %s\n", dbPath)

	// func New(name string, dbPath string, readOnly bool, config []byte, gatherer interface{}, logger log.Logger, metricsPrefix string, meterDBRegName string)
	db, err := factory.New("pebbledb", dbPath, true, []byte{}, registerer, logger, "test", "")
	if err != nil {
		fmt.Printf("❌ Failed to open database in readonly mode: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("✓ Successfully opened database in readonly mode!")

	// Try to read some keys
	iter := db.NewIterator()
	defer iter.Release()

	count := 0
	for iter.Next() && count < 10 {
		key := iter.Key()
		fmt.Printf("Key %d: %x (len=%d)\n", count, key[:min(len(key), 32)], len(key))
		count++
	}

	if err := iter.Error(); err != nil {
		fmt.Printf("❌ Iterator error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Successfully read %d keys from readonly database\n", count)
	fmt.Println("✓ Readonly mode is working correctly!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
