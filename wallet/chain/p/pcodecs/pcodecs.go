// Copyright (C) 2019-2026, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package pcodecs is the canonical construction site for the proto/p
// wire codecs used by the SDK's PVM wallet stack. proto/p carries no
// github.com/luxfi/codec import after Wave 2A (#101); construction is
// delegated to proto/zap_codec — the single canonical site where the
// wire codec backend is chosen.
//
// Wave 2G-Wallet (#101): the linearcodec/codec.Manager construction
// that previously lived here is gone. proto/zap_codec.New{PVMRuntime,
// PVMGenesis,Warp,Payload,Message} return ZAP-native little-endian
// codec managers — see that package's doc comment for the wire-format
// break vs the legacy linearcodec wire bytes (LP-023 ZAP-native
// activation, forward-only).
//
// Each consumer subpackage (sdk/wallet/chain/p, .../signer,
// .../builder, .../wallet, .../validatormanager) imports pcodecs and
// pins a package-level codec via init() rather than duplicating the
// construction at every Marshal / Parse call site.
package pcodecs

import (
	"github.com/luxfi/proto/p/block"
	"github.com/luxfi/proto/p/txs"
	"github.com/luxfi/proto/p/warp"
	"github.com/luxfi/proto/p/warp/message"
	"github.com/luxfi/proto/p/warp/payload"
	"github.com/luxfi/proto/zap_codec"
)

// NewPVMRuntimeCodec constructs the ZAP-native PVM runtime tx codec.
// block.RegisterTypes pulls in the full Apricot/Banff/Durango/Quasar
// block + tx type set in the canonical wire-byte order required by
// proto/p (block.RegisterTypes is a superset of txs.RegisterTypes —
// see proto/p/block/codec.go).
//
// Budget is proto/zap_codec.MaxSize (1 MiB) — sufficient for runtime
// txs. For genesis blobs see NewPVMGenesisCodec.
func NewPVMRuntimeCodec() (txs.Codec, error) {
	cm := zap_codec.NewPVMRuntime(txs.CodecVersion)
	if err := block.RegisterTypes(cm); err != nil {
		return nil, err
	}
	return cm, nil
}

// NewPVMGenesisCodec constructs the ZAP-native PVM genesis tx codec.
// Identical wire layout to the runtime codec but with the math.MaxInt32
// size budget required by PVM genesis blobs (full set of initial
// validator stake txs plus CreateChainTx entries for every primary-
// network chain).
func NewPVMGenesisCodec() (txs.Codec, error) {
	cm := zap_codec.NewPVMGenesis(txs.CodecVersion)
	if err := block.RegisterTypes(cm); err != nil {
		return nil, err
	}
	return cm, nil
}

// NewWarpCodec constructs the ZAP-native proto/p/warp codec.
// warp.RegisterTypes seeds the canonical signature + teleport types in
// the wire-byte order required by proto/p/warp. Used by
// fee.WarpComplexity for L1-validator tx complexity computations and by
// warp.ParseMessage for parsing observed warp message wire bytes.
//
// Budget is math.MaxInt because warp signature aggregates may grow
// with validator set size.
func NewWarpCodec() (warp.Codec, error) {
	cm := zap_codec.NewWarp(warp.CodecVersion)
	if err := warp.RegisterTypes(cm); err != nil {
		return nil, err
	}
	return cm, nil
}

// NewPayloadCodec constructs the ZAP-native proto/p/warp/payload
// codec. payload.RegisterTypes seeds the canonical AddressedCall +
// Hash payload types. Used by payload.ParseAddressedCall and
// payload.NewAddressedCall.
//
// Budget is payload.MaxMessageSize (the proto/p/warp/payload package's
// own max-message constant).
func NewPayloadCodec() (payload.Codec, error) {
	cm := zap_codec.NewPayload(payload.CodecVersion, payload.MaxMessageSize)
	if err := payload.RegisterTypes(cm); err != nil {
		return nil, err
	}
	return cm, nil
}

// NewMessageCodec constructs the ZAP-native proto/p/warp/message
// codec. message.RegisterTypes seeds the canonical ChainToL1Conversion
// / RegisterL1Validator / L1ValidatorRegistration / L1ValidatorWeight
// types. Used by message.Parse* on the AddressedCall payload bytes.
func NewMessageCodec() (message.Codec, error) {
	cm := zap_codec.NewMessage(message.CodecVersion)
	if err := message.RegisterTypes(cm); err != nil {
		return nil, err
	}
	return cm, nil
}
