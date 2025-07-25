// (c) 2024, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"context"
	"fmt"

	"github.com/luxfi/evm/warp/messages"

	"github.com/luxfi/node/database"
	engine "github.com/luxfi/node/consensus/engine/core"
	luxWarp "github.com/luxfi/node/vms/platformvm/warp"
	"github.com/luxfi/node/vms/platformvm/warp/payload"
)

const (
	ParseErrCode = iota + 1
	VerifyErrCode
)

// Verify verifies the signature of the message
// It also implements the lp118.Verifier interface
func (b *backend) Verify(ctx context.Context, unsignedMessage *luxWarp.UnsignedMessage, _ []byte) *engine.AppError {
	messageID := unsignedMessage.ID()
	// Known on-chain messages should be signed
	if _, err := b.GetMessage(messageID); err == nil {
		return nil
	} else if err != database.ErrNotFound {
		return &engine.AppError{
			Code:    ParseErrCode,
			Message: fmt.Sprintf("failed to get message %s: %s", messageID, err.Error()),
		}
	}

	parsed, err := payload.Parse(unsignedMessage.Payload)
	if err != nil {
		b.stats.IncMessageParseFail()
		return &engine.AppError{
			Code:    ParseErrCode,
			Message: "failed to parse payload: " + err.Error(),
		}
	}

	switch p := parsed.(type) {
	case *payload.AddressedCall:
		return b.verifyOffchainAddressedCall(p)
	case *payload.Hash:
		return b.verifyBlockMessage(ctx, p)
	default:
		b.stats.IncMessageParseFail()
		return &engine.AppError{
			Code:    ParseErrCode,
			Message: fmt.Sprintf("unknown payload type: %T", p),
		}
	}
}

// verifyBlockMessage returns nil if blockHashPayload contains the ID
// of an accepted block indicating it should be signed by the VM.
func (b *backend) verifyBlockMessage(ctx context.Context, blockHashPayload *payload.Hash) *engine.AppError {
	blockID := blockHashPayload.Hash
	_, err := b.blockClient.GetAcceptedBlock(ctx, blockID)
	if err != nil {
		b.stats.IncBlockValidationFail()
		return &engine.AppError{
			Code:    VerifyErrCode,
			Message: fmt.Sprintf("failed to get block %s: %s", blockID, err.Error()),
		}
	}

	return nil
}

// verifyOffchainAddressedCall verifies the addressed call message
func (b *backend) verifyOffchainAddressedCall(addressedCall *payload.AddressedCall) *engine.AppError {
	// Further, parse the payload to see if it is a known type.
	parsed, err := messages.Parse(addressedCall.Payload)
	if err != nil {
		b.stats.IncMessageParseFail()
		return &engine.AppError{
			Code:    ParseErrCode,
			Message: "failed to parse addressed call message: " + err.Error(),
		}
	}

	if len(addressedCall.SourceAddress) != 0 {
		return &engine.AppError{
			Code:    VerifyErrCode,
			Message: "source address should be empty for offchain addressed messages",
		}
	}

	switch p := parsed.(type) {
	case *messages.ValidatorUptime:
		if err := b.verifyUptimeMessage(p); err != nil {
			b.stats.IncUptimeValidationFail()
			return err
		}
	default:
		b.stats.IncMessageParseFail()
		return &engine.AppError{
			Code:    ParseErrCode,
			Message: fmt.Sprintf("unknown message type: %T", p),
		}
	}

	return nil
}

func (b *backend) verifyUptimeMessage(uptimeMsg *messages.ValidatorUptime) *engine.AppError {
	// FIXME: GetValidatorAndUptime method doesn't exist in validators.State interface
	// vdr, currentUptime, _, err := b.validatorReader.GetValidatorAndUptime(uptimeMsg.ValidationID)
	var err error
	if err == nil {
		err = fmt.Errorf("GetValidatorAndUptime not implemented")
	}
	b.stats.IncUptimeValidationFail()
	return &engine.AppError{
		Code:    VerifyErrCode,
		Message: fmt.Sprintf("uptime verification not implemented: %s", err.Error()),
	}
}
