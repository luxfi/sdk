// Copyright (C) 2026, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package aichain

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	ethereum "github.com/luxfi/geth"
	"github.com/luxfi/geth/common"
	gethtypes "github.com/luxfi/geth/core/types"
	"github.com/stretchr/testify/require"
)

// --- mock EVM backend (no live chain) ----------------------------------------

type mockBackend struct {
	chainID  *big.Int
	nonce    uint64
	tip      *big.Int
	baseFee  *big.Int
	gasEst   uint64
	callRet  []byte
	callErr  error
	sent     []*gethtypes.Transaction
	receipts map[common.Hash]*gethtypes.Receipt // by tx hash
	sendErr  error
	recAfter int // return receipt only after this many TransactionReceipt polls
	recPolls int
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		chainID:  big.NewInt(36911), // a Lux-style chain id
		nonce:    1,
		tip:      big.NewInt(1_000_000_000),
		baseFee:  big.NewInt(25_000_000_000),
		gasEst:   200_000,
		receipts: map[common.Hash]*gethtypes.Receipt{},
	}
}

func (m *mockBackend) ChainID(context.Context) (*big.Int, error) { return m.chainID, nil }
func (m *mockBackend) AcceptedNonceAt(context.Context, common.Address) (uint64, error) {
	return m.nonce, nil
}
func (m *mockBackend) SuggestGasTipCap(context.Context) (*big.Int, error) { return m.tip, nil }
func (m *mockBackend) EstimateBaseFee(context.Context) (*big.Int, error)  { return m.baseFee, nil }
func (m *mockBackend) EstimateGas(context.Context, ethereum.CallMsg) (uint64, error) {
	return m.gasEst, nil
}
func (m *mockBackend) SendTransaction(_ context.Context, tx *gethtypes.Transaction) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sent = append(m.sent, tx)
	// Auto-arrange a successful receipt for this tx unless recAfter delays it.
	m.receipts[tx.Hash()] = &gethtypes.Receipt{
		Status: gethtypes.ReceiptStatusSuccessful,
		TxHash: tx.Hash(),
	}
	return nil
}
func (m *mockBackend) TransactionReceipt(_ context.Context, h common.Hash) (*gethtypes.Receipt, error) {
	m.recPolls++
	if m.recPolls <= m.recAfter {
		return nil, errors.New("not yet")
	}
	r, ok := m.receipts[h]
	if !ok {
		return nil, errors.New("not found")
	}
	return r, nil
}
func (m *mockBackend) CallContract(_ context.Context, _ ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	return m.callRet, m.callErr
}

// --- fake receipt store -------------------------------------------------------

type fakeStore struct {
	aChain   common.Hash
	byIntent map[common.Hash]InferenceReceipt
	calls    int
	appearAt int // make the receipt visible only after this many lookups
}

func (f *fakeStore) AChainID() common.Hash { return f.aChain }
func (f *fakeStore) ReceiptByIntent(_ context.Context, id common.Hash) (InferenceReceipt, MerkleProof, bool, error) {
	f.calls++
	if f.calls <= f.appearAt {
		return InferenceReceipt{}, MerkleProof{}, false, nil
	}
	r, ok := f.byIntent[id]
	return r, MerkleProof{}, ok, nil
}

// A throwaway funded test key (well-known anvil key #0; never used on any real
// chain). Only its ability to sign is exercised.
const testKey = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

func fastClient(t *testing.T, b EVMBackend, opts ...Option) *Client {
	t.Helper()
	base := []Option{WithPolling(time.Millisecond, 2*time.Second)}
	c, err := NewClient(b, testKey, append(base, opts...)...)
	require.NoError(t, err)
	return c
}

func TestNewClientConstruction(t *testing.T) {
	r := require.New(t)

	// Read-only client (no key) is allowed.
	ro, err := NewClient(newMockBackend(), "")
	r.NoError(err)
	r.Equal(common.Address{}, ro.From())

	// Nil backend rejected.
	_, err = NewClient(nil, testKey)
	r.Error(err)

	// Bad key rejected.
	_, err = NewClient(newMockBackend(), "not-hex-zz")
	r.Error(err)

	// Signing client derives the expected address from the key.
	c := fastClient(t, newMockBackend())
	r.NotEqual(common.Address{}, c.From())
}

func TestGenerateDeterministic(t *testing.T) {
	r := require.New(t)
	b := newMockBackend()
	// Mock the inference precompile return: prompt+generated tokens, big-endian u32.
	b.callRet = EncodeGenerate(0, []uint32{1, 7, 13, 2, 4, 28}) // reuse encoder to build a token array body
	// strip the selector+nNew prefix EncodeGenerate added (we only want the token bytes as a return)
	b.callRet = b.callRet[8:]

	c := fastClient(t, b)
	out, err := c.GenerateDeterministic(context.Background(), 2, []uint32{1, 7, 13, 2})
	r.NoError(err)
	r.Equal([]uint32{1, 7, 13, 2, 4, 28}, out)

	// empty prompt rejected.
	_, err = c.GenerateDeterministic(context.Background(), 1, nil)
	r.Error(err)
}

