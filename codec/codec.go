package codec

import (
	"errors"
	"fmt"
)

var (
	ErrNotRegistered = errors.New("rita: codec not registered")

	Default = JSON

	Codecs = []string{
		"json",
		"msgpack",
		"protobuf",
		"binary",
	}

	Registry = &codecRegistry{
		m: map[string]Codec{
			"json":     JSON,
			"msgpack":  MsgPack,
			"protobuf": ProtoBuf,
			"binary":   Binary,
		},
	}
)

type codecRegistry struct {
	m map[string]Codec
}

func (c *codecRegistry) Get(name string) (Codec, error) {
	x, ok := c.m[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotRegistered, name)
	}
	return x, nil
}

type Codec interface {
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
}
