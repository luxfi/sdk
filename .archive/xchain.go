// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"context"
	"fmt"
	"math/big"

	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/sdk/wallet"
	"github.com/luxfi/vms/avm"
)

// XChainClient handles all X-Chain operations for asset management
type XChainClient struct {
	client   avm.Client
	wallet   *wallet.Wallet
	logger   log.Logger
	endpoint string
}

// NewXChainClient creates a new X-Chain client
func NewXChainClient(endpoint string, wallet *wallet.Wallet, logger log.Logger) (*XChainClient, error) {
	client := avm.NewClient(endpoint)

	return &XChainClient{
		client:   client,
		wallet:   wallet,
		logger:   logger,
		endpoint: endpoint,
	}, nil
}

// Asset Creation and Management

// CreateAsset creates a new asset on the X-Chain
func (x *XChainClient) CreateAsset(ctx context.Context, params *CreateAssetParams) (ids.ID, error) {
	x.logger.Info("creating asset",
		"name", params.Name,
		"symbol", params.Symbol,
		"denomination", params.Denomination,
	)

	// Create the initial state for the asset
	initialState := map[uint32][]avm.Verify{
		0: { // Feature extension for minting
			&avm.TransferableOutput{
				Asset: avm.Asset{ID: ids.Empty}, // Will be set to asset ID
				Out: &secp256k1fx.MintOutput{
					OutputOwners: secp256k1fx.OutputOwners{
						Threshold: params.MintThreshold,
						Addrs:     params.Minters,
					},
				},
			},
		},
	}

	// Add initial holders if specified
	if len(params.InitialHolders) > 0 {
		holders := make([]avm.Verify, 0, len(params.InitialHolders))
		for _, holder := range params.InitialHolders {
			holders = append(holders, &avm.TransferableOutput{
				Asset: avm.Asset{ID: ids.Empty}, // Will be set to asset ID
				Out: &secp256k1fx.TransferOutput{
					Amt: holder.Amount.Uint64(),
					OutputOwners: secp256k1fx.OutputOwners{
						Threshold: 1,
						Addrs:     []ids.ShortID{holder.Address},
					},
				},
			})
		}
		initialState[1] = holders // Feature extension for transferring
	}

	tx, err := x.wallet.X().IssueCreateAssetTx(
		params.Name,
		params.Symbol,
		uint8(params.Denomination),
		initialState,
	)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to issue create asset tx: %w", err)
	}

	return tx.ID(), nil
}

// CreateFixedCapAsset creates a fixed cap asset
func (x *XChainClient) CreateFixedCapAsset(ctx context.Context, params *CreateFixedCapAssetParams) (ids.ID, error) {
	x.logger.Info("creating fixed cap asset",
		"name", params.Name,
		"symbol", params.Symbol,
		"totalSupply", params.TotalSupply,
	)

	// Create initial state with all supply going to specified holders
	holders := make([]avm.Verify, 0, len(params.InitialHolders))
	for _, holder := range params.InitialHolders {
		holders = append(holders, &avm.TransferableOutput{
			Asset: avm.Asset{ID: ids.Empty}, // Will be set to asset ID
			Out: &secp256k1fx.TransferOutput{
				Amt: holder.Amount.Uint64(),
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs:     []ids.ShortID{holder.Address},
				},
			},
		})
	}

	initialState := map[uint32][]avm.Verify{
		0: holders, // No minting capability for fixed cap assets
	}

	tx, err := x.wallet.X().IssueCreateAssetTx(
		params.Name,
		params.Symbol,
		uint8(params.Denomination),
		initialState,
	)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to issue create fixed cap asset tx: %w", err)
	}

	return tx.ID(), nil
}

// CreateNFT creates a new NFT collection
func (x *XChainClient) CreateNFT(ctx context.Context, params *CreateNFTParams) (ids.ID, error) {
	x.logger.Info("creating NFT collection",
		"name", params.Name,
		"symbol", params.Symbol,
	)

	// NFT initial state - groups represent different NFT types/traits
	initialState := make(map[uint32][]avm.Verify)

	for groupID, group := range params.Groups {
		initialState[groupID] = []avm.Verify{
			&avm.TransferableOutput{
				Asset: avm.Asset{ID: ids.Empty}, // Will be set to asset ID
				Out: &nftfx.MintOutput{
					GroupID: group.ID,
					OutputOwners: secp256k1fx.OutputOwners{
						Threshold: group.MintThreshold,
						Addrs:     group.Minters,
					},
				},
			},
		}
	}

	tx, err := x.wallet.X().IssueCreateAssetTx(
		params.Name,
		params.Symbol,
		0, // NFTs have no denomination
		initialState,
	)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to issue create NFT tx: %w", err)
	}

	return tx.ID(), nil
}