func TestSubmitInferenceIntent(t *testing.T) {
	r := require.New(t)
	b := newMockBackend()
	store := &fakeStore{aChain: common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")}
	c := fastClient(t, b, WithReceiptStore(store))

	opts := SubmitOptions{
		ModelSpecHash: common.HexToHash("0x44"),
		PromptHash:    common.HexToHash("0x55"),
		N:             5,
		Threshold:     3,
		Fee:           big.NewInt(1_000_000),
	}
	id, txHash, err := c.SubmitInferenceIntent(context.Background(), opts)
	r.NoError(err)
	r.NotEqual(common.Hash{}, id)
	r.NotEqual(common.Hash{}, txHash)
	r.Len(b.sent, 1, "exactly one tx broadcast")
	r.Equal(AIBridgeAddress, *b.sent[0].To())

	// The id must equal a manual ComputeIntentID over the landed tx + opts +
	// configured chain ids (cChain = EVM chain id, aChain from the store).
	cChain := common.BigToHash(b.chainID)
	want := ComputeIntentID(cChain, store.aChain, txHash, 0, c.From(),
		opts.ModelSpecHash, opts.PromptHash, opts.N, opts.Threshold, opts.Fee)
	r.Equal(want, id, "derived intent id must match the pinned preimage")
}

func TestSubmitValidation(t *testing.T) {
	r := require.New(t)
	c := fastClient(t, newMockBackend())
	bad := []SubmitOptions{
		{PromptHash: common.HexToHash("0x55"), N: 5, Threshold: 3},                                                      // zero model
		{ModelSpecHash: common.HexToHash("0x44"), N: 5, Threshold: 3},                                                   // zero prompt
		{ModelSpecHash: common.HexToHash("0x44"), PromptHash: common.HexToHash("0x55"), N: 0, Threshold: 0},             // N=0
		{ModelSpecHash: common.HexToHash("0x44"), PromptHash: common.HexToHash("0x55"), N: 5, Threshold: 6},             // thr>N
		{ModelSpecHash: common.HexToHash("0x44"), PromptHash: common.HexToHash("0x55"), N: 5, Threshold: 2},             // below quorum floor (need 3)
		{ModelSpecHash: common.HexToHash("0x44"), PromptHash: common.HexToHash("0x55"), N: MaxFanout + 1, Threshold: 1}, // N over max
	}
	for i, o := range bad {
		_, _, err := c.SubmitInferenceIntent(context.Background(), o)
		r.Error(err, "case %d should fail validation", i)
	}
}

func TestWaitReceiptCompleted(t *testing.T) {
	r := require.New(t)
	b := newMockBackend()
	intentID := common.HexToHash("0xabc123")
	store := &fakeStore{
		appearAt: 2, // simulate a few polls before settlement
		byIntent: map[common.Hash]InferenceReceipt{
			intentID: {
				Status:              StatusCompleted,
				CanonicalOutputHash: common.HexToHash("0xfaceofa"),
				ModelSpecHash:       common.HexToHash("0x44"),
				PromptHash:          common.HexToHash("0x55"),
			},
		},
	}
	c := fastClient(t, b, WithReceiptStore(store))
	got, err := c.WaitReceipt(context.Background(), intentID)
	r.NoError(err)
	r.True(got.Completed())
	r.Equal(common.HexToHash("0xfaceofa"), got.CanonicalOutputHash)
	r.GreaterOrEqual(store.calls, 3)
}

func TestWaitReceiptFailedAndChallenged(t *testing.T) {
	r := require.New(t)
	for _, tc := range []struct {
		name   string
		status uint8
	}{{"failed", StatusFailed}, {"challenged", StatusChallenged}} {
		id := common.HexToHash("0x01")
		store := &fakeStore{byIntent: map[common.Hash]InferenceReceipt{id: {Status: tc.status}}}
		c := fastClient(t, newMockBackend(), WithReceiptStore(store))
		_, err := c.WaitReceipt(context.Background(), id)
		r.Error(err, tc.name)
	}
}

func TestWaitReceiptCompletedZeroOutputIsError(t *testing.T) {
	r := require.New(t)
	id := common.HexToHash("0x01")
	// Completed but zero output is NOT actionable.
	store := &fakeStore{byIntent: map[common.Hash]InferenceReceipt{id: {Status: StatusCompleted}}}
	c := fastClient(t, newMockBackend(), WithReceiptStore(store))
	_, err := c.WaitReceipt(context.Background(), id)
	r.Error(err)
}

func TestWaitReceiptTimeout(t *testing.T) {
	r := require.New(t)
	store := &fakeStore{appearAt: 1 << 30} // never appears
	c := fastClient(t, newMockBackend(), WithReceiptStore(store), WithPolling(time.Millisecond, 30*time.Millisecond))
	_, err := c.WaitReceipt(context.Background(), common.HexToHash("0xdead"))
	r.Error(err)
}

func TestWaitReceiptNeedsStore(t *testing.T) {
	r := require.New(t)
	c := fastClient(t, newMockBackend()) // no store
	_, err := c.WaitReceipt(context.Background(), common.HexToHash("0x1"))
	r.Error(err)
}

func TestInferEndToEnd(t *testing.T) {
	r := require.New(t)
	b := newMockBackend()
	model := ModelSpec{Name: "zenlm/zen-omni", Version: 3, WeightCommit: common.HexToHash("0xabc"), Quantization: "int8"}
	prompt := []byte("hello beluga")

	// The store must return a receipt bound to the SAME model/prompt the client
	// derives, keyed by the id the client will compute. Compute that id up front.
	cChain := common.BigToHash(b.chainID)
	aChain := common.HexToHash("0x2222")
	// We don't know txHash before send; capture it by pre-seeding the store with a
	// matcher. Simpler: use a store that returns a completed receipt for ANY id,
	// with matching bound fields.
	store := &anyIDStore{
		aChain: aChain,
		rec: InferenceReceipt{
			Status:              StatusCompleted,
			CanonicalOutputHash: common.HexToHash("0x0d0e0a0d"),
			ModelSpecHash:       model.Hash(),
			PromptHash:          PromptHash(prompt),
		},
	}
	c := fastClient(t, b, WithReceiptStore(store))
	res, err := c.Infer(context.Background(), model, prompt, 5, 3, big.NewInt(1_000_000))
	r.NoError(err)
	r.True(res.Receipt.Completed())
	r.Equal(common.HexToHash("0x0d0e0a0d"), res.Receipt.CanonicalOutputHash)
	r.NotEqual(common.Hash{}, res.IntentID)
	_ = cChain
}

func TestInferRejectsBindMismatch(t *testing.T) {
	r := require.New(t)
	b := newMockBackend()
	model := ModelSpec{Name: "m", Version: 1, WeightCommit: common.HexToHash("0x1")}
	// Store returns a completed receipt that binds a DIFFERENT model/prompt.
	store := &anyIDStore{rec: InferenceReceipt{
		Status:              StatusCompleted,
		CanonicalOutputHash: common.HexToHash("0x99"),
		ModelSpecHash:       common.HexToHash("0xdeadbeef"), // wrong
		PromptHash:          common.HexToHash("0xdeadbeef"),
	}}
	c := fastClient(t, b, WithReceiptStore(store))
	_, err := c.Infer(context.Background(), model, []byte("x"), 3, 2, nil)
	r.Error(err, "must reject a receipt binding a different model/prompt")
}

func TestRegisterModelAndGetModel(t *testing.T) {
	r := require.New(t)
	b := newMockBackend()
	c := fastClient(t, b)

	spec := ModelSpec{Name: "zenlm/zen-coder-flash", Version: 2, WeightCommit: common.HexToHash("0xc0de"), Quantization: "bf16-kat"}
	txHash, err := c.RegisterModel(context.Background(), spec)
	r.NoError(err)
	r.NotEqual(common.Hash{}, txHash)
	r.Len(b.sent, 1)
	r.Equal(ModelRegistryAddress, *b.sent[0].To())

	// Zero weight commitment rejected.
	_, err = c.RegisterModel(context.Background(), ModelSpec{Name: "x", Version: 1})
	r.Error(err)

	// GetModel decodes a getApproved return.
	gar := make([]byte, 64)
	gar[31] = 2
	copy(gar[32:64], spec.WeightCommit.Bytes())
	b.callRet = gar
	got, err := c.GetModel(context.Background(), spec.RegistryName())
	r.NoError(err)
	r.Equal(uint64(2), got.Version)
	r.Equal(spec.WeightCommit, got.WeightCommit)
}

func TestRegisterAndSubmitNeedKey(t *testing.T) {
	r := require.New(t)
	ro, err := NewClient(newMockBackend(), "") // read-only
	r.NoError(err)
	_, err = ro.RegisterModel(context.Background(), ModelSpec{WeightCommit: common.HexToHash("0x1")})
	r.Error(err)
	_, _, err = ro.SubmitInferenceIntent(context.Background(), SubmitOptions{
		ModelSpecHash: common.HexToHash("0x44"), PromptHash: common.HexToHash("0x55"), N: 3, Threshold: 2,
	})
	r.Error(err)
}

// anyIDStore returns the same receipt for any intent id (used to test the
// submit->wait->return flow where the pre-send tx hash is unknown).
type anyIDStore struct {
	aChain common.Hash
	rec    InferenceReceipt
}

func (s *anyIDStore) AChainID() common.Hash { return s.aChain }
func (s *anyIDStore) ReceiptByIntent(context.Context, common.Hash) (InferenceReceipt, MerkleProof, bool, error) {
	return s.rec, MerkleProof{}, true, nil
}

// Compile-time: mockBackend satisfies EVMBackend.
var _ EVMBackend = (*mockBackend)(nil)
