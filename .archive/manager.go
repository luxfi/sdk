// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/sdk/wallet"
)

// ChainManager provides unified access to all Lux chains
type ChainManager struct {
	pChain *PChainClient
	xChain *XChainClient
	cChain *CChainClient
	mChain *MChainClient
	qChain *QChainClient
	wallet *wallet.Wallet
	logger log.Logger
}

// NewChainManager creates a new chain manager
func NewChainManager(endpoint string, wallet *wallet.Wallet, logger log.Logger) (*ChainManager, error) {
	// Initialize P-Chain client
	pChain, err := NewPChainClient(endpoint+"/ext/P", wallet, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create P-Chain client: %w", err)
	}

	// Initialize X-Chain client
	xChain, err := NewXChainClient(endpoint+"/ext/X", wallet, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create X-Chain client: %w", err)
	}

	// Initialize C-Chain client
	cChain, err := NewCChainClient(endpoint+"/ext/bc/C/rpc", wallet, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create C-Chain client: %w", err)
	}

	// Initialize M-Chain client (wallet management)
	mChain, err := NewMChainClient(endpoint+"/ext/M", wallet, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create M-Chain client: %w", err)
	}

	// Initialize Q-Chain client (quantum-resistant)
	qChain, err := NewQChainClient(endpoint+"/ext/Q", wallet, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Q-Chain client: %w", err)
	}

	return &ChainManager{
		pChain: pChain,
		xChain: xChain,
		cChain: cChain,
		mChain: mChain,
		qChain: qChain,
		wallet: wallet,
		logger: logger,
	}, nil
}

// P-Chain Operations

// Stake stakes tokens on the primary network
func (cm *ChainManager) Stake(ctx context.Context, amount *big.Int, duration time.Duration) (ids.ID, error) {
	nodeID := cm.wallet.NodeID()

	params := &AddValidatorParams{
		NodeID:            nodeID,
		StakeAmount:       amount,
		StartTime:         time.Now().Add(30 * time.Second),
		EndTime:           time.Now().Add(duration),
		Duration:          duration,
		RewardAddress:     cm.wallet.P().Address(),
		DelegationFeeRate: 20000, // 2%
	}

	return cm.pChain.AddValidator(ctx, params)
}

// StakeOnSubnet stakes tokens on a specific subnet
func (cm *ChainManager) StakeOnSubnet(ctx context.Context, subnetID ids.ID, weight uint64, duration time.Duration) (ids.ID, error) {
	nodeID := cm.wallet.NodeID()

	params := &AddSubnetValidatorParams{
		NodeID:    nodeID,
		SubnetID:  subnetID,
		Weight:    weight,
		StartTime: time.Now().Add(30 * time.Second),
		EndTime:   time.Now().Add(duration),
	}

	return cm.pChain.AddSubnetValidator(ctx, params)
}

// Delegate delegates stake to a validator
func (cm *ChainManager) Delegate(ctx context.Context, nodeID ids.NodeID, amount *big.Int, duration time.Duration) (ids.ID, error) {
	params := &AddDelegatorParams{
		NodeID:        nodeID,
		StakeAmount:   amount,
		StartTime:     time.Now().Add(30 * time.Second),
		EndTime:       time.Now().Add(duration),
		Duration:      duration,
		RewardAddress: cm.wallet.P().Address(),
	}

	return cm.pChain.AddDelegator(ctx, params)
}

// Vote votes on a governance proposal
func (cm *ChainManager) Vote(ctx context.Context, proposalID ids.ID, vote bool, reason string) (ids.ID, error) {
	params := &VoteParams{
		ProposalID: proposalID,
		Vote:       vote,
		Reason:     reason,
	}

	return cm.pChain.Vote(ctx, params)
}

// X-Chain Asset Operations

