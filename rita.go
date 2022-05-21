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

// TypeRegistry sets an explicit type registry.
func TypeRegistry(types *types.Registry) RitaOption {
	return ritaOption(func(o *Rita) error {
		o.types = types
		return nil
	})
}

// Clock sets a clock implementation. Default it clock.Time.
func Clock(clock clock.Clock) RitaOption {
	return ritaOption(func(o *Rita) error {
		o.clock = clock
		return nil
	})
}

// ID sets a unique ID generator implementation. Default is id.NUID.
func ID(id id.ID) RitaOption {
	return ritaOption(func(o *Rita) error {
		o.id = id
		return nil
	})
}

type Rita struct {
	nc *nats.Conn
	js nats.JetStreamContext

	id    id.ID
	clock clock.Clock
	types *types.Registry
}

func (r *Rita) EventStore(name string) *EventStore {
	return &EventStore{
		name: name,
		rt:   r,
	}
}

// New initializes a new Rita instance with a NATS connection.
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

	return rt, nil
}
