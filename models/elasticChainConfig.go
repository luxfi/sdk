// Copyright (C) 2022, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package models

import (
	"time"

	"github.com/luxfi/ids"
)

type ElasticChainConfig struct {
	ChainID                  ids.ID // Chain identifier
	AssetID                  ids.ID
	InitialSupply            uint64
	MaxSupply                uint64
	MinConsumptionRate       uint64
	MaxConsumptionRate       uint64
	MinValidatorStake        uint64
	MaxValidatorStake        uint64
	MinStakeDuration         time.Duration
	MaxStakeDuration         time.Duration
	MinDelegationFee         uint32
	MinDelegatorStake        uint64
	MaxValidatorWeightFactor byte
	UptimeRequirement        uint32
}

// ElasticNetConfig is an alias for backward compatibility (deprecated)
type ElasticNetConfig = ElasticChainConfig
