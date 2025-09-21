// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/luxfi/database"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/cache"
	"github.com/luxfi/node/cache/lru"

	"github.com/luxfi/consensus/protocol/chain"
	luxWarp "github.com/luxfi/warp"
	"github.com/luxfi/warp/payload"

	"github.com/luxfi/evm/plugin/evm/validators/interfaces"
	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/log"
)

var (
	_                         Backend = (*backend)(nil)
	errParsingOffChainMessage         = errors.New("failed to parse off-chain message")

	messageCacheSize = 500
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

	// Verify verifies the signature of the message
	Verify(ctx context.Context, unsignedMessage *luxWarp.UnsignedMessage, _ []byte) error
}

// backend implements Backend, keeps track of warp messages, and generates message signatures.
type backend struct {
	networkID                 uint32
	sourceChainID             ids.ID
	db                        database.Database
	warpSigner                WarpSigner
	blockClient               BlockClient
	validatorReader           interfaces.ValidatorReader
	signatureCache            cache.Cacher[ids.ID, []byte]
	messageCache              *lru.Cache[ids.ID, *luxWarp.UnsignedMessage]
	offchainAddressedCallMsgs map[string]*luxWarp.UnsignedMessage
	stats                     *verifierStats
}

// NewBackend creates a new Backend, and initializes the signature cache and message tracking database.
func NewBackend(
	networkID uint32,
	sourceChainID ids.ID,
	warpSigner WarpSigner,
	blockClient BlockClient,
	validatorReader interfaces.ValidatorReader,
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
		signatureCache:            signatureCache,
		validatorReader:           validatorReader,
		messageCache:              lru.NewCache[ids.ID, *luxWarp.UnsignedMessage](messageCacheSize),
		stats:                     newVerifierStats(),
		offchainAddressedCallMsgs: make(map[string]*luxWarp.UnsignedMessage),
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
			return fmt.Errorf("wrong network ID at index %d", i)
		}

		// Compare source chain IDs
		if !bytes.Equal(unsignedMsg.SourceChainID[:], b.sourceChainID[:]) {
			return fmt.Errorf("wrong source chain ID at index %d", i)
		}

		_, err = payload.ParsePayload(unsignedMsg.Payload)
		if err != nil {
			return fmt.Errorf("%w at index %d as AddressedCall: %w", errParsingOffChainMessage, i, err)
		}
		messageID := unsignedMsg.ID()
		msgIDHash := ids.ID(crypto.Keccak256Hash(messageID[:]))
		b.offchainAddressedCallMsgs[msgIDHash.String()] = unsignedMsg
	}

	return nil
}

func (b *backend) AddMessage(unsignedMessage *luxWarp.UnsignedMessage) error {
	messageIDBytes := unsignedMessage.ID()
	messageID := ids.ID(crypto.Keccak256Hash(messageIDBytes[:]))
	log.Debug("Adding warp message to backend", "messageID", messageID)

	// In the case when a node restarts, and possibly changes its bls key, the cache gets emptied but the database does not.
	// So to avoid having incorrect signatures saved in the database after a bls key change, we save the full message in the database.
	// Whereas for the cache, after the node restart, the cache would be emptied so we can directly save the signatures.
	if err := b.db.Put(messageID[:], unsignedMessage.Bytes()); err != nil {
		return fmt.Errorf("failed to put warp signature in db: %w", err)
	}

	if _, err := b.signMessage(unsignedMessage); err != nil {
		return fmt.Errorf("failed to sign warp message: %w", err)
	}
	return nil
}

func (b *backend) GetMessageSignature(ctx context.Context, unsignedMessage *luxWarp.UnsignedMessage) ([]byte, error) {
	messageIDBytes := unsignedMessage.ID()
	messageID := ids.ID(crypto.Keccak256Hash(messageIDBytes[:]))

	log.Debug("Getting warp message from backend", "messageID", messageID)
	if sig, ok := b.signatureCache.Get(messageID); ok {
		return sig, nil
	}

	if err := b.Verify(ctx, unsignedMessage, nil); err != nil {
		return nil, fmt.Errorf("failed to validate warp message: %w", err)
	}
	return b.signMessage(unsignedMessage)
}

func (b *backend) GetBlockSignature(ctx context.Context, blockID ids.ID) ([]byte, error) {
	log.Debug("Getting block from backend", "blockID", blockID)

	blockHashPayload, err := payload.NewHash(blockID[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create new block hash payload: %w", err)
	}

	unsignedMessage, err := luxWarp.NewUnsignedMessage(b.networkID, b.sourceChainID, blockHashPayload.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to create new unsigned warp message: %w", err)
	}

	messageIDBytes := unsignedMessage.ID()
	messageID := ids.ID(crypto.Keccak256Hash(messageIDBytes[:]))
	if sig, ok := b.signatureCache.Get(messageID); ok {
		return sig, nil
	}

	if err := b.verifyBlockMessage(ctx, blockHashPayload); err != nil {
		return nil, fmt.Errorf("failed to validate block message: %w", err)
	}

	sig, err := b.signMessage(unsignedMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to sign block message: %w", err)
	}
	return sig, nil
}

func (b *backend) GetMessage(messageID ids.ID) (*luxWarp.UnsignedMessage, error) {
	if message, ok := b.messageCache.Get(messageID); ok {
		return message, nil
	}
	messageIDStr := messageID.String()
	if message, ok := b.offchainAddressedCallMsgs[messageIDStr]; ok {
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
	// TODO: implement signing with luxfi/warp
	// For now, return empty signature
	sig := make([]byte, 64)

	messageIDBytes := unsignedMessage.ID()
	messageID := ids.ID(crypto.Keccak256Hash(messageIDBytes[:]))
	b.signatureCache.Put(messageID, sig)
	return sig, nil
}
