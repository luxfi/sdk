// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txstest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/ids"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/platformvm/config"
	"github.com/luxfi/sdk/node/vms/platformvm/fx"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/platformvm/warp/message"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/node/wallet/chain/p/builder"
	"github.com/luxfi/sdk/node/wallet/chain/p/signer"
	"github.com/luxfi/sdk/node/wallet/chain/p/wallet"
	"github.com/luxfi/sdk/node/wallet/net/primary/common"
)

func NewWallet(
	t testing.TB,
	ctx *consensusctx.Context,
	cfg *config.Config,
	state state.State,
	kc *secp256k1fx.Keychain,
	netIDs []ids.ID,
	validationIDs []ids.ID,
	chainIDs []ids.ID,
) wallet.Wallet {
	return NewWalletWithOptions(
		t,
		ctx,
		WalletConfig{
			Config:        cfg,
			InternalCfg:   nil, // No dynamic fees by default
		},
		state,
		kc,
		netIDs,
		validationIDs,
		chainIDs,
	)
}

type WalletConfig struct {
	Config      *config.Config
	InternalCfg *config.Internal // Optional: for dynamic fees
}

func NewWalletWithOptions(
	t testing.TB,
	ctx *consensusctx.Context,
	wCfg WalletConfig,
	state state.State,
	kc *secp256k1fx.Keychain,
	netIDs []ids.ID,
	validationIDs []ids.ID,
	chainIDs []ids.ID,
) wallet.Wallet {
	var (
		require = require.New(t)
		addrs   = kc.Addresses()
		utxos   = common.NewUTXOs()
	)

	pChainUTXOs, err := lux.GetAllUTXOs(state, addrs)
	require.NoError(err)

	for _, utxo := range pChainUTXOs {
		require.NoError(utxos.AddUTXO(
			context.Background(),
			constants.PlatformChainID,
			constants.PlatformChainID,
			utxo,
		))
	}

	// Add cross-chain UTXOs from shared memory for import transactions
	if sm, ok := ctx.SharedMemory.(interface {
		Indexed(chainID ids.ID, addrs [][]byte, startAddr, startUTXO []byte, limit int) ([][]byte, []byte, []byte, error)
	}); ok && len(chainIDs) > 0 {
		// Convert addresses to [][]byte for SharedMemory API
		addrsList := addrs.List()
		addrsBytes := make([][]byte, len(addrsList))
		for i, addr := range addrsList {
			addrsBytes[i] = addr.Bytes()
		}

		for _, chainID := range chainIDs {
			// Indexed returns UTXOs that chainID has put in our (P-Chain's) shared memory
			// for us to import. These were exported from chainID to P-Chain.
			atomicUTXOs, _, _, err := sm.Indexed(
				chainID, // The source chain we're importing from
				addrsBytes,
				nil,
				nil,
				100, // reasonable limit for test wallets
			)
			if err != nil {
				// If error getting atomic UTXOs, skip this chain but don't fail
				// Some tests may not have atomic UTXOs set up
				continue
			}

			for _, utxoBytes := range atomicUTXOs {
				var utxo lux.UTXO
				_, err := txs.Codec.Unmarshal(utxoBytes, &utxo)
				if err != nil {
					continue // Skip malformed UTXOs
				}

				require.NoError(utxos.AddUTXO(
					context.Background(),
					chainID,
					constants.PlatformChainID,
					&utxo,
				))
			}
		}
	}

	owners := make(map[ids.ID]fx.Owner, len(netIDs)+len(validationIDs))
	for _, netID := range netIDs {
		owner, err := state.GetNetOwner(netID)
		require.NoError(err)
		owners[netID] = owner
	}
	for _, validationID := range validationIDs {
		l1Validator, err := state.GetL1Validator(validationID)
		require.NoError(err)

		var owner message.PChainOwner
		_, err = txs.Codec.Unmarshal(l1Validator.DeactivationOwner, &owner)
		require.NoError(err)
		owners[validationID] = &secp256k1fx.OutputOwners{
			Threshold: owner.Threshold,
			Addrs:     owner.Addresses,
		}
	}

	backend := wallet.NewBackend(
		common.NewChainUTXOs(constants.PlatformChainID, utxos),
		owners,
	)
	builderContext := newContext(ctx, ctx.NetworkID, ctx.XAssetID, wCfg.Config, wCfg.InternalCfg, state.GetTimestamp())
	kcAdapter := &keychainAdapter{kc: kc}
	return wallet.New(
		&client{
			backend: backend,
		},
		builder.New(
			addrs,
			builderContext,
			backend,
		),
		signer.New(
			kcAdapter,
			backend,
		),
	)
}

type client struct {
	backend wallet.Backend
}

func (c *client) IssueTx(
	tx *txs.Tx,
	options ...common.Option,
) error {
	ops := common.NewOptions(options)
	ctx := ops.Context()
	return c.backend.AcceptTx(ctx, tx)
}