// Asset Trading Operations

// Send sends an asset to another address
func (x *XChainClient) Send(ctx context.Context, params *SendParams) (ids.ID, error) {
	x.logger.Info("sending asset",
		"assetID", params.AssetID,
		"amount", params.Amount,
		"to", params.To,
	)

	tx, err := x.wallet.X().IssueSendTx(
		params.AssetID,
		params.Amount.Uint64(),
		params.To,
		params.Memo,
	)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to issue send tx: %w", err)
	}

	return tx.ID(), nil
}

// CreateOrder creates a limit order for asset trading
func (x *XChainClient) CreateOrder(ctx context.Context, params *CreateOrderParams) (ids.ID, error) {
	x.logger.Info("creating order",
		"sellAsset", params.SellAsset,
		"sellAmount", params.SellAmount,
		"buyAsset", params.BuyAsset,
		"buyAmount", params.BuyAmount,
	)

	// Create a transaction with both inputs and outputs for the trade
	inputs := []*avm.TransferableInput{{
		UTXOID: avm.UTXOID{
			TxID:        params.InputTxID,
			OutputIndex: params.InputIndex,
		},
		Asset: avm.Asset{ID: params.SellAsset},
		In: &secp256k1fx.TransferInput{
			Amt: params.SellAmount.Uint64(),
			Input: secp256k1fx.Input{
				SigIndices: []uint32{0},
			},
		},
	}}

	outputs := []*avm.TransferableOutput{
		{
			Asset: avm.Asset{ID: params.BuyAsset},
			Out: &secp256k1fx.TransferOutput{
				Amt: params.BuyAmount.Uint64(),
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs:     []ids.ShortID{params.Receiver},
				},
			},
		},
	}

	// If partial fills are allowed, add change output
	if params.SellAmount.Cmp(params.MinSellAmount) > 0 {
		outputs = append(outputs, &avm.TransferableOutput{
			Asset: avm.Asset{ID: params.SellAsset},
			Out: &secp256k1fx.TransferOutput{
				Amt: params.SellAmount.Uint64() - params.MinSellAmount.Uint64(),
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs:     []ids.ShortID{params.Receiver},
				},
			},
		})
	}

	tx := &avm.Tx{
		UnsignedTx: &avm.BaseTx{
			BaseTx: avm.BaseTx{
				NetworkID:    x.wallet.NetworkID(),
				BlockchainID: x.wallet.X().BlockchainID(),
				Ins:          inputs,
				Outs:         outputs,
			},
		},
	}

	signedTx, err := x.wallet.X().SignTx(tx)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to sign order tx: %w", err)
	}

	txID, err := x.client.IssueTx(ctx, signedTx.Bytes())
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to issue order tx: %w", err)
	}

	return txID, nil
}

// Asset Operations

// MintAsset mints new units of a variable cap asset
func (x *XChainClient) MintAsset(ctx context.Context, params *MintAssetParams) (ids.ID, error) {
	x.logger.Info("minting asset",
		"assetID", params.AssetID,
		"amount", params.Amount,
		"to", params.To,
	)

	tx, err := x.wallet.X().IssueMintTx(
		params.AssetID,
		params.Amount.Uint64(),
		params.To,
		params.MintInput,
	)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to issue mint tx: %w", err)
	}

	return tx.ID(), nil
}

// MintNFT mints a new NFT
func (x *XChainClient) MintNFT(ctx context.Context, params *MintNFTParams) (ids.ID, error) {
	x.logger.Info("minting NFT",
		"assetID", params.AssetID,
		"groupID", params.GroupID,
		"to", params.To,
	)

	tx, err := x.wallet.X().IssueMintNFTTx(
		params.AssetID,
		params.GroupID,
		params.Payload,
		params.To,
		params.MintInput,
	)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to issue mint NFT tx: %w", err)
	}

	return tx.ID(), nil
}

// Cross-Chain Operations

