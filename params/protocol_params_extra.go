// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package params

import "github.com/luxfi/geth/common"

// Re-export missing constants from geth params that might not exist in current version
const (
	// BlobTxBlobGasPerBlob is the gas consumption of a single data blob (== blob byte size)
	BlobTxBlobGasPerBlob = 1 << 17 // 131072
	// BlobTxTargetBlobGasPerBlock is the target blob gas per block
	// Calculated as 3 blobs per block * gas per blob
	BlobTxTargetBlobGasPerBlock = 3 * BlobTxBlobGasPerBlob
	// BlobTxBlobGaspriceUpdateFraction is the fraction for blob gas price updates
	BlobTxBlobGaspriceUpdateFraction = 3338477
	// MaxBlobGasPerBlock is the max blob gas per block (6 blobs)
	MaxBlobGasPerBlock = 6 * BlobTxBlobGasPerBlob
)

// BeaconRootsStorageAddress is the address of the beacon roots storage contract
var BeaconRootsStorageAddress = common.HexToAddress("0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02")
