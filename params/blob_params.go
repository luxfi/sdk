// Copyright 2023 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

// Blob gas constants - these values are defined by EIP-4844
const (
	BlobTxBytesPerFieldElement         = 32       // Size of each field element in bytes
	BlobTxFieldElementsPerBlob         = 4096     // Number of field elements stored in a single data blob
	BlobTxBlobGasPerBlob               = 1 << 17  // Gas consumption of a single data blob (== 131072)
	BlobTxMinBlobGasprice              = 1        // Minimum gas price for data blobs
	BlobTxBlobGaspriceUpdateFraction   = 3338477  // Controls the maximum rate of change for blob gas price
	BlobTxPointEvaluationPrecompileGas = 50000    // Gas price for the point evaluation precompile
	BlobTxTargetBlobGasPerBlock        = 3 * BlobTxBlobGasPerBlob // Target consumable blob gas for data blobs per block (for 1559-like pricing)
	MaxBlobGasPerBlock                 = 6 * BlobTxBlobGasPerBlob // Maximum consumable blob gas for data blobs per block
)