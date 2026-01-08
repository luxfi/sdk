// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package secp256k1fx

import (
	"context"

	"github.com/luxfi/sdk/node/vms/components/verify"
)

var _ verify.State = (*MintOutput)(nil)

type MintOutput struct {
	verify.IsState `json:"-"`

	OutputOwners `serialize:"true"`
}

func (out *MintOutput) InitializeWithContext(ctx context.Context) error {
	return nil
}

func (out *MintOutput) Verify() error {
	if out == nil {
		return ErrNilOutput
	}

	return out.OutputOwners.Verify()
}
