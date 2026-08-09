// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txstest

import (
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/ids"
	wkeychain "github.com/luxfi/keychain"
	"github.com/luxfi/math/set"
	"github.com/luxfi/proto/p/config"
	"github.com/luxfi/proto/p/state"
	"github.com/luxfi/runtime"
	"github.com/luxfi/sdk/wallet/chain/p/builder"
	"github.com/luxfi/sdk/wallet/chain/p/signer"
	"github.com/luxfi/utxo/secp256k1fx"
)

func NewWalletFactory(
	rt *runtime.Runtime,
	cfg *config.Config,
	state state.State,
) *WalletFactory {
	return &WalletFactory{
		rt:    rt,
		cfg:   cfg,
		state: state,
	}
}

// NewWalletFactoryWithAssets creates a wallet factory with explicit asset IDs
func NewWalletFactoryWithAssets(
	rt *runtime.Runtime,
	cfg *config.Config,
	state state.State,
	utxoAssetID ids.ID,
) *WalletFactory {
	if rt == nil {
		rt = &runtime.Runtime{}
	}
	rt.UTXOAssetID = utxoAssetID
	return &WalletFactory{
		rt:    rt,
		cfg:   cfg,
		state: state,
	}
}

type WalletFactory struct {
	rt    *runtime.Runtime
	cfg   *config.Config
	state state.State
}

// keychainAdapter adapts secp256k1fx.Keychain (utils/crypto keychain) to wallet keychain
type keychainAdapter struct {
	kc *secp256k1fx.Keychain
}

func (k *keychainAdapter) Get(addr ids.ShortID) (wkeychain.Signer, bool) {
	utilsSigner, ok := k.kc.Get(addr)
	if !ok {
		return nil, false
	}
	return utilsSigner.(wkeychain.Signer), true
}

func (k *keychainAdapter) Addresses() set.Set[ids.ShortID] {
	return k.kc.Addresses()
}

func (w *WalletFactory) NewWallet(keys ...*secp256k1.PrivateKey) (builder.Builder, signer.Signer) {
	var (
		kc      = secp256k1fx.NewKeychain(keys...)
		addrSet = kc.AddressSet()
		backend = newBackend(addrSet, w.state)
		// Extract networkID and UTXOAssetID from context
		networkID   = w.rt.NetworkID
		utxoAssetID = w.rt.UTXOAssetID
	)

	context := newContext(w.rt, networkID, utxoAssetID, w.cfg, nil, w.state.GetTimestamp())
	kcAdapter := &keychainAdapter{kc: kc}

	return builder.New(addrSet, context, backend), signer.New(kcAdapter, backend)
}
