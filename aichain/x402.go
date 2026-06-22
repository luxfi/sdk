// Copyright (C) 2026, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package aichain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/common"
)

// x402.go is the x402 (HTTP 402 "Payment Required") machine-payment layer for
// pay-per-cognition. It sits ALONGSIDE the inference client (client.go) and joins
// an off-chain x402 settlement to an on-chain Proof-of-Thought (PoT) receipt:
//
//	x402 flow:  client wants inference -> a resource server answers 402 with
//	            PaymentRequirements -> client returns a SIGNED PaymentPayload (the
//	            X-PAYMENT header) -> a Facilitator verifies + settles it -> the
//	            server serves the result + a SettlementReceipt (X-PAYMENT-RESPONSE).
//
// The load-bearing value is paymentHash: the EIP-712 digest of the signed
// authorization (see BuildPaymentPayload). It is the SAME value the on-chain
// ProofOfThoughtRegistry stores as ThoughtReceipt.paymentHash, so a paid x402
// inference can be registered on-chain and later proven payment<->thought via
// VerifyPaidReceipt / ComputeReceiptId.
//
// The SDK ships NO live HTTP transport (the x402 wire is an HTTP concern outside
// this on-chain SDK); callers wire their own Facilitator (a real one calls the
// facilitator service), and tests use MockFacilitator. This keeps the SDK
// decoupled from any single x402 facilitator while still offering the one-call
// InferPaid convenience and the budget gate.

// ---------------------------------------------------------------------------
// EIP-712 domain + type hashes (the ONLY place the x402 signing preimage lives)
// ---------------------------------------------------------------------------

// The x402 authorization is signed EIP-712-style. The domain is x402-specific (NOT
// the EVM tx domain) so an x402 signature can never be replayed as a transaction
// and vice-versa. We bind name+version+chainId so a payment authorized for one
// network cannot be settled on another.
//
// EIP-712 digest:
//
//	paymentHash = keccak256( 0x19 || 0x01 || domainSeparator || hashStruct(msg) )
//
//	domainSeparator = keccak256(
//	    EIP712_DOMAIN_TYPEHASH ||
//	    keccak256("x402PaymentAuthorization") ||      // name
//	    keccak256("1") ||                             // version
//	    u256(chainId) )
//
//	hashStruct(msg) = keccak256(
//	    PAYMENT_AUTH_TYPEHASH ||
//	    from(32) || to(32) || value(32) || token(32) ||
//	    u256(validAfter) || u256(validBefore) || nonce(32) )
//
// where addresses are left-padded to 32 bytes (EIP-712 encodeData), value/times
// are uint256, and nonce/token/from/to are encoded as their 32-byte words. This
// is the EXACT preimage the TS side reproduces (golden-vector parity test).
const (
	eip712DomainType = "EIP712Domain(string name,string version,uint256 chainId)"
	paymentAuthType  = "PaymentAuthorization(address from,address to,uint256 value,address token,uint256 validAfter,uint256 validBefore,bytes32 nonce)"

	x402DomainName    = "x402PaymentAuthorization"
	x402DomainVersion = "1"
)

// eip712Prefix is the 2-byte \x19\x01 EIP-712 prefix.
var eip712Prefix = []byte{0x19, 0x01}

// ---------------------------------------------------------------------------
// x402 wire types
// ---------------------------------------------------------------------------

// X402Scheme is the x402 payment scheme. `exact` is a fixed per-request charge;
// `upto` caps a charge that the facilitator may settle for less (e.g. metered by
// actual tokens). The signed authorization is identical; the scheme tells the
// facilitator how to interpret MaxAmount vs the settled amount.
type X402Scheme string

const (
	// SchemeExact: settle exactly MaxAmount.
	SchemeExact X402Scheme = "exact"
	// SchemeUpto: settle at most MaxAmount (the facilitator may charge less).
	SchemeUpto X402Scheme = "upto"
)

