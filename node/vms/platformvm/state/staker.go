// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"bytes"
	"time"

	"github.com/google/btree"

	"github.com/luxfi/ids"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
)

// StakerIterator is an iterator for Staker objects.
// Iterators should be released when they are no longer needed.
type StakerIterator interface {
	// Next advances the iterator to the next staker.
	// Returns false if there are no more stakers.
	Next() bool

	// Value returns the current staker.
	// Should only be called after Next() returns true.
	Value() *Staker

	// Release frees any resources associated with the iterator.
	// Must be called when the iterator is no longer needed.
	Release()
}

var _ btree.LessFunc[*Staker] = (*Staker).Less

// Staker contains all information required to represent a validator or
// delegator in the current and pending validator sets.
// Invariant: Staker's size is bounded to prevent OOM DoS attacks.
type Staker struct {
	TxID            ids.ID
	NodeID          ids.NodeID
	PublicKey       *bls.PublicKey
	ChainID         ids.ID
	Weight          uint64
	StartTime       time.Time
	EndTime         time.Time
	PotentialReward uint64

	// NextTime is the next time this staker will be moved from a validator set.
	// If the staker is in the pending validator set, NextTime will equal
	// StartTime. If the staker is in the current validator set, NextTime will
	// equal EndTime.
	NextTime time.Time

	// Priority specifies how to break ties between stakers with the same
	// NextTime. This ensures that stakers created by the same transaction type
	// are grouped together. The ordering of these groups is documented in
	// [priorities.go] and depends on if the stakers are in the pending or
	// current validator set.
	Priority txs.Priority

	// ValidatorNFT contains NFT information if this validator is using NFT staking
	ValidatorNFT *txs.ValidatorNFTInfo
}

// A *Staker is considered to be less than another *Staker when:
//
//  1. If its NextTime is before the other's.
//  2. If the NextTimes are the same, the *Staker with the lesser priority is the
//     lesser one.
//  3. If the priorities are also the same, the one with the lesser txID is
//     lesser.
func (s *Staker) Less(than *Staker) bool {
	if s.NextTime.Before(than.NextTime) {
		return true
	}
	if than.NextTime.Before(s.NextTime) {
		return false
	}

	if s.Priority < than.Priority {
		return true
	}
	if than.Priority < s.Priority {
		return false
	}

	return bytes.Compare(s.TxID[:], than.TxID[:]) == -1
}

func NewCurrentStaker(
	txID ids.ID,
	staker txs.Staker,
	startTime time.Time,
	potentialReward uint64,
) (*Staker, error) {
	publicKey, _, err := staker.PublicKey()
	if err != nil {
		return nil, err
	}
	endTime := staker.EndTime()
	return &Staker{
		TxID:            txID,
		NodeID:          staker.NodeID(),
		PublicKey:       publicKey,
		ChainID:           staker.ChainID(),
		Weight:          staker.Weight(),
		StartTime:       startTime,
		EndTime:         endTime,
		PotentialReward: potentialReward,
		NextTime:        endTime,
		Priority:        staker.CurrentPriority(),
	}, nil
}

func NewPendingStaker(txID ids.ID, staker txs.ScheduledStaker) (*Staker, error) {
	publicKey, _, err := staker.PublicKey()
	if err != nil {
		return nil, err
	}
	startTime := staker.StartTime()
	return &Staker{
		TxID:      txID,
		NodeID:    staker.NodeID(),
		PublicKey: publicKey,
		ChainID:     staker.ChainID(),
		Weight:    staker.Weight(),
		StartTime: startTime,
		EndTime:   staker.EndTime(),
		NextTime:  startTime,
		Priority:  staker.PendingPriority(),
	}, nil
}
