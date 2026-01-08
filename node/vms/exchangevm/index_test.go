// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package exchangevm

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/database"
	"github.com/luxfi/metric"

	"github.com/luxfi/database/memdb"

	"github.com/luxfi/database/prefixdb"

	"github.com/luxfi/database/versiondb"

	"github.com/luxfi/ids"

	"github.com/luxfi/sdk/utils"

	"github.com/luxfi/const"

	"github.com/luxfi/crypto/secp256k1"

	"github.com/luxfi/sdk/node/vms/exchangevm/txs"

	"github.com/luxfi/sdk/node/vms/components/index"

	"github.com/luxfi/sdk/node/vms/components/lux"

	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

func TestIndexTransaction_Ordered(t *testing.T) {
	require := require.New(t)

	env := setup(t, &envConfig{fork: durango, indexTransactions: true})
	env.vm.Lock.Unlock()

	key := keys[0]
	addr := key.PublicKey().Address()
	txAssetID := lux.Asset{ID: env.genesisTx.ID()}

	var txs []*txs.Tx
	for i := 0; i < 5; i++ {
		// make utxo
		utxoID := lux.UTXOID{
			TxID: ids.GenerateTestID(),
		}
		utxo := buildUTXO(utxoID, txAssetID, addr)
		env.vm.state.AddUTXO(utxo)
		require.NoError(env.vm.state.Commit())

		// make transaction
		tx := buildTX(env.consensusCtx.ChainID, utxoID, txAssetID, addr)
		require.NoError(tx.SignSECP256K1Fx(env.vm.parser.Codec(), [][]*secp256k1.PrivateKey{{key}}))

		issueAndAccept(require, env.vm, tx)

		txs = append(txs, tx)
	}

	// Debug: try reading directly from addressTxsIndexer
	txIDs, err := env.vm.addressTxsIndexer.Read(addr[:], txAssetID.ID, 0, 5)
	require.NoError(err, "Failed to read from indexer")
	t.Logf("Read %d transactions from indexer for addr %s", len(txIDs), addr)

	// for each tx check its indexed at right index
	for i, tx := range txs {
		assertIndexedTX(t, env.vm.baseDB, uint64(i), addr, txAssetID.ID, tx.ID())
	}
	assertLatestIdx(t, env.vm.baseDB, addr, txAssetID.ID, 5)
}

func TestIndexTransaction_MultipleTransactions(t *testing.T) {
	require := require.New(t)

	env := setup(t, &envConfig{fork: durango, indexTransactions: true})
	env.vm.Lock.Unlock()

	addressTxMap := map[ids.ShortID]*txs.Tx{}
	txAssetID := lux.Asset{ID: env.genesisTx.ID()}

	for _, key := range keys {
		addr := key.PublicKey().Address()

		// make utxo
		utxoID := lux.UTXOID{
			TxID: ids.GenerateTestID(),
		}
		utxo := buildUTXO(utxoID, txAssetID, addr)
		env.vm.state.AddUTXO(utxo)
		require.NoError(env.vm.state.Commit())

		// make transaction
		tx := buildTX(env.consensusCtx.ChainID, utxoID, txAssetID, addr)
		require.NoError(tx.SignSECP256K1Fx(env.vm.parser.Codec(), [][]*secp256k1.PrivateKey{{key}}))

		// issue transaction
		issueAndAccept(require, env.vm, tx)

		addressTxMap[addr] = tx
	}

	// ensure length is same as keys length
	require.Len(addressTxMap, len(keys))

	// for each *UniqueTx check its indexed at right index for the right address
	for addr, tx := range addressTxMap {
		assertIndexedTX(t, env.vm.baseDB, 0, addr, txAssetID.ID, tx.ID())
		assertLatestIdx(t, env.vm.baseDB, addr, txAssetID.ID, 1)
	}
}

