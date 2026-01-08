// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"bytes"
	"errors"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/crypto/bls/signer/localsigner"
)

const pChainHeight uint64 = 1337

var (
	errTest  = errors.New("non-nil error")
	testVdrs []*testValidator
)

type testValidator struct {
	nodeID ids.NodeID
	sk     *localsigner.LocalSigner
	vdr    *Validator
}

func (v *testValidator) Compare(other *testValidator) int {
	return bytes.Compare(v.vdr.PublicKeyBytes, other.vdr.PublicKeyBytes)
}

func newTestValidator() *testValidator {
	sk, err := localsigner.New()
	if err != nil {
		panic(err)
	}

	nodeID := ids.GenerateTestNodeID()
	pk := sk.PublicKey()
	return &testValidator{
		nodeID: nodeID,
		sk:     sk,
		vdr: &Validator{
			PublicKey:      pk,
			PublicKeyBytes: bls.PublicKeyToUncompressedBytes(pk),
			Weight:         3,
			NodeIDs:        []ids.NodeID{nodeID},
		},
	}
}

func init() {
	testVdrs = []*testValidator{
		newTestValidator(),
		newTestValidator(),
		newTestValidator(),
	}
	utils.Sort(testVdrs)
}
