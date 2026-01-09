// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package c

import (
	"context"
	"math/big"
	"time"

	"github.com/luxfi/evm/plugin/evm/client"
	"github.com/luxfi/geth/ethclient"

	"github.com/luxfi/ids"
	"github.com/luxfi/rpc"
	"github.com/luxfi/sdk/wallet/primary/common"
	"github.com/luxfi/vm/vms/secp256k1fx"

	ethcommon "github.com/luxfi/geth/common"
)

var _ Wallet = (*wallet)(nil)

type Wallet interface {
	// Builder returns the builder that will be used to create the transactions.
	Builder() Builder

	// Signer returns the signer that will be used to sign the transactions.
	Signer() Signer

	// IssueImportTx creates, signs, and issues an import transaction that
	// attempts to consume all the available UTXOs and import the funds to [to].
	//
	// - [chainID] specifies the chain to be importing funds from.
	// - [to] specifies where to send the imported funds to.
	IssueImportTx(
		chainID ids.ID,
		to ethcommon.Address,
		options ...common.Option,
	) (*Tx, error)

	// IssueExportTx creates, signs, and issues an export transaction that
	// attempts to send all the provided [outputs] to the requested [chainID].
	//
	// - [chainID] specifies the chain to be exporting the funds to.
	// - [outputs] specifies the outputs to send to the [chainID].
	IssueExportTx(
		chainID ids.ID,
		outputs []*secp256k1fx.TransferOutput,
		options ...common.Option,
	) (*Tx, error)

	// IssueUnsignedAtomicTx signs and issues the unsigned tx.
	IssueUnsignedAtomicTx(
		utx UnsignedAtomicTx,
		options ...common.Option,
	) (*Tx, error)

	// IssueAtomicTx issues the signed tx.
	IssueAtomicTx(
		tx *Tx,
		options ...common.Option,
	) error
}

func NewWallet(
	builder Builder,
	signer Signer,
	luxClient client.Client,
	ethClient *ethclient.Client,
	backend Backend,
) Wallet {
	return &wallet{
		Backend:   backend,
		builder:   builder,
		signer:    signer,
		luxClient: luxClient,
		ethClient: ethClient,
	}
}

type wallet struct {
	Backend
	builder   Builder
	signer    Signer
	luxClient client.Client
	ethClient *ethclient.Client
}

func (w *wallet) Builder() Builder {
	return w.builder
}

func (w *wallet) Signer() Signer {
	return w.signer
}

func (w *wallet) IssueImportTx(
	chainID ids.ID,
	to ethcommon.Address,
	options ...common.Option,
) (*Tx, error) {
	baseFee, err := w.baseFee(options)
	if err != nil {
		return nil, err
	}

	utx, err := w.builder.NewImportTx(chainID, to, baseFee, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedAtomicTx(utx, options...)
}

func (w *wallet) IssueExportTx(
	chainID ids.ID,
	outputs []*secp256k1fx.TransferOutput,
	options ...common.Option,
) (*Tx, error) {
	baseFee, err := w.baseFee(options)
	if err != nil {
		return nil, err
	}

	utx, err := w.builder.NewExportTx(chainID, outputs, baseFee, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedAtomicTx(utx, options...)
}

func (w *wallet) IssueUnsignedAtomicTx(
	utx UnsignedAtomicTx,
	options ...common.Option,
) (*Tx, error) {
	ops := common.NewOptions(options)
	ctx := ops.Context()
	tx, err := SignUnsignedAtomic(ctx, w.signer, utx)
	if err != nil {
		return nil, err
	}

	return tx, w.IssueAtomicTx(tx, options...)
}

func (w *wallet) IssueAtomicTx(
	tx *Tx,
	options ...common.Option,
) error {
	ops := common.NewOptions(options)
	ctx := ops.Context()
	startTime := time.Now()
	// C-Chain transaction issuance not yet implemented.
	// This would require EVM client integration.
	txID := tx.ID
	_ = ctx // Suppress unused warning

	issuanceDuration := time.Since(startTime)
	if f := ops.PostIssuanceFunc(); f != nil {
		f(txID)
	}

	if ops.AssumeDecided() {
		return w.Backend.AcceptAtomicTx(ctx, tx)
	}

	if err := awaitTxAccepted(w.luxClient, ctx, txID, ops.PollFrequency()); err != nil {
		return err
	}

	if f := ops.ConfirmationHandler(); f != nil {
		totalDuration := time.Since(startTime)
		confirmationDuration := totalDuration - issuanceDuration

		f(common.ConfirmationReceipt{
			ChainAlias:           Alias,
			TxID:                 txID,
			IssuanceDuration:     issuanceDuration,
			ConfirmationDuration: confirmationDuration,
			TotalDuration:        totalDuration,
		})
	}

	return w.Backend.AcceptAtomicTx(ctx, tx)
}

func (w *wallet) baseFee(options []common.Option) (*big.Int, error) {
	ops := common.NewOptions(options)
	baseFee := ops.BaseFee(nil)
	if baseFee != nil {
		return baseFee, nil
	}

	// For now, return a default base fee
	return big.NewInt(25000000000), nil // 25 gwei
}

// awaitTxAccepted polls for transaction acceptance.
//
// TODO: Implement proper GetAtomicTxStatus for C-Chain.
// This requires querying the EVM for atomic transaction status.
// For now, this stub assumes immediate acceptance.
func awaitTxAccepted(
	c client.Client,
	ctx context.Context,
	txID ids.ID,
	freq time.Duration,
	options ...rpc.Option,
) error {
	// Stub: C-Chain atomic tx status polling not yet implemented.
	// When implemented, this should poll c.GetAtomicTxStatus(ctx, txID)
	// until the transaction is accepted or the context is cancelled.
	_ = c
	_ = txID
	_ = freq
	_ = options
	return nil
}