func TestIndexTransaction_MultipleAddresses(t *testing.T) {
	require := require.New(t)

	env := setup(t, &envConfig{fork: durango, indexTransactions: true})
	env.vm.Lock.Unlock()

	addrs := make([]ids.ShortID, len(keys))
	for i, key := range keys {
		addrs[i] = key.PublicKey().Address()
	}
	utils.Sort(addrs)

	txAssetID := lux.Asset{ID: env.genesisTx.ID()}

	key := keys[0]
	addr := key.PublicKey().Address()

	// make utxo
	utxoID := lux.UTXOID{
		TxID: ids.GenerateTestID(),
	}
	utxo := buildUTXO(utxoID, txAssetID, addr)
	env.vm.state.AddUTXO(utxo)
	require.NoError(env.vm.state.Commit())

	// make transaction
	tx := buildTX(env.consensusCtx.ChainID, utxoID, txAssetID, addrs...)
	require.NoError(tx.SignSECP256K1Fx(env.vm.parser.Codec(), [][]*secp256k1.PrivateKey{{key}}))

	issueAndAccept(require, env.vm, tx)

	assertIndexedTX(t, env.vm.baseDB, 0, addr, txAssetID.ID, tx.ID())
	assertLatestIdx(t, env.vm.baseDB, addr, txAssetID.ID, 1)
}

func TestIndexer_Read(t *testing.T) {
	require := require.New(t)

	env := setup(t, &envConfig{fork: durango, indexTransactions: true})
	env.vm.Lock.Unlock()

	// generate test address and asset IDs
	assetID := ids.GenerateTestID()
	addr := ids.GenerateTestShortID()

	// setup some fake txs under the above generated address and asset IDs
	testTxs := initTestTxIndex(t, env.vm.baseDB, addr, assetID, 25)
	require.Len(testTxs, 25)

	// read the pages, 5 items at a time
	var (
		cursor   uint64
		pageSize uint64 = 5
	)
	for cursor < 25 {
		txIDs, err := env.vm.addressTxsIndexer.Read(addr[:], assetID, cursor, pageSize)
		require.NoError(err)
		require.Len(txIDs, 5)
		require.Equal(txIDs, testTxs[cursor:cursor+pageSize])
		cursor += pageSize
	}
}

func TestIndexingNewInitWithIndexingEnabled(t *testing.T) {
	require := require.New(t)

	db := memdb.New()

	// start with indexing enabled
	_, err := index.NewIndexer(db, nil, "", metric.NewNoOp().Registry(), true)
	require.NoError(err)

	// now disable indexing with allow-incomplete set to false
	_, err = index.NewNoIndexer(db, false)
	require.ErrorIs(err, index.ErrCausesIncompleteIndex)

	// now disable indexing with allow-incomplete set to true
	_, err = index.NewNoIndexer(db, true)
	require.NoError(err)
}

func TestIndexingNewInitWithIndexingDisabled(t *testing.T) {
	require := require.New(t)

	db := memdb.New()

	// disable indexing with allow-incomplete set to false
	_, err := index.NewNoIndexer(db, false)
	require.NoError(err)

	// It's not OK to have an incomplete index when allowIncompleteIndices is false
	_, err = index.NewIndexer(db, nil, "", metric.NewNoOp().Registry(), false)
	require.ErrorIs(err, index.ErrIndexingRequiredFromGenesis)

	// It's OK to have an incomplete index when allowIncompleteIndices is true
	_, err = index.NewIndexer(db, nil, "", metric.NewNoOp().Registry(), true)
	require.NoError(err)

	// It's OK to have an incomplete index when indexing currently disabled
	_, err = index.NewNoIndexer(db, false)
	require.NoError(err)

	// It's OK to have an incomplete index when allowIncompleteIndices is true
	_, err = index.NewNoIndexer(db, true)
	require.NoError(err)
}

