package rita

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nuid"
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

	tests := []struct {
		Name string
		Run  func(t *testing.T, es EventStore, subject string)
	}{
		{
			"append-load-no-occ",
			func(t *testing.T, es EventStore, subject string) {
				ctx := context.Background()
				event1 := &Event{
					ID:   nuid.Next(),
					Type: "order-created",
					Time: time.Now().Local(),
				}

				seq, err := es.Append(ctx, subject, event1)
				is.NoErr(err)
				is.Equal(seq, uint64(1))

				events, lseq, err := es.Load(ctx, subject)
				is.NoErr(err)

				is.Equal(seq, lseq)
				is.Equal(len(events), 1)
				is.Equal(*events[0], Event{
					ID:   event1.ID,
					Type: event1.Type,
					Time: event1.Time,
					Seq:  1,
					Data: []byte{},
				})
			},
		},
		{
			"append-load-with-occ",
			func(t *testing.T, es EventStore, subject string) {
				ctx := context.Background()

				event1 := &Event{
					ID:   nuid.Next(),
					Type: "order-created",
					Time: time.Now().Local(),
				}

				seq, err := es.Append(ctx, subject, event1, ExpectSequence(0))
				is.NoErr(err)
				is.Equal(seq, uint64(1))

				event2 := &Event{
					ID:   nuid.Next(),
					Type: "order-shipped",
					Time: time.Now().Local(),
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
			func(t *testing.T, es EventStore, subject string) {
				ctx := context.Background()

				event1 := &Event{
					ID:   nuid.Next(),
					Type: "order-created",
					Time: time.Now().Local(),
				}

				seq, err := es.Append(ctx, subject, event1, ExpectSequence(0))
				is.NoErr(err)
				is.Equal(seq, uint64(1))

				event2 := &Event{
					ID:   nuid.Next(),
					Type: "order-shipped",
					Time: time.Now().Local(),
				}

				seq, err = es.Append(ctx, subject, event2, ExpectSequence(1))
				is.NoErr(err)
				is.Equal(seq, uint64(2))

				events, lseq, err := es.Load(ctx, subject, AfterSequence(1))
				is.NoErr(err)

				is.Equal(seq, lseq)
				is.Equal(len(events), 1)
				is.Equal(*events[0], Event{
					ID:   event2.ID,
					Type: event2.Type,
					Time: event2.Time,
					Seq:  2,
					Data: []byte{},
				})
			},
		},
	}

	srv := newNatsServer()
	defer shutdownNatsServer(srv)

	nc, _ := nats.Connect(srv.ClientURL())

	r, err := New(nc)
	is.NoErr(err)

	for i, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// Recreate store each time.
			_ = r.DeleteEventStore("orders")
			es, err := r.CreateEventStore(&EventStoreConfig{
				Name:    "orders",
				Storage: nats.MemoryStorage,
			})
			is.NoErr(err)

			subject := fmt.Sprintf("orders.%d", i)
			t.Log(subject)
			test.Run(t, es, subject)
		})
	}
}
