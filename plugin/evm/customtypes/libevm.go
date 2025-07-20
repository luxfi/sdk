// (c) 2025, Hanzo Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package customtypes

import (
	"io"

	"github.com/luxfi/geth/rlp"
)

// TODO: RegisterExtras API doesn't exist in go-ethereum v1.16.1
// var extras = ethtypes.RegisterExtras[
// 	HeaderExtra, *HeaderExtra,
// 	ethtypes.NOOPBlockBodyHooks, *ethtypes.NOOPBlockBodyHooks,
// 	noopStateAccountExtras,
// ]()

type noopStateAccountExtras struct{}

// EncodeRLP implements the [rlp.Encoder] interface.
func (noopStateAccountExtras) EncodeRLP(w io.Writer) error { return nil }

// DecodeRLP implements the [rlp.Decoder] interface.
func (*noopStateAccountExtras) DecodeRLP(s *rlp.Stream) error { return nil }
