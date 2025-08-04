// Copyright (C) 2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/vms/platformvm"
	"github.com/luxfi/sdk/wallet"
	"github.com/luxfi/log"
)

// PChainClient handles all P-Chain operations
type PChainClient struct {
	client   platformvm.Client
	wallet   *wallet.Wallet
	logger   log.Logger
	endpoint string
}

// NewPChainClient creates a new P-Chain client
func NewPChainClient(endpoint string, wallet *wallet.Wallet, logger log.Logger) (*PChainClient, error) {
	client := platformvm.NewClient(endpoint)
	
	return &PChainClient{
		client:   client,
		wallet:   wallet,
		logger:   logger,
		endpoint: endpoint,
	}, nil
}

// Staking Operations

// AddValidator adds a validator to the primary network
func (p *PChainClient) AddValidator(ctx context.Context, params *AddValidatorParams) (ids.ID, error) {
	p.logger.Info("adding validator", 
		"nodeID", params.NodeID,
		"stake", params.StakeAmount,
		"duration", params.Duration,
	)

	tx, err := p.wallet.P().IssueAddValidatorTx(
		&platformvm.Validator{
			NodeID: params.NodeID,
			Start:  uint64(params.StartTime.Unix()),
			End:    uint64(params.EndTime.Unix()),
			Wght:   params.StakeAmount.Uint64(),
		},
		params.RewardAddress,
		params.DelegationFeeRate,
	)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to issue add validator tx: %w", err)
	}

	return tx.ID(), nil
}

// AddSubnetValidator adds a validator to a subnet
func (p *PChainClient) AddSubnetValidator(ctx context.Context, params *AddSubnetValidatorParams) (ids.ID, error) {
	p.logger.Info("adding subnet validator",
		"nodeID", params.NodeID,
		"subnetID", params.SubnetID,
		"weight", params.Weight,
	)

	tx, err := p.wallet.P().IssueAddSubnetValidatorTx(
		&platformvm.SubnetValidator{
			Validator: platformvm.Validator{
				NodeID: params.NodeID,
				Start:  uint64(params.StartTime.Unix()),
				End:    uint64(params.EndTime.Unix()),
				Wght:   params.Weight,
			},
			Subnet: params.SubnetID,
		},
	)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to issue add subnet validator tx: %w", err)
	}

	return tx.ID(), nil
}

// AddDelegator adds a delegator to a validator
func (p *PChainClient) AddDelegator(ctx context.Context, params *AddDelegatorParams) (ids.ID, error) {
	p.logger.Info("adding delegator",
		"nodeID", params.NodeID,
		"stake", params.StakeAmount,
		"duration", params.Duration,
	)

	tx, err := p.wallet.P().IssueAddDelegatorTx(
		&platformvm.Validator{
			NodeID: params.NodeID,
			Start:  uint64(params.StartTime.Unix()),
			End:    uint64(params.EndTime.Unix()),
			Wght:   params.StakeAmount.Uint64(),
		},
		params.RewardAddress,
	)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to issue add delegator tx: %w", err)
	}

	return tx.ID(), nil
}

// Delegation Operations

// GetPendingValidators returns pending validators
func (p *PChainClient) GetPendingValidators(ctx context.Context, subnetID *ids.ID) ([]*platformvm.Validator, error) {
	var subnet ids.ID
	if subnetID != nil {
		subnet = *subnetID
	}

	validators, err := p.client.GetPendingValidators(ctx, subnet, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending validators: %w", err)
	}

	return validators, nil
}

// GetCurrentValidators returns current validators
func (p *PChainClient) GetCurrentValidators(ctx context.Context, subnetID *ids.ID) ([]*platformvm.Validator, error) {
	var subnet ids.ID
	if subnetID != nil {
		subnet = *subnetID
	}

	validators, err := p.client.GetCurrentValidators(ctx, subnet, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get current validators: %w", err)
	}

	return validators, nil
}

// GetStake returns the staked amount for an address
func (p *PChainClient) GetStake(ctx context.Context, addresses []string) (*GetStakeResponse, error) {
	stake, err := p.client.GetStake(ctx, addresses)
	if err != nil {
		return nil, fmt.Errorf("failed to get stake: %w", err)
	}

	return &GetStakeResponse{
		Staked:     big.NewInt(int64(stake.Staked)),
		Stakeds:    stake.Stakeds,
		Outputs:    stake.Outputs,
		Encoding:   stake.Encoding,
	}, nil
}

// Subnet Operations

// CreateSubnet creates a new subnet
func (p *PChainClient) CreateSubnet(ctx context.Context, params *CreateSubnetParams) (ids.ID, error) {
	p.logger.Info("creating subnet", "controlKeys", len(params.ControlKeys))

	tx, err := p.wallet.P().IssueCreateSubnetTx(
		params.ControlKeys,
		params.Threshold,
	)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to issue create subnet tx: %w", err)
	}

	return tx.ID(), nil
}

