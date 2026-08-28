// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package aichain

import (
	"fmt"
	"math/big"

	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/common"
)

// InferenceReceipt is the cross-chain settlement receipt produced by the A-Chain
// when a Tier-2 inference task settles. Its canonical encoding is PINNED
// byte-for-byte (Encode / ReceiptEncodedLen = 355) and identical to
// chains/aivm/receipts.go AInferenceReceipt; its hash is
// keccak(DomainReceipt || Encode()), the leaf the receipt_root commits to.
//
// FeePaid is a non-negative *big.Int (nil == zero on the wire). Field order here
// is presentation; the WIRE order is fixed by Encode and MUST NOT be reordered.
type InferenceReceipt struct {
	Version             uint16         `json:"version"`
	IntentID            common.Hash    `json:"intentId"` // source C-chain intent id
	TaskID              common.Hash    `json:"taskId"`   // A-chain task id
	CChainID            common.Hash    `json:"cChainId"`
	AChainID            common.Hash    `json:"aChainId"`
	Requester           common.Address `json:"requester"`
	ModelSpecHash       common.Hash    `json:"modelSpecHash"`
	PromptHash          common.Hash    `json:"promptHash"`
	CanonicalOutputHash common.Hash    `json:"canonicalOutputHash"` // zero if Failed
	Status              uint8          `json:"status"`
	N                   uint16         `json:"n"`
	Threshold           uint16         `json:"threshold"`
	WinnersRoot         common.Hash    `json:"winnersRoot"`
	OperatorsRoot       common.Hash    `json:"operatorsRoot"`
	FeePaid             *big.Int       `json:"feePaid"`
	SettledAtHeight     uint64         `json:"settledAtHeight"`
}

// Encode produces the canonical fixed-width encoding (exactly ReceiptEncodedLen =
// 355 bytes) in the PINNED order, byte-identical to chains/aivm:
//
//	u16be(Version) || IntentID(32) || TaskID(32) || CChainID(32) || AChainID(32) ||
//	Requester(20) || ModelSpecHash(32) || PromptHash(32) || CanonicalOutputHash(32) ||
//	u8(Status) || u16be(N) || u16be(Threshold) || WinnersRoot(32) ||
//	OperatorsRoot(32) || u256be(FeePaid,32) || u64be(SettledAtHeight)
func (r InferenceReceipt) Encode() []byte {
	buf := make([]byte, 0, ReceiptEncodedLen)
	buf = append(buf, u16be(r.Version)...)
	buf = append(buf, r.IntentID.Bytes()...)
	buf = append(buf, r.TaskID.Bytes()...)
	buf = append(buf, r.CChainID.Bytes()...)
	buf = append(buf, r.AChainID.Bytes()...)
	buf = append(buf, r.Requester.Bytes()...)
	buf = append(buf, r.ModelSpecHash.Bytes()...)
	buf = append(buf, r.PromptHash.Bytes()...)
	buf = append(buf, r.CanonicalOutputHash.Bytes()...)
	buf = append(buf, r.Status)
	buf = append(buf, u16be(r.N)...)
	buf = append(buf, u16be(r.Threshold)...)
	buf = append(buf, r.WinnersRoot.Bytes()...)
	buf = append(buf, r.OperatorsRoot.Bytes()...)
	buf = append(buf, u256be(r.FeePaid)...)
	buf = append(buf, u64be(r.SettledAtHeight)...)
	return buf
}

// Hash is the receipt commitment: keccak256(DomainReceipt || Encode()). This is
// the leaf value (pre-leaf-hash) that goes into the receipt_root and that an
// exported proof proves membership of. Identical to chains/aivm
// AInferenceReceipt.Hash.
func (r InferenceReceipt) Hash() common.Hash {
	return common.BytesToHash(crypto.Keccak256(append([]byte(DomainReceipt), r.Encode()...)))
}

// Completed reports whether the receipt is the only actionable shape C-side:
// Status == StatusCompleted with a non-zero canonical output. A Pending,
// Failed, or Challenged receipt — or a zero output — is never credited.
func (r InferenceReceipt) Completed() bool {
	return r.Status == StatusCompleted && r.CanonicalOutputHash != (common.Hash{})
}

// DecodeReceipt parses a canonical 355-byte receipt encoding back into an
// InferenceReceipt. It is the inverse of Encode and rejects any input that is
// not EXACTLY ReceiptEncodedLen bytes (a corrupt/short/over-long frame is never
// reinterpreted into a credit — the same fail-secure discipline the precompile's
// DecodeReceipt applies).
func DecodeReceipt(b []byte) (InferenceReceipt, error) {
	var r InferenceReceipt
	if len(b) != ReceiptEncodedLen {
		return r, fmt.Errorf("aichain: receipt length %d, want %d", len(b), ReceiptEncodedLen)
	}
	o := 0
	rd16 := func() uint16 { v := uint16(b[o])<<8 | uint16(b[o+1]); o += 2; return v }
	rd32 := func() common.Hash { h := common.BytesToHash(b[o : o+32]); o += 32; return h }
	r.Version = rd16()
	r.IntentID = rd32()
	r.TaskID = rd32()
	r.CChainID = rd32()
	r.AChainID = rd32()
	r.Requester = common.BytesToAddress(b[o : o+20])
	o += 20
	r.ModelSpecHash = rd32()
	r.PromptHash = rd32()
	r.CanonicalOutputHash = rd32()
	r.Status = b[o]
	o++
	r.N = rd16()
	r.Threshold = rd16()
	r.WinnersRoot = rd32()
	r.OperatorsRoot = rd32()
	r.FeePaid = new(big.Int).SetBytes(b[o : o+32])
	o += 32
	var h uint64
	for i := 0; i < 8; i++ {
		h = h<<8 | uint64(b[o+i])
	}
	r.SettledAtHeight = h
	return r, nil
}
