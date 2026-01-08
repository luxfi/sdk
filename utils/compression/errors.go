// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package compression

import "errors"

var (
	ErrInvalidMaxSizeCompressor = errors.New("invalid compressor max size")
	ErrDecompressedMsgTooLarge  = errors.New("decompressed msg too large")
	ErrMsgTooLarge              = errors.New("msg too large to be compressed")
)
