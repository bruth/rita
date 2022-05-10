package rita

import (
	"testing"
	"time"

	"github.com/bruth/rita/codec"
	"github.com/bruth/rita/internal/pb"
	"github.com/bruth/rita/testutil"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNewRegistry(t *testing.T) {
	// Base case.
	type A struct{}

	// Not serializable.
	type B struct {
		C chan int
	}

	tests := map[string]struct {
		Init func() any
		Err  bool
	}{
		"base": {
			func() any { return &A{} },
			false,
		},
		"no-init": {
			nil,
			true,
		},
		"non-pointer": {
			func() any { return A{} },
			true,
		},
		"not-serializable": {
			func() any { return &B{} },
			true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewTypeRegistry(map[string]*Type{
				"a": {
					Init: test.Init,
				},
			})
			if err != nil && !test.Err {
				t.Errorf("unexpected error: %s", err)
			} else if err == nil && test.Err {
				t.Errorf("expected error")
			}
		})
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	is := testutil.NewIs(t)

	ty := map[string]*Type{
		"a": {
			Init: func() any {
				return &pb.A{}
			},
		},
	}

	for k := range codec.Codecs {
		t.Run(k, func(t *testing.T) {
			rt, err := NewTypeRegistry(ty, Codec(k))
			is.NoErr(err)

			v1 := pb.A{
				S: "foo",
				I: 5,
				B: true,
				F: 1.4,
				A: &pb.A{
					I: 10,
				},
				T: timestamppb.New(time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)),
			}

			// Support both struct value and pointers.
			tt, err := rt.Lookup(&v1)
			is.NoErr(err)
			is.Equal(tt, "a")

			b, err := rt.Marshal(&v1)
			is.NoErr(err)

			x, err := rt.Init("a")
			is.NoErr(err)

			err = rt.Unmarshal(b, x)
			is.NoErr(err)

			v2 := x.(*pb.A)
			if !proto.Equal(&v1, v2) {
				t.Error("v1 and v2 differ")
			}
		})
	}
}

func BenchmarkInit(b *testing.B) {
	type T struct{}

	r, _ := NewTypeRegistry(map[string]*Type{
		"a": {
			Init: func() any { return &T{} },
		},
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = r.Init("a")
	}
}

func BenchmarkLookup(b *testing.B) {
	type T struct{}

	r, _ := NewTypeRegistry(map[string]*Type{
		"a": {
			Init: func() any { return &T{} },
		},
	})

	v := &T{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = r.Lookup(v)
	}
}
