// Copyright (C) 2020-2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

// Example: Deploy a subnet using the SDK deploy package
//
// This example demonstrates how to deploy a subnet with a blockchain
// to any Lux network endpoint using the SDK deploy package.
//
// Usage:
//
//	go run main.go --endpoint http://127.0.0.1:9628 --genesis /path/to/genesis.json --name ZOO
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/deploy"
)

func main() {
	// Parse command line flags
	endpoint := flag.String("endpoint", "http://127.0.0.1:9650", "RPC endpoint URI")
	genesisFile := flag.String("genesis", "", "Path to genesis.json file")
	chainName := flag.String("name", "MyChain", "Name of the blockchain")
	flag.Parse()

	if *genesisFile == "" {
		fmt.Println("Error: --genesis flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Read genesis file
	genesisBytes, err := os.ReadFile(*genesisFile)
	if err != nil {
		fmt.Printf("Failed to read genesis file: %v\n", err)
		os.Exit(1)
	}

	// Create deployer with EWOQ (genesis) key for local development
	deployer, err := deploy.New(
		deploy.WithEndpoint(*endpoint),
		deploy.WithKeychain(deploy.EWOQKeychain()),
		deploy.WithTimeout(5*time.Minute),
	)
	if err != nil {
		fmt.Printf("Failed to create deployer: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("=== Deploying Subnet ===\n")
	fmt.Printf("Endpoint:   %s\n", deployer.Endpoint())
	fmt.Printf("Chain Name: %s\n", *chainName)
	fmt.Printf("VM ID:      %s\n", deploy.SubnetEVMID)

	// Get the control key address from the keychain
	kc := deploy.EWOQKeychain()
	addrs := kc.Addresses().List()
	if len(addrs) == 0 {
		fmt.Println("Error: no addresses in keychain")
		os.Exit(1)
	}
	controlKey := addrs[0]
	fmt.Printf("Control:    %s\n", controlKey)
	fmt.Println()

	ctx := context.Background()

	// Deploy subnet with blockchain
	result, err := deployer.DeploySubnet(
		ctx,
		genesisBytes,
		deploy.SubnetEVMID,
		*chainName,
		[]ids.ShortID{controlKey},
		1, // threshold
	)
	if err != nil {
		fmt.Printf("Failed to deploy subnet: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("=== Success ===\n")
	fmt.Printf("Subnet ID:     %s\n", result.SubnetID)
	fmt.Printf("Blockchain ID: %s\n", result.BlockchainID)
	fmt.Printf("RPC Endpoint:  %s/ext/bc/%s/rpc\n", *endpoint, result.BlockchainID)
}
