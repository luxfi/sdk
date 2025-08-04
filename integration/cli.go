// Copyright (C) 2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/blockchain"
	"github.com/luxfi/cli/pkg/config"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/subnet"
	"github.com/luxfi/sdk/internal/logging"
)

// CLIIntegration provides integration with the Lux CLI
type CLIIntegration struct {
	logger  logging.Logger
	cliPath string
	app     *application.Application
}

// NewCLIIntegration creates a new CLI integration
func NewCLIIntegration(logger logging.Logger) (*CLIIntegration, error) {
	// Find lux CLI binary
	cliPath, err := exec.LookPath("lux")
	if err != nil {
		// Try to find it in the parent directory
		cliPath = filepath.Join("..", "cli", "lux")
		if _, err := os.Stat(cliPath); err != nil {
			return nil, fmt.Errorf("lux CLI binary not found: %w", err)
		}
	}

	// Initialize application
	app := application.New()

	return &CLIIntegration{
		logger:  logger,
		cliPath: cliPath,
		app:     app,
	}, nil
}

// CreateBlockchain creates a new blockchain using CLI functionality
func (c *CLIIntegration) CreateBlockchain(ctx context.Context, name string, vmType string) error {
	c.logger.Info("creating blockchain with CLI", "name", name, "vmType", vmType)

	// Use CLI functionality to create blockchain
	cmd := exec.CommandContext(ctx, c.cliPath, "blockchain", "create", name, "--vm", vmType)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create blockchain: %w", err)
	}

	return nil
}

// DeployBlockchain deploys a blockchain to a network
func (c *CLIIntegration) DeployBlockchain(ctx context.Context, blockchainName string, network string) error {
	c.logger.Info("deploying blockchain with CLI", "blockchain", blockchainName, "network", network)

	// Use CLI functionality to deploy blockchain
	cmd := exec.CommandContext(ctx, c.cliPath, "blockchain", "deploy", blockchainName, "--network", network)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to deploy blockchain: %w", err)
	}

	return nil
}

// CreateKey creates a new key using CLI functionality
func (c *CLIIntegration) CreateKey(ctx context.Context, keyName string) (*key.SoftKey, error) {
	c.logger.Info("creating key with CLI", "name", keyName)

	// Use CLI key management
	softKey, err := key.NewSoftKey()
	if err != nil {
		return nil, fmt.Errorf("failed to create key: %w", err)
	}

	// Save the key
	if err := softKey.Save(keyName); err != nil {
		return nil, fmt.Errorf("failed to save key: %w", err)
	}

	return softKey, nil
}

// LaunchNetwork launches a network using CLI
func (c *CLIIntegration) LaunchNetwork(ctx context.Context, networkType string) error {
	c.logger.Info("launching network with CLI", "type", networkType)

	var cmd *exec.Cmd
	switch networkType {
	case "local":
		cmd = exec.CommandContext(ctx, c.cliPath, "network", "start", "--local")
	case "testnet":
		cmd = exec.CommandContext(ctx, c.cliPath, "network", "start", "--testnet")
	case "mainnet":
		cmd = exec.CommandContext(ctx, c.cliPath, "network", "start", "--mainnet")
	default:
		return fmt.Errorf("unsupported network type: %s", networkType)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to launch network: %w", err)
	}

	return nil
}

// GetSubnetInfo gets information about a subnet
func (c *CLIIntegration) GetSubnetInfo(ctx context.Context, subnetName string) (*subnet.Subnet, error) {
	c.logger.Info("getting subnet info", "name", subnetName)

	// Use CLI SDK functionality
	sc := subnet.New(&subnet.SubnetParams{
		Name: subnetName,
	})

	return sc, nil
}

// ValidatorOperations provides validator management operations
func (c *CLIIntegration) AddValidator(ctx context.Context, nodeID string, subnetID string, weight uint64) error {
	c.logger.Info("adding validator", "nodeID", nodeID, "subnet", subnetID, "weight", weight)

	cmd := exec.CommandContext(ctx, c.cliPath, "blockchain", "addValidator", 
		"--nodeID", nodeID,
		"--subnet", subnetID,
		"--weight", fmt.Sprintf("%d", weight),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add validator: %w", err)
	}

	return nil
}