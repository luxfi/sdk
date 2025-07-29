// (c) 2023, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"context"
	"errors"
	"fmt"
	"github.com/luxfi/node/cache"
	"github.com/luxfi/node/database"
	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/network/p2p/lp118"
	"github.com/luxfi/node/consensus/chain"
	"github.com/luxfi/node/consensus/validators"
	luxWarp "github.com/luxfi/node/vms/platformvm/warp"
	"github.com/luxfi/node/vms/platformvm/warp/payload"
	"github.com/luxfi/geth/log"
)

var (
	_                         Backend = &backend{}
	errParsingOffChainMessage         = errors.New("failed to parse off-chain message")

	messageCacheSize = 500
	batchSize        = 4096
)

type BlockClient interface {
	GetAcceptedBlock(ctx context.Context, blockID ids.ID) (chain.Block, error)
}

// Backend tracks signature-eligible warp messages and provides an interface to fetch them.
// The backend is also used to query for warp message signatures by the signature request handler.
type Backend interface {
	// AddMessage signs [unsignedMessage] and adds it to the warp backend database
	AddMessage(unsignedMessage *luxWarp.UnsignedMessage) error

	// GetMessageSignature validates the message and returns the signature of the requested message.
	GetMessageSignature(ctx context.Context, message *luxWarp.UnsignedMessage) ([]byte, error)

	// GetBlockSignature returns the signature of a hash payload containing blockID if it's the ID of an accepted block.
	GetBlockSignature(ctx context.Context, blockID ids.ID) ([]byte, error)

	// GetMessage retrieves the [unsignedMessage] from the warp backend database if available
	GetMessage(messageHash ids.ID) (*luxWarp.UnsignedMessage, error)

	lp118.Verifier
}

// backend implements Backend, keeps track of warp messages, and generates message signatures.
type backend struct {
	networkID                 uint32
	sourceChainID             ids.ID
	db                        database.Database
	warpSigner                luxWarp.Signer
	blockClient               BlockClient
	messageSignatureCache     cache.Cacher[ids.ID, []byte]
	blockSignatureCache       cache.Cacher[ids.ID, []byte]
	messageCache              cache.Cacher[ids.ID, *luxWarp.UnsignedMessage]
	offchainAddressedCallMsgs map[ids.ID]*luxWarp.UnsignedMessage
	stats                     *verifierStats
	validatorReader           validators.State
}

// NewBackend creates a new Backend, and initializes the signature cache and message tracking database.
func NewBackend(
	networkID uint32,
	sourceChainID ids.ID,
	warpSigner luxWarp.Signer,
	blockClient BlockClient,
	validatorReader validators.State,
	db database.Database,
	signatureCache cache.Cacher[ids.ID, []byte],
	offchainMessages [][]byte,
) (Backend, error) {
	b := &backend{
		networkID:                 networkID,
		sourceChainID:             sourceChainID,
		db:                        db,
		warpSigner:                warpSigner,
		blockClient:               blockClient,
		messageSignatureCache:     signatureCache,
		blockSignatureCache:       signatureCache,
		messageCache:              &cache.Empty[ids.ID, *luxWarp.UnsignedMessage]{},
		offchainAddressedCallMsgs: make(map[ids.ID]*luxWarp.UnsignedMessage),
		stats:                     newVerifierStats(),
		validatorReader:           validatorReader,
	}
	return b, b.initOffChainMessages(offchainMessages)
}

func (b *backend) initOffChainMessages(offchainMessages [][]byte) error {
	for i, offchainMsg := range offchainMessages {
		unsignedMsg, err := luxWarp.ParseUnsignedMessage(offchainMsg)
		if err != nil {
			return fmt.Errorf("%w at index %d: %w", errParsingOffChainMessage, i, err)
		}

		if unsignedMsg.NetworkID != b.networkID {
			return fmt.Errorf("%w at index %d", luxWarp.ErrWrongNetworkID, i)
		}

		if unsignedMsg.SourceChainID != b.sourceChainID {
			return fmt.Errorf("%w at index %d", luxWarp.ErrWrongSourceChainID, i)
		}

		_, err = payload.ParseAddressedCall(unsignedMsg.Payload)
		if err != nil {
			return fmt.Errorf("%w at index %d as AddressedCall: %w", errParsingOffChainMessage, i, err)
		}
		b.offchainAddressedCallMsgs[unsignedMsg.ID()] = unsignedMsg
	}

	return nil
}

