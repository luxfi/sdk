// Copyright (C) 2019-2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

// Package p provides P-Chain wallet backend implementation.
package p

import (
	"context"
	"sync"

	"github.com/luxfi/constants"
	"github.com/luxfi/database"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/protocol/p/fx"
	"github.com/luxfi/protocol/p/txs"
	"github.com/luxfi/sdk/wallet/chain/p/builder"
	"github.com/luxfi/sdk/wallet/chain/p/signer"
	"github.com/luxfi/sdk/wallet/primary/common"
	lux "github.com/luxfi/utxo"
)

var _ Backend = (*backend)(nil)

// Backend defines the full interface required to support a P-chain wallet.
type Backend interface {
	builder.Backend
	signer.Backend

	AcceptTx(ctx context.Context, tx *txs.Tx) error
}

type backend struct {
	common.ChainUTXOs

	context *builder.Context

	chainOwnerLock sync.RWMutex
	chainOwner     map[ids.ID]fx.Owner // chainID -> owner
}

func NewBackend(context *builder.Context, utxos common.ChainUTXOs, chainTxs map[ids.ID]*txs.Tx) Backend {
	chainOwner := make(map[ids.ID]fx.Owner)
	for txID, tx := range chainTxs { // first get owners from the CreateNetworkTx
		createNetworkTx, ok := tx.Unsigned.(*txs.CreateNetworkTx)
		if !ok {
			continue
		}
		chainOwner[txID] = createNetworkTx.Owner
	}
	for _, tx := range chainTxs { // then check for TransferChainOwnershipTx
		transferChainOwnershipTx, ok := tx.Unsigned.(*txs.TransferChainOwnershipTx)
		if !ok {
			continue
		}
		chainOwner[transferChainOwnershipTx.Chain] = transferChainOwnershipTx.Owner
	}
	return &backend{
		ChainUTXOs: utxos,
		context:    context,
		chainOwner: chainOwner,
	}
}

func (b *backend) AcceptTx(ctx context.Context, tx *txs.Tx) error {
	txID := tx.ID()
	v := &backendVisitor{
		b:    b,
		ctx:  ctx,
		txID: txID,
	}
	err := tx.Unsigned.Visit(v)
	if err != nil {
		return err
	}

	producedUTXOSlice := tx.UTXOs()
	return b.addUTXOs(ctx, constants.PlatformChainID, producedUTXOSlice)
}

// backendVisitor handles accepting of transactions for the backend
type backendVisitor struct {
	b    *backend
	ctx  context.Context
	txID ids.ID
}

func (v *backendVisitor) AdvanceTimeTx(*txs.AdvanceTimeTx) error {
	return nil
}

func (v *backendVisitor) RewardValidatorTx(*txs.RewardValidatorTx) error {
	return nil
}

func (v *backendVisitor) AddValidatorTx(tx *txs.AddValidatorTx) error {
	return v.baseTx(&tx.BaseTx)
}

func (v *backendVisitor) AddChainValidatorTx(tx *txs.AddChainValidatorTx) error {
	return v.baseTx(&tx.BaseTx)
}

func (v *backendVisitor) AddDelegatorTx(tx *txs.AddDelegatorTx) error {
	return v.baseTx(&tx.BaseTx)
}

func (v *backendVisitor) CreateChainTx(tx *txs.CreateChainTx) error {
	return v.baseTx(&tx.BaseTx)
}

func (v *backendVisitor) CreateNetworkTx(tx *txs.CreateNetworkTx) error {
	v.b.setChainOwner(v.txID, tx.Owner)
	return v.baseTx(&tx.BaseTx)
}

func (v *backendVisitor) ImportTx(tx *txs.ImportTx) error {
	err := v.b.removeUTXOs(v.ctx, tx.SourceChain, tx.InputUTXOs())
	if err != nil {
		return err
	}
	return v.baseTx(&tx.BaseTx)
}