// ExportAsset exports an asset from X-Chain to another chain
func (x *XChainClient) ExportAsset(ctx context.Context, params *ExportAssetParams) (ids.ID, error) {
	x.logger.Info("exporting asset",
		"assetID", params.AssetID,
		"amount", params.Amount,
		"targetChain", params.TargetChainID,
	)

	tx, err := x.wallet.X().IssueExportTx(
		params.TargetChainID,
		[]*avm.TransferableOutput{{
			Asset: avm.Asset{ID: params.AssetID},
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

// ImportAsset imports an asset from another chain to X-Chain
func (x *XChainClient) ImportAsset(ctx context.Context, params *ImportAssetParams) (ids.ID, error) {
	x.logger.Info("importing asset",
		"sourceChain", params.SourceChainID,
		"to", params.To,
	)

	tx, err := x.wallet.X().IssueImportTx(
		params.SourceChainID,
		params.To,
	)
	if err != nil {
		return ids.Empty, fmt.Errorf("failed to issue import tx: %w", err)
	}

	return tx.ID(), nil
}

// Query Operations

// GetBalance returns the balance of an address for a specific asset
func (x *XChainClient) GetBalance(ctx context.Context, address string, assetID ids.ID) (*big.Int, error) {
	balance, err := x.client.GetBalance(ctx, address, assetID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return big.NewInt(int64(balance.Balance)), nil
}

// GetAllBalances returns all asset balances for an address
func (x *XChainClient) GetAllBalances(ctx context.Context, address string) (map[ids.ID]*big.Int, error) {
	balances, err := x.client.GetAllBalances(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to get all balances: %w", err)
	}

	result := make(map[ids.ID]*big.Int)
	for _, balance := range balances.Balances {
		assetID, err := ids.FromString(balance.AssetID)
		if err != nil {
			continue
		}
		result[assetID] = big.NewInt(int64(balance.Balance))
	}

	return result, nil
}

// GetAssetDescription returns information about an asset
func (x *XChainClient) GetAssetDescription(ctx context.Context, assetID ids.ID) (*AssetDescription, error) {
	info, err := x.client.GetAssetDescription(ctx, assetID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get asset description: %w", err)
	}

	return &AssetDescription{
		AssetID:      assetID,
		Name:         info.Name,
		Symbol:       info.Symbol,
		Denomination: info.Denomination,
	}, nil
}

// GetUTXOs returns UTXOs for addresses
func (x *XChainClient) GetUTXOs(ctx context.Context, addresses []string) ([]*avm.UTXO, error) {
	utxos, err := x.client.GetUTXOs(ctx, addresses, "", 0, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get UTXOs: %w", err)
	}

	return utxos.UTXOs, nil
}

// Parameter types

type CreateAssetParams struct {
	Name           string
	Symbol         string
	Denomination   int
	InitialHolders []AssetHolder
	Minters        []ids.ShortID
	MintThreshold  uint32
}

type CreateFixedCapAssetParams struct {
	Name           string
	Symbol         string
	Denomination   int
	TotalSupply    *big.Int
	InitialHolders []AssetHolder
}

type CreateNFTParams struct {
	Name   string
	Symbol string
	Groups []NFTGroup
}

type AssetHolder struct {
	Address ids.ShortID
	Amount  *big.Int
}

type NFTGroup struct {
	ID            uint32
	Minters       []ids.ShortID
	MintThreshold uint32
}

type SendParams struct {
	AssetID ids.ID
	Amount  *big.Int
	To      ids.ShortID
	Memo    []byte
}

type CreateOrderParams struct {
	SellAsset     ids.ID
	SellAmount    *big.Int
	MinSellAmount *big.Int
	BuyAsset      ids.ID
	BuyAmount     *big.Int
	Receiver      ids.ShortID
	InputTxID     ids.ID
	InputIndex    uint32
}

type MintAssetParams struct {
	AssetID   ids.ID
	Amount    *big.Int
	To        ids.ShortID
	MintInput *avm.TransferableInput
}

type MintNFTParams struct {
	AssetID   ids.ID
	GroupID   uint32
	Payload   []byte
	To        ids.ShortID
	MintInput *avm.TransferableInput
}

type ExportAssetParams struct {
	AssetID       ids.ID
	Amount        *big.Int
	To            ids.ShortID
	TargetChainID ids.ID
}

type ImportAssetParams struct {
	SourceChainID ids.ID
	To            ids.ShortID
}

type AssetDescription struct {
	AssetID      ids.ID
	Name         string
	Symbol       string
	Denomination uint8
}
