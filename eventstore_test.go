package rita

import (
	"context"
	"fmt"
	"testing"

	"github.com/bruth/rita/id"
	"github.com/bruth/rita/testutil"
	"github.com/bruth/rita/types"
	"github.com/nats-io/nats.go"
)

func TestEventStore(t *testing.T) {
	is := testutil.NewIs(t)

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
				devent := OrderPlaced{ID: "123"}
				seq, err := es.Append(ctx, subject, &Event{
					Data: &devent,
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
				is.Equal(*data, devent)
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

				seq, err := es.Append(ctx, subject, &Event{Data: &OrderPlaced{ID: "123"}}, ExpectSequence(0))
				is.NoErr(err)
				is.Equal(seq, uint64(1))

				seq, err = es.Append(ctx, subject, &Event{Data: &OrderShipped{ID: "123"}}, ExpectSequence(1))
				is.NoErr(err)
				is.Equal(seq, uint64(2))

				events, lseq, err := es.Load(ctx, subject, AfterSequence(1))
				is.NoErr(err)

				is.Equal(seq, lseq)
				is.Equal(len(events), 1)
				is.Equal(events[0].Type, "order-shipped")
			},
		},
		{
			"duplicate-append",
			func(t *testing.T, es *EventStore, subject string) {
				ctx := context.Background()

				e := &Event{
					ID:   id.NUID.New(),
					Data: &OrderPlaced{ID: "123"},
				}

				seq, err := es.Append(ctx, subject, e)
				is.NoErr(err)
				is.Equal(seq, uint64(1))

				// Append same event with same ID, expect the same response.
				seq, err = es.Append(ctx, subject, e)
				is.NoErr(err)
				is.Equal(seq, uint64(1))

				// Append same event with same ID, expect the same response... again.
				seq, err = es.Append(ctx, subject, e)
				is.NoErr(err)
				is.Equal(seq, uint64(1))
			},
		},
	}

	srv := testutil.NewNatsServer()
	defer testutil.ShutdownNatsServer(srv)

	nc, _ := nats.Connect(srv.ClientURL())

	tr, err := types.NewRegistry(map[string]*types.Type{
		"order-placed": {
			Init: func() any { return &OrderPlaced{} },
		},
		"order-shipped": {
			Init: func() any { return &OrderShipped{} },
		},
	})
	is.NoErr(err)

	r, err := New(nc, TypeRegistry(tr))
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
