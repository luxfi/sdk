// (c) 2019-2020, Lux Industries, Inc.
//
// This file is a derived work, based on the go-ethereum library whose original
// notices appear below.
//
// It is distributed under a license compatible with the licensing terms of the
// original code from which it is derived.
//
// Much love to the original authors for their work.
// **********
// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package consensus implements different Ethereum consensus engines.
package consensus

import (
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/evm/commontype"
	"github.com/luxfi/evm/core/types"
	"github.com/luxfi/evm/interfaces"
)

// Use interfaces from the interfaces package
type ChainHeaderReader = interfaces.ChainHeaderReader
type ChainReader = interfaces.ChainReader
type Engine = interfaces.Engine

// PoW is a consensus engine based on proof-of-work (deprecated).
type PoW interface {
	Engine

	// Hashrate returns the current mining hashrate of a PoW consensus engine.
	Hashrate() float64
}

// PoS is a consensus engine based on proof-of-stake (delegated to Lux).
type PoS interface {
	Engine
}

// FeeConfig defines the interface for chain fee configuration
type FeeConfig interface {
	GetBaseFee(timestamp uint64) *big.Int
	GetMaxGasLimit() *big.Int
}