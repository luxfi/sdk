// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package stakeable

import (
	"errors"

	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/sdk/node/vms/components/lux"
)

var (
	errInvalidLocktime      = errors.New("invalid locktime")
	errNestedStakeableLocks = errors.New("shouldn't nest stakeable locks")
)

type LockOut struct {
	Locktime            uint64 `serialize:"true" json:"locktime"`
	lux.TransferableOut `serialize:"true" json:"output"`
}

func (s *LockOut) InitCtx(ctx *consensusctx.Context) {
	// Initialize the context for the underlying output if it supports it
	if contextOutput, ok := s.TransferableOut.(interface{ InitCtx(*consensusctx.Context) }); ok {
		contextOutput.InitCtx(ctx)
	}
}

func (s *LockOut) Addresses() [][]byte {
	if addressable, ok := s.TransferableOut.(lux.Addressable); ok {
		return addressable.Addresses()
	}
	return nil
}

func (s *LockOut) Verify() error {
	if s.Locktime == 0 {
		return errInvalidLocktime
	}
	if _, nested := s.TransferableOut.(*LockOut); nested {
		return errNestedStakeableLocks
	}
	return s.TransferableOut.Verify()
}

type LockIn struct {
	Locktime           uint64 `serialize:"true" json:"locktime"`
	lux.TransferableIn `serialize:"true" json:"input"`
}

func (s *LockIn) Verify() error {
	if s.Locktime == 0 {
		return errInvalidLocktime
	}
	if _, nested := s.TransferableIn.(*LockIn); nested {
		return errNestedStakeableLocks
	}
	return s.TransferableIn.Verify()
}