func TestIndexingAllowIncomplete(t *testing.T) {
	require := require.New(t)

	baseDB := memdb.New()
	db := versiondb.New(baseDB)
	// disabled indexer will persist idxEnabled as false
	_, err := index.NewNoIndexer(db, false)
	require.NoError(err)

	// we initialize with indexing enabled now and allow incomplete indexing as false
	_, err = index.NewIndexer(db, nil, "", metric.NewNoOp().Registry(), false)
	// we should get error because:
	// - indexing was disabled previously
	// - node now is asked to enable indexing with allow incomplete set to false
	require.ErrorIs(err, index.ErrIndexingRequiredFromGenesis)
}

func buildUTXO(utxoID lux.UTXOID, txAssetID lux.Asset, addr ids.ShortID) *lux.UTXO {
	return &lux.UTXO{
		UTXOID: utxoID,
		Asset:  txAssetID,
		Out: &secp256k1fx.TransferOutput{
			Amt: startBalance,
			OutputOwners: secp256k1fx.OutputOwners{
				Threshold: 1,
				Addrs:     []ids.ShortID{addr},
			},
		},
	}
}

func buildTX(chainID ids.ID, utxoID lux.UTXOID, txAssetID lux.Asset, address ...ids.ShortID) *txs.Tx {
	return &txs.Tx{Unsigned: &txs.BaseTx{
		BaseTx: lux.BaseTx{
			NetworkID:    constants.UnitTestID,
			BlockchainID: chainID,
			Ins: []*lux.TransferableInput{{
				UTXOID: utxoID,
				Asset:  txAssetID,
				In: &secp256k1fx.TransferInput{
					Amt:   startBalance,
					Input: secp256k1fx.Input{SigIndices: []uint32{0}},
				},
			}},
			Outs: []*lux.TransferableOutput{{
				Asset: txAssetID,
				Out: &secp256k1fx.TransferOutput{
					Amt: startBalance - testTxFee,
					OutputOwners: secp256k1fx.OutputOwners{
						Threshold: 1,
						Addrs:     address,
					},
				},
			}},
		},
	}}
}

func assertLatestIdx(t *testing.T, db database.Database, sourceAddress ids.ShortID, assetID ids.ID, expectedIdx uint64) {
	require := require.New(t)

	addressDB := prefixdb.New(sourceAddress[:], db)
	assetDB := prefixdb.New(assetID[:], addressDB)

	expectedIdxBytes := database.PackUInt64(expectedIdx)
	idxBytes, err := assetDB.Get([]byte("idx"))
	require.NoError(err)
	require.Equal(expectedIdxBytes, idxBytes)
}

func assertIndexedTX(t *testing.T, db database.Database, index uint64, sourceAddress ids.ShortID, assetID ids.ID, transactionID ids.ID) {
	require := require.New(t)

	addressDB := prefixdb.New(sourceAddress[:], db)
	assetDB := prefixdb.New(assetID[:], addressDB)

	idxBytes := database.PackUInt64(index)
	txID, err := database.GetID(assetDB, idxBytes)
	require.NoError(err)
	require.Equal(transactionID, txID)
}

// Sets up test tx IDs in DB in the following structure for the indexer to pick
// them up:
//
//	[address] prefix DB
//	  [assetID] prefix DB
//	    - "idx": 2
//	    - 0: txID1
//	    - 1: txID1
func initTestTxIndex(t *testing.T, db database.Database, address ids.ShortID, assetID ids.ID, txCount int) []ids.ID {
	require := require.New(t)

	testTxs := make([]ids.ID, txCount)
	for i := 0; i < txCount; i++ {
		testTxs[i] = ids.GenerateTestID()
	}

	addressPrefixDB := prefixdb.New(address[:], db)
	assetPrefixDB := prefixdb.New(assetID[:], addressPrefixDB)

	for i, txID := range testTxs {
		idxBytes := database.PackUInt64(uint64(i))
		txID := txID
		require.NoError(assetPrefixDB.Put(idxBytes, txID[:]))
	}

	idxBytes := database.PackUInt64(uint64(len(testTxs)))
	require.NoError(assetPrefixDB.Put([]byte("idx"), idxBytes))
	return testTxs
}
