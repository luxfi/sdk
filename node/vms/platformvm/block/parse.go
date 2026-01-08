// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package block

import "github.com/luxfi/sdk/node/codec"

func Parse(c codec.Manager, b []byte) (Block, error) {
	var blk Block
	if _, err := c.Unmarshal(b, &blk); err != nil {
		return nil, err
	}
	return blk, blk.initialize(b)
}