// CreateAsset creates a new asset on X-Chain
func (cm *ChainManager) CreateAsset(ctx context.Context, name, symbol string, supply *big.Int) (ids.ID, error) {
	params := &CreateAssetParams{
		Name:         name,
		Symbol:       symbol,
		Denomination: 9, // Default to 9 decimals like LUX
		InitialHolders: []AssetHolder{{
			Address: cm.wallet.X().Address(),
			Amount:  supply,
		}},
		Minters:       []ids.ShortID{cm.wallet.X().Address()},
		MintThreshold: 1,
	}

	return cm.xChain.CreateAsset(ctx, params)
}

// SendAsset sends an asset on X-Chain
func (cm *ChainManager) SendAsset(ctx context.Context, assetID ids.ID, amount *big.Int, to ids.ShortID) (ids.ID, error) {
	params := &SendParams{
		AssetID: assetID,
		Amount:  amount,
		To:      to,
	}

	return cm.xChain.Send(ctx, params)
}

// TradeAssets creates a limit order for asset trading
func (cm *ChainManager) TradeAssets(ctx context.Context, sellAsset ids.ID, sellAmount *big.Int, buyAsset ids.ID, buyAmount *big.Int) (ids.ID, error) {
	// Get UTXOs for the sell asset
	utxos, err := cm.xChain.GetUTXOs(ctx, []string{cm.wallet.X().Address().String()})
	if err != nil {
		return ids.Empty, err
	}

	// Find suitable UTXO
	var inputTxID ids.ID
	var inputIndex uint32
	for _, utxo := range utxos {
		if utxo.AssetID() == sellAsset && utxo.Amount() >= sellAmount.Uint64() {
			inputTxID = utxo.TxID
			inputIndex = utxo.OutputIndex
			break
		}
	}

	params := &CreateOrderParams{
		SellAsset:     sellAsset,
		SellAmount:    sellAmount,
		MinSellAmount: sellAmount, // No partial fills for simplicity
		BuyAsset:      buyAsset,
		BuyAmount:     buyAmount,
		Receiver:      cm.wallet.X().Address(),
		InputTxID:     inputTxID,
		InputIndex:    inputIndex,
	}

	return cm.xChain.CreateOrder(ctx, params)
}

// Cross-Chain Operations

// TransferCrossChain transfers assets between chains
func (cm *ChainManager) TransferCrossChain(ctx context.Context, params *CrossChainTransferParams) (ids.ID, error) {
	// First export from source chain
	var exportTxID ids.ID
	var err error

	switch params.SourceChain {
	case "P":
		exportParams := &ExportParams{
			Amount:        params.Amount,
			To:            params.To,
			TargetChainID: cm.getChainID(params.TargetChain),
		}
		exportTxID, err = cm.pChain.ExportLUX(ctx, exportParams)
	case "X":
		exportParams := &ExportAssetParams{
			AssetID:       params.AssetID,
			Amount:        params.Amount,
			To:            params.To,
			TargetChainID: cm.getChainID(params.TargetChain),
		}
		exportTxID, err = cm.xChain.ExportAsset(ctx, exportParams)
	case "C":
		exportTxID, err = cm.cChain.Export(ctx, params.Amount, params.To, cm.getChainID(params.TargetChain))
	default:
		return ids.Empty, fmt.Errorf("unsupported source chain: %s", params.SourceChain)
	}

	if err != nil {
		return ids.Empty, fmt.Errorf("failed to export: %w", err)
	}

	// Wait for export to be accepted
	time.Sleep(2 * time.Second)

	// Then import to target chain
	var importTxID ids.ID

	switch params.TargetChain {
	case "P":
		importParams := &ImportParams{
			SourceChainID: cm.getChainID(params.SourceChain),
			To:            params.To,
		}
		importTxID, err = cm.pChain.ImportLUX(ctx, importParams)
	case "X":
		importParams := &ImportAssetParams{
			SourceChainID: cm.getChainID(params.SourceChain),
			To:            params.To,
		}
		importTxID, err = cm.xChain.ImportAsset(ctx, importParams)
	case "C":
		importTxID, err = cm.cChain.Import(ctx, cm.getChainID(params.SourceChain), params.To)
	default:
		return ids.Empty, fmt.Errorf("unsupported target chain: %s", params.TargetChain)
	}

	if err != nil {
		return ids.Empty, fmt.Errorf("failed to import: %w", err)
	}

	cm.logger.Info("cross-chain transfer completed",
		"exportTx", exportTxID,
		"importTx", importTxID,
	)

	return importTxID, nil
}