// CreateChain creates a new chain in a subnet
func (p *PChainClient) CreateChain(ctx context.Context, params *CreateChainParams) (ids.ID, error) {
	p.logger.Info("creating chain",
		"subnet", params.SubnetID,
		"chainName", params.ChainName,
		"vmID", params.VMID,
	)

	tx, err := p.wallet.P().IssueCreateChainTx(
		params.SubnetID,
		params.GenesisData,
		params.VMID,
		params.FxIDs,
		params.ChainName,
	)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to issue create chain tx: %w", err)
	}

	return tx.ID(), nil
}

// Voting Operations

// Vote submits a vote on a proposal
func (p *PChainClient) Vote(ctx context.Context, params *VoteParams) (ids.ID, error) {
	p.logger.Info("submitting vote",
		"proposalID", params.ProposalID,
		"vote", params.Vote,
	)

	// TODO: Implement voting logic based on governance system
	// This would involve creating and signing a vote transaction

	return ids.Empty, fmt.Errorf("voting not yet implemented")
}

// Cross-Chain Operations

// ExportLUX exports LUX from P-Chain to another chain
func (p *PChainClient) ExportLUX(ctx context.Context, params *ExportParams) (ids.ID, error) {
	p.logger.Info("exporting LUX",
		"amount", params.Amount,
		"to", params.To,
		"targetChain", params.TargetChainID,
	)

	tx, err := p.wallet.P().IssueExportTx(
		params.TargetChainID,
		[]*platformvm.TransferableOutput{{
			Asset: platformvm.Asset{ID: ids.Empty}, // LUX asset ID
			Out: &secp256k1fx.TransferOutput{
				Amt: params.Amount.Uint64(),
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs:     []ids.ShortID{params.To},
				},
			},
		}},
	)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to issue export tx: %w", err)
	}

	return tx.ID(), nil
}

// ImportLUX imports LUX from another chain to P-Chain
func (p *PChainClient) ImportLUX(ctx context.Context, params *ImportParams) (ids.ID, error) {
	p.logger.Info("importing LUX",
		"from", params.SourceChainID,
		"to", params.To,
	)

	tx, err := p.wallet.P().IssueImportTx(
		params.SourceChainID,
		params.To,
	)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to issue import tx: %w", err)
	}

	return tx.ID(), nil
}

// Query Operations

// GetBalance returns the balance of an address
func (p *PChainClient) GetBalance(ctx context.Context, address string) (*big.Int, error) {
	balance, err := p.client.GetBalance(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return big.NewInt(int64(balance.Balance)), nil
}

// GetUTXOs returns UTXOs for addresses
func (p *PChainClient) GetUTXOs(ctx context.Context, addresses []string) ([]*platformvm.UTXO, error) {
	utxos, err := p.client.GetUTXOs(ctx, addresses, "", 0, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get UTXOs: %w", err)
	}

	return utxos.UTXOs, nil
}

// GetHeight returns the current block height
func (p *PChainClient) GetHeight(ctx context.Context) (uint64, error) {
	height, err := p.client.GetHeight(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get height: %w", err)
	}

	return height, nil
}

// GetMinStake returns minimum stake requirements
func (p *PChainClient) GetMinStake(ctx context.Context) (*MinStakeInfo, error) {
	minValidatorStake, err := p.client.GetMinStake(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get min stake: %w", err)
	}

	return &MinStakeInfo{
		MinValidatorStake: big.NewInt(int64(minValidatorStake.MinValidatorStake)),
		MinDelegatorStake: big.NewInt(int64(minValidatorStake.MinDelegatorStake)),
	}, nil
}

// GetRewardUTXOs returns reward UTXOs for a transaction
func (p *PChainClient) GetRewardUTXOs(ctx context.Context, txID ids.ID) ([]*platformvm.UTXO, error) {
	utxos, err := p.client.GetRewardUTXOs(ctx, txID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reward UTXOs: %w", err)
	}

	return utxos, nil
}

// Parameter types

type AddValidatorParams struct {
	NodeID            ids.NodeID
	StakeAmount       *big.Int
	StartTime         time.Time
	EndTime           time.Time
	Duration          time.Duration
	RewardAddress     ids.ShortID
	DelegationFeeRate uint32
}

type AddSubnetValidatorParams struct {
	NodeID    ids.NodeID
	SubnetID  ids.ID
	Weight    uint64
	StartTime time.Time
	EndTime   time.Time
}

type AddDelegatorParams struct {
	NodeID        ids.NodeID
	StakeAmount   *big.Int
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	RewardAddress ids.ShortID
}

type CreateSubnetParams struct {
	ControlKeys []ids.ShortID
	Threshold   uint32
}

type CreateChainParams struct {
	SubnetID    ids.ID
	GenesisData []byte
	VMID        ids.ID
	FxIDs       []ids.ID
	ChainName   string
}

type VoteParams struct {
	ProposalID ids.ID
	Vote       bool
	Reason     string
}

type ExportParams struct {
	Amount        *big.Int
	To            ids.ShortID
	TargetChainID ids.ID
}

type ImportParams struct {
	SourceChainID ids.ID
	To            ids.ShortID
}

type GetStakeResponse struct {
	Staked   *big.Int
	Stakeds  map[ids.ID]uint64
	Outputs  []*platformvm.UTXO
	Encoding string
}

type MinStakeInfo struct {
	MinValidatorStake *big.Int
	MinDelegatorStake *big.Int
}