// PaymentRequirements is what a resource server returns in its HTTP 402 response:
// what it costs, in what token, on what network, to whom, and until when. The
// client turns this into a signed PaymentPayload.
type PaymentRequirements struct {
	Scheme      X402Scheme     `json:"scheme"`
	Network     *big.Int       `json:"network"`     // EVM chain id the token/settlement lives on
	MaxAmount   *big.Int       `json:"maxAmount"`   // amount (exact) or cap (upto), in token base units
	Token       common.Address `json:"token"`       // ERC-20 payment token (zero == native)
	PayTo       common.Address `json:"payTo"`       // payee (resource server's receiving address)
	Resource    string         `json:"resource"`    // the resource being paid for (e.g. model id / URL)
	Description string         `json:"description"` // human-readable description
	Nonce       common.Hash    `json:"nonce"`       // server-chosen unique nonce (replay protection)
	ValidUntil  uint64         `json:"validUntil"`  // unix seconds the authorization is valid until (validBefore)
}

func (r PaymentRequirements) validate() error {
	if r.Scheme != SchemeExact && r.Scheme != SchemeUpto {
		return fmt.Errorf("aichain/x402: unknown scheme %q", r.Scheme)
	}
	if r.Network == nil || r.Network.Sign() <= 0 {
		return errors.New("aichain/x402: missing/zero network (chain id)")
	}
	if r.MaxAmount == nil || r.MaxAmount.Sign() <= 0 {
		return errors.New("aichain/x402: missing/zero maxAmount")
	}
	if r.PayTo == (common.Address{}) {
		return errors.New("aichain/x402: zero payTo")
	}
	if r.Nonce == (common.Hash{}) {
		return errors.New("aichain/x402: zero nonce")
	}
	if r.ValidUntil == 0 {
		return errors.New("aichain/x402: zero validUntil")
	}
	return nil
}

// PaymentPayload is the signed authorization the client sends in the X-PAYMENT
// header. It is an EIP-3009-shaped transferWithAuthorization payload: a signature
// over (from, to, value, validAfter, validBefore, nonce) bound to (token, chain
// id) via the EIP-712 domain. The facilitator recovers `from` from the signature
// and settles the transfer.
type PaymentPayload struct {
	Scheme      X402Scheme     `json:"scheme"`
	Network     *big.Int       `json:"network"`
	From        common.Address `json:"from"`        // payer (recovered-equals the signer)
	To          common.Address `json:"to"`          // payee (== requirements.PayTo)
	Value       *big.Int       `json:"value"`       // amount authorized (== requirements.MaxAmount)
	Token       common.Address `json:"token"`       // payment token
	ValidAfter  uint64         `json:"validAfter"`  // unix seconds the auth becomes valid (0 == immediately)
	ValidBefore uint64         `json:"validBefore"` // unix seconds the auth expires (== requirements.ValidUntil)
	Nonce       common.Hash    `json:"nonce"`       // == requirements.Nonce
	Signature   []byte         `json:"signature"`   // 65-byte secp256k1 [R||S||V]
}

// SettlementReceipt is what the facilitator returns after Verify+Settle — the
// X-PAYMENT-RESPONSE the resource server echoes. PaymentHash is the EIP-712 digest
// of the authorization (the x402<->PoT join key); TxHash is the on-chain
// settlement tx (zero for an off-chain/escrowed settlement).
type SettlementReceipt struct {
	PaymentHash common.Hash    `json:"paymentHash"` // EIP-712 digest of the payload (== PoT paymentHash)
	TxHash      common.Hash    `json:"txHash"`      // settlement tx (zero if off-chain)
	Settled     bool           `json:"settled"`
	Payer       common.Address `json:"payer"`
	Payee       common.Address `json:"payee"`
	Amount      *big.Int       `json:"amount"` // amount actually settled (<= payload.Value for `upto`)
}

// ---------------------------------------------------------------------------
// EIP-712 digest (paymentHash)
// ---------------------------------------------------------------------------

// domainSeparator computes the EIP-712 domain separator for the x402 domain bound
// to chainID.
func x402DomainSeparator(chainID *big.Int) common.Hash {
	return common.BytesToHash(crypto.Keccak256(
		crypto.Keccak256([]byte(eip712DomainType)),
		crypto.Keccak256([]byte(x402DomainName)),
		crypto.Keccak256([]byte(x402DomainVersion)),
		u256be(chainID),
	))
}

