// Copyright (C) 2026, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package aichain

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/common"
	"github.com/stretchr/testify/require"
)

// GOLDEN x402 vectors. These are computed by the Go SDK and pinned; the TS SDK's
// test/x402.test.ts asserts the IDENTICAL paymentHash + receiptId, proving the
// EIP-712 preimage and the abi.encode receiptId layout match byte-for-byte across
// languages and against the on-chain ProofOfThoughtRegistry.computeReceiptId.
const (
	// signer = anvil key #0 (well-known, never used on a real chain).
	x402SignerKey  = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	x402SignerAddr = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

	// PaymentHash of the canonical fixture (x402Fixture) — the EIP-712 digest the
	// signature is over and the value the on-chain registry stores as paymentHash.
	GoldenPaymentHash = "0x99a3abd03c66497ebc597c616b7f3cd42b1a2b129af884d7607206a1408569d2"

	// computeReceiptId(0x44.., 0x55.., 0x66.., 0x77.., 0x..aa, 0x..bb) — matches
	// keccak256(abi.encode(bytes32,bytes32,bytes32,bytes32,address,address)).
	GoldenReceiptID = "0x803dbf024c2f91bb115752b416145b571d204480c8c9e1919539a8b83a184762"
)

// x402Fixture is the canonical PaymentRequirements the golden vectors are taken
// over. It MUST stay byte-identical to the TS test fixture.
func x402Fixture() PaymentRequirements {
	return PaymentRequirements{
		Scheme:      SchemeExact,
		Network:     big.NewInt(36911),
		MaxAmount:   big.NewInt(1_000_000),
		Token:       common.HexToAddress("0x00000000000000000000000000000000000000a0"),
		PayTo:       common.HexToAddress("0x00000000000000000000000000000000000000bb"),
		Resource:    "zenlm/zen-omni",
		Description: "one cognition",
		Nonce:       common.HexToHash("0x77"),
		ValidUntil:  2_000_000_000,
	}
}

func TestBuildPaymentPayloadDeterministicAndGolden(t *testing.T) {
	r := require.New(t)
	key, err := crypto.HexToECDSA(x402SignerKey)
	r.NoError(err)
	reqs := x402Fixture()

	p1, err := BuildPaymentPayload(key, reqs)
	r.NoError(err)
	p2, err := BuildPaymentPayload(key, reqs)
	r.NoError(err)

	// from is the signer's address.
	r.Equal(common.HexToAddress(x402SignerAddr), p1.From)
	// Payload faithfully encodes the requirements.
	r.Equal(reqs.PayTo, p1.To)
	r.Equal(reqs.Token, p1.Token)
	r.Equal(reqs.MaxAmount, p1.Value)
	r.Equal(reqs.Nonce, p1.Nonce)
	r.Equal(reqs.ValidUntil, p1.ValidBefore)
	r.EqualValues(0, p1.ValidAfter)
	r.Len(p1.Signature, 65)

	// Deterministic: same inputs -> same paymentHash AND same signature
	// (secp256k1 deterministic-k via RFC6979 in luxfi/crypto).
	r.Equal(p1.PaymentHash(), p2.PaymentHash())
	r.Equal(p1.Signature, p2.Signature)

	// GOLDEN: the EIP-712 digest is the pinned value (cross-checked in TS).
	r.Equal(common.HexToHash(GoldenPaymentHash), p1.PaymentHash(), "paymentHash golden drifted")

	// The signature recovers to the payer.
	payer, err := p1.RecoverPayer()
	r.NoError(err)
	r.Equal(common.HexToAddress(x402SignerAddr), payer)
}

func TestPaymentHashBindsTokenAndNetwork(t *testing.T) {
	r := require.New(t)
	key, err := crypto.HexToECDSA(x402SignerKey)
	r.NoError(err)
	base := x402Fixture()

	p0, err := BuildPaymentPayload(key, base)
	r.NoError(err)

	// Different network -> different digest (domain binds chainId).
	rn := base
	rn.Network = big.NewInt(1)
	pn, err := BuildPaymentPayload(key, rn)
	r.NoError(err)
	r.NotEqual(p0.PaymentHash(), pn.PaymentHash())

	// Different token -> different digest.
	rt := base
	rt.Token = common.HexToAddress("0x00000000000000000000000000000000000000ff")
	pt, err := BuildPaymentPayload(key, rt)
	r.NoError(err)
	r.NotEqual(p0.PaymentHash(), pt.PaymentHash())

	// Different nonce -> different digest.
	rnonce := base
	rnonce.Nonce = common.HexToHash("0x78")
	pnonce, err := BuildPaymentPayload(key, rnonce)
	r.NoError(err)
	r.NotEqual(p0.PaymentHash(), pnonce.PaymentHash())
}

