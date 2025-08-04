// Copyright (C) 2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package address

import (
	"errors"

	"github.com/btcsuite/btcd/btcutil/bech32"
)

var (
	errBits5To8 = errors.New("unable to convert address from 5-bit to 8-bit formatting")
	errBits8To5 = errors.New("unable to convert address from 8-bit to 5-bit formatting")
)

// FormatBech32 formats a byte slice as a bech32 address with the given HRP
func FormatBech32(hrp string, data []byte) (string, error) {
	if hrp == "" {
		return "", errors.New("empty HRP")
	}
	if len(data) == 0 {
		return "", errors.New("empty data")
	}

	// Convert 8-bit bytes to 5-bit groups as required by bech32
	fiveBits, err := bech32.ConvertBits(data, 8, 5, true)
	if err != nil {
		return "", errBits8To5
	}

	return bech32.Encode(hrp, fiveBits)
}

// ParseBech32 parses a bech32 address and returns the HRP and data
func ParseBech32(addr string) (string, []byte, error) {
	if addr == "" {
		return "", nil, errors.New("empty address")
	}

	hrp, decoded, err := bech32.Decode(addr)
	if err != nil {
		return "", nil, err
	}

	// Convert 5-bit groups back to 8-bit bytes
	addrBytes, err := bech32.ConvertBits(decoded, 5, 8, true)
	if err != nil {
		return "", nil, errBits5To8
	}

	return hrp, addrBytes, nil
}