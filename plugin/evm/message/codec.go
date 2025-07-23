// (c) 2019-2022, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package message

import (
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
)

const (
	Version        = uint16(0)
	maxMessageSize = 2*interfaces.MiB - 64*interfaces.KiB // Subtract 64 KiB from p2p network cap to leave room for encoding overhead from Lux
)

var (
	Codec interfaces.Codec
)

func init() {
	Codec = interfaces.NewManager(maxMessageSize)
	c := linearinterfaces.NewDefault()

	// Skip registration to keep registeredTypes unchanged after legacy gossip deprecation
	c.SkipRegistrations(1)

	errs := interfaces.Errs{}
	errs.Add(
		// Types for state sync frontier consensus
		c.RegisterType(SyncSummary{}),

		// state sync types
		c.RegisterType(BlockRequest{}),
		c.RegisterType(BlockResponse{}),
		c.RegisterType(LeafsRequest{}),
		c.RegisterType(LeafsResponse{}),
		c.RegisterType(CodeRequest{}),
		c.RegisterType(CodeResponse{}),

		// Warp request types
		c.RegisterType(MessageSignatureRequest{}),
		c.RegisterType(BlockSignatureRequest{}),
		c.RegisterType(SignatureResponse{}),

		Codec.RegisterCodec(Version, c),
	)

	if errs.Errored() {
		panic(errs.Err)
	}
}
