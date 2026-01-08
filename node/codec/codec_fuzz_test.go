// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package codec_test

import (
	"bytes"
	"math"
	"testing"

	"github.com/luxfi/sdk/node/codec/linearcodec"
	"github.com/luxfi/sdk/node/codec/reflectcodec"
	"github.com/luxfi/sdk/utils/wrappers"
)

// FuzzLinearCodecMarshal tests round-trip marshaling/unmarshaling with linear codec
func FuzzLinearCodecMarshal(f *testing.F) {
	// Seed corpus with known patterns
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{1, 2, 3, 4})
	f.Add([]byte{255, 255, 255, 255})
	f.Add([]byte{0, 0, 0, 1, 255})
	f.Add([]byte{128, 64, 32, 16, 8, 4, 2, 1})
	f.Add(bytes.Repeat([]byte{0xff}, 100))

	codec := linearcodec.NewDefault()

	f.Fuzz(func(t *testing.T, data []byte) {
		// Test that unmarshaling doesn't panic
		p := wrappers.Packer{Bytes: data, MaxSize: 10 * 1024 * 1024} // 10MB limit
		var result []byte
		err := codec.UnmarshalFrom(&p, &result)

		// If unmarshal succeeded, test round-trip
		if err == nil && result != nil {
			// Marshal back
			p2 := wrappers.Packer{MaxSize: 10 * 1024 * 1024}
			if err := codec.MarshalInto(result, &p2); err == nil {
				// Unmarshal again and verify round-trip
				var result2 []byte
				err = codec.UnmarshalFrom(&p2, &result2)
				if err == nil && !bytes.Equal(result, result2) {
					t.Errorf("Round-trip mismatch: original=%x, result=%x", result, result2)
				}
			}
		}
	})
}

// FuzzReflectCodecStruct tests reflect codec with struct marshaling
func FuzzReflectCodecStruct(f *testing.F) {
	type TestStruct struct {
		Field1 uint32   `serialize:"true"`
		Field2 []byte   `serialize:"true"`
		Field3 bool     `serialize:"true"`
		Field4 uint64   `serialize:"true"`
		Field5 string   `serialize:"true"`
	}

	// Seed corpus with edge cases
	f.Add(uint32(0), []byte{}, false, uint64(0), "")
	f.Add(uint32(42), []byte{1, 2, 3}, true, uint64(1234567890), "test")
	f.Add(uint32(0xFFFFFFFF), []byte{255}, false, uint64(0xFFFFFFFFFFFFFFFF), "edge")
	f.Add(uint32(math.MaxUint32), bytes.Repeat([]byte{0xaa}, 256), true, uint64(math.MaxUint64), "max")

	// Create a linearCodec that uses reflectcodec internally
	codec := linearcodec.New([]string{reflectcodec.DefaultTagName})

	f.Fuzz(func(t *testing.T, field1 uint32, field2 []byte, field3 bool, field4 uint64, field5 string) {
		// Limit slice and string lengths to avoid OOM
		if len(field2) > 10000 {
			field2 = field2[:10000]
		}
		if len(field5) > 10000 {
			field5 = field5[:10000]
		}

		original := TestStruct{
			Field1: field1,
			Field2: field2,
			Field3: field3,
			Field4: field4,
			Field5: field5,
		}

		// Marshal
		p := wrappers.Packer{MaxSize: 10 * 1024 * 1024} // 10MB limit
		err := codec.MarshalInto(original, &p)
		if err != nil {
			// Expected for invalid inputs
			return
		}

		// Unmarshal
		var result TestStruct
		p2 := wrappers.Packer{Bytes: p.Bytes, MaxSize: 10 * 1024 * 1024}
		err = codec.UnmarshalFrom(&p2, &result)
		if err != nil {
			t.Errorf("Unmarshal failed after successful marshal: %v", err)
			return
		}

		// Verify round-trip
		if result.Field1 != original.Field1 {
			t.Errorf("Field1 mismatch: got %v, want %v", result.Field1, original.Field1)
		}
		if !bytes.Equal(result.Field2, original.Field2) {
			t.Errorf("Field2 mismatch: got %x, want %x", result.Field2, original.Field2)
		}
		if result.Field3 != original.Field3 {
			t.Errorf("Field3 mismatch: got %v, want %v", result.Field3, original.Field3)
		}
		if result.Field4 != original.Field4 {
			t.Errorf("Field4 mismatch: got %v, want %v", result.Field4, original.Field4)
		}
		if result.Field5 != original.Field5 {
			t.Errorf("Field5 mismatch: got %q, want %q", result.Field5, original.Field5)
		}
	})
}

