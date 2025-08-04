// Copyright (C) 2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/luxfi/sdk/blockchain"
	"github.com/luxfi/sdk/config"
	"github.com/luxfi/sdk/internal/logging"
	"github.com/luxfi/sdk/network"
)

func main() {
	// Create logger
	logger := logging.NewLogger("info")
	
	// Create network configuration for mainnet
	networkConfig := &config.NetworkConfig{
		NetworkID:     96369, // Lux mainnet ID
		APIEndpoint:   "https://api.mainnet.lux.network",
		P2PPort:       9651,
		HTTPPort:      9650,
		StakingPort:   9651,
		LogLevel:      "info",
	}

	// Create network manager
	nm, err := network.NewNetworkManager(networkConfig, logger)
	if err != nil {
		log.Fatalf("Failed to create network manager: %v", err)
	}

	// Create blockchain builder
	builder := blockchain.NewBuilder(logger)

	// Boot mainnet
	fmt.Println("Booting Lux Mainnet...")
	fmt.Printf("Network ID: %d\n", networkConfig.NetworkID)
	fmt.Printf("API Endpoint: %s\n", networkConfig.APIEndpoint)
	
	// Create mainnet network parameters
	mainnetParams := &network.NetworkParams{
		Name:             "lux-mainnet",
		Type:             network.NetworkTypeMainnet,
		NumNodes:         21, // Mainnet has 21 validators
		EnableMonitoring: true,
		HTTPPort:         networkConfig.HTTPPort,
		StakingPort:      networkConfig.StakingPort,
		EnableStaking:    true,
	}

	// Create the mainnet network
	ctx := context.Background()
	mainnet, err := nm.CreateNetwork(ctx, mainnetParams)
	if err != nil {
		log.Fatalf("Failed to create mainnet: %v", err)
	}

	fmt.Printf("\nMainnet created:\n")
	fmt.Printf("- ID: %s\n", mainnet.ID)
	fmt.Printf("- Name: %s\n", mainnet.Name)
	fmt.Printf("- Type: %s\n", mainnet.Type)
	fmt.Printf("- Status: %s\n", mainnet.Status)
	fmt.Printf("- Nodes: %d\n", len(mainnet.Nodes))

	// Display node information
	fmt.Println("\nValidator Nodes:")
	for i, node := range mainnet.Nodes {
		fmt.Printf("%d. %s - %s (Stake: %d LUX)\n", i+1, node.NodeID, node.Status, node.StakeAmount)
	}

	// Create P-Chain configuration
	fmt.Println("\nCreating P-Chain configuration...")
	pChainParams := &blockchain.CreateParams{
		Name:    "P-Chain",
		Type:    blockchain.TypeL1,
		VMType:  blockchain.VMTypeEVM,
		ChainID: networkConfig.ChainID(),
	}

	pChain, err := builder.CreateBlockchain(ctx, pChainParams)
	if err != nil {
		log.Fatalf("Failed to create P-Chain: %v", err)
	}

	fmt.Printf("\nP-Chain created:\n")
	fmt.Printf("- ID: %s\n", pChain.ID)
	fmt.Printf("- Name: %s\n", pChain.Name)
	fmt.Printf("- Chain ID: %s\n", pChain.ChainID.String())
	fmt.Printf("- Status: %s\n", pChain.Status)

	// Deploy P-Chain to mainnet
	fmt.Println("\nDeploying P-Chain to mainnet...")
	err = builder.Deploy(ctx, pChain, mainnet)
	if err != nil {
		log.Fatalf("Failed to deploy P-Chain: %v", err)
	}

	fmt.Println("P-Chain deployed successfully!")

	// Create C-Chain configuration
	fmt.Println("\nCreating C-Chain configuration...")
	cChainParams := &blockchain.CreateParams{
		Name:    "C-Chain",
		Type:    blockchain.TypeL1,
		VMType:  blockchain.VMTypeEVM,
		ChainID: networkConfig.ChainID(),
	}

	cChain, err := builder.CreateBlockchain(ctx, cChainParams)
	if err != nil {
		log.Fatalf("Failed to create C-Chain: %v", err)
	}

	fmt.Printf("\nC-Chain created:\n")
	fmt.Printf("- ID: %s\n", cChain.ID)
	fmt.Printf("- Name: %s\n", cChain.Name)
	fmt.Printf("- Chain ID: %s\n", cChain.ChainID.String())
	fmt.Printf("- Status: %s\n", cChain.Status)

	// Deploy C-Chain to mainnet
	fmt.Println("\nDeploying C-Chain to mainnet...")
	err = builder.Deploy(ctx, cChain, mainnet)
	if err != nil {
		log.Fatalf("Failed to deploy C-Chain: %v", err)
	}

	fmt.Println("C-Chain deployed successfully!")

	// Create X-Chain configuration
	fmt.Println("\nCreating X-Chain configuration...")
	xChainParams := &blockchain.CreateParams{
		Name:    "X-Chain",
		Type:    blockchain.TypeL1,
		VMType:  blockchain.VMTypeTokenVM,
		ChainID: networkConfig.ChainID(),
	}

	xChain, err := builder.CreateBlockchain(ctx, xChainParams)
	if err != nil {
		log.Fatalf("Failed to create X-Chain: %v", err)
	}

	fmt.Printf("\nX-Chain created:\n")
	fmt.Printf("- ID: %s\n", xChain.ID)
	fmt.Printf("- Name: %s\n", xChain.Name)
	fmt.Printf("- Chain ID: %s\n", xChain.ChainID.String())
	fmt.Printf("- Status: %s\n", xChain.Status)

	// Deploy X-Chain to mainnet
	fmt.Println("\nDeploying X-Chain to mainnet...")
	err = builder.Deploy(ctx, xChain, mainnet)
	if err != nil {
		log.Fatalf("Failed to deploy X-Chain: %v", err)
	}

	fmt.Println("X-Chain deployed successfully!")

	// Display summary
	fmt.Println("\n=== Lux Mainnet Boot Summary ===")
	fmt.Printf("Network: %s (ID: %s)\n", mainnet.Name, mainnet.ID)
	fmt.Printf("Status: %s\n", mainnet.Status)
	fmt.Printf("Validators: %d\n", len(mainnet.Nodes))
	fmt.Printf("Chains deployed: %d\n", len(mainnet.ChainIDs))
	fmt.Printf("- P-Chain: %s\n", pChain.ID)
	fmt.Printf("- C-Chain: %s\n", cChain.ID)
	fmt.Printf("- X-Chain: %s\n", xChain.ID)

	// Wait for interrupt signal
	fmt.Println("\nMainnet is running. Press Ctrl+C to shutdown...")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	// Shutdown
	fmt.Println("\nShutting down mainnet...")
	err = nm.StopNetwork(ctx, mainnet.ID)
	if err != nil {
		log.Printf("Error stopping network: %v", err)
	}

	fmt.Println("Mainnet shutdown complete.")
}