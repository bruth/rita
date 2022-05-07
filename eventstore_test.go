package rita

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
)

func newIs(t *testing.T) *Is {
	return &Is{t}
}

type Is struct {
	t *testing.T
}

func (is *Is) Equal(a, b any) {
	if d := cmp.Diff(a, b); d != "" {
		is.t.Error(d)
	}
}

func (is *Is) NoErr(err error) {
	if err != nil {
		is.t.Error(err)
	}
}

func (is *Is) True(t bool) {
	if !t {
		is.t.Error("expected true")
	}
}

func newNatsServer() *server.Server {
	opts := natsserver.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true
	return natsserver.RunServer(&opts)
}

func shutdownNatsServer(s *server.Server) {
	var sd string
	if config := s.JetStreamConfig(); config != nil {
		sd = config.StoreDir
	}
	s.Shutdown()
	if sd != "" {
		os.RemoveAll(sd)
	}
	s.WaitForShutdown()
}

func TestEventStore(t *testing.T) {
	is := newIs(t)

	type OrderPlaced struct {
		ID string
	}

	type OrderShipped struct {
		ID string
	}

	tests := []struct {
		Name string
		Run  func(t *testing.T, es *EventStore, subject string)
	}{
		{
			"append-load-no-occ",
			func(t *testing.T, es *EventStore, subject string) {
				ctx := context.Background()
				seq, err := es.Append(ctx, subject, &OrderPlaced{
					ID: "123",
				})
				is.NoErr(err)
				is.Equal(seq, uint64(1))

				events, lseq, err := es.Load(ctx, subject)
				is.NoErr(err)

				is.Equal(seq, lseq)
				is.Equal(len(events), 1)

				is.True(events[0].ID != "")
				is.True(!events[0].Time.IsZero())
				is.Equal(events[0].Type, "order-placed")
				data, ok := events[0].Data.(*OrderPlaced)
				is.True(ok)
				is.Equal(*data, OrderPlaced{ID: "123"})
			},
		},
		{
			"append-load-with-occ",
			func(t *testing.T, es *EventStore, subject string) {
				ctx := context.Background()

				event1 := &Event{
					Data: &OrderPlaced{ID: "123"},
					Meta: map[string]string{
						"geo": "eu",
					},
				}

				seq, err := es.Append(ctx, subject, event1, ExpectSequence(0))
				is.NoErr(err)
				is.Equal(seq, uint64(1))

				event2 := &Event{
					Data: &OrderShipped{ID: "123"},
				}

				seq, err = es.Append(ctx, subject, event2, ExpectSequence(1))
				is.NoErr(err)
				is.Equal(seq, uint64(2))

				events, lseq, err := es.Load(ctx, subject)
				is.NoErr(err)

				is.Equal(seq, lseq)
				is.Equal(len(events), 2)
			},
		},
		{
			"append-load-partial",
			func(t *testing.T, es *EventStore, subject string) {
				ctx := context.Background()

				seq, err := es.Append(ctx, subject, &OrderPlaced{ID: "123"}, ExpectSequence(0))
				is.NoErr(err)
				is.Equal(seq, uint64(1))

				seq, err = es.Append(ctx, subject, &OrderShipped{ID: "123"}, ExpectSequence(1))
				is.NoErr(err)
				is.Equal(seq, uint64(2))

				events, lseq, err := es.Load(ctx, subject, AfterSequence(1))
				is.NoErr(err)

				is.Equal(seq, lseq)
				is.Equal(len(events), 1)
				is.Equal(events[0].Type, "order-shipped")
			},
		},
	}

	srv := newNatsServer()
	defer shutdownNatsServer(srv)

	nc, _ := nats.Connect(srv.ClientURL())

	tr, err := NewTypeRegistry(map[string]*Type{
		"order-placed": {
			Init: func() any { return &OrderPlaced{} },
		},
		"order-shipped": {
			Init: func() any { return &OrderShipped{} },
		},
	})
	is.NoErr(err)

	r, err := New(nc, tr)
	is.NoErr(err)

	for i, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// Recreate the store for each test.
			_ = r.EventStores.Delete("orders")

			es, err := r.EventStores.Create(&EventStoreConfig{
				Name:    "orders",
				Storage: nats.MemoryStorage,
			})
			is.NoErr(err)

			subject := fmt.Sprintf("orders.%d", i)
			test.Run(t, es, subject)
		})
	}
}
