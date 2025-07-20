// (c) 2024, Hanzo Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package customtypes

import (
	"math/big"

	ethtypes "github.com/luxfi/geth/core/types"
)

func BlockGasCost(b *ethtypes.Block) *big.Int {
	cost := GetHeaderExtra(b.Header()).BlockGasCost
	if cost == nil {
		return nil
	}
	return new(big.Int).Set(cost)
}
