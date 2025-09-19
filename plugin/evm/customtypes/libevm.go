// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package customtypes

import (
	"io"

	"github.com/luxfi/geth/rlp"
)

// TODO: RegisterExtras not available in current geth version
// var extras = ethtypes.RegisterExtras[
// 	HeaderExtra, *HeaderExtra,
// 	ethtypes.NOOPBlockBodyHooks, *ethtypes.NOOPBlockBodyHooks,
// 	noopStateAccountExtras,
// ]()

// Mock extras struct for compatibility
type mockExtras struct {
	Header mockHeaderExtras
}

type mockHeaderExtras struct{}

func (m mockHeaderExtras) Set(header interface{}, extra *HeaderExtra) {}
func (m mockHeaderExtras) Get(header interface{}) *HeaderExtra        { return nil }

var extras = mockExtras{
	Header: mockHeaderExtras{},
}

type noopStateAccountExtras struct{}

// EncodeRLP implements the [rlp.Encoder] interface.
func (noopStateAccountExtras) EncodeRLP(w io.Writer) error { return nil }

// DecodeRLP implements the [rlp.Decoder] interface.
func (*noopStateAccountExtras) DecodeRLP(s *rlp.Stream) error { return nil }