// hashStruct computes the EIP-712 hashStruct of the PaymentAuthorization message.
func (p PaymentPayload) hashStruct() common.Hash {
	return common.BytesToHash(crypto.Keccak256(
		crypto.Keccak256([]byte(paymentAuthType)),
		word32(p.From.Bytes()),
		word32(p.To.Bytes()),
		u256be(p.Value),
		word32(p.Token.Bytes()),
		u256be(new(big.Int).SetUint64(p.ValidAfter)),
		u256be(new(big.Int).SetUint64(p.ValidBefore)),
		p.Nonce.Bytes(),
	))
}

// PaymentHash is the EIP-712 signing digest of the authorization:
// keccak256(0x1901 || domainSeparator(network) || hashStruct(msg)). This is the
// value the signature is over AND the value stored on-chain as
// ThoughtReceipt.paymentHash, so it is the canonical x402<->PoT join key.
func (p PaymentPayload) PaymentHash() common.Hash {
	return common.BytesToHash(crypto.Keccak256(
		eip712Prefix,
		x402DomainSeparator(p.Network).Bytes(),
		p.hashStruct().Bytes(),
	))
}

// ---------------------------------------------------------------------------
// build + sign
// ---------------------------------------------------------------------------

// BuildPaymentPayload constructs the x402 authorization from a server's
// PaymentRequirements and EIP-712-signs it with `signer`. The payload binds
// from=signer-address, to=PayTo, value=MaxAmount, token, validBefore=ValidUntil,
// nonce, on the requirements' network. The returned payload's PaymentHash() is
// the EIP-712 digest the signature is over (and the PoT join key). Deterministic:
// the same (signer, requirements) always yields the same payload + paymentHash.
func BuildPaymentPayload(signer *ecdsa.PrivateKey, reqs PaymentRequirements) (PaymentPayload, error) {
	if signer == nil {
		return PaymentPayload{}, errors.New("aichain/x402: nil signer")
	}
	if err := reqs.validate(); err != nil {
		return PaymentPayload{}, err
	}
	from := common.BytesToAddress(crypto.PubkeyToAddress(signer.PublicKey).Bytes())
	p := PaymentPayload{
		Scheme:      reqs.Scheme,
		Network:     new(big.Int).Set(reqs.Network),
		From:        from,
		To:          reqs.PayTo,
		Value:       new(big.Int).Set(reqs.MaxAmount),
		Token:       reqs.Token,
		ValidAfter:  0,
		ValidBefore: reqs.ValidUntil,
		Nonce:       reqs.Nonce,
	}
	sig, err := crypto.Sign(p.PaymentHash().Bytes(), signer)
	if err != nil {
		return PaymentPayload{}, fmt.Errorf("aichain/x402: sign authorization: %w", err)
	}
	p.Signature = sig
	return p, nil
}

// RecoverPayer recovers the address that signed the authorization from the
// signature over PaymentHash(). It is the facilitator's check that `from` really
// authorized the payment. Errors on a malformed signature or a recovered address
// that does not equal payload.From.
func (p PaymentPayload) RecoverPayer() (common.Address, error) {
	if len(p.Signature) != 65 {
		return common.Address{}, fmt.Errorf("aichain/x402: signature %d bytes, want 65", len(p.Signature))
	}
	pub, err := crypto.SigToPub(p.PaymentHash().Bytes(), p.Signature)
	if err != nil {
		return common.Address{}, fmt.Errorf("aichain/x402: recover: %w", err)
	}
	got := common.BytesToAddress(crypto.PubkeyToAddress(*pub).Bytes())
	if got != p.From {
		return common.Address{}, fmt.Errorf("aichain/x402: signer %s != payload.from %s", got.Hex(), p.From.Hex())
	}
	return got, nil
}

