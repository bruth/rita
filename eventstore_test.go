package rita

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nuid"
)

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
	is := is.New(t)

	srv := newNatsServer()
	defer shutdownNatsServer(srv)

	nc, _ := nats.Connect(srv.ClientURL())
	js, _ := nc.JetStream()
	_, err := js.AddStream(&nats.StreamConfig{
		Name: "orders",
		Subjects: []string{
			"orders.>",
		},
		Storage: nats.MemoryStorage,
	})
	is.NoErr(err)

	store, err := NewEventStore(nc, "orders")
	is.NoErr(err)

	ctx := context.Background()

	tests := []struct {
		Name string
		Run  func(t *testing.T, subject string)
	}{
		{
			"append-load-no-occ",
			func(t *testing.T, subject string) {
				event1 := &Event{
					ID:   nuid.Next(),
					Type: "order-created",
					Time: time.Now().Local(),
				}

				seq, err := store.Append(ctx, subject, event1)
				is.NoErr(err)
				is.Equal(seq, uint64(1))

				events, lseq, err := store.Load(ctx, subject)
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
			func(t *testing.T, subject string) {
				event1 := &Event{
					ID:   nuid.Next(),
					Type: "order-created",
					Time: time.Now().Local(),
				}

				seq, err := store.Append(ctx, subject, event1, ExpectSequence(0))
				is.NoErr(err)
				is.Equal(seq, uint64(2))

				event2 := &Event{
					ID:   nuid.Next(),
					Type: "order-shipped",
					Time: time.Now().Local(),
				}

				seq, err = store.Append(ctx, subject, event2, ExpectSequence(2))
				is.NoErr(err)
				is.Equal(seq, uint64(3))

				events, lseq, err := store.Load(ctx, subject)
				is.NoErr(err)

				is.Equal(seq, lseq)
				is.Equal(len(events), 2)
			},
		},
		{
			"append-load-partial",
			func(t *testing.T, subject string) {
				event1 := &Event{
					ID:   nuid.Next(),
					Type: "order-created",
					Time: time.Now().Local(),
				}

				seq, err := store.Append(ctx, subject, event1, ExpectSequence(0))
				is.NoErr(err)
				is.Equal(seq, uint64(4))

				event2 := &Event{
					ID:   nuid.Next(),
					Type: "order-shipped",
					Time: time.Now().Local(),
				}

				seq, err = store.Append(ctx, subject, event2, ExpectSequence(4))
				is.NoErr(err)
				is.Equal(seq, uint64(5))

				events, lseq, err := store.Load(ctx, subject, AfterSequence(4))
				is.NoErr(err)

				is.Equal(seq, lseq)
				is.Equal(len(events), 1)
				is.Equal(*events[0], Event{
					ID:   event2.ID,
					Type: event2.Type,
					Time: event2.Time,
					Seq:  5,
					Data: []byte{},
				})
			},
		},
	}

	for i, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			subject := fmt.Sprintf("orders.%d", i)
			t.Log(subject)
			test.Run(t, subject)
		})
	}
}
