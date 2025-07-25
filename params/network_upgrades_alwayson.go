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

func (n MandatoryNetworkUpgrades) IsApricotPhase1(uint64) bool  { return true }
func (n MandatoryNetworkUpgrades) IsApricotPhase2(uint64) bool  { return true }
func (n *MandatoryNetworkUpgrades) IsApricotPhase3(uint64) bool { return true }
func (n MandatoryNetworkUpgrades) IsApricotPhase4(uint64) bool  { return true }
func (n MandatoryNetworkUpgrades) IsApricotPhase5(uint64) bool  { return true }
func (n MandatoryNetworkUpgrades) IsApricotPhase6(uint64) bool  { return true }

// Convenience helpers used by some estimators
func (n MandatoryNetworkUpgrades) IsApricotPhasePost6(uint64) bool { return true }
func (n MandatoryNetworkUpgrades) IsApricotPhasePre6(uint64) bool  { return false } // always AFTER P6

func (n MandatoryNetworkUpgrades) IsBanff(uint64) bool   { return true }
func (n MandatoryNetworkUpgrades) IsCortina(uint64) bool { return true }
func (n MandatoryNetworkUpgrades) IsDurango(uint64) bool { return true }
func (n MandatoryNetworkUpgrades) IsEtna(uint64) bool    { return true }

// Already on; returns true for any timestamp
func (n *MandatoryNetworkUpgrades) IsFortuna(uint64) bool { return true }
func (n *MandatoryNetworkUpgrades) IsGranite(uint64) bool { return true }

// -----------------------------------------------------------------------------
// FEE RECIPIENT WHITELIST
// -----------------------------------------------------------------------------
// Empty slice -> any address is valid coinbase
func (c *ChainConfig) AllowedFeeRecipientsList() []common.Address { return nil }