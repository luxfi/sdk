// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package params

// EIP-4844 blob gas constants not present in upstream geth
const (
	// BlobTxTargetBlobGasPerBlock is the target blob gas per block
	// Calculated as 3 blobs per block * gas per blob
	BlobTxTargetBlobGasPerBlock = 3 * (1 << 17) // 3 * BlobTxBlobGasPerBlob
)