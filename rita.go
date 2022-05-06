package rita

import "github.com/nats-io/nats.go"

type Rita interface {
	EventStoreManager
}

type rita struct {
	*eventStoreManager
}

func New(nc *nats.Conn) (Rita, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}

	esm := &eventStoreManager{
		nc: nc,
		js: js,
	}

	return &rita{
		eventStoreManager: esm,
	}, nil
}
