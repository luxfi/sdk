// Copyright (C) 2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

package contract

import (
	"fmt"

	"github.com/luxfi/formatting"
	"github.com/luxfi/ids"
	"github.com/luxfi/rpc"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/vm/vms/platformvm/txs"
)

type getTxArgs struct {
	TxID     ids.ID              `json:"txID"`
	Encoding formatting.Encoding `json:"encoding"`
}

type formattedTx struct {
	Tx       string              `json:"tx"`
	Encoding formatting.Encoding `json:"encoding"`
}

// GetBlockchainTx retrieves a blockchain CreateChainTx from the network.
func GetBlockchainTx(endpoint string, blockchainID ids.ID) (*txs.CreateChainTx, error) {
	requester := rpc.NewEndpointRequester(endpoint + "/ext/P")
	ctx, cancel := utils.GetAPIContext()
	defer cancel()

	reply := &formattedTx{}
	if err := requester.SendRequest(ctx, "platform.getTx", &getTxArgs{
		TxID:     blockchainID,
		Encoding: formatting.Hex,
	}, reply); err != nil {
		return nil, err
	}

	txBytes, err := formatting.Decode(reply.Encoding, reply.Tx)
	if err != nil {
		return nil, err
	}

	var tx txs.Tx
	if _, err = txs.Codec.Unmarshal(txBytes, &tx); err != nil {
		return nil, fmt.Errorf("failed unmarshaling the createChainTx: %w", err)
	}
	createChainTx, ok := tx.Unsigned.(*txs.CreateChainTx)
	if !ok {
		return nil, fmt.Errorf("expected a CreateChainTx, got %T", tx.Unsigned)
	}
	return createChainTx, nil
}