// FuzzCodecSize tests Size calculation consistency
func FuzzCodecSize(f *testing.F) {
	// Seed corpus with edge cases
	f.Add(uint32(0))
	f.Add(uint32(1))
	f.Add(uint32(127))
	f.Add(uint32(128))
	f.Add(uint32(255))
	f.Add(uint32(256))
	f.Add(uint32(1000))
	f.Add(uint32(65535))
	f.Add(uint32(65536))
	f.Add(uint32(0xFFFFFFFF))

	codec := linearcodec.NewDefault()

	f.Fuzz(func(t *testing.T, value uint32) {
		// Calculate size
		size, err := codec.Size(value)
		if err != nil {
			return
		}

		// Marshal and verify size matches
		p := wrappers.Packer{MaxSize: 1024 * 1024}
		err = codec.MarshalInto(value, &p)
		if err != nil {
			t.Errorf("Marshal failed but Size succeeded: %v", err)
			return
		}

		if len(p.Bytes) != size {
			t.Errorf("Size mismatch: Size() returned %d, Marshal produced %d bytes", size, len(p.Bytes))
		}
	})
}

// FuzzCodecNestedStructs tests codec with deeply nested structures
func FuzzCodecNestedStructs(f *testing.F) {
	type Inner struct {
		A uint16  `serialize:"true"`
		B []byte  `serialize:"true"`
	}

	type Outer struct {
		X uint32  `serialize:"true"`
		Y []Inner `serialize:"true"`
		Z string  `serialize:"true"`
	}

	// Seed corpus
	f.Add(uint32(0), []byte{}, "")
	f.Add(uint32(100), []byte{1, 2, 3, 4, 5}, "nested")
	f.Add(uint32(0xFFFFFFFF), bytes.Repeat([]byte{0xde, 0xad}, 50), "deep")

	// Create a linearCodec that uses reflectcodec internally
	codec := linearcodec.New([]string{reflectcodec.DefaultTagName})

	f.Fuzz(func(t *testing.T, x uint32, innerData []byte, z string) {
		// Limit data sizes
		if len(innerData) > 1000 {
			innerData = innerData[:1000]
		}
		if len(z) > 1000 {
			z = z[:1000]
		}

		// Create nested structure
		inners := []Inner{}
		if len(innerData) > 0 {
			for i := 0; i < len(innerData) && i < 10; i++ {
				inners = append(inners, Inner{
					A: uint16(innerData[i]),
					B: innerData[i:min(i+5, len(innerData))],
				})
			}
		}

		original := Outer{
			X: x,
			Y: inners,
			Z: z,
		}

		// Marshal
		p := wrappers.Packer{MaxSize: 10 * 1024 * 1024}
		err := codec.MarshalInto(original, &p)
		if err != nil {
			return // Expected for some inputs
		}

		// Unmarshal
		var result Outer
		p2 := wrappers.Packer{Bytes: p.Bytes, MaxSize: 10 * 1024 * 1024}
		err = codec.UnmarshalFrom(&p2, &result)
		if err != nil {
			t.Errorf("Unmarshal failed after successful marshal: %v", err)
			return
		}

		// Basic validation
		if result.X != original.X {
			t.Errorf("X mismatch: got %v, want %v", result.X, original.X)
		}
		if len(result.Y) != len(original.Y) {
			t.Errorf("Y length mismatch: got %v, want %v", len(result.Y), len(original.Y))
		}
		if result.Z != original.Z {
			t.Errorf("Z mismatch: got %q, want %q", result.Z, original.Z)
		}
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
