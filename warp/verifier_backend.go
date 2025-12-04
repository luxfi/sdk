// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"context"
	"errors"
	"fmt"

	"github.com/luxfi/evm/warp/messages"

	"github.com/luxfi/crypto"
	"github.com/luxfi/database"
	"github.com/luxfi/ids"
	luxWarp "github.com/luxfi/warp"
	"github.com/luxfi/warp/payload"
)

const (
	ParseErrCode = iota + 1
	VerifyErrCode
)

// Verify verifies the signature of the message
// It also implements the lp118.Verifier interface
func (b *backend) Verify(ctx context.Context, unsignedMessage *luxWarp.UnsignedMessage, _ []byte) error {
	messageIDBytes := unsignedMessage.ID()
	messageID := ids.ID(crypto.Keccak256Hash(messageIDBytes[:]))
	// Known on-chain messages should be signed
	if _, err := b.GetMessage(messageID); err == nil {
		return nil
	} else if err != database.ErrNotFound {
		return fmt.Errorf("failed to get message %s: %w", messageID, err)
	}

	parsed, err := payload.ParsePayload(unsignedMessage.Payload)
	if err != nil {
		b.stats.IncMessageParseFail()
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	switch p := parsed.(type) {
	case *payload.AddressedCall:
		return b.verifyOffchainAddressedCall(p)
	case *payload.Hash:
		return b.verifyBlockMessage(ctx, p)
	default:
		b.stats.IncMessageParseFail()
		return fmt.Errorf("unknown payload type: %T", p)
	}
}

// verifyBlockMessage returns nil if blockHashPayload contains the ID
// of an accepted block indicating it should be signed by the VM.
func (b *backend) verifyBlockMessage(ctx context.Context, blockHashPayload *payload.Hash) error {
	// Convert []byte to ids.ID
	var blockID ids.ID
	copy(blockID[:], blockHashPayload.Hash)
	_, err := b.blockClient.GetAcceptedBlock(ctx, blockID)
	if err != nil {
		b.stats.IncBlockValidationFail()
		return fmt.Errorf("failed to get block %s: %w", blockID, err)
	}

	return nil
}

// verifyOffchainAddressedCall verifies the addressed call message
func (b *backend) verifyOffchainAddressedCall(addressedCall *payload.AddressedCall) error {
	// Further, parse the payload to see if it is a known type.
	parsed, err := messages.Parse(addressedCall.Payload)
	if err != nil {
		b.stats.IncMessageParseFail()
		return fmt.Errorf("failed to parse addressed call message: %w", err)
	}

	if len(addressedCall.SourceAddress) != 0 {
		return errors.New("source address should be empty for offchain addressed messages")
	}

	switch p := parsed.(type) {
	case *messages.ValidatorUptime:
		if err := b.verifyUptimeMessage(p); err != nil {
			b.stats.IncUptimeValidationFail()
			return err
		}
	default:
		b.stats.IncMessageParseFail()
		return fmt.Errorf("unknown message type: %T", p)
	}

	return nil
}

func (b *backend) verifyUptimeMessage(uptimeMsg *messages.ValidatorUptime) error {
	vdr, currentUptime, _, err := b.validatorReader.GetValidatorAndUptime(uptimeMsg.ValidationID)
	if err != nil {
		return fmt.Errorf("failed to get uptime for validationID %s: %w", uptimeMsg.ValidationID, err)
	}

	currentUptimeSeconds := uint64(currentUptime.Seconds())
	// verify the current uptime against the total uptime in the message
	if currentUptimeSeconds < uptimeMsg.TotalUptime {
		return fmt.Errorf("current uptime %d is less than queried uptime %d for nodeID %s", currentUptimeSeconds, uptimeMsg.TotalUptime, vdr.NodeID)
	}

	return nil
}
