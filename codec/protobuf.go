package codec

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

var (
	ProtoBuf Codec = &protoBufCodec{}
)

type protoBufCodec struct{}

func (*protoBufCodec) Name() string {
	return "protobuf"
}

func (*protoBufCodec) Marshal(v interface{}) ([]byte, error) {
	m, ok := v.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("%w: not a proto.Message", proto.Error)
	}
	return proto.Marshal(m)
}

func (*protoBufCodec) Unmarshal(b []byte, v interface{}) error {
	m, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("%w: not a proto.Message", proto.Error)
	}
	return proto.Unmarshal(b, m)
}
