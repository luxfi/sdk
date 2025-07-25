// (c) 2023, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/metrics"
	"github.com/luxfi/evm/plugin/evm/message"
	syncHandlers "github.com/luxfi/evm/sync/handlers"
	syncStats "github.com/luxfi/evm/sync/handlers/stats"
	"github.com/luxfi/geth/trie"
	"github.com/luxfi/geth/triedb"
	"github.com/luxfi/evm/warp"
	warpHandlers "github.com/luxfi/evm/warp/handlers"
)

var _ message.RequestHandler = &networkHandler{}

type networkHandler struct {
	stateTrieLeafsRequestHandler *syncHandlers.LeafsRequestHandler
	blockRequestHandler          *syncHandlers.BlockRequestHandler
	codeRequestHandler           *syncHandlers.CodeRequestHandler
	signatureRequestHandler      *warpHandlers.SignatureRequestHandler
}

// newNetworkHandler constructs the handler for serving network requests.
func newNetworkHandler(
	provider syncHandlers.SyncDataProvider,
	diskDB ethdb.KeyValueReader,
	evmTrieDB *triedb.Database,
	warpBackend warp.Backend,
	networkCodec interfaces.Codec,
) message.RequestHandler {
	syncStats := syncStats.NewHandlerStats(metrics.Enabled)
	return &networkHandler{
		stateTrieLeafsRequestHandler: syncHandlers.NewLeafsRequestHandler(evmTrieDB, provider, networkCodec, syncStats),
		blockRequestHandler:          syncHandlers.NewBlockRequestHandler(provider, networkCodec, syncStats),
		codeRequestHandler:           syncHandlers.NewCodeRequestHandler(diskDB, networkCodec, syncStats),
		signatureRequestHandler:      warpHandlers.NewSignatureRequestHandler(warpBackend, networkCodec),
	}
}

func (n networkHandler) HandleStateTrieLeafsRequest(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, leafsRequest message.LeafsRequest) ([]byte, error) {
	return n.stateTrieLeafsRequestHandler.OnLeafsRequest(ctx, nodeID, requestID, leafsRequest)
}

func (n networkHandler) HandleBlockRequest(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, blockRequest message.BlockRequest) ([]byte, error) {
	return n.blockRequestHandler.OnBlockRequest(ctx, nodeID, requestID, blockRequest)
}

func (n networkHandler) HandleCodeRequest(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, codeRequest message.CodeRequest) ([]byte, error) {
	return n.codeRequestHandler.OnCodeRequest(ctx, nodeID, requestID, codeRequest)
}

func (n networkHandler) HandleMessageSignatureRequest(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, messageSignatureRequest message.MessageSignatureRequest) ([]byte, error) {
	return n.signatureRequestHandler.OnMessageSignatureRequest(ctx, nodeID, requestID, messageSignatureRequest)
}

func (n networkHandler) HandleBlockSignatureRequest(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, blockSignatureRequest message.BlockSignatureRequest) ([]byte, error) {
	return n.signatureRequestHandler.OnBlockSignatureRequest(ctx, nodeID, requestID, blockSignatureRequest)
}
