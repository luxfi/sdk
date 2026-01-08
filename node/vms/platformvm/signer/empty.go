// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package signer

import "github.com/luxfi/crypto/bls"

var _ Signer = (*Empty)(nil)

type Empty struct{}

func (*Empty) Verify() error {
	return nil
}

func (*Empty) Key() *bls.PublicKey {
	return nil
}
