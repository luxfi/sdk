// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package exchangevm

import (
	"cmp"
	"fmt"

	"github.com/luxfi/sdk/node/codec"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/sdk/utils/formatting/address"
	"github.com/luxfi/sdk/node/vms/exchangevm/fxs"
	"github.com/luxfi/sdk/node/vms/exchangevm/txs"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

// Genesis represents the genesis state of the XVM
type Genesis struct {
	Txs []*GenesisAsset `serialize:"true"`
}

// GenesisAsset represents an asset in the genesis block
type GenesisAsset struct {
	Alias             string            `serialize:"true"`
	txs.CreateAssetTx `serialize:"true"`
}

// Compare implements utils.Sortable for GenesisAsset
func (g *GenesisAsset) Compare(other *GenesisAsset) int {
	return cmp.Compare(g.Alias, other.Alias)
}

// AssetInitialState describes the initial state of an asset
type AssetInitialState struct {
	FixedCap    []GenesisHolder
	VariableCap []GenesisOwners
}

// GenesisAssetDefinition describes a genesis asset and its initial state
type GenesisAssetDefinition struct {
	Name         string
	Symbol       string
	Denomination byte
	InitialState AssetInitialState
	Memo         []byte
}

// GenesisHolder describes how much asset is owned by an address
type GenesisHolder struct {
	Amount  uint64
	Address string
}

// GenesisOwners describes who can perform an action
type GenesisOwners struct {
	Threshold uint32
	Minters   []string
}

// NewGenesis creates a new Genesis from genesis data
func NewGenesis(
	networkID uint32,
	genesisData map[string]GenesisAssetDefinition,
) (*Genesis, error) {
	g := &Genesis{}
	for assetAlias, assetDefinition := range genesisData {
		asset := GenesisAsset{
			Alias: assetAlias,
			CreateAssetTx: txs.CreateAssetTx{
				BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
					NetworkID:    networkID,
					BlockchainID: ids.Empty,
					Memo:         assetDefinition.Memo,
				}},
				Name:         assetDefinition.Name,
				Symbol:       assetDefinition.Symbol,
				Denomination: assetDefinition.Denomination,
			},
		}

		initialState := &txs.InitialState{
			FxIndex: 0, // secp256k1fx
		}
		for _, holder := range assetDefinition.InitialState.FixedCap {
			_, addrbuff, err := address.ParseBech32(holder.Address)
			if err != nil {
				return nil, fmt.Errorf("problem parsing holder address: %w", err)
			}
			addr, err := ids.ToShortID(addrbuff)
			if err != nil {
				return nil, fmt.Errorf("problem parsing holder address: %w", err)
			}
			initialState.Outs = append(initialState.Outs, &secp256k1fx.TransferOutput{
				Amt: holder.Amount,
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs:     []ids.ShortID{addr},
				},
			})
		}
		for _, owners := range assetDefinition.InitialState.VariableCap {
			out := &secp256k1fx.MintOutput{
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: owners.Threshold,
				},
			}
			for _, addrStr := range owners.Minters {
				_, addrBytes, err := address.ParseBech32(addrStr)
				if err != nil {
					return nil, fmt.Errorf("problem parsing minters address: %w", err)
				}
				addr, err := ids.ToShortID(addrBytes)
				if err != nil {
					return nil, fmt.Errorf("problem parsing minters address: %w", err)
				}
				out.Addrs = append(out.Addrs, addr)
			}
			out.Sort()

			initialState.Outs = append(initialState.Outs, out)
		}

		if len(initialState.Outs) > 0 {
			codec, err := newGenesisCodec()
			if err != nil {
				return nil, err
			}
			initialState.Sort(codec)
			asset.States = append(asset.States, initialState)
		}

		utils.Sort(asset.States)
		g.Txs = append(g.Txs, &asset)
	}
	utils.Sort(g.Txs)

	return g, nil
}

// Bytes serializes the Genesis to bytes using the XVM genesis codec
func (g *Genesis) Bytes() ([]byte, error) {
	codec, err := newGenesisCodec()
	if err != nil {
		return nil, err
	}
	return codec.Marshal(txs.CodecVersion, g)
}

func newGenesisCodec() (codec.Manager, error) {
	parser, err := txs.NewParser(
		[]fxs.Fx{
			&secp256k1fx.Fx{},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("problem creating parser: %w", err)
	}
	return parser.GenesisCodec(), nil
}
