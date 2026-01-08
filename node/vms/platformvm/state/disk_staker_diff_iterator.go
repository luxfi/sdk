// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"encoding/binary"
	"fmt"

	"github.com/luxfi/database"
	"github.com/luxfi/ids"
)

const (
	// startDiffKey = [netID] + [inverseHeight]
	startDiffKeyLength = ids.IDLen + database.Uint64Size
	// diffKey = [netID] + [inverseHeight] + [nodeID]
	diffKeyLength = startDiffKeyLength + ids.NodeIDLen
	// diffKeyNodeIDOffset = [netIDLen] + [inverseHeightLen]
	diffKeyNodeIDOffset = ids.IDLen + database.Uint64Size

	// weightValue = [isNegative] + [weight] + [validationID]
	weightValueLength = database.BoolSize + database.Uint64Size + ids.IDLen
)

var (
	errUnexpectedDiffKeyLength     = fmt.Errorf("expected diff key length %d", diffKeyLength)
	errUnexpectedWeightValueLength = fmt.Errorf("expected weight value length %d", weightValueLength)
)

// marshalStartDiffKey is used to determine the starting key when iterating.
//
// Invariant: the result is a prefix of [marshalDiffKey] when called with the
// same arguments.
func marshalStartDiffKey(netID ids.ID, height uint64) []byte {
	key := make([]byte, startDiffKeyLength)
	copy(key, netID[:])
	packIterableHeight(key[ids.IDLen:], height)
	return key
}

func marshalDiffKey(netID ids.ID, height uint64, nodeID ids.NodeID) []byte {
	key := make([]byte, diffKeyLength)
	copy(key, netID[:])
	packIterableHeight(key[ids.IDLen:], height)
	copy(key[diffKeyNodeIDOffset:], nodeID.Bytes())
	return key
}

func unmarshalDiffKey(key []byte) (ids.ID, uint64, ids.NodeID, error) {
	if len(key) != diffKeyLength {
		return ids.Empty, 0, ids.EmptyNodeID, errUnexpectedDiffKeyLength
	}
	var (
		netID  ids.ID
		nodeID ids.NodeID
	)
	copy(netID[:], key)
	height := unpackIterableHeight(key[ids.IDLen:])
	copy(nodeID[:], key[diffKeyNodeIDOffset:])
	return netID, height, nodeID, nil
}

func marshalWeightDiff(diff *ValidatorWeightDiff) []byte {
	value := make([]byte, weightValueLength)
	if diff.Decrease {
		value[0] = database.BoolTrue
	}
	binary.BigEndian.PutUint64(value[database.BoolSize:], diff.Amount)
	copy(value[database.BoolSize+database.Uint64Size:], diff.ValidationID[:])
	return value
}

func unmarshalWeightDiff(value []byte) (*ValidatorWeightDiff, error) {
	if len(value) != weightValueLength {
		return nil, errUnexpectedWeightValueLength
	}
	var validationID ids.ID
	copy(validationID[:], value[database.BoolSize+database.Uint64Size:])
	return &ValidatorWeightDiff{
		Decrease:     value[0] == database.BoolTrue,
		Amount:       binary.BigEndian.Uint64(value[database.BoolSize:]),
		ValidationID: validationID,
	}, nil
}

// Note: [height] is encoded as a bit flipped big endian number so that
// iterating lexicographically results in iterating in decreasing heights.
//
// Invariant: [key] has sufficient length
func packIterableHeight(key []byte, height uint64) {
	binary.BigEndian.PutUint64(key, ^height)
}

// Because we bit flip the height when constructing the key, we must remember to
// bip flip again here.
//
// Invariant: [key] has sufficient length
func unpackIterableHeight(key []byte) uint64 {
	return ^binary.BigEndian.Uint64(key)
}
