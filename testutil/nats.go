package testutil

import (
	"os"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
)

func NewNatsServer() *server.Server {
	opts := natsserver.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true
	return natsserver.RunServer(&opts)
}

func ShutdownNatsServer(s *server.Server) {
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

// ReadSubject reads N messages from a subject.
func ReadSubject(js nats.JetStream, subject string, num int, opts ...nats.SubOpt) ([]*nats.Msg, error) {
	opts = append(opts, nats.OrderedConsumer())
	sub, err := js.SubscribeSync(
		subject,
		opts...,
	)
	if err != nil {
		return nil, err
	}

	var msgs []*nats.Msg
	var n int
	for {
		n++
		msg, err := sub.NextMsg(100 * time.Millisecond)
		if err != nil {
			return nil, err
		}

		msgs = append(msgs, msg)
		if n == num {
			break
		}
	}

	return msgs, nil
}
