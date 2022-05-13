package codec

import (
	"encoding"
	"fmt"
)

var (
	Binary Codec = &binaryCodec{}
)

type binaryCodec struct{}

func (c *binaryCodec) Marshal(v interface{}) ([]byte, error) {
	// Check for native implementation.
	if m, ok := v.(encoding.BinaryMarshaler); ok {
		return m.MarshalBinary()
	}

	// Otherwise assume byte slice.
	b, ok := v.([]byte)
	if !ok {
		return nil, fmt.Errorf("value not []byte")
	}

	return b, nil
}

func (c *binaryCodec) Unmarshal(b []byte, v interface{}) error {
	// Check for native implementation.
	if u, ok := v.(encoding.BinaryUnmarshaler); ok {
		return u.UnmarshalBinary(b)
	}

	// Otherwise assume byte slice.
	bp, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("value must be *[]byte")
	}

	*bp = append((*bp)[:0], b...)
	return nil
}
