package pb

import "google.golang.org/protobuf/proto"

func (a *A) MarshalBinary() ([]byte, error) {
	return proto.Marshal(a)
}

func (a *A) UnmarshalBinary(b []byte) error {
	return proto.Unmarshal(b, a)
}