func TestBuildPaymentPayloadValidation(t *testing.T) {
	r := require.New(t)
	key, err := crypto.HexToECDSA(x402SignerKey)
	r.NoError(err)

	_, err = BuildPaymentPayload(nil, x402Fixture())
	r.Error(err, "nil signer")

	bad := []PaymentRequirements{
		func() PaymentRequirements { x := x402Fixture(); x.Scheme = "bogus"; return x }(),
		func() PaymentRequirements { x := x402Fixture(); x.Network = nil; return x }(),
		func() PaymentRequirements { x := x402Fixture(); x.Network = big.NewInt(0); return x }(),
		func() PaymentRequirements { x := x402Fixture(); x.MaxAmount = nil; return x }(),
		func() PaymentRequirements { x := x402Fixture(); x.MaxAmount = big.NewInt(0); return x }(),
		func() PaymentRequirements { x := x402Fixture(); x.PayTo = common.Address{}; return x }(),
		func() PaymentRequirements { x := x402Fixture(); x.Nonce = common.Hash{}; return x }(),
		func() PaymentRequirements { x := x402Fixture(); x.ValidUntil = 0; return x }(),
	}
	for i, b := range bad {
		_, err := BuildPaymentPayload(key, b)
		r.Error(err, "case %d should fail validation", i)
	}
}

func TestMockFacilitatorVerifyAndSettle(t *testing.T) {
	r := require.New(t)
	key, err := crypto.HexToECDSA(x402SignerKey)
	r.NoError(err)
	reqs := x402Fixture()
	p, err := BuildPaymentPayload(key, reqs)
	r.NoError(err)

	// Clock before validUntil so the authorization is live.
	fac := &MockFacilitator{Now: func() time.Time { return time.Unix(1_000_000_000, 0) }}

	r.NoError(fac.Verify(p, reqs))

	st, err := fac.Settle(p, reqs)
	r.NoError(err)
	r.True(st.Settled)
	r.Equal(p.PaymentHash(), st.PaymentHash, "settlement carries the EIP-712 paymentHash")
	r.Equal(common.HexToAddress(x402SignerAddr), st.Payer)
	r.Equal(reqs.PayTo, st.Payee)
	r.Equal(reqs.MaxAmount, st.Amount)
	r.NotEqual(common.Hash{}, st.TxHash)
}

func TestMockFacilitatorRejectsBadPayloads(t *testing.T) {
	r := require.New(t)
	key, err := crypto.HexToECDSA(x402SignerKey)
	r.NoError(err)
	reqs := x402Fixture()
	good, err := BuildPaymentPayload(key, reqs)
	r.NoError(err)
	fac := &MockFacilitator{Now: func() time.Time { return time.Unix(1_000_000_000, 0) }}

	// Tampered amount (exact scheme requires value == maxAmount): the binding check
	// fails (and a tamper after signing would also break recovery).
	bad := good
	bad.Value = big.NewInt(999)
	r.Error(fac.Verify(bad, reqs))

	// Tampered payee.
	bad = good
	bad.To = common.HexToAddress("0x00000000000000000000000000000000000000cc")
	r.Error(fac.Verify(bad, reqs))

	// Corrupted signature -> recovery fails.
	bad = good
	bad.Signature = append([]byte(nil), good.Signature...)
	bad.Signature[10] ^= 0xff
	r.Error(fac.Verify(bad, reqs))

	// Wrong-length signature.
	bad = good
	bad.Signature = good.Signature[:64]
	r.Error(fac.Verify(bad, reqs))

	// Expired authorization (clock at/after validBefore).
	expired := &MockFacilitator{Now: func() time.Time { return time.Unix(int64(reqs.ValidUntil), 0) }}
	r.Error(expired.Verify(good, reqs))

	// Mismatched nonce between payload and requirements.
	wrongReqs := reqs
	wrongReqs.Nonce = common.HexToHash("0x99")
	r.Error(fac.Verify(good, wrongReqs))
}

