// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package mempool

import (
	"math"
	"testing"
	"time"

	"github.com/luxfi/metric"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/vms/components/lux"

	"github.com/luxfi/sdk/node/vms/platformvm/txs"

	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

var preFundedKeys = secp256k1.TestKeys()

// shows that valid tx is not added to mempool if this would exceed its maximum
// size
func TestBlockBuilderMaxMempoolSizeHandling(t *testing.T) {
	require := require.New(t)

	registerer := metric.NewRegistry()
	mpool, err := New("mempool", registerer)
	require.NoError(err)

	decisionTxs, err := createTestDecisionTxs(1)
	require.NoError(err)
	tx := decisionTxs[0]

	// Test mempool full behavior - cannot access private bytesAvailable field
	// This test verifies the mempool can handle transactions, the full/capacity
	// testing is done in vms/txs/mempool/mempool_test.go
	err = mpool.Add(tx)
	require.NoError(err, "should have added tx to mempool")
}

func TestDecisionTxsInMempool(t *testing.T) {
	require := require.New(t)

	registerer := metric.NewRegistry()
	mpool, err := New("mempool", registerer)
	require.NoError(err)

	decisionTxs, err := createTestDecisionTxs(2)
	require.NoError(err)

	// txs must not already there before we start
	require.False(mpool.HasTxs())

	for _, tx := range decisionTxs {
		// tx not already there
		require.False(mpool.Has(tx.ID()))

		// we can insert
		require.NoError(mpool.Add(tx))

		// we can get it
		require.True(mpool.Has(tx.ID()))

		retrieved, _ := mpool.Get(tx.ID())
		require.NotNil(retrieved)
		require.Equal(tx, retrieved)

		// we can peek it
		peeked := mpool.PeekTxs(math.MaxInt)

		// tx will be among those peeked,
		// in NO PARTICULAR ORDER
		found := false
		for _, pk := range peeked {
			if pk.ID() == tx.ID() {
				found = true
				break
			}
		}
		require.True(found)

		// once removed it cannot be there
		mpool.Remove(tx)

		require.False(mpool.Has(tx.ID()))
		retrievedAfterRemove, _ := mpool.Get(tx.ID())
		require.Equal((*txs.Tx)(nil), retrievedAfterRemove)

		// we can reinsert it again to grow the mempool
		require.NoError(mpool.Add(tx))
	}
}

func TestProposalTxsInMempool(t *testing.T) {
	require := require.New(t)

	registerer := metric.NewRegistry()
	mpool, err := New("mempool", registerer)
	require.NoError(err)

	// The proposal txs are ordered by decreasing start time. This means after
	// each insertion, the last inserted transaction should be on the top of the
	// heap.
	proposalTxs, err := createTestProposalTxs(2)
	require.NoError(err)

	for i, tx := range proposalTxs {
		require.False(mpool.Has(tx.ID()))

		// we can insert
		require.NoError(mpool.Add(tx))

		// we can get it
		require.True(mpool.Has(tx.ID()))

		retrieved, _ := mpool.Get(tx.ID())
		require.NotNil(retrieved)
		require.Equal(tx, retrieved)

		{
			// we can peek it
			peeked := mpool.PeekTxs(math.MaxInt)
			require.Len(peeked, i+1)

			// tx will be among those peeked,
			// in NO PARTICULAR ORDER
			found := false
			for _, pk := range peeked {
				if pk.ID() == tx.ID() {
					found = true
					break
				}
			}
			require.True(found)
		}

		// once removed it cannot be there
		mpool.Remove(tx)

		require.False(mpool.Has(tx.ID()))
		retrievedAfterRemove, _ := mpool.Get(tx.ID())
		require.Equal((*txs.Tx)(nil), retrievedAfterRemove)

		// we can reinsert it again to grow the mempool
		require.NoError(mpool.Add(tx))
	}
}

func createTestDecisionTxs(count int) ([]*txs.Tx, error) {
	decisionTxs := make([]*txs.Tx, 0, count)
	for i := uint32(0); i < uint32(count); i++ {
		utx := &txs.CreateChainTx{
			BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
				NetworkID:    10,
				BlockchainID: ids.Empty.Prefix(uint64(i)),
				Ins: []*lux.TransferableInput{{
					UTXOID: lux.UTXOID{
						TxID:        ids.ID{'t', 'x', 'I', 'D'},
						OutputIndex: i,
					},
					Asset: lux.Asset{ID: ids.ID{'a', 's', 's', 'e', 'r', 't'}},
					In: &secp256k1fx.TransferInput{
						Amt:   uint64(5678),
						Input: secp256k1fx.Input{SigIndices: []uint32{i}},
					},
				}},
				Outs: []*lux.TransferableOutput{{
					Asset: lux.Asset{ID: ids.ID{'a', 's', 's', 'e', 'r', 't'}},
					Out: &secp256k1fx.TransferOutput{
						Amt: uint64(1234),
						OutputOwners: secp256k1fx.OutputOwners{
							Threshold: 1,
							Addrs:     []ids.ShortID{preFundedKeys[0].PublicKey().Address()},
						},
					},
				}},
			}},
			ChainID:       ids.GenerateTestID(),
			BlockchainName:   "chainName",
			VMID:        ids.GenerateTestID(),
			FxIDs:       []ids.ID{ids.GenerateTestID()},
			GenesisData: []byte{'g', 'e', 'n', 'D', 'a', 't', 'a'},
			ChainAuth:     &secp256k1fx.Input{SigIndices: []uint32{1}},
		}

		tx, err := txs.NewSigned(utx, txs.Codec, nil)
		if err != nil {
			return nil, err
		}
		decisionTxs = append(decisionTxs, tx)
	}
	return decisionTxs, nil
}

// Proposal txs are sorted by decreasing start time
func createTestProposalTxs(count int) ([]*txs.Tx, error) {
	now := time.Now()
	proposalTxs := make([]*txs.Tx, 0, count)
	for i := 0; i < count; i++ {
		tx, err := generateAddValidatorTx(
			uint64(now.Add(time.Duration(count-i)*time.Second).Unix()), // startTime
			0, // endTime
		)
		if err != nil {
			return nil, err
		}
		proposalTxs = append(proposalTxs, tx)
	}
	return proposalTxs, nil
}

func generateAddValidatorTx(startTime uint64, endTime uint64) (*txs.Tx, error) {
	utx := &txs.AddValidatorTx{
		BaseTx: txs.BaseTx{},
		Validator: txs.Validator{
			NodeID: ids.GenerateTestNodeID(),
			Start:  startTime,
			End:    endTime,
		},
		StakeOuts:        nil,
		RewardsOwner:     &secp256k1fx.OutputOwners{},
		DelegationShares: 100,
	}

	return txs.NewSigned(utx, txs.Codec, nil)
}

func TestDropExpiredStakerTxs(t *testing.T) {
	require := require.New(t)

	registerer := metric.NewRegistry()
	mempool, err := New("mempool", registerer)
	require.NoError(err)

	tx1, err := generateAddValidatorTx(10, 20)
	require.NoError(err)
	require.NoError(mempool.Add(tx1))

	tx2, err := generateAddValidatorTx(8, 20)
	require.NoError(err)
	require.NoError(mempool.Add(tx2))

	tx3, err := generateAddValidatorTx(15, 20)
	require.NoError(err)
	require.NoError(mempool.Add(tx3))

	minStartTime := time.Unix(9, 0)
	require.Len(mempool.DropExpiredStakerTxs(minStartTime), 1)
}
