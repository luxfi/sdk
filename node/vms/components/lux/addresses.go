// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package lux

import (
	"errors"
	"fmt"
	consensusctx "github.com/luxfi/consensus/context"

	"github.com/luxfi/const"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/utils/formatting/address"
)

var (
	_ AddressManager = (*addressManager)(nil)

	ErrMismatchedChainIDs = errors.New("mismatched chainIDs")
)

// BCLookup provides blockchain alias lookup
type BCLookup interface {
	Lookup(string) (ids.ID, error)
	PrimaryAlias(ids.ID) (string, error)
}

type AddressManager interface {
	// ParseLocalAddress takes in an address for this chain and produces the ID
	ParseLocalAddress(addrStr string) (ids.ShortID, error)

	// ParseAddress takes in an address and produces the ID of the chain it's
	// for and the ID of the address
	ParseAddress(addrStr string) (ids.ID, ids.ShortID, error)

	// FormatLocalAddress takes in a raw address and produces the formatted
	// address for this chain
	FormatLocalAddress(addr ids.ShortID) (string, error)

	// FormatAddress takes in a chainID and a raw address and produces the
	// formatted address for that chain
	FormatAddress(chainID ids.ID, addr ids.ShortID) (string, error)
}

type addressManager struct {
	ctx *consensusctx.Context
}

func NewAddressManager(ctx *consensusctx.Context) AddressManager {
	return &addressManager{
		ctx: ctx,
	}
}

func (a *addressManager) ParseLocalAddress(addrStr string) (ids.ShortID, error) {
	chainID, addr, err := a.ParseAddress(addrStr)
	if err != nil {
		return ids.ShortID{}, err
	}
	expectedChainID := a.ctx.ChainID
	if chainID != expectedChainID {
		return ids.ShortID{}, fmt.Errorf(
			"%w: expected %q but got %q",
			ErrMismatchedChainIDs,
			expectedChainID,
			chainID,
		)
	}
	return addr, nil
}

func (a *addressManager) ParseAddress(addrStr string) (ids.ID, ids.ShortID, error) {
	chainIDAlias, hrp, addrBytes, err := address.Parse(addrStr)
	if err != nil {
		return ids.Empty, ids.ShortID{}, err
	}

	// Try to parse chainIDAlias as an ID directly since consensus context doesn't have BCLookup
	chainID, err := ids.FromString(chainIDAlias)
	if err != nil {
		return ids.ID{}, ids.ShortID{}, fmt.Errorf("failed to parse chain ID %q: %w", chainIDAlias, err)
	}

	networkID := a.ctx.NetworkID
	expectedHRP := constants.GetHRP(networkID)
	if hrp != expectedHRP {
		return ids.Empty, ids.ShortID{}, fmt.Errorf(
			"expected hrp %q but got %q",
			expectedHRP,
			hrp,
		)
	}

	addr, err := ids.ToShortID(addrBytes)
	if err != nil {
		return ids.Empty, ids.ShortID{}, err
	}
	return chainID, addr, nil
}

func (a *addressManager) FormatLocalAddress(addr ids.ShortID) (string, error) {
	chainID := a.ctx.ChainID
	return a.FormatAddress(chainID, addr)
}

func (a *addressManager) FormatAddress(chainID ids.ID, addr ids.ShortID) (string, error) {
	// Use ChainID directly - consensus context doesn't have BCLookup
	chainIDAlias := chainID.String()
	hrp := constants.GetHRP(a.ctx.NetworkID)
	return address.Format(chainIDAlias, hrp, addr.Bytes())
}

func ParseLocalAddresses(a AddressManager, addrStrs []string) (set.Set[ids.ShortID], error) {
	addrs := make(set.Set[ids.ShortID], len(addrStrs))
	for _, addrStr := range addrStrs {
		addr, err := a.ParseLocalAddress(addrStr)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse address %q: %w", addrStr, err)
		}
		addrs.Add(addr)
	}
	return addrs, nil
}

// ParseServiceAddress get address ID from address string, being it either localized (using address manager,
// doing also components validations), or not localized.
// If both attempts fail, reports error from localized address parsing
func ParseServiceAddress(a AddressManager, addrStr string) (ids.ShortID, error) {
	addr, err := ids.ShortFromString(addrStr)
	if err == nil {
		return addr, nil
	}

	addr, err = a.ParseLocalAddress(addrStr)
	if err != nil {
		return addr, fmt.Errorf("couldn't parse address %q: %w", addrStr, err)
	}
	return addr, nil
}

// ParseServiceAddress get addresses IDs from addresses strings, being them either localized or not
func ParseServiceAddresses(a AddressManager, addrStrs []string) (set.Set[ids.ShortID], error) {
	addrs := set.NewSet[ids.ShortID](len(addrStrs))
	for _, addrStr := range addrStrs {
		addr, err := ParseServiceAddress(a, addrStr)
		if err != nil {
			return nil, err
		}
		addrs.Add(addr)
	}
	return addrs, nil
}