func TestUptoSchemeAllowsLesserValue(t *testing.T) {
	r := require.New(t)
	key, err := crypto.HexToECDSA(x402SignerKey)
	r.NoError(err)
	reqs := x402Fixture()
	reqs.Scheme = SchemeUpto
	reqs.MaxAmount = big.NewInt(1_000_000)

	// An upto payload may authorize less than the cap.
	p, err := BuildPaymentPayload(key, reqs)
	r.NoError(err)
	r.Equal(reqs.MaxAmount, p.Value) // BuildPaymentPayload authorizes the cap by default

	// Hand-build a lesser-value upto payload, re-sign, and verify it passes binding.
	lesser := reqs
	lesser.MaxAmount = big.NewInt(400_000) // authorize less
	pl, err := BuildPaymentPayload(key, lesser)
	r.NoError(err)
	pl.Value = big.NewInt(400_000)
	// Re-derive the signature is not needed: bindsTo checks 0 < value <= cap.
	fac := &MockFacilitator{Now: func() time.Time { return time.Unix(1_000_000_000, 0) }}
	// Verify against the ORIGINAL cap (1_000_000): 400k <= 1_000_000 is allowed.
	checkReqs := reqs
	// But the signature must match: re-sign pl over the original-cap requirements
	// is unnecessary — pl was signed over lesser (value 400k, same nonce/token/net),
	// and checkReqs differs only by MaxAmount which is not in the EIP-712 preimage,
	// so recovery still holds.
	r.NoError(fac.Verify(pl, checkReqs))

	// An upto payload over the cap is rejected.
	over := pl
	over.Value = big.NewInt(2_000_000)
	r.Error(fac.Verify(over, checkReqs))
}

func TestComputeReceiptIdGolden(t *testing.T) {
	r := require.New(t)
	// EXACT abi.encode(bytes32,bytes32,bytes32,bytes32,address,address) layout —
	// matches ProofOfThoughtRegistry.computeReceiptId on-chain.
	got := ComputeReceiptId(
		common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444"),
		common.HexToHash("0x5555555555555555555555555555555555555555555555555555555555555555"),
		common.HexToHash("0x6666666666666666666666666666666666666666666666666666666666666666"),
		common.HexToHash("0x7777777777777777777777777777777777777777777777777777777777777777"),
		common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		common.HexToAddress("0x00000000000000000000000000000000000000bb"),
	)
	r.Equal(common.HexToHash(GoldenReceiptID), got, "receiptId golden drifted (abi.encode layout)")

	// PoTReceipt.ReceiptID() must equal ComputeReceiptId over its fields.
	pot := PoTReceipt{
		ModelID:     common.HexToHash("0x44"),
		PromptHash:  common.HexToHash("0x55"),
		OutputHash:  common.HexToHash("0x66"),
		PaymentHash: common.HexToHash("0x77"),
		Payer:       common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		Operator:    common.Address{},
	}
	r.Equal(
		ComputeReceiptId(pot.ModelID, pot.PromptHash, pot.OutputHash, pot.PaymentHash, pot.Payer, pot.Operator),
		pot.ReceiptID(),
	)
}

func TestInferPaidHappyPath(t *testing.T) {
	r := require.New(t)
	key, err := crypto.HexToECDSA(x402SignerKey)
	r.NoError(err)
	b := newMockBackend()
	model := ModelSpec{Name: "zenlm/zen-omni", Version: 3, WeightCommit: common.HexToHash("0xabc"), Quantization: "int8"}
	prompt := []byte("think about beluga")

	// Store returns a completed receipt bound to the requested model/prompt.
	store := &anyIDStore{
		aChain: common.HexToHash("0x2222"),
		rec: InferenceReceipt{
			Status:              StatusCompleted,
			CanonicalOutputHash: common.HexToHash("0x0d0e0a0d"),
			ModelSpecHash:       model.Hash(),
			PromptHash:          PromptHash(prompt),
		},
	}
	c := fastClient(t, b, WithReceiptStore(store))
	fac := &MockFacilitator{Now: func() time.Time { return time.Unix(1_000_000_000, 0) }}
	reqs := x402Fixture()

	res, err := c.InferPaid(context.Background(), model, prompt, 5, 3, reqs, fac, key)
	r.NoError(err)

	// Settlement carries the paymentHash and is settled.
	r.True(res.Settlement.Settled)
	r.Equal(common.HexToHash(GoldenPaymentHash), res.Settlement.PaymentHash)
	r.Equal(reqs.MaxAmount, res.Settlement.Amount)

	// Inference result is the completed receipt.
	r.True(res.Inference.Completed())
	r.Equal(common.HexToHash("0x0d0e0a0d"), res.Inference.CanonicalOutputHash)

	// PoT join is assembled: paymentHash == settlement, outputHash == receipt.
	r.Equal(res.Settlement.PaymentHash, res.PoT.PaymentHash)
	r.Equal(res.Inference.CanonicalOutputHash, res.PoT.OutputHash)
	r.Equal(model.Hash(), res.PoT.ModelID)
	r.Equal(PromptHash(prompt), res.PoT.PromptHash)
	r.Equal(res.Inference.Hash(), res.PoT.QuorumProof)
	r.NotEqual(common.Hash{}, res.PoT.ReceiptID())

	// VerifyPaidReceipt proves the join.
	ok, err := c.VerifyPaidReceipt(context.Background(), res.PoT, res.Settlement)
	r.NoError(err)
	r.True(ok)
}

