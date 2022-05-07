package codec

import (
	"github.com/vmihailenco/msgpack/v5"
)

var (
	MsgPack Codec = &msgpackCodec{}
)

type msgpackCodec struct{}

func (*msgpackCodec) Marshal(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (*msgpackCodec) Unmarshal(b []byte, v interface{}) error {
	return msgpack.Unmarshal(b, v)
}
