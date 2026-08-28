// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package aichain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"time"

	ethereum "github.com/luxfi/geth"
	"github.com/luxfi/geth/common"
	gethtypes "github.com/luxfi/geth/core/types"
)

// EVMBackend is the SUBSET of luxfi/evm/ethclient.Client this SDK needs: enough
// to read chain id / nonce / fees, send a tx, poll a receipt, and make an
// eth_call. luxfi/evm/ethclient.Client satisfies it directly (see DialClient),
// and a test can supply a mock without a live chain. Keeping the surface minimal
// is the decoupling: the SDK depends on these eight methods, not on the whole
// node client.
type EVMBackend interface {
	ChainID(ctx context.Context) (*big.Int, error)
	AcceptedNonceAt(ctx context.Context, account common.Address) (uint64, error)
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
	EstimateBaseFee(ctx context.Context) (*big.Int, error)
	EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error)
	SendTransaction(ctx context.Context, tx *gethtypes.Transaction) error
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*gethtypes.Receipt, error)
	CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

// ReceiptStore reads committed A-Chain receipts for the high-level WaitReceipt /
// GetReceipt convenience. It is intentionally SEPARATE from EVMBackend (the
// settlement read path is not the EVM tx path): an implementation may poll the
// A-Chain RPC directly, or watch the aivmbridge receipt-root checkpoint and the
// A->C export. ReceiptByIntent returns the committed receipt + its inclusion
// proof for an intent id, or (false) if not yet settled.
//
// The SDK ships no live ReceiptStore (it depends on the node-local A-Chain RPC
// shape, out of scope here); callers wire their own, and tests use a fake. This
// keeps the SDK decoupled from any single A-Chain transport while still offering
// the one-call Infer convenience.
type ReceiptStore interface {
	ReceiptByIntent(ctx context.Context, intentID common.Hash) (receipt InferenceReceipt, proof MerkleProof, found bool, err error)
}

// Client is the high-level A-Chain (Beluga) inference client over an EVM
// JSON-RPC endpoint. It signs and sends txs to the AI precompiles and reads
// results.
//
//   - Tier-1 (GenerateDeterministic): the deterministic in-consensus inference
//     precompile (0x0300…0003) answers INSIDE one EVM call — synchronous, no
//     quorum, small models. Use it via an eth_call (no tx, no gas spent) or a tx
//     if you want the call recorded.
//   - Tier-2 (SubmitInferenceIntent / WaitReceipt / Infer): a large model on the
//     A-Chain settled by an M-of-N quorum — ASYNCHRONOUS. Submit returns an
//     intent id; the result arrives later as a committed A-Chain receipt the
//     bridge can verify.
type Client struct {
	backend EVMBackend
	store   ReceiptStore // optional; required only for WaitReceipt/Infer/GetReceipt

	key  *ecdsa.PrivateKey
	from common.Address

	chainID *big.Int // cached after first use

	// PollInterval / PollTimeout bound the tx-receipt and settlement polling.
	PollInterval time.Duration
	PollTimeout  time.Duration
}

// Option configures a Client.
type Option func(*Client)

// WithReceiptStore wires the A-Chain receipt read path (enables WaitReceipt /
// Infer / GetReceipt).
func WithReceiptStore(s ReceiptStore) Option { return func(c *Client) { c.store = s } }

// WithChainID pre-sets the EVM chain id, skipping the eth_chainId round-trip
// (useful in tests / offline signing).
func WithChainID(id *big.Int) Option { return func(c *Client) { c.chainID = new(big.Int).Set(id) } }

// WithPolling overrides the receipt/settlement polling cadence and deadline.
func WithPolling(interval, timeout time.Duration) Option {
	return func(c *Client) { c.PollInterval, c.PollTimeout = interval, timeout }
}