func TestInferPaidSettleFailsBeforeInference(t *testing.T) {
	r := require.New(t)
	key, err := crypto.HexToECDSA(x402SignerKey)
	r.NoError(err)
	b := newMockBackend()
	c := fastClient(t, b, WithReceiptStore(&anyIDStore{}))
	// Clock AT validUntil -> facilitator rejects as expired; inference never runs.
	reqs := x402Fixture()
	fac := &MockFacilitator{Now: func() time.Time { return time.Unix(int64(reqs.ValidUntil), 0) }}

	_, err = c.InferPaid(context.Background(), ModelSpec{Name: "m", Version: 1, WeightCommit: common.HexToHash("0x1")},
		[]byte("x"), 3, 2, reqs, fac, key)
	r.Error(err)
	r.Len(b.sent, 0, "no inference tx broadcast when settlement fails")

	// nil facilitator rejected.
	_, err = c.InferPaid(context.Background(), ModelSpec{WeightCommit: common.HexToHash("0x1")},
		[]byte("x"), 3, 2, reqs, nil, key)
	r.Error(err)
}

func TestVerifyPaidReceiptMatchesAndRejects(t *testing.T) {
	r := require.New(t)
	c := fastClient(t, newMockBackend())
	ph := common.HexToHash(GoldenPaymentHash)

	pot := PoTReceipt{PaymentHash: ph, ModelID: common.HexToHash("0x44")}
	st := SettlementReceipt{Settled: true, PaymentHash: ph}

	// Match.
	ok, err := c.VerifyPaidReceipt(context.Background(), pot, st)
	r.NoError(err)
	r.True(ok)

	// Mismatched paymentHash -> false (no error; it is a clean "not paid by this").
	stWrong := SettlementReceipt{Settled: true, PaymentHash: common.HexToHash("0xdeadbeef")}
	ok, err = c.VerifyPaidReceipt(context.Background(), pot, stWrong)
	r.NoError(err)
	r.False(ok)

	// Unsettled -> error.
	_, err = c.VerifyPaidReceipt(context.Background(), pot, SettlementReceipt{Settled: false, PaymentHash: ph})
	r.Error(err)

	// Zero paymentHash -> error.
	_, err = c.VerifyPaidReceipt(context.Background(), PoTReceipt{}, SettlementReceipt{Settled: true})
	r.Error(err)
}

