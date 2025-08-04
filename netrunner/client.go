// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package netrunner provides integration with the Lux netrunner tool
// for managing test networks and blockchain deployments.
package netrunner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luxfi/netrunner-sdk/rpcpb"
	netrunner "github.com/luxfi/netrunner-sdk"
	"github.com/luxfi/log"
)

// Client wraps the netrunner-sdk client with additional functionality
type Client struct {
	client netrunner.Client
	logger log.Logger
	config *Config
}

// Config holds configuration for the netrunner client
type Config struct {
	Endpoint    string
	DialTimeout time.Duration
	LogLevel    string
}

// DefaultConfig returns default netrunner configuration
func DefaultConfig() *Config {
	return &Config{
		Endpoint:    "localhost:8080",
		DialTimeout: 30 * time.Second,
		LogLevel:    "info",
	}
}

// NewClient creates a new netrunner client
func NewClient(config *Config, logger log.Logger) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}
	
	if logger == nil {
		logger = log.NewNoOpLogger()
	}

	// Create netrunner-sdk config
	sdkConfig := netrunner.Config{
		LogLevel:    config.LogLevel,
		Endpoint:    config.Endpoint,
		DialTimeout: config.DialTimeout,
	}

	// Create netrunner client
	client, err := netrunner.New(sdkConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create netrunner client: %w", err)
	}

	return &Client{
		client: client,
		logger: logger,
		config: config,
	}, nil
}

// Start starts a new network with the given configuration
func (c *Client) Start(ctx context.Context, execPath string, opts ...netrunner.OpOption) (*rpcpb.StartResponse, error) {
	c.logger.Info("starting network", "execPath", execPath)
	resp, err := c.client.Start(ctx, execPath, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start network: %w", err)
	}
	c.logger.Info("network started", "clusterInfo", resp.ClusterInfo)
	return resp, nil
}

// Stop stops the running network
func (c *Client) Stop(ctx context.Context) error {
	c.logger.Info("stopping network")
	_, err := c.client.Stop(ctx)
	if err != nil {
		return fmt.Errorf("failed to stop network: %w", err)
	}
	c.logger.Info("network stopped")
	return nil
}

// Health checks the health of the network
func (c *Client) Health(ctx context.Context) (*rpcpb.HealthResponse, error) {
	return c.client.Health(ctx)
}

// Status returns the current network status
func (c *Client) Status(ctx context.Context) (*rpcpb.StatusResponse, error) {
	return c.client.Status(ctx)
}

// URIs returns the URIs of all nodes in the network
func (c *Client) URIs(ctx context.Context) ([]string, error) {
	return c.client.URIs(ctx)
}

// CreateBlockchains creates new blockchains with the given specifications
func (c *Client) CreateBlockchains(ctx context.Context, specs []*rpcpb.BlockchainSpec) (*rpcpb.CreateBlockchainsResponse, error) {
	c.logger.Info("creating blockchains", "count", len(specs))
	resp, err := c.client.CreateBlockchains(ctx, specs)
	if err != nil {
		return nil, fmt.Errorf("failed to create blockchains: %w", err)
	}
	c.logger.Info("blockchains created")
	return resp, nil
}

// CreateSubnets creates new subnets
func (c *Client) CreateSubnets(ctx context.Context, opts ...netrunner.OpOption) (*rpcpb.CreateSubnetsResponse, error) {
	c.logger.Info("creating subnets")
	resp, err := c.client.CreateSubnets(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create subnets: %w", err)
	}
	c.logger.Info("subnets created")
	return resp, nil
}

// AddNode adds a new node to the network
func (c *Client) AddNode(ctx context.Context, name string, execPath string, opts ...netrunner.OpOption) (*rpcpb.AddNodeResponse, error) {
	c.logger.Info("adding node", "name", name)
	resp, err := c.client.AddNode(ctx, name, execPath, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to add node: %w", err)
	}
	c.logger.Info("node added", "clusterInfo", resp.ClusterInfo)
	return resp, nil
}

// RemoveNode removes a node from the network
func (c *Client) RemoveNode(ctx context.Context, name string) error {
	c.logger.Info("removing node", "name", name)
	_, err := c.client.RemoveNode(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to remove node: %w", err)
	}
	c.logger.Info("node removed", "name", name)
	return nil
}

// RestartNode restarts a node in the network
func (c *Client) RestartNode(ctx context.Context, name string, opts ...netrunner.OpOption) error {
	c.logger.Info("restarting node", "name", name)
	_, err := c.client.RestartNode(ctx, name, opts...)
	if err != nil {
		return fmt.Errorf("failed to restart node: %w", err)
	}
	c.logger.Info("node restarted", "name", name)
	return nil
}

// WaitForHealthy waits for all nodes in the network to be healthy
func (c *Client) WaitForHealthy(ctx context.Context, timeout time.Duration) error {
	c.logger.Info("waiting for network to be healthy", "timeout", timeout)
	
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.New("timeout waiting for network to be healthy")
		case <-ticker.C:
			health, err := c.Health(ctx)
			if err != nil {
				c.logger.Debug("health check failed", "error", err)
				continue
			}
			// Check if health response indicates healthy
			if health.GetClusterInfo() != nil {
				c.logger.Info("network is healthy")
				return nil
			}
		}
	}
}

// Close closes the netrunner client connection
func (c *Client) Close() error {
	return c.client.Close()
}