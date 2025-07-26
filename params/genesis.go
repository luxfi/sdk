//go:build !legacy_upgrades_off
// Copyright 2025, Lux Industries, Inc.

package params

import "github.com/ethereum/go-ethereum/common"

// -----------------------------------------------------------------------------
// ALWAYS-ON UPGRADE FLAGS
// -----------------------------------------------------------------------------
// We launched post-Durango (Nov-2023). All Avalanche forks up to Granite are
// considered *already active*. These helpers are kept only because upstream
// code and unit-tests still expect them to exist. Feel free to delete this
// file entirely once those dependencies are gone.

func (c *ChainConfig) IsApricotPhase1(uint64) bool  { return true }
func (c *ChainConfig) IsApricotPhase2(uint64) bool  { return true }
func (c *ChainConfig) IsApricotPhase3(uint64) bool { return true }
func (c *ChainConfig) IsApricotPhase4(uint64) bool  { return true }
func (c *ChainConfig) IsApricotPhase5(uint64) bool  { return true }
func (c *ChainConfig) IsApricotPhase6(uint64) bool  { return true }

// Convenience helpers used by some estimators
func (c *ChainConfig) IsApricotPhasePost6(uint64) bool { return true }
func (c *ChainConfig) IsApricotPhasePre6(uint64) bool  { return false } // always AFTER P6

func (c *ChainConfig) IsBanff(uint64) bool   { return true }
func (c *ChainConfig) IsCortina(uint64) bool { return true }
func (c *ChainConfig) IsDurango(uint64) bool { return true }
func (c *ChainConfig) IsEtna(uint64) bool    { return true }

// Already on; returns true for any timestamp
func (c *ChainConfig) IsFortuna(uint64) bool { return true }
func (c *ChainConfig) IsGranite(uint64) bool { return true }

// -----------------------------------------------------------------------------
// FEE RECIPIENT WHITELIST
// -----------------------------------------------------------------------------
// Empty slice -> any address is valid coinbase
func (c *ChainConfig) AllowedFeeRecipientsList() []common.Address { return nil }