func (v *backendVisitor) ExportTx(tx *txs.ExportTx) error {
	for i, out := range tx.ExportedOutputs {
		err := v.b.AddUTXO(
			v.ctx,
			tx.DestinationChain,
			&lux.UTXO{
				UTXOID: lux.UTXOID{
					TxID:        v.txID,
					OutputIndex: uint32(len(tx.Outs) + i),
				},
				Asset: lux.Asset{ID: out.AssetID()},
				Out:   out.Out,
			},
		)
		if err != nil {
			return err
		}
	}
	return v.baseTx(&tx.BaseTx)
}

func (v *backendVisitor) RemoveChainValidatorTx(tx *txs.RemoveChainValidatorTx) error {
	return v.baseTx(&tx.BaseTx)
}

func (v *backendVisitor) TransformChainTx(tx *txs.TransformChainTx) error {
	return v.baseTx(&tx.BaseTx)
}

func (v *backendVisitor) AddPermissionlessValidatorTx(tx *txs.AddPermissionlessValidatorTx) error {
	return v.baseTx(&tx.BaseTx)
}

func (v *backendVisitor) AddPermissionlessDelegatorTx(tx *txs.AddPermissionlessDelegatorTx) error {
	return v.baseTx(&tx.BaseTx)
}

func (v *backendVisitor) TransferChainOwnershipTx(tx *txs.TransferChainOwnershipTx) error {
	v.b.setChainOwner(tx.Chain, tx.Owner)
	return v.baseTx(&tx.BaseTx)
}

func (v *backendVisitor) BaseTx(tx *txs.BaseTx) error {
	return v.baseTx(tx)
}

func (v *backendVisitor) ConvertNetworkToL1Tx(*txs.ConvertNetworkToL1Tx) error {
	return nil
}

func (v *backendVisitor) RegisterL1ValidatorTx(*txs.RegisterL1ValidatorTx) error {
	return nil
}

func (v *backendVisitor) SetL1ValidatorWeightTx(tx *txs.SetL1ValidatorWeightTx) error {
	return v.baseTx(&tx.BaseTx)
}

func (v *backendVisitor) IncreaseL1ValidatorBalanceTx(tx *txs.IncreaseL1ValidatorBalanceTx) error {
	return v.baseTx(&tx.BaseTx)
}

func (v *backendVisitor) DisableL1ValidatorTx(tx *txs.DisableL1ValidatorTx) error {
	return v.baseTx(&tx.BaseTx)
}

func (v *backendVisitor) baseTx(tx *txs.BaseTx) error {
	return v.b.removeUTXOs(v.ctx, constants.PlatformChainID, tx.InputIDs())
}

func (b *backend) addUTXOs(ctx context.Context, destinationChainID ids.ID, utxos []*lux.UTXO) error {
	for _, utxo := range utxos {
		if err := b.AddUTXO(ctx, destinationChainID, utxo); err != nil {
			return err
		}
	}
	return nil
}

func (b *backend) removeUTXOs(ctx context.Context, sourceChain ids.ID, utxoIDs set.Set[ids.ID]) error {
	for utxoID := range utxoIDs {
		if err := b.RemoveUTXO(ctx, sourceChain, utxoID); err != nil {
			return err
		}
	}
	return nil
}

func (b *backend) GetOwner(_ context.Context, ownerID ids.ID) (fx.Owner, error) {
	b.chainOwnerLock.RLock()
	defer b.chainOwnerLock.RUnlock()

	owner, exists := b.chainOwner[ownerID]
	if !exists {
		return nil, database.ErrNotFound
	}
	return owner, nil
}

func (b *backend) GetNetOwner(_ context.Context, chainID ids.ID) (fx.Owner, error) {
	return b.GetOwner(context.Background(), chainID)
}

func (b *backend) setChainOwner(chainID ids.ID, owner fx.Owner) {
	b.chainOwnerLock.Lock()
	defer b.chainOwnerLock.Unlock()

	b.chainOwner[chainID] = owner
}