func (b *backend) Clear() error {
	b.messageSignatureCache.Flush()
	b.blockSignatureCache.Flush()
	b.messageCache.Flush()
	return database.Clear(b.db, batchSize)
}

func (b *backend) AddMessage(unsignedMessage *luxWarp.UnsignedMessage) error {
	messageID := unsignedMessage.ID()
	log.Debug("Adding warp message to backend", "messageID", messageID)

	// In the case when a node restarts, and possibly changes its bls key, the cache gets emptied but the database does not.
	// So to avoid having incorrect signatures saved in the database after a bls key change, we save the full message in the database.
	// Whereas for the cache, after the node restart, the cache would be emptied so we can directly save the signatures.
	if err := b.db.Put(messageID[:], unsignedMessage.Bytes()); err != nil {
		return fmt.Errorf("failed to put warp signature in db: %w", err)
	}

	sig, err := b.signMessage(unsignedMessage)
	if err != nil {
		return fmt.Errorf("failed to sign warp message: %w", err)
	}
	b.messageSignatureCache.Put(messageID, sig)
	return nil
}

func (b *backend) GetMessageSignature(ctx context.Context, unsignedMessage *luxWarp.UnsignedMessage) ([]byte, error) {
	messageID := unsignedMessage.ID()

	log.Debug("Getting warp message from backend", "messageID", messageID)
	if sig, ok := b.messageSignatureCache.Get(messageID); ok {
		return sig, nil
	}

	if err := b.Verify(ctx, unsignedMessage, nil); err != nil {
		return nil, fmt.Errorf("failed to validate warp message: %w", err)
	}
	return b.signMessage(unsignedMessage)
}

func (b *backend) GetBlockSignature(ctx context.Context, blockID ids.ID) ([]byte, error) {
	log.Debug("Getting block from backend", "blockID", blockID)

	blockHashPayload, err := payload.NewHash(blockID)
	if err != nil {
		return nil, fmt.Errorf("failed to create new block hash payload: %w", err)
	}

	unsignedMessage, err := luxWarp.NewUnsignedMessage(b.networkID, b.sourceChainID, blockHashPayload.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to create new unsigned warp message: %w", err)
	}

	if sig, ok := b.blockSignatureCache.Get(unsignedMessage.ID()); ok {
		return sig, nil
	}

	if err := b.verifyBlockMessage(ctx, blockHashPayload); err != nil {
		return nil, fmt.Errorf("failed to validate block message: %w", err)
	}

	sig, err := b.signMessage(unsignedMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to sign block message: %w", err)
	}
	b.blockSignatureCache.Put(unsignedMessage.ID(), sig)
	return sig, nil
}

func (b *backend) GetMessage(messageID ids.ID) (*luxWarp.UnsignedMessage, error) {
	if message, ok := b.messageCache.Get(messageID); ok {
		return message, nil
	}
	if message, ok := b.offchainAddressedCallMsgs[messageID]; ok {
		return message, nil
	}

	unsignedMessageBytes, err := b.db.Get(messageID[:])
	if err != nil {
		return nil, err
	}

	unsignedMessage, err := luxWarp.ParseUnsignedMessage(unsignedMessageBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse unsigned message %s: %w", messageID.String(), err)
	}
	b.messageCache.Put(messageID, unsignedMessage)

	return unsignedMessage, nil
}

func (b *backend) signMessage(unsignedMessage *luxWarp.UnsignedMessage) ([]byte, error) {
	sig, err := b.warpSigner.Sign(unsignedMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to sign warp message: %w", err)
	}

	// Cache the signature 
	b.messageSignatureCache.Put(unsignedMessage.ID(), sig)
	return sig, nil
}
