package rita

import (
	"github.com/bruth/rita/clock"
	"github.com/bruth/rita/id"
	"github.com/bruth/rita/types"
	"github.com/nats-io/nats.go"
)

type ritaOption func(o *Rita) error

func (f ritaOption) addOption(o *Rita) error {
	return f(o)
}

// RegistryOption models a option when creating a type registry.
type RitaOption interface {
	addOption(o *Rita) error
}

func TypeRegistry(types *types.Registry) RitaOption {
	return ritaOption(func(o *Rita) error {
		o.types = types
		return nil
	})
}

func Clock(clock clock.Clock) RitaOption {
	return ritaOption(func(o *Rita) error {
		o.clock = clock
		return nil
	})
}

func ID(id id.ID) RitaOption {
	return ritaOption(func(o *Rita) error {
		o.id = id
		return nil
	})
}

type Rita struct {
	EventStores *EventStoreManager

	nc *nats.Conn
	js nats.JetStreamContext

	id    id.ID
	clock clock.Clock
	types *types.Registry
}

// New initializes a new Rita instance given a NATS connection and the type registry.
func New(nc *nats.Conn, opts ...RitaOption) (*Rita, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}

	rt := &Rita{
		nc:    nc,
		js:    js,
		id:    id.NUID,
		clock: clock.Time,
	}

	for _, o := range opts {
		if err := o.addOption(rt); err != nil {
			return nil, err
		}
	}

	rt.EventStores = &EventStoreManager{
		rt: rt,
	}

	return rt, nil
}
