// Copyright (C) 2022, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

package models

import (
	"time"

	"github.com/luxfi/ids"
)

type ElasticNetConfig struct {
	NetID                    ids.ID
	SubnetID                 ids.ID // Deprecated: use NetID
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

// ElasticChainConfig is an alias for backward compatibility
type ElasticChainConfig = ElasticNetConfig
