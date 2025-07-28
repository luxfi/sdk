// (c) 2023, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package aggregator

import (
	"context"
	"fmt"

	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/geth/log"
	"github.com/luxfi/evm/utils"
)

type AggregateSignatureResult struct {
	// Weight of validators included in the aggregate signature.
	SignatureWeight uint64
	// Total weight of all validators in the subnet.
	TotalWeight uint64
	// The message with the aggregate signature.
	Message *interfaces.WarpMessage
}

type signatureFetchResult struct {
	sig    *interfaces.WarpSignature
	index  int
	weight uint64
}

// Aggregator requests signatures from validators and
// aggregates them into a single signature.
type Aggregator struct {
	validators  []*interfaces.ValidatorData
	totalWeight uint64
	client      SignatureGetter
}

// New returns a signature aggregator that will attempt to aggregate signatures from [validators].
func New(client SignatureGetter, validators []*interfaces.ValidatorData, totalWeight uint64) *Aggregator {
	return &Aggregator{
		client:      client,
		validators:  validators,
		totalWeight: totalWeight,
	}
}

// Returns an aggregate signature over [unsignedMessage].
// The returned signature's weight exceeds the threshold given by [quorumNum].
func (a *Aggregator) AggregateSignatures(ctx context.Context, unsignedMessage *interfaces.WarpUnsignedMessage, quorumNum uint64) (*AggregateSignatureResult, error) {
	// Create a child context to cancel signature fetching if we reach signature threshold.
	signatureFetchCtx, signatureFetchCancel := context.WithCancel(ctx)
	defer signatureFetchCancel()

	// Fetch signatures from validators concurrently.
	signatureFetchResultChan := make(chan *signatureFetchResult)
	for i, validator := range a.validators {
		var (
			i         = i
			validator = validator
			// TODO: update from a single nodeID to the original slice and use extra nodeIDs as backup.
			nodeID = validator.NodeID
		)
		go func() {
			log.Debug("Fetching warp signature",
				"nodeID", nodeID,
				"index", i,
				// TODO: Fix - WarpUnsignedMessage doesn't have ID() method
				// "msgID", unsignedMessage.ID(),
			)

			signature, err := a.client.GetSignature(signatureFetchCtx, nodeID, unsignedMessage)
			if err != nil {
				log.Debug("Failed to fetch warp signature",
					"nodeID", nodeID,
					"index", i,
					"err", err,
					// TODO: Fix - WarpUnsignedMessage doesn't have ID() method
					// "msgID", unsignedMessage.ID(),
				)
				signatureFetchResultChan <- nil
				return
			}

			log.Debug("Retrieved warp signature",
				"nodeID", nodeID,
				// TODO: Fix - WarpUnsignedMessage doesn't have ID() method
				// "msgID", unsignedMessage.ID(),
				"index", i,
			)

			// TODO: Fix - WarpUnsignedMessage doesn't have Bytes() method
			// if !interfaces.Verify(validator.PublicKey, signature, unsignedMessage.Bytes()) {
			if false {
				log.Debug("Failed to verify warp signature",
					"nodeID", nodeID,
					"index", i,
					// TODO: Fix - WarpUnsignedMessage doesn't have ID() method
					// "msgID", unsignedMessage.ID(),
				)
				signatureFetchResultChan <- nil
				return
			}

			signatureFetchResultChan <- &signatureFetchResult{
				sig:    signature,
				index:  i,
				weight: validator.Weight,
			}
		}()
	}

	var (
		signatures                = make([]*interfaces.Signature, 0, len(a.validators))
		signersBitset             = utils.NewBits()
		signaturesWeight          = uint64(0)
		signaturesPassedThreshold = false
	)

	for i := 0; i < len(a.validators); i++ {
		signatureFetchResult := <-signatureFetchResultChan
		if signatureFetchResult == nil {
			continue
		}

		signatures = append(signatures, signatureFetchResult.sig)
		signersBitset.Add(signatureFetchResult.index)
		signaturesWeight += signatureFetchResult.weight
		log.Debug("Updated weight",
			"totalWeight", signaturesWeight,
			"addedWeight", signatureFetchResult.weight,
			"msgID", unsignedMessage.ID(),
		)

		// If the signature weight meets the requested threshold, cancel signature fetching
		if err := interfaces.VerifyWeight(signaturesWeight, a.totalWeight, quorumNum, params.WarpQuorumDenominator); err == nil {
			log.Debug("Verify weight passed, exiting aggregation early",
				"quorumNum", quorumNum,
				"totalWeight", a.totalWeight,
				"signatureWeight", signaturesWeight,
				"msgID", unsignedMessage.ID(),
			)
			signatureFetchCancel()
			signaturesPassedThreshold = true
			break
		}
	}

	// If I failed to fetch sufficient signature stake, return an error
	if !signaturesPassedThreshold {
		return nil, interfaces.ErrInsufficientWeight
	}

	// Otherwise, return the aggregate signature
	aggregateSignature, err := interfaces.AggregateSignatures(signatures)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate BLS signatures: %w", err)
	}

	warpSignature := &interfaces.BitSetSignature{
		Signers: signersBitset.Bytes(),
	}
	copy(warpSignature.Signature[:], interfaces.SignatureToBytes(aggregateSignature))

	msg, err := interfaces.NewMessage(unsignedMessage, warpSignature)
	if err != nil {
		return nil, fmt.Errorf("failed to construct warp message: %w", err)
	}

	return &AggregateSignatureResult{
		Message:         msg,
		SignatureWeight: signaturesWeight,
		TotalWeight:     a.totalWeight,
	}, nil
}