func TestGenerateWithBudgetAllowsWithin(t *testing.T) {
	r := require.New(t)
	key, err := crypto.HexToECDSA(x402SignerKey)
	r.NoError(err)
	b := newMockBackend()
	model := ModelSpec{Name: "zenlm/zen-omni", Version: 3, WeightCommit: common.HexToHash("0xabc"), Quantization: "int8"}
	prompt := []byte("hello")
	store := &anyIDStore{rec: InferenceReceipt{
		Status:              StatusCompleted,
		CanonicalOutputHash: common.HexToHash("0x0d"),
		ModelSpecHash:       model.Hash(),
		PromptHash:          PromptHash(prompt),
	}}
	c := fastClient(t, b, WithReceiptStore(store))
	fac := &MockFacilitator{Now: func() time.Time { return time.Unix(1_000_000_000, 0) }}
	reqs := x402Fixture() // MaxAmount 1_000_000

	guard := NewBudgetGuard(BudgetPolicy{
		MaxPerRequest: big.NewInt(2_000_000),
		MaxPerHour:    big.NewInt(3_000_000),
		AllowedModels: []common.Hash{model.Hash()},
		RequirePoT:    true,
	}, func() time.Time { return time.Unix(1_000_000_000, 0) })

	// First request (1M) within both caps -> ok.
	res, err := c.GenerateWithBudget(context.Background(), model, prompt, 5, 3, reqs, guard, fac, key)
	r.NoError(err)
	r.True(res.Settlement.Settled)
	r.True(res.Inference.Completed())

	// Second request (1M, total 2M) still within hourly cap (3M) -> ok.
	_, err = c.GenerateWithBudget(context.Background(), model, prompt, 5, 3, reqs, guard, fac, key)
	r.NoError(err)

	// Third request (1M, total 3M) hits exactly the cap -> ok (cap is inclusive).
	_, err = c.GenerateWithBudget(context.Background(), model, prompt, 5, 3, reqs, guard, fac, key)
	r.NoError(err)

	// Fourth request would push to 4M > 3M hourly cap -> rejected, no tx.
	before := len(b.sent)
	_, err = c.GenerateWithBudget(context.Background(), model, prompt, 5, 3, reqs, guard, fac, key)
	r.Error(err)
	r.Len(b.sent, before, "over-hourly-budget request must not broadcast inference")
}

func TestGenerateWithBudgetRejectsOverPerRequest(t *testing.T) {
	r := require.New(t)
	key, err := crypto.HexToECDSA(x402SignerKey)
	r.NoError(err)
	b := newMockBackend()
	model := ModelSpec{Name: "m", Version: 1, WeightCommit: common.HexToHash("0xabc")}
	c := fastClient(t, b, WithReceiptStore(&anyIDStore{}))
	fac := &MockFacilitator{Now: func() time.Time { return time.Unix(1_000_000_000, 0) }}
	reqs := x402Fixture() // MaxAmount 1_000_000

	// Per-request cap below the request amount -> rejected BEFORE paying.
	guard := NewBudgetGuard(BudgetPolicy{MaxPerRequest: big.NewInt(500_000)}, nil)
	_, err = c.GenerateWithBudget(context.Background(), model, []byte("x"), 3, 2, reqs, guard, fac, key)
	r.Error(err)
	r.Len(b.sent, 0, "over-per-request budget must not pay or infer")
}

func TestGenerateWithBudgetRejectsDisallowedModel(t *testing.T) {
	r := require.New(t)
	key, err := crypto.HexToECDSA(x402SignerKey)
	r.NoError(err)
	b := newMockBackend()
	allowed := ModelSpec{Name: "allowed", Version: 1, WeightCommit: common.HexToHash("0xa11")}
	other := ModelSpec{Name: "other", Version: 1, WeightCommit: common.HexToHash("0x07")}
	c := fastClient(t, b, WithReceiptStore(&anyIDStore{}))
	fac := &MockFacilitator{Now: func() time.Time { return time.Unix(1_000_000_000, 0) }}
	reqs := x402Fixture()

	guard := NewBudgetGuard(BudgetPolicy{AllowedModels: []common.Hash{allowed.Hash()}}, nil)
	_, err = c.GenerateWithBudget(context.Background(), other, []byte("x"), 3, 2, reqs, guard, fac, key)
	r.Error(err, "model not in allow-list must be rejected")
	r.Len(b.sent, 0)

	// nil guard rejected.
	_, err = c.GenerateWithBudget(context.Background(), allowed, []byte("x"), 3, 2, reqs, nil, fac, key)
	r.Error(err)
}

func TestSpendTrackerRollingWindow(t *testing.T) {
	r := require.New(t)
	now := time.Unix(1_000_000_000, 0)
	clock := func() time.Time { return now }
	st := newSpendTracker(time.Hour, clock)
	cap := big.NewInt(1000)

	r.NoError(st.reserve(big.NewInt(600), cap)) // 600
	r.NoError(st.reserve(big.NewInt(400), cap)) // 1000 (== cap, allowed)
	r.Error(st.reserve(big.NewInt(1), cap))     // 1001 > cap

	// Advance past the window: old spend ages out, capacity returns.
	now = now.Add(time.Hour + time.Second)
	r.NoError(st.reserve(big.NewInt(1000), cap))
}
