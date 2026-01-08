// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package exchangevm

import (
	"errors"
	"fmt"
	"maps"
	"net/http"

	"github.com/luxfi/ids"
	"github.com/luxfi/math"
	"github.com/luxfi/sdk/node/api"
	"github.com/luxfi/sdk/utils/formatting"
	"github.com/luxfi/sdk/utils/linked"
	"github.com/luxfi/log"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/node/vms/txs/mempool"
	"github.com/luxfi/sdk/node/vms/exchangevm/txs"
)

type WalletService struct {
	vm         *VM
	pendingTxs *linked.Hashmap[ids.ID, *txs.Tx]
}

// update refreshes the UTXO set, removing spent UTXOs from pending transactions
func (w *WalletService) update(utxos []*lux.UTXO) ([]*lux.UTXO, error) {
	// For now, return UTXOs as-is
	// TODO: Filter out UTXOs that have been spent by pending transactions
	return utxos, nil
}

func (w *WalletService) decided(txID ids.ID) {
	if !w.pendingTxs.Delete(txID) {
		return
	}

	w.vm.log.Info("tx decided over wallet API",
		log.Stringer("txID", txID),
	)
	for {
		txID, tx, ok := w.pendingTxs.Oldest()
		if !ok {
			return
		}

		err := w.vm.network.IssueTxFromRPCWithoutVerification(tx)
		if err == nil {
			w.vm.log.Info("issued tx to mempool over wallet API",
				log.Stringer("txID", txID),
			)
			return
		}
		if errors.Is(err, mempool.ErrDuplicateTx) {
			return
		}

		w.pendingTxs.Delete(txID)
		w.vm.log.Warn("dropping tx issued over wallet API",
			log.Stringer("txID", txID),
			log.String("error", err.Error()),
		)
	}
}

func (w *WalletService) issue(tx *txs.Tx) (ids.ID, error) {
	txID := tx.ID()
	w.vm.log.Info("issuing tx over wallet API",
		log.Stringer("txID", txID),
	)

	if _, ok := w.pendingTxs.Get(txID); ok {
		w.vm.log.Warn("issuing duplicate tx over wallet API",
			log.Stringer("txID", txID),
		)
		return txID, nil
	}

	if w.pendingTxs.Len() == 0 {
		if err := w.vm.network.IssueTxFromRPCWithoutVerification(tx); err == nil {
			w.vm.log.Info("issued tx to mempool over wallet API",
				log.Stringer("txID", txID),
			)
		} else if !errors.Is(err, mempool.ErrDuplicateTx) {
			w.vm.log.Warn("failed to issue tx over wallet API",
				log.Stringer("txID", txID),
				log.String("error", err.Error()),
			)
			return ids.Empty, err
		}
	} else {
		w.vm.log.Info("enqueueing tx over wallet API",
			log.Stringer("txID", txID),
		)
	}

	w.pendingTxs.Put(txID, tx)
	return txID, nil
}

// IssueTx attempts to issue a transaction into consensus
func (w *WalletService) IssueTx(_ *http.Request, args *api.FormattedTx, reply *api.JSONTxID) error {
	w.vm.log.Warn("deprecated API called",
		log.String("service", "wallet"),
		log.String("method", "issueTx"),
		log.String("tx", args.Tx),
	)

	txBytes, err := formatting.Decode(args.Encoding, args.Tx)
	if err != nil {
		return fmt.Errorf("problem decoding transaction: %w", err)
	}

	tx, err := w.vm.parser.ParseTx(txBytes)
	if err != nil {
		return err
	}

	w.vm.Lock.Lock()
	defer w.vm.Lock.Unlock()

	txID, err := w.issue(tx)
	reply.TxID = txID
	return err
}

// Send returns the ID of the newly created transaction
func (w *WalletService) Send(r *http.Request, args *SendArgs, reply *JSONTxIDChangeAddr) error {
	return w.SendMultiple(r, &SendMultipleArgs{
		JSONSpendHeader: args.JSONSpendHeader,
		Outputs:         []SendOutput{args.SendOutput},
		Memo:            args.Memo,
	}, reply)
}

