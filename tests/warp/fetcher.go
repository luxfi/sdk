// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"context"
	"fmt"

	"github.com/luxfi/ids"
	"github.com/luxfi/crypto/bls"
	luxWarp "github.com/luxfi/warp"
	"github.com/luxfi/warp/payload"
	warpBackend "github.com/luxfi/evm/warp"
)

type apiFetcher struct {
	clients map[ids.NodeID]warpBackend.Client
}

func NewAPIFetcher(clients map[ids.NodeID]warpBackend.Client) *apiFetcher {
	return &apiFetcher{
		clients: clients,
	}
}

func (f *apiFetcher) GetSignature(ctx context.Context, nodeID ids.NodeID, unsignedWarpMessage *luxWarp.UnsignedMessage) (*bls.Signature, error) {
	client, ok := f.clients[nodeID]
	if !ok {
		return nil, fmt.Errorf("no warp client for nodeID: %s", nodeID)
	}
	var signatureBytes []byte
	parsedPayload, err := payload.ParsePayload(unsignedWarpMessage.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse unsigned message payload: %w", err)
	}
	switch p := parsedPayload.(type) {
	case *payload.AddressedCall:
			msgID := unsignedWarpMessage.ID()
		signatureBytes, err = client.GetMessageSignature(ctx, msgID)
	case *payload.Hash:
			blockID, _ := ids.ToID(p.Hash)
		signatureBytes, err = client.GetBlockSignature(ctx, blockID)
	}
	if err != nil {
		return nil, err
	}

	signature, err := bls.SignatureFromBytes(signatureBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse signature from client %s: %w", nodeID, err)
	}
	return signature, nil
}