// bindsTo reports whether the payload faithfully encodes the requirements (the
// facilitator's structural check before settling). It does NOT check the
// signature (RecoverPayer does that) or time validity.
func (p PaymentPayload) bindsTo(reqs PaymentRequirements) error {
	if p.Scheme != reqs.Scheme {
		return fmt.Errorf("aichain/x402: payload scheme %q != required %q", p.Scheme, reqs.Scheme)
	}
	if p.Network == nil || reqs.Network == nil || p.Network.Cmp(reqs.Network) != 0 {
		return errors.New("aichain/x402: payload network != required network")
	}
	if p.To != reqs.PayTo {
		return fmt.Errorf("aichain/x402: payload to %s != payTo %s", p.To.Hex(), reqs.PayTo.Hex())
	}
	if p.Token != reqs.Token {
		return errors.New("aichain/x402: payload token != required token")
	}
	if p.Nonce != reqs.Nonce {
		return errors.New("aichain/x402: payload nonce != required nonce")
	}
	if p.ValidBefore != reqs.ValidUntil {
		return errors.New("aichain/x402: payload validBefore != required validUntil")
	}
	// `exact` must authorize exactly MaxAmount; `upto` must authorize at most it.
	switch reqs.Scheme {
	case SchemeExact:
		if p.Value == nil || p.Value.Cmp(reqs.MaxAmount) != 0 {
			return errors.New("aichain/x402: exact scheme requires value == maxAmount")
		}
	case SchemeUpto:
		if p.Value == nil || p.Value.Sign() <= 0 || p.Value.Cmp(reqs.MaxAmount) > 0 {
			return errors.New("aichain/x402: upto scheme requires 0 < value <= maxAmount")
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Facilitator
// ---------------------------------------------------------------------------

// Facilitator is the x402 verify+settle service. A real implementation calls the
// facilitator HTTP endpoint (or settles on-chain itself); MockFacilitator settles
// deterministically in-process for tests. Verify is the cheap pre-check (valid
// signature, well-formed, binds the requirements, not expired); Settle performs
// the transfer and returns the SettlementReceipt carrying the paymentHash.
type Facilitator interface {
	Verify(payload PaymentPayload, reqs PaymentRequirements) error
	Settle(payload PaymentPayload, reqs PaymentRequirements) (SettlementReceipt, error)
}

// MockFacilitator is an in-process Facilitator for tests + local dev. It runs the
// real verification (signature recovery, binding, expiry) so tests exercise the
// genuine path, then "settles" by returning a deterministic receipt (no chain).
// Now is injectable for deterministic expiry tests (zero == time.Now).
type MockFacilitator struct {
	Now func() time.Time
}

// Verify runs the genuine x402 verification: the payload binds the requirements,
// the signature recovers to payload.From, and the authorization is not expired.
func (m *MockFacilitator) Verify(payload PaymentPayload, reqs PaymentRequirements) error {
	if err := reqs.validate(); err != nil {
		return err
	}
	if err := payload.bindsTo(reqs); err != nil {
		return err
	}
	if _, err := payload.RecoverPayer(); err != nil {
		return err
	}
	now := uint64(m.now().Unix())
	if payload.ValidAfter != 0 && now < payload.ValidAfter {
		return fmt.Errorf("aichain/x402: authorization not yet valid (validAfter=%d now=%d)", payload.ValidAfter, now)
	}
	if payload.ValidBefore != 0 && now >= payload.ValidBefore {
		return fmt.Errorf("aichain/x402: authorization expired (validBefore=%d now=%d)", payload.ValidBefore, now)
	}
	return nil
}

// Settle verifies then returns a deterministic settlement receipt. The settled
// amount equals payload.Value (the mock charges the full authorized amount even
// for `upto`). PaymentHash is the EIP-712 digest (the PoT join key); TxHash is
// derived deterministically from the paymentHash so a settlement is reproducible
// in tests (a real facilitator returns the actual on-chain tx hash).
func (m *MockFacilitator) Settle(payload PaymentPayload, reqs PaymentRequirements) (SettlementReceipt, error) {
	if err := m.Verify(payload, reqs); err != nil {
		return SettlementReceipt{}, err
	}
	payer, err := payload.RecoverPayer()
	if err != nil {
		return SettlementReceipt{}, err
	}
	ph := payload.PaymentHash()
	return SettlementReceipt{
		PaymentHash: ph,
		TxHash:      common.BytesToHash(crypto.Keccak256([]byte("x402/mock-settle"), ph.Bytes())),
		Settled:     true,
		Payer:       payer,
		Payee:       payload.To,
		Amount:      new(big.Int).Set(payload.Value),
	}, nil
}

func (m *MockFacilitator) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

// Compile-time: MockFacilitator satisfies Facilitator.
var _ Facilitator = (*MockFacilitator)(nil)

// ---------------------------------------------------------------------------
// Proof-of-Thought receiptId (matches ProofOfThoughtRegistry.computeReceiptId)
// ---------------------------------------------------------------------------

// ComputeReceiptId reproduces the on-chain ProofOfThoughtRegistry.computeReceiptId
// for a paid thought:
//
//	keccak256( abi.encode(modelId, promptHash, outputHash, paymentHash, payer, operator) )
//
// abi.encode of (bytes32,bytes32,bytes32,bytes32,address,address) is six 32-byte
// words (the two addresses left-padded to 32 bytes). This is the deterministic id
// under which the receipt is registered on-chain; the SDK computes it so a paid
// inference can be located/verified on-chain without a round-trip.
func ComputeReceiptId(modelID, promptHash, outputHash, paymentHash common.Hash, payer, operator common.Address) common.Hash {
	buf := make([]byte, 0, 6*32)
	buf = append(buf, modelID.Bytes()...)
	buf = append(buf, promptHash.Bytes()...)
	buf = append(buf, outputHash.Bytes()...)
	buf = append(buf, paymentHash.Bytes()...)
	buf = append(buf, word32(payer.Bytes())...)
	buf = append(buf, word32(operator.Bytes())...)
	return common.BytesToHash(crypto.Keccak256(buf))
}

// ---------------------------------------------------------------------------
// PoT-linkable inference receipt
// ---------------------------------------------------------------------------

// PoTReceipt is the SDK's view of a Proof-of-Thought record: the join of a paid
// x402 settlement (PaymentHash), a model output (OutputHash), and the producing
// proof (QuorumProof), keyed by the on-chain ReceiptId. It is what a caller
// registers via ProofOfThoughtRegistry.register and what VerifyPaidReceipt checks
// against a SettlementReceipt.
type PoTReceipt struct {
	ModelID     common.Hash    `json:"modelId"`
	PromptHash  common.Hash    `json:"promptHash"`
	OutputHash  common.Hash    `json:"outputHash"`
	PaymentHash common.Hash    `json:"paymentHash"`
	QuorumProof common.Hash    `json:"quorumProof"`
	Payer       common.Address `json:"payer"`
	Operator    common.Address `json:"operator"`
	Cost        *big.Int       `json:"cost"`
}

// ReceiptID is the deterministic on-chain id for this PoT record (==
// ComputeReceiptId over its fields).
func (r PoTReceipt) ReceiptID() common.Hash {
	return ComputeReceiptId(r.ModelID, r.PromptHash, r.OutputHash, r.PaymentHash, r.Payer, r.Operator)
}

// ---------------------------------------------------------------------------
// Client x402 methods
// ---------------------------------------------------------------------------

// PaidResult is the outcome of an InferPaid call: the inference result joined to
// its x402 settlement and the PoT record that binds them on-chain.
type PaidResult struct {
	IntentID   common.Hash
	TxHash     common.Hash
	Inference  InferenceReceipt
	Settlement SettlementReceipt
	PoT        PoTReceipt // ready to register on ProofOfThoughtRegistry (PoT.ReceiptID())
}

// InferPaid runs the full x402 pay-per-cognition flow for a Tier-2 inference:
//
//  1. build + sign the x402 PaymentPayload from `reqs` with `signer`;
//  2. have the facilitator Settle it (verify signature, charge) -> SettlementReceipt;
//  3. run the quorum inference (Infer) for (model, prompt);
//  4. assemble the PoT join (paymentHash from the settlement, outputHash from the
//     receipt) so the result can be registered on-chain.
//
// Payment happens BEFORE inference (x402 is pay-first). The returned
// Settlement.PaymentHash is the on/off-chain join key; PoT.ReceiptID() is the id
// the result registers under on ProofOfThoughtRegistry. Errors from settle abort
// before any inference is requested.
func (c *Client) InferPaid(
	ctx context.Context,
	model ModelSpec,
	prompt []byte,
	n, threshold uint16,
	reqs PaymentRequirements,
	facilitator Facilitator,
	signer *ecdsa.PrivateKey,
) (PaidResult, error) {
	if facilitator == nil {
		return PaidResult{}, errors.New("aichain/x402: nil facilitator")
	}
	payload, err := BuildPaymentPayload(signer, reqs)
	if err != nil {
		return PaidResult{}, err
	}
	settlement, err := facilitator.Settle(payload, reqs)
	if err != nil {
		return PaidResult{}, fmt.Errorf("aichain/x402: settle: %w", err)
	}
	if !settlement.Settled {
		return PaidResult{}, errors.New("aichain/x402: facilitator returned unsettled receipt")
	}
	// Pay-first: only now request the cognition. reqs.MaxAmount funds the A escrow.
	res, err := c.Infer(ctx, model, prompt, n, threshold, reqs.MaxAmount)
	if err != nil {
		return PaidResult{IntentID: res.IntentID, TxHash: res.TxHash, Settlement: settlement}, err
	}
	pot := PoTReceipt{
		ModelID:     res.Receipt.ModelSpecHash,
		PromptHash:  res.Receipt.PromptHash,
		OutputHash:  res.Receipt.CanonicalOutputHash,
		PaymentHash: settlement.PaymentHash,
		QuorumProof: res.Receipt.Hash(), // the settled receipt's commitment is the Tier-2 quorum proof
		Payer:       settlement.Payer,
		Operator:    common.Address{}, // operator (quorum aggregate) is filled by the registrar; 0 here
		Cost:        settlement.Amount,
	}
	return PaidResult{
		IntentID:   res.IntentID,
		TxHash:     res.TxHash,
		Inference:  res.Receipt,
		Settlement: settlement,
		PoT:        pot,
	}, nil
}

// VerifyPaidReceipt proves the payment<->thought join: it checks that the PoT
// receipt's PaymentHash equals the settlement's PaymentHash (the value the
// signature was over and that the on-chain registry stores). A true result means
// "this recorded thought was paid for by exactly this settlement". It also
// guards against a zero paymentHash (which the on-chain registry rejects).
func (c *Client) VerifyPaidReceipt(_ context.Context, pot PoTReceipt, settlement SettlementReceipt) (bool, error) {
	if !settlement.Settled {
		return false, errors.New("aichain/x402: settlement is not settled")
	}
	if pot.PaymentHash == (common.Hash{}) || settlement.PaymentHash == (common.Hash{}) {
		return false, errors.New("aichain/x402: zero paymentHash")
	}
	if pot.PaymentHash != settlement.PaymentHash {
		return false, nil
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// budget gate
// ---------------------------------------------------------------------------

// BudgetPolicy is a CLIENT-SIDE spend guard checked BEFORE any payment is made.
// It caps the per-request amount and the rolling per-hour spend, optionally
// restricts which models may be paid for, and can require that a PoT receipt be
// produced. It is advisory (the client's own discipline), independent of and
// complementary to the on-chain escrow.
type BudgetPolicy struct {
	// MaxPerRequest caps a single request's amount (nil/zero == no per-request cap).
	MaxPerRequest *big.Int
	// MaxPerHour caps the rolling 1-hour spend (nil/zero == no hourly cap).
	MaxPerHour *big.Int
	// AllowedModels, if non-empty, whitelists model spec hashes that may be paid
	// for (a request for any other model is rejected before paying).
	AllowedModels []common.Hash
	// RequirePoT, when true, requires the paid result to carry a non-zero PoT
	// paymentHash<->settlement join (InferPaid always produces one; this asserts it).
	RequirePoT bool
}

func (b *BudgetPolicy) allows(model ModelSpec) error {
	if len(b.AllowedModels) == 0 {
		return nil
	}
	h := model.Hash()
	for _, m := range b.AllowedModels {
		if m == h {
			return nil
		}
	}
	return fmt.Errorf("aichain/x402: model %s not in budget allow-list", h.Hex())
}

// spendTracker is an in-memory rolling-window spend ledger enforcing the hourly
// cap. It is safe for concurrent use. Spends older than the window are pruned on
// each check, so memory stays bounded by the request rate within one hour.
type spendTracker struct {
	mu     sync.Mutex
	window time.Duration
	now    func() time.Time
	events []spendEvent
}

type spendEvent struct {
	at     time.Time
	amount *big.Int
}

func newSpendTracker(window time.Duration, now func() time.Time) *spendTracker {
	if now == nil {
		now = time.Now
	}
	return &spendTracker{window: window, now: now}
}

// spentInWindow returns the total spend within the trailing window (pruning older
// events as a side effect).
func (s *spendTracker) spentInWindow() *big.Int {
	cutoff := s.now().Add(-s.window)
	kept := s.events[:0]
	sum := new(big.Int)
	for _, e := range s.events {
		if e.at.After(cutoff) {
			kept = append(kept, e)
			sum.Add(sum, e.amount)
		}
	}
	s.events = kept
	return sum
}

// reserve checks that recording `amount` would not exceed `cap` within the
// window, and if so records it. A nil/zero cap means unlimited. It is atomic
// (check+record under the lock) so concurrent requests cannot both slip past the
// cap.
func (s *spendTracker) reserve(amount, cap *big.Int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cap != nil && cap.Sign() > 0 {
		projected := new(big.Int).Add(s.spentInWindow(), amount)
		if projected.Cmp(cap) > 0 {
			return fmt.Errorf("aichain/x402: hourly budget exceeded (would spend %s, cap %s)", projected, cap)
		}
	}
	s.events = append(s.events, spendEvent{at: s.now(), amount: new(big.Int).Set(amount)})
	return nil
}

// BudgetGuard enforces a BudgetPolicy across many GenerateWithBudget calls on one
// Client, holding the rolling hourly spend. Construct with NewBudgetGuard; it is
// safe for concurrent use. The guard is SEPARATE from the Client so the same
// client can serve different budgets (one and only one spend ledger per policy).
type BudgetGuard struct {
	policy  BudgetPolicy
	tracker *spendTracker
}

// NewBudgetGuard builds a guard for a policy. `now` is injectable for tests (nil
// == time.Now). The hourly window is fixed at one hour (the policy's unit).
func NewBudgetGuard(policy BudgetPolicy, now func() time.Time) *BudgetGuard {
	return &BudgetGuard{policy: policy, tracker: newSpendTracker(time.Hour, now)}
}

// check enforces the per-request + per-model caps and reserves the hourly spend.
// It is called BEFORE paying; on success the amount is already recorded against
// the hourly cap, so an over-budget request never reaches the facilitator.
func (g *BudgetGuard) check(model ModelSpec, amount *big.Int) error {
	if amount == nil || amount.Sign() <= 0 {
		return errors.New("aichain/x402: non-positive request amount")
	}
	if err := g.policy.allows(model); err != nil {
		return err
	}
	if mp := g.policy.MaxPerRequest; mp != nil && mp.Sign() > 0 && amount.Cmp(mp) > 0 {
		return fmt.Errorf("aichain/x402: request amount %s exceeds per-request cap %s", amount, mp)
	}
	return g.tracker.reserve(amount, g.policy.MaxPerHour)
}

// GenerateWithBudget is the budget-gated InferPaid: it enforces `guard`'s policy
// (per-request cap, hourly cap, model allow-list) BEFORE building or settling any
// payment, then runs the full paid inference. An over-budget or disallowed
// request is rejected with NO payment made. If the policy RequirePoT, the result
// is checked to carry a valid PoT join before returning.
//
// This is the Tier-2 entrypoint agents use to pay-per-cognition safely: the guard
// is the agent's spending discipline, the facilitator settles, and the PoT join
// makes every paid thought accountable on-chain.
func (c *Client) GenerateWithBudget(
	ctx context.Context,
	model ModelSpec,
	prompt []byte,
	n, threshold uint16,
	reqs PaymentRequirements,
	guard *BudgetGuard,
	facilitator Facilitator,
	signer *ecdsa.PrivateKey,
) (PaidResult, error) {
	if guard == nil {
		return PaidResult{}, errors.New("aichain/x402: nil budget guard")
	}
	if err := reqs.validate(); err != nil {
		return PaidResult{}, err
	}
	// Gate on the amount the requirements will actually charge BEFORE paying.
	if err := guard.check(model, reqs.MaxAmount); err != nil {
		return PaidResult{}, err
	}
	res, err := c.InferPaid(ctx, model, prompt, n, threshold, reqs, facilitator, signer)
	if err != nil {
		return res, err
	}
	if guard.policy.RequirePoT {
		ok, verr := c.VerifyPaidReceipt(ctx, res.PoT, res.Settlement)
		if verr != nil {
			return res, fmt.Errorf("aichain/x402: PoT required but verify failed: %w", verr)
		}
		if !ok {
			return res, errors.New("aichain/x402: PoT required but paymentHash join did not verify")
		}
	}
	return res, nil
}
