// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package pcodecs is the canonical construction site for the proto/p
// wire codecs used by the SDK's PVM wallet stack. proto/p carries no
// github.com/luxfi/codec import after Wave 2A (#101); the
// linearcodec-backed codec.Manager instances live here so consumers
// can pick their codec implementation (linearcodec today, zapcodec
// in a future wave) without touching proto/p.
//
// Each consumer subpackage (sdk/wallet/chain/p, .../signer,
// .../builder, .../wallet, .../validatormanager) imports pcodecs and
// pins a package-level codec via init() rather than duplicating the
// linearcodec.NewDefault + codec.NewManager + RegisterTypes wiring at
// every Marshal / Parse call site.
//
// Mirrors proto/internal/pcodectest, which serves the same role for
// proto/p's own in-tree tests. pcodecs is the production-side
// equivalent — pcodectest is in proto/internal (test-only,
// linearcodec.NewDefault construction); pcodecs is in the SDK
// (production, same construction, same wire bytes).
package pcodecs

import (
	"math"

	"github.com/luxfi/codec"
	"github.com/luxfi/codec/linearcodec"
	"github.com/luxfi/proto/p/block"
	"github.com/luxfi/proto/p/txs"
	"github.com/luxfi/proto/p/warp"
	"github.com/luxfi/proto/p/warp/message"
	"github.com/luxfi/proto/p/warp/payload"
)

// NewPVMRuntimeCodec constructs the linearcodec-backed PVM runtime tx
// codec. block.RegisterTypes pulls in the full Apricot/Banff/Durango/
// Quasar block + tx type set in the canonical wire-byte order
// required by proto/p (block.RegisterTypes is a superset of
// txs.RegisterTypes — see proto/p/block/codec.go).
//
// Budget is codec.NewDefaultManager() — sufficient for runtime txs.
// For genesis blobs see NewPVMGenesisCodec.
func NewPVMRuntimeCodec() (txs.Codec, error) {
	c := linearcodec.NewDefault()
	cm := codec.NewDefaultManager()
	if err := block.RegisterTypes(c); err != nil {
		return nil, err
	}
	if err := cm.RegisterCodec(txs.CodecVersion, c); err != nil {
		return nil, err
	}
	return cm, nil
}

// NewPVMGenesisCodec constructs the linearcodec-backed PVM genesis tx
// codec. Identical wire-byte order to the runtime codec but with the
// math.MaxInt32 size budget required by PVM genesis blobs (full set of
// initial validator stake txs plus CreateChainTx entries for every
// primary-network chain).
func NewPVMGenesisCodec() (txs.Codec, error) {
	c := linearcodec.NewDefault()
	cm := codec.NewManager(math.MaxInt32)
	if err := block.RegisterTypes(c); err != nil {
		return nil, err
	}
	if err := cm.RegisterCodec(txs.CodecVersion, c); err != nil {
		return nil, err
	}
	return cm, nil
}

// NewWarpCodec constructs the linearcodec-backed proto/p/warp codec.
// warp.RegisterTypes seeds the canonical signature + teleport types
// in the wire-byte order required by proto/p/warp. Used by
// fee.WarpComplexity for L1-validator tx complexity computations and
// by warp.ParseMessage for parsing observed warp message wire bytes.
//
// Budget is math.MaxInt because warp signature aggregates may grow
// with validator set size.
func NewWarpCodec() (warp.Codec, error) {
	c := linearcodec.NewDefault()
	cm := codec.NewManager(math.MaxInt)
	if err := warp.RegisterTypes(c); err != nil {
		return nil, err
	}
	if err := cm.RegisterCodec(warp.CodecVersion, c); err != nil {
		return nil, err
	}
	return cm, nil
}

// NewPayloadCodec constructs the linearcodec-backed proto/p/warp/payload
// codec. payload.RegisterTypes seeds the canonical AddressedCall +
// Hash payload types. Used by payload.ParseAddressedCall and
// payload.NewAddressedCall.
//
// Budget is payload.MaxMessageSize (the proto/p/warp/payload
// package's own max-message constant).
func NewPayloadCodec() (payload.Codec, error) {
	c := linearcodec.NewDefault()
	cm := codec.NewManager(payload.MaxMessageSize)
	if err := payload.RegisterTypes(c); err != nil {
		return nil, err
	}
	if err := cm.RegisterCodec(payload.CodecVersion, c); err != nil {
		return nil, err
	}
	return cm, nil
}

// NewMessageCodec constructs the linearcodec-backed proto/p/warp/message
// codec. message.RegisterTypes seeds the canonical ChainToL1Conversion
// / RegisterL1Validator / L1ValidatorRegistration / L1ValidatorWeight
// types. Used by message.Parse* on the AddressedCall payload bytes.
func NewMessageCodec() (message.Codec, error) {
	c := linearcodec.NewDefault()
	cm := codec.NewManager(math.MaxInt)
	if err := message.RegisterTypes(c); err != nil {
		return nil, err
	}
	if err := cm.RegisterCodec(message.CodecVersion, c); err != nil {
		return nil, err
	}
	return cm, nil
}