// Wallet Operations via M-Chain

// CreateWallet creates a new wallet
func (cm *ChainManager) CreateWallet(ctx context.Context, name string) (*WalletInfo, error) {
	return cm.mChain.CreateWallet(ctx, name)
}

// ListWallets lists all wallets
func (cm *ChainManager) ListWallets(ctx context.Context) ([]*WalletInfo, error) {
	return cm.mChain.ListWallets(ctx)
}

// CreateMultisigWallet creates a multisig wallet
func (cm *ChainManager) CreateMultisigWallet(ctx context.Context, name string, owners []ids.ShortID, threshold uint32) (*MultisigWalletInfo, error) {
	return cm.mChain.CreateMultisigWallet(ctx, name, owners, threshold)
}

// Query Operations

// GetBalance returns balance across all chains
func (cm *ChainManager) GetBalance(ctx context.Context, address string) (*BalanceInfo, error) {
	balances := &BalanceInfo{
		Address: address,
		Chains:  make(map[string]*ChainBalance),
	}

	// P-Chain balance
	pBalance, err := cm.pChain.GetBalance(ctx, address)
	if err == nil {
		balances.Chains["P"] = &ChainBalance{
			LUX: pBalance,
		}
	}

	// X-Chain balances
	xBalances, err := cm.xChain.GetAllBalances(ctx, address)
	if err == nil {
		balances.Chains["X"] = &ChainBalance{
			Assets: xBalances,
		}
	}

	// C-Chain balance
	cBalance, err := cm.cChain.GetBalance(ctx, address)
	if err == nil {
		balances.Chains["C"] = &ChainBalance{
			LUX: cBalance,
		}
	}

	return balances, nil
}

// GetValidators returns all validators
func (cm *ChainManager) GetValidators(ctx context.Context) (*ValidatorInfo, error) {
	current, err := cm.pChain.GetCurrentValidators(ctx, nil)
	if err != nil {
		return nil, err
	}

	pending, err := cm.pChain.GetPendingValidators(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &ValidatorInfo{
		Current: current,
		Pending: pending,
	}, nil
}

// Helper methods

func (cm *ChainManager) getChainID(chain string) ids.ID {
	switch chain {
	case "P":
		return ids.Empty // Platform chain ID
	case "X":
		return ids.Empty // X-Chain ID
	case "C":
		return ids.Empty // C-Chain ID
	default:
		return ids.Empty
	}
}

// P returns the P-Chain client
func (cm *ChainManager) P() *PChainClient {
	return cm.pChain
}

// X returns the X-Chain client
func (cm *ChainManager) X() *XChainClient {
	return cm.xChain
}

// C returns the C-Chain client
func (cm *ChainManager) C() *CChainClient {
	return cm.cChain
}

// M returns the M-Chain client
func (cm *ChainManager) M() *MChainClient {
	return cm.mChain
}

// Q returns the Q-Chain client
func (cm *ChainManager) Q() *QChainClient {
	return cm.qChain
}

// Parameter types

type CrossChainTransferParams struct {
	SourceChain string
	TargetChain string
	AssetID     ids.ID
	Amount      *big.Int
	To          ids.ShortID
}

type BalanceInfo struct {
	Address string
	Chains  map[string]*ChainBalance
}

type ChainBalance struct {
	LUX    *big.Int
	Assets map[ids.ID]*big.Int
}

type ValidatorInfo struct {
	Current []*platformvm.Validator
	Pending []*platformvm.Validator
}
