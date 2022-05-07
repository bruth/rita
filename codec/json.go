package codec

import "encoding/json"

var (
	JSON Codec = &jsonCodec{}
)

type jsonCodec struct{}

func (*jsonCodec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)

}
func (*jsonCodec) Unmarshal(b []byte, v interface{}) error {
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, v)
}
