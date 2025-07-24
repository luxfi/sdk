// (c) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Etna defines constants used after the Etna upgrade.
package etna

import "github.com/luxfi/evm/utils"

// MinBaseFee is the minimum base fee specified in LIP-125 that is allowed after
// the Etna upgrade.
//
// See: https://github.com/lux-foundation/LIPs/tree/main/LIPs/125-basefee-reduction
//
// This value modifies the previously used `ap4.MinBaseFee`.
const MinBaseFee = utils.GWei
