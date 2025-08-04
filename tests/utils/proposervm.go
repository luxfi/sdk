// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/crypto"
	"github.com/luxfi/log"
	ethparams "github.com/luxfi/geth/params"
	"github.com/luxfi/evm/ethclient"
	"github.com/luxfi/evm/plugin/evm/upgrade/legacy"
)

const numTriggerTxs = 2 // Number of txs needed to activate the proposer VM fork

// IssueTxsToActivateProposerVMFork issues transactions at the current
// timestamp, which should be after the ProposerVM activation time (aka
// ApricotPhase4). This should generate a PostForkBlock because its parent block
// (genesis) has a timestamp (0) that is greater than or equal to the fork
// activation time of 0. Therefore, subsequent blocks should be built with
// BuildBlockWithContext.
func IssueTxsToActivateProposerVMFork(
	ctx context.Context, chainID *big.Int, fundedKey *ecdsa.PrivateKey,
	client ethclient.Client,
) error {
	addr := crypto.PubkeyToAddress(fundedKey.PublicKey)
	nonce, err := client.NonceAt(ctx, addr, nil)
	if err != nil {
		return err
	}

	newHeads := make(chan *types.Header, 1)
	sub, err := client.SubscribeNewHead(ctx, newHeads)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	gasPrice := big.NewInt(legacy.BaseFee)
	txSigner := types.LatestSignerForChainID(chainID)
	for i := 0; i < numTriggerTxs; i++ {
		tx := types.NewTransaction(
			nonce, addr, common.Big1, ethparams.TxGas, gasPrice, nil)
		triggerTx, err := types.SignTx(tx, txSigner, fundedKey)
		if err != nil {
			return err
		}
		if err := client.SendTransaction(ctx, triggerTx); err != nil {
			return err
		}
		<-newHeads // wait for block to be accepted
		nonce++
	}
	log.Info(
		"Built sufficient blocks to activate proposerVM fork",
		"txCount", numTriggerTxs,
	)
	return nil
}
