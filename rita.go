package rita

import (
	"github.com/nats-io/nats.go"
)

type Rita struct {
	Types       *TypeRegistry
	EventStores *EventStoreManager

	nc *nats.Conn
	js nats.JetStreamContext

	id    ID
	clock Clock
}

// New initializes a new Rita instance given a NATS connection and the type registry.
func New(nc *nats.Conn, tr *TypeRegistry) (*Rita, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}

	rt := &Rita{
		Types: tr,
		nc:    nc,
		js:    js,
		id:    NUID,
		clock: Time,
	}

	rt.EventStores = &EventStoreManager{
		rt: rt,
	}

	return rt, nil
}
