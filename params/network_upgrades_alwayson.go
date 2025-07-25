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

func (n NetworkUpgrades) IsApricotPhase1(uint64) bool  { return true }
func (n NetworkUpgrades) IsApricotPhase2(uint64) bool  { return true }
func (n *NetworkUpgrades) IsApricotPhase3(uint64) bool { return true }
func (n NetworkUpgrades) IsApricotPhase4(uint64) bool  { return true }
func (n NetworkUpgrades) IsApricotPhase5(uint64) bool  { return true }
func (n NetworkUpgrades) IsApricotPhase6(uint64) bool  { return true }

// Convenience helpers used by some estimators
func (n NetworkUpgrades) IsApricotPhasePost6(uint64) bool { return true }
func (n NetworkUpgrades) IsApricotPhasePre6(uint64) bool  { return false } // always AFTER P6

func (n NetworkUpgrades) IsBanff(uint64) bool   { return true }
func (n NetworkUpgrades) IsCortina(uint64) bool { return true }
func (n NetworkUpgrades) IsDurango(uint64) bool { return true }
func (n NetworkUpgrades) IsEtna(uint64) bool    { return true }

// Already on; returns true for any timestamp
func (n *NetworkUpgrades) IsFortuna(uint64) bool { return true }
func (n *NetworkUpgrades) IsGranite(uint64) bool { return true }

// -----------------------------------------------------------------------------
// FEE RECIPIENT WHITELIST
// -----------------------------------------------------------------------------
// Empty slice -> any address is valid coinbase
func (c *ChainConfig) AllowedFeeRecipients() []common.Address { return nil }