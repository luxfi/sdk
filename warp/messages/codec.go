// (c) 2024, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package messages

import (
	"errors"

	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
)

const (
	CodecVersion = 0

	MaxMessageSize = 24 * interfaces.KiB
)

var Codec interfaces.Codec

func init() {
	Codec = interfaces.NewManager(MaxMessageSize)
	lc := linearinterfaces.NewDefault()

	err := errors.Join(
		lc.RegisterType(&ValidatorUptime{}),
		Codec.RegisterCodec(CodecVersion, lc),
	)
	if err != nil {
		panic(err)
	}
}