// NewClient builds a Client over the given backend, signing with privKeyHex (a
// 0x-optional hex secp256k1 key). For a read-only client (eth_call Tier-1, or
// reads), pass an empty key.
func NewClient(backend EVMBackend, privKeyHex string, opts ...Option) (*Client, error) {
	if backend == nil {
		return nil, errors.New("aichain: nil backend")
	}
	c := &Client{
		backend:      backend,
		PollInterval: 2 * time.Second,
		PollTimeout:  5 * time.Minute,
	}
	if privKeyHex != "" {
		key, addr, err := parseKey(privKeyHex)
		if err != nil {
			return nil, err
		}
		c.key, c.from = key, addr
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// From returns the signer address (zero for a read-only client).
func (c *Client) From() common.Address { return c.from }

// chainIDOf returns the cached chain id, fetching once if needed.
func (c *Client) chainIDOf(ctx context.Context) (*big.Int, error) {
	if c.chainID != nil {
		return c.chainID, nil
	}
	id, err := c.backend.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("aichain: chain id: %w", err)
	}
	c.chainID = id
	return id, nil
}

// SubmitOptions parametrise a Tier-2 inference intent.
type SubmitOptions struct {
	// ModelSpecHash is the canonical model id (ModelSpec.Hash). Required, non-zero.
	ModelSpecHash common.Hash
	// PromptHash commits to the prompt (PromptHash(prompt)). Required, non-zero.
	PromptHash common.Hash
	// N is the fan-out (independent provider executions), in [1, 256].
	N uint16
	// Threshold is the M-of-N agreement, in [1, N] (the A-Chain also requires
	// floor(N/2)+1 <= Threshold).
	Threshold uint16
	// Fee funds the A escrow + burn (256-bit, non-negative).
	Fee *big.Int
	// Routing is an opaque transport hint (NOT consensus state, NOT in the id).
	Routing common.Hash
	// GasLimit overrides gas estimation when non-zero.
	GasLimit uint64
}

func (o SubmitOptions) validate() error {
	if o.ModelSpecHash == (common.Hash{}) {
		return errors.New("aichain: zero modelSpecHash")
	}
	if o.PromptHash == (common.Hash{}) {
		return errors.New("aichain: zero promptHash")
	}
	if o.N == 0 || o.N > MaxFanout {
		return fmt.Errorf("aichain: N=%d out of [1,%d]", o.N, MaxFanout)
	}
	if o.Threshold == 0 || o.Threshold > o.N {
		return fmt.Errorf("aichain: threshold=%d out of [1,%d]", o.Threshold, o.N)
	}
	if min := o.N/2 + 1; o.Threshold < min {
		return fmt.Errorf("aichain: threshold=%d below quorum floor %d (=floor(N/2)+1)", o.Threshold, min)
	}
	return nil
}

// SubmitInferenceIntent sends a Tier-2 intent tx to the aivmbridge precompile
// (Pattern A) and returns the deterministic intent id (re-derived from the LANDED
// tx hash + call index, exactly as the precompile derives it) and the tx hash.
//
// The intent is the START of an async quorum settlement; use WaitReceipt (or the
// one-call Infer) to obtain the result. The single-call submit places the intent
// at call index 0 of its tx — the SDK derives the id on that assumption (a
// contract that batches multiple submits in one tx must compute ids itself).
func (c *Client) SubmitInferenceIntent(ctx context.Context, opts SubmitOptions) (intentID common.Hash, txHash common.Hash, err error) {
	if c.key == nil {
		return common.Hash{}, common.Hash{}, errors.New("aichain: SubmitInferenceIntent needs a signing key")
	}
	if err := opts.validate(); err != nil {
		return common.Hash{}, common.Hash{}, err
	}
	data := EncodeSubmitInferenceIntent(opts.ModelSpecHash, opts.PromptHash, opts.N, opts.Threshold, opts.Fee, opts.Routing)
	txHash, err = c.sendTx(ctx, AIBridgeAddress, data, opts.GasLimit)
	if err != nil {
		return common.Hash{}, common.Hash{}, err
	}
	// Wait for the tx to land so we know its committed hash, then re-derive the id
	// against this client's chain ids (cChainID = EVM chain id; aChainID supplied
	// by the receipt store at settle time, but the id binds the C side — see note).
	rcpt, err := c.waitTxReceipt(ctx, txHash)
	if err != nil {
		return common.Hash{}, txHash, err
	}
	id, err := c.deriveIntentID(ctx, rcpt.TxHash, 0, opts)
	if err != nil {
		return common.Hash{}, txHash, err
	}
	return id, txHash, nil
}

// deriveIntentID reconstructs the intent id from the landed tx + the submit
// options. cChainID is this EVM's chain id (32-byte left-padded), aChainID is the
// configured A-Chain id; both are part of the pinned preimage. If the caller has
// not configured an A-Chain id (via the receipt store / options) the SDK uses the
// EVM chain id for both only as a best-effort local correlation key — the
// authoritative id is the one the precompile commits. Callers that need the
// exact cross-chain id should read it back from the receipt (receipt.IntentID).
func (c *Client) deriveIntentID(ctx context.Context, txHash common.Hash, callIndex uint32, opts SubmitOptions) (common.Hash, error) {
	id, err := c.chainIDOf(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	cChain := common.BigToHash(id)
	aChain := c.aChainID(cChain)
	fee := opts.Fee
	if fee == nil {
		fee = big.NewInt(0)
	}
	return ComputeIntentID(cChain, aChain, txHash, callIndex, c.from, opts.ModelSpecHash, opts.PromptHash, opts.N, opts.Threshold, fee), nil
}

// aChainID returns the configured A-Chain id for preimage derivation. With no
// receipt store configured we fall back to the C chain id (local correlation
// only); when a store is present and exposes an A-Chain id, that is used. The
// fallback is documented and the authoritative id always comes from the receipt.
func (c *Client) aChainID(cChain common.Hash) common.Hash {
	if p, ok := c.store.(interface{ AChainID() common.Hash }); ok {
		if id := p.AChainID(); id != (common.Hash{}) {
			return id
		}
	}
	return cChain
}

// WaitReceipt polls the A-Chain receipt store for the committed receipt of
// intentID until it is Completed (or the deadline / ctx fires). It returns the
// receipt only when actionable (Status==Completed, non-zero output); a Failed or
// Challenged terminal receipt returns an error carrying the receipt.
func (c *Client) WaitReceipt(ctx context.Context, intentID common.Hash) (InferenceReceipt, error) {
	if c.store == nil {
		return InferenceReceipt{}, errors.New("aichain: WaitReceipt needs a ReceiptStore (WithReceiptStore)")
	}
	deadline := time.Now().Add(c.PollTimeout)
	tick := time.NewTicker(c.PollInterval)
	defer tick.Stop()
	for {
		r, _, found, err := c.store.ReceiptByIntent(ctx, intentID)
		if err != nil {
			return InferenceReceipt{}, fmt.Errorf("aichain: receipt lookup: %w", err)
		}
		if found {
			switch r.Status {
			case StatusCompleted:
				if r.CanonicalOutputHash == (common.Hash{}) {
					return r, fmt.Errorf("aichain: receipt completed but output is zero (intent %s)", intentID.Hex())
				}
				return r, nil
			case StatusFailed:
				return r, fmt.Errorf("aichain: inference failed (intent %s)", intentID.Hex())
			case StatusChallenged:
				return r, fmt.Errorf("aichain: inference challenged (intent %s)", intentID.Hex())
			}
			// Unknown/Pending -> keep polling.
		}
		select {
		case <-ctx.Done():
			return InferenceReceipt{}, ctx.Err()
		case <-tick.C:
			if time.Now().After(deadline) {
				return InferenceReceipt{}, fmt.Errorf("aichain: timed out waiting for receipt (intent %s)", intentID.Hex())
			}
		}
	}
}

// GetReceipt does a single (non-blocking) read of the committed receipt for an
// intent id. found==false means not yet settled.
func (c *Client) GetReceipt(ctx context.Context, intentID common.Hash) (receipt InferenceReceipt, proof MerkleProof, found bool, err error) {
	if c.store == nil {
		return InferenceReceipt{}, MerkleProof{}, false, errors.New("aichain: GetReceipt needs a ReceiptStore")
	}
	return c.store.ReceiptByIntent(ctx, intentID)
}

// InferResult is the outcome of a one-call Tier-2 Infer.
type InferResult struct {
	IntentID common.Hash
	TxHash   common.Hash
	Receipt  InferenceReceipt
}

// Infer is the one-call Tier-2 convenience: submit an intent for `model` over
// `prompt`, wait for the quorum to settle, and return the canonical result.
//
// SETTLEMENT IS ASYNC. Unlike GenerateDeterministic (Tier-1, answered in one EVM
// call), Infer submits a C-Chain intent and BLOCKS polling the A-Chain until an
// M-of-N quorum settles a receipt — seconds to minutes, bounded by PollTimeout.
// The returned Receipt.CanonicalOutputHash is the agreed output digest; fetch the
// full output bytes from your provider/DA layer keyed by that hash (the chain
// commits the digest, not the bytes).
func (c *Client) Infer(ctx context.Context, model ModelSpec, prompt []byte, n, threshold uint16, fee *big.Int) (InferResult, error) {
	opts := SubmitOptions{
		ModelSpecHash: model.Hash(),
		PromptHash:    PromptHash(prompt),
		N:             n,
		Threshold:     threshold,
		Fee:           fee,
	}
	intentID, txHash, err := c.SubmitInferenceIntent(ctx, opts)
	if err != nil {
		return InferResult{}, err
	}
	r, err := c.WaitReceipt(ctx, intentID)
	if err != nil {
		return InferResult{IntentID: intentID, TxHash: txHash}, err
	}
	// Bind the settled receipt to what we asked for (defense in depth: the store
	// is untrusted transport).
	if r.ModelSpecHash != opts.ModelSpecHash || r.PromptHash != opts.PromptHash {
		return InferResult{IntentID: intentID, TxHash: txHash, Receipt: r},
			fmt.Errorf("aichain: receipt binds different model/prompt than requested")
	}
	return InferResult{IntentID: intentID, TxHash: txHash, Receipt: r}, nil
}

// RegisterModel adopts a model version in the registry precompile (0x0300…0002).
// The caller must be a registry admin (governance); a non-admin tx reverts on
// chain. Returns the tx hash.
func (c *Client) RegisterModel(ctx context.Context, spec ModelSpec) (common.Hash, error) {
	if c.key == nil {
		return common.Hash{}, errors.New("aichain: RegisterModel needs a signing key")
	}
	if spec.WeightCommit == (common.Hash{}) {
		return common.Hash{}, errors.New("aichain: zero weight commitment")
	}
	return c.sendTx(ctx, ModelRegistryAddress, EncodeRegisterModel(spec), 0)
}

// GetModel reads the registry's currently-adopted (version, weight commitment)
// for a model name via eth_call (no tx). A zero weight commitment means the
// model is not adopted.
func (c *Client) GetModel(ctx context.Context, name common.Hash) (ApprovedModel, error) {
	ret, err := c.backend.CallContract(ctx, ethereum.CallMsg{
		To:   &ModelRegistryAddress,
		Data: EncodeGetApproved(name),
	}, nil)
	if err != nil {
		return ApprovedModel{}, fmt.Errorf("aichain: getApproved call: %w", err)
	}
	return DecodeGetApprovedResult(ret)
}

// GenerateDeterministic runs the Tier-1 in-consensus inference precompile
// (0x0300…0003) via eth_call and returns the full token sequence (prompt +
// generated). This is SYNCHRONOUS and answered inside the single call — no
// quorum, no async settlement, no gas spent (it is an eth_call). Use it for the
// small deterministic model; use Infer for large Tier-2 models.
func (c *Client) GenerateDeterministic(ctx context.Context, nNew uint32, promptTokens []uint32) ([]uint32, error) {
	if len(promptTokens) == 0 {
		return nil, errors.New("aichain: empty prompt")
	}
	ret, err := c.backend.CallContract(ctx, ethereum.CallMsg{
		From: c.from,
		To:   &InferenceAddress,
		Data: EncodeGenerate(nNew, promptTokens),
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("aichain: generate call: %w", err)
	}
	return DecodeGenerateResult(ret)
}

// ---------------------------------------------------------------------------
// tx plumbing
// ---------------------------------------------------------------------------

// sendTx builds, signs, and broadcasts an EIP-1559 tx to `to` with `data`,
// returning the tx hash. gasLimit==0 triggers gas estimation (+ a 25% headroom).
func (c *Client) sendTx(ctx context.Context, to common.Address, data []byte, gasLimit uint64) (common.Hash, error) {
	chainID, err := c.chainIDOf(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	nonce, err := c.backend.AcceptedNonceAt(ctx, c.from)
	if err != nil {
		return common.Hash{}, fmt.Errorf("aichain: nonce: %w", err)
	}
	tip, err := c.backend.SuggestGasTipCap(ctx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("aichain: gas tip: %w", err)
	}
	baseFee, err := c.backend.EstimateBaseFee(ctx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("aichain: base fee: %w", err)
	}
	// maxFee = 2*baseFee + tip (standard headroom for one base-fee doubling).
	feeCap := new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), tip)

	if gasLimit == 0 {
		est, err := c.backend.EstimateGas(ctx, ethereum.CallMsg{
			From: c.from, To: &to, Data: data, GasFeeCap: feeCap, GasTipCap: tip,
		})
		if err != nil {
			return common.Hash{}, fmt.Errorf("aichain: estimate gas: %w", err)
		}
		gasLimit = est + est/4 // +25% headroom
	}

	tx := gethtypes.NewTx(&gethtypes.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: tip,
		GasFeeCap: feeCap,
		Gas:       gasLimit,
		To:        &to,
		Value:     big.NewInt(0),
		Data:      data,
	})
	signed, err := gethtypes.SignTx(tx, gethtypes.LatestSignerForChainID(chainID), c.key)
	if err != nil {
		return common.Hash{}, fmt.Errorf("aichain: sign: %w", err)
	}
	if err := c.backend.SendTransaction(ctx, signed); err != nil {
		return common.Hash{}, fmt.Errorf("aichain: send: %w", err)
	}
	return signed.Hash(), nil
}

// waitTxReceipt polls until the tx is mined, returning its receipt. It fails if
// the tx reverted (Status != 1).
func (c *Client) waitTxReceipt(ctx context.Context, txHash common.Hash) (*gethtypes.Receipt, error) {
	deadline := time.Now().Add(c.PollTimeout)
	tick := time.NewTicker(c.PollInterval)
	defer tick.Stop()
	for {
		rcpt, err := c.backend.TransactionReceipt(ctx, txHash)
		if err == nil && rcpt != nil {
			if rcpt.Status != gethtypes.ReceiptStatusSuccessful {
				return rcpt, fmt.Errorf("aichain: tx %s reverted", txHash.Hex())
			}
			return rcpt, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-tick.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("aichain: timed out waiting for tx %s", txHash.Hex())
			}
		}
	}
}
