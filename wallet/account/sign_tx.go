// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package account

import "fmt"

// TxAuthEnvelope is the wallet-side view of a transaction-authorization
// envelope. Callers (typically protocol/tx assemblers) implement this
// interface to expose the canonical 48-byte signing digest and to attach
// the wallet's signature back to the envelope after Sign returns.
//
// Defined as an interface here rather than imported from a protocol
// package because the wallet account package is the canonical lowest
// layer — protocol/tx packages depend on account, not the other way
// around. Any envelope type that satisfies these two methods can be
// signed by SignTx.
//
// The 48-byte (SHAKE256-384) digest size is the strict-PQ
// MinHashOutputBits floor; envelopes that produce a different digest
// size must rehash before exposing it through SigningDigest.
type TxAuthEnvelope interface {
	// SigningDigest returns the canonical 48-byte digest the wallet
	// signs. Must be deterministic for a fixed envelope state.
	SigningDigest() ([AccountIDSize]byte, error)

	// AttachSignature wires (schemeID, pubkey, sig) back into the
	// envelope. Implementations should store the triple verbatim; the
	// consensus verifier reconstructs the AccountID and scheme from
	// these bytes.
	AttachSignature(scheme WalletSchemeID, pubkey, sig []byte) error
}

// SignTx is the high-level entry point: ask the envelope for its
// canonical digest, sign the digest with the account, attach the
// signature back to the envelope. Returns an error if any of the three
// steps fails.
//
// Wallet callers should prefer SignTx over calling account.Sign
// directly. Direct callers must also remember to AttachSignature, and
// errors at that step are silent failures (no signed envelope) which
// SignTx surfaces explicitly.
func SignTx(account *PQAccount, env TxAuthEnvelope) error {
	if account == nil {
		return fmt.Errorf("account: SignTx called with nil PQAccount")
	}
	if env == nil {
		return fmt.Errorf("account: SignTx called with nil TxAuthEnvelope")
	}

	digest, err := env.SigningDigest()
	if err != nil {
		return fmt.Errorf("account: SigningDigest: %w", err)
	}

	sig, err := account.Sign(digest)
	if err != nil {
		return fmt.Errorf("account: Sign: %w", err)
	}

	if err := env.AttachSignature(account.SchemeID, account.PublicKey, sig); err != nil {
		return fmt.Errorf("account: AttachSignature: %w", err)
	}
	return nil
}
