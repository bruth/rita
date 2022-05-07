package codec

import (
	"testing"
	"time"
)

func BenchmarkMsgPackMarshal(b *testing.B) {
	type T struct {
		String string
		Int    int
		Bool   bool
		Float  float32
		Struct *T
		Time   time.Time
		Bytes  []byte
	}

	v1 := &T{
		String: "foo",
		Int:    5,
		Bool:   true,
		Float:  1.4,
		Struct: &T{
			Int: 10,
		},
		Time:  time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC),
		Bytes: []byte(`{"foo": "bar", "baz": 3.4}`),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		MsgPack.Marshal(v1)
	}
}

func BenchmarkMsgPackUnmarshal(b *testing.B) {
	type T struct {
		String string
		Int    int
		Bool   bool
		Float  float32
		Struct *T
		Time   time.Time
		Bytes  []byte
	}

	v1 := &T{
		String: "foo",
		Int:    5,
		Bool:   true,
		Float:  1.4,
		Struct: &T{
			Int: 10,
		},
		Time:  time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC),
		Bytes: []byte(`{"foo": "bar", "baz": 3.4}`),
	}

	y, _ := MsgPack.Marshal(v1)
	var v2 T

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		MsgPack.Unmarshal(y, &v2)
	}
}
