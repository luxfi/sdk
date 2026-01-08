// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"encoding/json"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/chains"
	"github.com/luxfi/sdk/utils/units"
)

var Default = Config{
	Network:                       DefaultNetwork,
	BlockCacheSize:                64 * units.MiB,
	TxCacheSize:                   128 * units.MiB,
	TransformedNetTxCacheSize:  4 * units.MiB,
	RewardUTXOsCacheSize:          2048,
	ChainCacheSize:                2048,
	ChainDBCacheSize:              2048,
	BlockIDCacheSize:              8192,
	FxOwnerCacheSize:              4 * units.MiB,
	NetToL1ConversionCacheSize: 4 * units.MiB,
	L1WeightsCacheSize:            16 * units.KiB,
	L1InactiveValidatorsCacheSize: 256 * units.KiB,
	L1NetIDNodeIDCacheSize:     16 * units.KiB,
	ChecksumsEnabled:              false,
	MempoolPruneFrequency:         30 * time.Minute,
	TxFee:                         units.MilliLux,
	CreateAssetTxFee:              units.MilliLux,
	CreateNetTxFee:             units.Lux,
	CreateBlockchainTxFee:         units.Lux,
	AddPrimaryNetworkValidatorFee: 0,
	AddPrimaryNetworkDelegatorFee: 0,
}

// Config contains all of the user-configurable parameters of the PlatformVM.
type Config struct {
	Network                       Network          `json:"network"`
	BlockCacheSize                int              `json:"block-cache-size"`
	TxCacheSize                   int              `json:"tx-cache-size"`
	TransformedNetTxCacheSize  int              `json:"transformed-subnet-tx-cache-size"`
	RewardUTXOsCacheSize          int              `json:"reward-utxos-cache-size"`
	ChainCacheSize                int              `json:"chain-cache-size"`
	ChainDBCacheSize              int              `json:"chain-db-cache-size"`
	BlockIDCacheSize              int              `json:"block-id-cache-size"`
	FxOwnerCacheSize              int              `json:"fx-owner-cache-size"`
	NetToL1ConversionCacheSize int              `json:"subnet-to-l1-conversion-cache-size"`
	L1WeightsCacheSize            int              `json:"l1-weights-cache-size"`
	L1InactiveValidatorsCacheSize int              `json:"l1-inactive-validators-cache-size"`
	L1NetIDNodeIDCacheSize     int              `json:"l1-subnet-id-node-id-cache-size"`
	ChecksumsEnabled              bool             `json:"checksums-enabled"`
	MempoolPruneFrequency         time.Duration    `json:"mempool-prune-frequency"`
	SybilProtectionEnabled        bool             `json:"sybil-protection-enabled"`
	TrackedChains              set.Set[ids.ID]  `json:"tracked-chains"`
	Chains                        chains.Manager   `json:"-"`
	
	// Transaction fees
	TxFee                         uint64           `json:"tx-fee"`
	CreateAssetTxFee              uint64           `json:"create-asset-tx-fee"`
	CreateNetTxFee             uint64           `json:"create-subnet-tx-fee"`
	CreateBlockchainTxFee         uint64           `json:"create-blockchain-tx-fee"`
	AddPrimaryNetworkValidatorFee uint64           `json:"add-primary-network-validator-fee"`
	AddPrimaryNetworkDelegatorFee uint64           `json:"add-primary-network-delegator-fee"`
}

// GetConfig returns a Config from the provided json encoded bytes. If a
// configuration is not provided in the bytes, the default value is set. If
// empty bytes are provided, the default config is returned.
func GetConfig(b []byte) (*Config, error) {
	ec := Default

	// An empty slice is invalid json, so handle that as a special case.
	if len(b) == 0 {
		return &ec, nil
	}

	return &ec, json.Unmarshal(b, &ec)
}
