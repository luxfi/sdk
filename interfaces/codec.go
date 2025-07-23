// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package interfaces

// Codec defines serialization operations
type Codec interface {
	// Marshal serializes an object into bytes
	Marshal(v interface{}) ([]byte, error)
	
	// Unmarshal deserializes bytes into an object
	Unmarshal(b []byte, v interface{}) error
}

// CodecRegistry manages codecs
type CodecRegistry interface {
	// RegisterCodec registers a codec for a type
	RegisterCodec(version uint16, codec Codec) error
	
	// GetCodec gets a codec for a version
	GetCodec(version uint16) (Codec, error)
}

// Marshal is a helper function to marshal an object
func Marshal(codec Codec, v interface{}) ([]byte, error) {
	// Use the codec's Marshal method
	return codec.Marshal(v)
}

// Unmarshal is a helper function to unmarshal bytes
func Unmarshal(codec Codec, b []byte, v interface{}) error {
	// Use the codec's Unmarshal method
	return codec.Unmarshal(b, v)
}