// SendMultiple sends a transaction with multiple outputs.
func (w *WalletService) SendMultiple(_ *http.Request, args *SendMultipleArgs, reply *JSONTxIDChangeAddr) error {
	w.vm.log.Warn("deprecated API called",
		log.String("service", "wallet"),
		log.String("method", "sendMultiple"),
		"username", args.Username,
	)

	// Validate the memo field
	memoBytes := []byte(args.Memo)
	if l := len(memoBytes); l > lux.MaxMemoSize {
		return fmt.Errorf("max memo length is %d but provided memo field is length %d",
			lux.MaxMemoSize,
			l)
	} else if len(args.Outputs) == 0 {
		return errNoOutputs
	}

	// Parse the from addresses
	fromAddrs, err := lux.ParseServiceAddresses(w.vm, args.From)
	if err != nil {
		return fmt.Errorf("couldn't parse 'From' addresses: %w", err)
	}

	w.vm.Lock.Lock()
	defer w.vm.Lock.Unlock()

	// Load user's UTXOs/keys
	utxos, kc, err := w.vm.LoadUser(args.Username, args.Password, fromAddrs)
	if err != nil {
		return err
	}

	utxos, err = w.update(utxos)
	if err != nil {
		return err
	}

	// Parse the change address.
	if len(kc.Keys) == 0 {
		return errNoKeys
	}
	defaultAddr, err := publicKeyToAddress(kc.Keys[0].PublicKey())
	if err != nil {
		return err
	}
	changeAddr, err := w.vm.selectChangeAddr(defaultAddr, args.ChangeAddr)
	if err != nil {
		return err
	}

	// Calculate required input amounts and create the desired outputs
	// String repr. of asset ID --> asset ID
	assetIDs := make(map[string]ids.ID)
	// Asset ID --> amount of that asset being sent
	amounts := make(map[ids.ID]uint64)
	// Outputs of our tx
	outs := []*lux.TransferableOutput{}
	for _, output := range args.Outputs {
		if output.Amount == 0 {
			return errZeroAmount
		}
		assetID, ok := assetIDs[output.AssetID] // Asset ID of next output
		if !ok {
			assetID, err = w.vm.lookupAssetID(output.AssetID)
			if err != nil {
				return fmt.Errorf("couldn't find asset %s", output.AssetID)
			}
			assetIDs[output.AssetID] = assetID
		}
		currentAmount := amounts[assetID]
		newAmount, err := math.Add64(currentAmount, uint64(output.Amount))
		if err != nil {
			return fmt.Errorf("problem calculating required spend amount: %w", err)
		}
		amounts[assetID] = newAmount

		// Parse the to address
		to, err := lux.ParseServiceAddress(w.vm, output.To)
		if err != nil {
			return fmt.Errorf("problem parsing to address %q: %w", output.To, err)
		}

		// Create the Output
		outs = append(outs, &lux.TransferableOutput{
			Asset: lux.Asset{ID: assetID},
			Out: &secp256k1fx.TransferOutput{
				Amt: uint64(output.Amount),
				OutputOwners: secp256k1fx.OutputOwners{
					Locktime:  0,
					Threshold: 1,
					Addrs:     []ids.ShortID{to},
				},
			},
		})
	}

	amountsWithFee := maps.Clone(amounts)

	amountWithFee, err := math.Add64(amounts[w.vm.feeAssetID], w.vm.TxFee)
	if err != nil {
		return fmt.Errorf("problem calculating required spend amount: %w", err)
	}
	amountsWithFee[w.vm.feeAssetID] = amountWithFee

	amountsSpent, ins, keys, err := w.vm.Spend(
		utxos,
		kc,
		amountsWithFee,
	)
	if err != nil {
		return err
	}

	// Add the required change outputs
	for assetID, amountWithFee := range amountsWithFee {
		amountSpent := amountsSpent[assetID]

		if amountSpent > amountWithFee {
			outs = append(outs, &lux.TransferableOutput{
				Asset: lux.Asset{ID: assetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: amountSpent - amountWithFee,
					OutputOwners: secp256k1fx.OutputOwners{
						Locktime:  0,
						Threshold: 1,
						Addrs:     []ids.ShortID{changeAddr},
					},
				},
			})
		}
	}

	codec := w.vm.parser.Codec()
	lux.SortTransferableOutputs(outs, codec)

	tx := &txs.Tx{Unsigned: &txs.BaseTx{BaseTx: lux.BaseTx{
		NetworkID:    w.vm.consensusCtx.NetworkID,
		BlockchainID: w.vm.consensusCtx.ChainID,
		Outs:         outs,
		Ins:          ins,
		Memo:         memoBytes,
	}}}
	if err := tx.SignSECP256K1Fx(codec, keys); err != nil {
		return err
	}

	txID, err := w.issue(tx)
	if err != nil {
		return fmt.Errorf("problem issuing transaction: %w", err)
	}

	reply.TxID = txID
	reply.ChangeAddr, err = w.vm.FormatLocalAddress(changeAddr)
	return err
}
