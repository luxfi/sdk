// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package json provides JSON serialization utilities for numeric types.
package json

import "strconv"

const Null = "null"

// Uint32 is a uint32 that can be JSON marshaled as a string.
type Uint32 uint32

func (u Uint32) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strconv.FormatUint(uint64(u), 10) + `"`), nil
}

func (u *Uint32) UnmarshalJSON(b []byte) error {
	str := string(b)
	if str == Null {
		return nil
	}
	if len(str) >= 2 {
		if lastIndex := len(str) - 1; str[0] == '"' && str[lastIndex] == '"' {
			str = str[1:lastIndex]
		}
	}
	val, err := strconv.ParseUint(str, 10, 32)
	*u = Uint32(val)
	return err
}

// Uint64 is a uint64 that can be JSON marshaled as a string.
type Uint64 uint64

func (u Uint64) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strconv.FormatUint(uint64(u), 10) + `"`), nil
}

func (u *Uint64) UnmarshalJSON(b []byte) error {
	str := string(b)
	if str == Null {
		return nil
	}
	if len(str) >= 2 {
		if lastIndex := len(str) - 1; str[0] == '"' && str[lastIndex] == '"' {
			str = str[1:lastIndex]
		}
	}
	val, err := strconv.ParseUint(str, 10, 64)
	*u = Uint64(val)
	return err
}

// Float64 is a float64 that can be JSON marshaled as a string.
type Float64 float64

func (f Float64) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strconv.FormatFloat(float64(f), 'f', 4, 64) + `"`), nil
}

func (f *Float64) UnmarshalJSON(b []byte) error {
	str := string(b)
	if str == Null {
		return nil
	}
	if len(str) >= 2 {
		if lastIndex := len(str) - 1; str[0] == '"' && str[lastIndex] == '"' {
			str = str[1:lastIndex]
		}
	}
	val, err := strconv.ParseFloat(str, 64)
	*f = Float64(val)
	return err
}
