// Copyright (C) 2019-2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package primary

import (
	"context"

	gethcommon "github.com/luxfi/geth/common"

	"github.com/luxfi/constants"
	"github.com/luxfi/ids"
	"github.com/luxfi/keychain"
	"github.com/luxfi/math/set"
	"github.com/luxfi/protocol/p/txs"
	"github.com/luxfi/sdk/wallet/chain/c"
	"github.com/luxfi/sdk/wallet/chain/p"
	"github.com/luxfi/sdk/wallet/chain/x"
	"github.com/luxfi/sdk/wallet/primary/common"
	"github.com/luxfi/utxo/secp256k1fx"

	pbuilder "github.com/luxfi/sdk/wallet/chain/p/builder"
	psigner "github.com/luxfi/sdk/wallet/chain/p/signer"
	xbuilder "github.com/luxfi/sdk/wallet/chain/x/builder"
	xsigner "github.com/luxfi/sdk/wallet/chain/x/signer"
)

var _ Wallet = (*wallet)(nil)

// KeychainAdapter adapts secp256k1fx.Keychain to wallet/keychain.Keychain and c.EthKeychain interfaces.
// This allows secp256k1fx.Keychain to be used with MakeWallet.
type KeychainAdapter struct {
	*secp256k1fx.Keychain
}

// Addresses implements wallet/keychain.Keychain
func (kc *KeychainAdapter) Addresses() set.Set[ids.ShortID] {
	return kc.Keychain.Addrs
}

// Get implements wallet/keychain.Keychain (returns wallet/keychain.Signer, not utils/crypto/keychain.Signer)
func (kc *KeychainAdapter) Get(addr ids.ShortID) (keychain.Signer, bool) {
	signer, ok := kc.Keychain.Get(addr)
	if !ok {
		return nil, false
	}
	// secp256k1fx.luxSigner already implements wallet/keychain.Signer
	return signer.(keychain.Signer), true
}

// GetEth implements c.EthKeychain
func (kc *KeychainAdapter) GetEth(addr gethcommon.Address) (keychain.Signer, bool) {
	signer, ok := kc.Keychain.GetEth(addr)
	if !ok {
		return nil, false
	}
	// secp256k1fx.luxSigner already implements wallet/keychain.Signer
	return signer.(keychain.Signer), true
}

// EthAddresses implements c.EthKeychain
func (kc *KeychainAdapter) EthAddresses() set.Set[gethcommon.Address] {
	return kc.Keychain.EthAddrs
}

// NewKeychainAdapter creates a KeychainAdapter from a secp256k1fx.Keychain
func NewKeychainAdapter(kc *secp256k1fx.Keychain) *KeychainAdapter {
	return &KeychainAdapter{Keychain: kc}
}

// Wallet provides chain wallets for the primary network.
type Wallet interface {
	P() p.Wallet
	X() x.Wallet
	C() c.Wallet
}

type wallet struct {
	p p.Wallet
	x x.Wallet
	c c.Wallet
}

func (w *wallet) P() p.Wallet {
	return w.p
}

func (w *wallet) X() x.Wallet {
	return w.x
}

func (w *wallet) C() c.Wallet {
	return w.c
}

// Creates a new default wallet
func NewWallet(p p.Wallet, x x.Wallet, c c.Wallet) Wallet {
	return &wallet{
		p: p,
		x: x,
		c: c,
	}
}

// Creates a Wallet with the given set of options
func NewWalletWithOptions(w Wallet, options ...common.Option) Wallet {
	return NewWallet(
		p.NewWalletWithOptions(w.P(), options...),
		x.NewWalletWithOptions(w.X(), options...),
		c.NewWalletWithOptions(w.C(), options...),
	)
}

type WalletConfig struct {
	// Base URI to use for all node requests.
	URI string // required
	// Keys to use for signing all transactions.
	LUXKeychain keychain.Keychain // required
	EthKeychain c.EthKeychain     // required
	// Set of P-chain transactions that the wallet should know about to be able
	// to generate transactions.
	PChainTxs map[ids.ID]*txs.Tx // optional
	// Set of P-chain transactions that the wallet should fetch to be able to
	// generate transactions.
	PChainTxsToFetch set.Set[ids.ID] // optional
}

// MakeWallet returns a wallet that supports issuing transactions to the chains
// living in the primary network.
//
// On creation, the wallet attaches to the provided uri and fetches all UTXOs
// that reference any of the provided keys. If the UTXOs are modified through an
// external issuance process, such as another instance of the wallet, the UTXOs
// may become out of sync. The wallet will also fetch all requested P-chain
// transactions.
//
// The wallet manages all state locally, and performs all tx signing locally.
func MakeWallet(ctx context.Context, config *WalletConfig) (Wallet, error) {
	luxAddrs := config.LUXKeychain.Addresses()
	luxState, err := FetchState(ctx, config.URI, luxAddrs)
	if err != nil {
		return nil, err
	}

	// Eth state fetching disabled for now
	// ethAddrs := config.EthKeychain.EthAddresses()
	// ethState, err := FetchEthState(ctx, config.URI, ethAddrs)
	// if err != nil {
	// 	return nil, err
	// }

	pChainTxs := config.PChainTxs
	if pChainTxs == nil {
		pChainTxs = make(map[ids.ID]*txs.Tx)
	}

	for txID := range config.PChainTxsToFetch {
		txBytes, err := luxState.PClient.GetTx(ctx, txID)
		if err != nil {
			return nil, err
		}
		tx, err := txs.Parse(txs.Codec, txBytes)
		if err != nil {
			return nil, err
		}
		pChainTxs[txID] = tx
	}

	pUTXOs := common.NewChainUTXOs(constants.PlatformChainID, luxState.UTXOs)
	pBackend := p.NewBackend(luxState.PCTX, pUTXOs, pChainTxs)
	pBuilder := pbuilder.New(luxAddrs, luxState.PCTX, pBackend)
	pSigner := psigner.New(config.LUXKeychain, pBackend)

	xChainID := luxState.XCTX.BlockchainID
	xUTXOs := common.NewChainUTXOs(xChainID, luxState.UTXOs)
	xBackend := x.NewBackend(luxState.XCTX, xUTXOs)
	xBuilder := xbuilder.New(luxAddrs, luxState.XCTX, xBackend)
	xSigner := xsigner.New(config.LUXKeychain, xBackend)

	// C-chain wallet disabled - requires full EVM client implementation
	// cChainID := luxState.CCTX.BlockchainID
	// cUTXOs := common.NewChainUTXOs(cChainID, luxState.UTXOs)
	// cBackend := c.NewBackend(cUTXOs, nil)
	// cBuilder := c.NewBuilder(luxAddrs, config.EthKeychain.EthAddresses(), luxState.CCTX, cBackend)
	// cSigner := c.NewSigner(config.LUXKeychain, config.EthKeychain, cBackend)

	pClient := p.NewClient(luxState.PClient, pBackend)

	return NewWallet(
		p.NewWallet(pClient, pBuilder, pSigner),
		x.NewWallet(xBuilder, xSigner, xBackend),
		nil, // C-chain wallet not yet implemented
	), nil
}

// MakePChainWallet returns a wallet that only supports issuing P-chain transactions.
// This is an alias for MakeWallet for backward compatibility.
func MakePChainWallet(ctx context.Context, config *WalletConfig) (Wallet, error) {
	return MakeWallet(ctx, config)
}
