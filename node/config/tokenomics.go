// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

// TokenomicsConfig defines the token economics for the Lux Network
type TokenomicsConfig struct {
	// Total supply: 2T tokens
	TotalSupply uint64

	// Airdrop configuration for legacy token holders
	AirdropConfig AirdropConfig

	// Staking requirements
	StakingConfig StakingConfig

	// Chain-specific allocations
	ChainAllocations map[string]uint64
}

// AirdropConfig defines airdrop parameters
type AirdropConfig struct {
	Enabled          bool
	SnapshotDate     string
	ConversionRatio  float64 // Legacy token to LUX conversion ratio
	VestingPeriod    uint64  // in seconds
	ClaimPeriod      uint64  // in seconds
}

// StakingConfig defines staking parameters
type StakingConfig struct {
	// Minimum stake: 1M LUX for validators
	MinimumValidatorStake uint64

	// NFT staking tiers
	NFTStakingEnabled bool
	NFTTiers          []NFTTier

	// Delegation parameters
	MinimumDelegatorStake uint64
	MaxDelegationRatio    uint32

	// Chain-specific validator requirements
	ChainValidatorRequirements map[string]ChainStakeRequirement

	// Combined staking (NFT + delegation + staked) to reach minimum
	AllowCombinedStaking bool
}

// NFTTier represents different validator NFT tiers
type NFTTier struct {
	Name              string
	RequiredLUX       uint64 // Base LUX requirement with NFT
	StakingMultiplier uint32 // Reward multiplier percentage
	MaxValidators     uint32 // Max validators in this tier
}

// ChainStakeRequirement defines chain-specific staking requirements
type ChainStakeRequirement struct {
	ChainID               string
	MinimumStake          uint64
	RequiresSpecialAccess bool   // e.g., B-chain bridge validators
	AccessRequirements    string // Description of special requirements
}

// DefaultTokenomicsConfig returns the default tokenomics configuration
func DefaultTokenomicsConfig() *TokenomicsConfig {
	return &TokenomicsConfig{
		// 10 Billion total supply (with 9 decimals = 10^19 units)
		TotalSupply: 10_000_000_000_000_000_000, // 10B tokens with 9 decimals

		AirdropConfig: AirdropConfig{
			Enabled:          true,
			SnapshotDate:    "2025-01-01T00:00:00Z",
			ConversionRatio: 1000.0, // 1 legacy token = 1000 LUX
			VestingPeriod:    365 * 24 * 60 * 60, // 1 year
			ClaimPeriod:      90 * 24 * 60 * 60,  // 90 days to claim
		},

		StakingConfig: StakingConfig{
			// 1M LUX minimum for validators (can be combined)
			MinimumValidatorStake: 1_000_000_000_000_000, // 1M tokens with 9 decimals
			
			NFTStakingEnabled: true,
			NFTTiers: []NFTTier{
				{
					Name:              "Genesis",
					RequiredLUX:       500_000_000_000_000,  // 500K LUX with Genesis NFT
					StakingMultiplier: 200,             // 2x rewards
					MaxValidators:     100,
				},
				{
					Name:              "Pioneer",
					RequiredLUX:       750_000_000_000_000,  // 750K LUX with Pioneer NFT
					StakingMultiplier: 150,             // 1.5x rewards
					MaxValidators:     500,
				},
				{
					Name:              "Standard",
					RequiredLUX:       1_000_000_000_000_000, // 1M LUX standard
					StakingMultiplier: 100,              // 1x rewards
					MaxValidators:     0,                // Unlimited
				},
			},

			// Delegation minimum: 25K LUX
			MinimumDelegatorStake: 25_000 * 1e9,
			MaxDelegationRatio:    10, // 10x validator stake

			// Allow combined staking (NFT value + delegated + staked = 1M+)
			AllowCombinedStaking: true,

			// Chain-specific requirements
			ChainValidatorRequirements: map[string]ChainStakeRequirement{
				"P": {
					ChainID:      "P",
					MinimumStake: 1_000_000 * 1e9, // 1M LUX standard
				},
				"C": {
					ChainID:      "C",
					MinimumStake: 1_000_000 * 1e9, // 1M LUX standard
				},
				"X": {
					ChainID:      "X",
					MinimumStake: 1_000_000 * 1e9, // 1M LUX standard
				},
				"A": {
					ChainID:      "A",
					MinimumStake: 1_000_000 * 1e9, // 1M LUX for AI chain
				},
				"B": {
					ChainID:               "B",
					MinimumStake:          100_000_000 * 1e9, // 100M LUX for bridge validators
					RequiresSpecialAccess: true,
					AccessRequirements:    "Bridge validator requires 100M LUX stake and KYC verification",
				},
				"Z": {
					ChainID:      "Z",
					MinimumStake: 1_000_000 * 1e9, // 1M LUX standard
				},
			},
		},

		// Chain allocations (percentage of total supply)
		ChainAllocations: map[string]uint64{
			"P-Chain": 1_500_000_000_000_000_000,  // 1.5B for staking/governance (15%)
			"X-Chain": 2_000_000_000_000_000_000,  // 2B for exchanges/liquidity (20%)
			"C-Chain": 3_000_000_000_000_000_000,  // 3B for smart contracts/DeFi (30%)
			"A-Chain": 1_500_000_000_000_000_000,  // 1.5B for attestation operations (15%)
			"B-Chain": 1_000_000_000_000_000_000,  // 1B for bridge liquidity (10%)
			"Z-Chain": 1_000_000_000_000_000_000,  // 1B for privacy/ZK operations (10%)
		},
	}
}

// GetMinimumStake returns the minimum stake based on NFT ownership
func (c *TokenomicsConfig) GetMinimumStake(hasNFT bool, nftTier string) uint64 {
	if !hasNFT || !c.StakingConfig.NFTStakingEnabled {
		return c.StakingConfig.MinimumValidatorStake
	}

	for _, tier := range c.StakingConfig.NFTTiers {
		if tier.Name == nftTier {
			return tier.RequiredLUX
		}
	}

	return c.StakingConfig.MinimumValidatorStake
}

// GetStakingMultiplier returns the reward multiplier for a given NFT tier
func (c *TokenomicsConfig) GetStakingMultiplier(nftTier string) uint32 {
	if !c.StakingConfig.NFTStakingEnabled {
		return 100 // Default 1x multiplier
	}

	for _, tier := range c.StakingConfig.NFTTiers {
		if tier.Name == nftTier {
			return tier.StakingMultiplier
		}
	}

	return 100 // Default 1x multiplier
}