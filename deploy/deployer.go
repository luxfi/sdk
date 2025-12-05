// Copyright (C) 2020-2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

// Package deploy provides a clean, composable interface for deploying
// subnets and blockchains to any Lux network endpoint.
package deploy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/node/vms/platformvm/txs"
	"github.com/luxfi/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/wallet/primary"
)

var (
	// ErrNoEndpoint is returned when no endpoint is configured
	ErrNoEndpoint = errors.New("no endpoint configured")
	// ErrNoKeychain is returned when no keychain is configured
	ErrNoKeychain = errors.New("no keychain configured")
	// ErrInvalidThreshold is returned when threshold exceeds control key count
	ErrInvalidThreshold = errors.New("threshold cannot exceed number of control keys")
)

// Deployer provides a clean interface for deploying subnets and blockchains.
// It wraps the node wallet functionality with a simpler, more composable API.
type Deployer struct {
	endpoint string
	kc       *primary.KeychainAdapter
	timeout  time.Duration
}

// Option configures a Deployer
type Option func(*Deployer)

// WithEndpoint sets the RPC endpoint URI
func WithEndpoint(uri string) Option {
	return func(d *Deployer) {
		d.endpoint = uri
	}
}

// WithKeychain sets the keychain adapter for signing transactions.
// Use KeychainFromPrivateKey or EWOQKeychain to create one.
func WithKeychain(kc *primary.KeychainAdapter) Option {
	return func(d *Deployer) {
		d.kc = kc
	}
}

// WithTimeout sets the operation timeout
func WithTimeout(timeout time.Duration) Option {
	return func(d *Deployer) {
		d.timeout = timeout
	}
}

// New creates a new Deployer with the given options
func New(opts ...Option) (*Deployer, error) {
	d := &Deployer{
		timeout: 2 * time.Minute, // default timeout
	}
	for _, opt := range opts {
		opt(d)
	}

	if d.endpoint == "" {
		return nil, ErrNoEndpoint
	}
	if d.kc == nil {
		return nil, ErrNoKeychain
	}

	return d, nil
}

// Endpoint returns the configured endpoint
func (d *Deployer) Endpoint() string {
	return d.endpoint
}

// wallet creates a new wallet connected to the endpoint
func (d *Deployer) wallet(ctx context.Context, preloadTxs ...ids.ID) (primary.Wallet, error) {
	config := &primary.WalletConfig{
		URI:         d.endpoint,
		LUXKeychain: d.kc,
		EthKeychain: d.kc,
	}

	// Add preload transactions if specified
	if len(preloadTxs) > 0 {
		config.PChainTxsToFetch = set.Of(preloadTxs...)
	}

	return primary.MakeWallet(ctx, config)
}

// CreateSubnet creates a new subnet with the given control keys and threshold.
// Returns the subnet ID on success.
func (d *Deployer) CreateSubnet(ctx context.Context, controlKeys []ids.ShortID, threshold uint32) (ids.ID, error) {
	if int(threshold) > len(controlKeys) {
		return ids.Empty, ErrInvalidThreshold
	}

	wallet, err := d.wallet(ctx)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to create wallet: %w", err)
	}

	owners := &secp256k1fx.OutputOwners{
		Threshold: threshold,
		Addrs:     controlKeys,
		Locktime:  0,
	}

	tx, err := wallet.P().IssueCreateNetTx(owners)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to create subnet: %w", err)
	}

	return tx.ID(), nil
}

// CreateBlockchain creates a new blockchain on the given subnet.
// Returns the blockchain ID on success.
func (d *Deployer) CreateBlockchain(
	ctx context.Context,
	subnetID ids.ID,
	genesis []byte,
	vmID ids.ID,
	name string,
) (ids.ID, error) {
	wallet, err := d.wallet(ctx, subnetID)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to create wallet: %w", err)
	}

	tx, err := wallet.P().IssueCreateChainTx(subnetID, genesis, vmID, nil, name)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to create blockchain: %w", err)
	}

	return tx.ID(), nil
}

// DeploySubnetResult contains the result of a full subnet deployment
type DeploySubnetResult struct {
	SubnetID     ids.ID
	BlockchainID ids.ID
}

// DeploySubnet creates a subnet and blockchain in one operation.
// This is the most common deployment pattern.
func (d *Deployer) DeploySubnet(
	ctx context.Context,
	genesis []byte,
	vmID ids.ID,
	chainName string,
	controlKeys []ids.ShortID,
	threshold uint32,
) (*DeploySubnetResult, error) {
	// Create subnet first
	subnetID, err := d.CreateSubnet(ctx, controlKeys, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to create subnet: %w", err)
	}

	// Wait briefly for subnet to be accepted
	time.Sleep(2 * time.Second)

	// Create blockchain on the subnet
	blockchainID, err := d.CreateBlockchain(ctx, subnetID, genesis, vmID, chainName)
	if err != nil {
		return nil, fmt.Errorf("failed to create blockchain: %w", err)
	}

	return &DeploySubnetResult{
		SubnetID:     subnetID,
		BlockchainID: blockchainID,
	}, nil
}

// AddValidator adds a validator to a subnet.
func (d *Deployer) AddValidator(
	ctx context.Context,
	subnetID ids.ID,
	nodeID ids.NodeID,
	weight uint64,
	startTime time.Time,
	endTime time.Time,
) (ids.ID, error) {
	wallet, err := d.wallet(ctx, subnetID)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to create wallet: %w", err)
	}

	validator := &txs.NetValidator{
		Validator: txs.Validator{
			NodeID: nodeID,
			Start:  uint64(startTime.Unix()),
			End:    uint64(endTime.Unix()),
			Wght:   weight,
		},
		Net: subnetID,
	}

	tx, err := wallet.P().IssueAddNetValidatorTx(validator)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to add validator: %w", err)
	}

	return tx.ID(), nil
}

// RemoveValidator removes a validator from a subnet.
func (d *Deployer) RemoveValidator(
	ctx context.Context,
	subnetID ids.ID,
	nodeID ids.NodeID,
) (ids.ID, error) {
	wallet, err := d.wallet(ctx, subnetID)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to create wallet: %w", err)
	}

	tx, err := wallet.P().IssueRemoveNetValidatorTx(nodeID, subnetID)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to remove validator: %w", err)
	}

	return tx.ID(), nil
}
