package codec

import "errors"

var (
	ErrCodecNotRegistered = errors.New("rita: codec not registered")

	Codecs = map[string]Codec{
		"json":     JSON,
		"msgpack":  MsgPack,
		"protobuf": ProtoBuf,
	}

	Mimes = map[string]string{
		"json":     "application/json",
		"msgpack":  "application/msgpack",
		"protobuf": "application/protobuf",
	}
)

type Codec interface {
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
}
