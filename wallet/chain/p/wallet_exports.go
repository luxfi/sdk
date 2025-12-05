// Copyright (C) 2019-2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package p

import (
	pwallet "github.com/luxfi/sdk/wallet/chain/p/wallet"
)

// Re-export wallet types so that importers of wallet/chain/p can access them
type Wallet = pwallet.Wallet

var (
	NewWallet            = pwallet.New
	NewWalletWithOptions = pwallet.WithOptions
)
