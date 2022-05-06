package rita

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	eventTypeHdr    = "Rita-Event-Type"
	eventTimeHdr    = "Rita-Event-Time"
	eventTimeFormat = time.RFC3339Nano
)

var (
	ErrSequenceConflict  = errors.New("rita: sequence conflict")
	ErrEventIDRequired   = errors.New("rita: event id required")
	ErrEventTypeRequired = errors.New("rita: event type required")
)

// Pack an event into a NATS message. The advantage of using NATS headers
// is that the server supports creating a consumer that _only_ gets the headers
// without the data as an optimization for some use cases.
func packEvent(subject string, event *Event, eventTime time.Time) *nats.Msg {
	msg := nats.NewMsg(subject)
	msg.Data = event.Data
	msg.Header.Set(nats.MsgIdHdr, event.ID)
	msg.Header.Set(eventTypeHdr, event.Type)
	msg.Header.Set(eventTimeHdr, eventTime.Format(eventTimeFormat))
	return msg
}

// Unpack an event from a NATS message.
func unpackEvent(msg *nats.Msg) (*Event, error) {
	md, err := msg.Metadata()
	if err != nil {
		return nil, fmt.Errorf("unpack: failed to get metadata: %s", err)
	}

	eventTime, err := time.Parse(eventTimeFormat, msg.Header.Get(eventTimeHdr))
	if err != nil {
		return nil, fmt.Errorf("unpack: failed to parse event time: %s", err)
	}

	return &Event{
		ID:   msg.Header.Get(nats.MsgIdHdr),
		Type: msg.Header.Get(eventTypeHdr),
		Time: eventTime,
		Data: msg.Data,
		Seq:  md.Sequence.Stream,
	}, nil
}

// Event is a wrapper type for encoded events.
type Event struct {
	// ID of the event. NUID, ULID, XID, or UUID are recommended.
	// This will be used as the NATS msg ID for de-duplication.
	ID string

	// Type is a unique name for the event itself.
	Type string

	// Time is the time of when the event occurred which may be different
	// from the time the event is appended to the store. If no time is provided,
	// the current time will be used.
	Time time.Time

	// Data is the encoded event data.
	Data []byte

	// Seq is the sequence where this event exists in the stream.
	Seq uint64
}

// EventStore provides an event store abstraction on a NATS stream.
type EventStore interface {
	// Load fetches all events for a specific subject. This is expected to be a concrete
	// subject (without wildcards) so that the sequence would be fed in as the expected
	// sequence for a subsequent append operation.
	Load(ctx context.Context, subject string, opts ...LoadOption) ([]*Event, uint64, error)

	// Append appends a new event to the subject's event sequence. It returns the resulting
	// sequence number of the appended event.
	Append(ctx context.Context, subject string, event *Event, opts ...AppendOption) (uint64, error)
}

type appendOpts struct {
	expSeq *uint64
}

type appendOptFn func(o *appendOpts) error

func (f appendOptFn) appendOpt(o *appendOpts) error {
	return f(o)
}

// AppendOption is an option for the event store Append operation.
type AppendOption interface {
	appendOpt(o *appendOpts) error
}

// ExpectSequence indicates that the expected sequence of the subject sequence should
// be the value provided. If not, a conflict is indicated.
func ExpectSequence(seq uint64) AppendOption {
	return appendOptFn(func(o *appendOpts) error {
		o.expSeq = &seq
		return nil
	})
}

type loadOpts struct {
	afterSeq *uint64
}

type loadOptFn func(o *loadOpts) error

func (f loadOptFn) loadOpt(o *loadOpts) error {
	return f(o)
}

// LoadOption is an option for the event store Load operation.
type LoadOption interface {
	loadOpt(o *loadOpts) error
}

// AfterSequence specifies the sequence of the first event that should be fetched
// from the sequence up to the end of the sequence. This useful when partially applied
// state has been derived up to a specific sequence and only the latest events need
// to be fetched.
func AfterSequence(seq uint64) LoadOption {
	return loadOptFn(func(o *loadOpts) error {
		o.afterSeq = &seq
		return nil
	})
}

type natsApiError struct {
	Code        int    `json:"code"`
	ErrCode     uint16 `json:"err_code"`
	Description string `json:"description"`
}

type natsGetMsgRequest struct {
	LastBySubject string `json:"last_by_subj"`
}

type natsGetMsgResponse struct {
	Type    string         `json:"type"`
	Error   *natsApiError  `json:"error"`
	Message *natsStoredMsg `json:"message"`
}

type natsStoredMsg struct {
	Sequence uint64 `json:"seq"`
}

type eventStore struct {
	stream string
	nc     *nats.Conn
	js     nats.JetStream
}

// lastSeqForSubject queries the JS API to identify the current latest sequence for a subject.
// This is used as an best-guess indicator of the current end of the even history.
func (s *eventStore) lastMsgForSubject(ctx context.Context, subject string) (*natsStoredMsg, error) {
	rsubject := fmt.Sprintf("$JS.API.STREAM.MSG.GET.%s", s.stream)

	data, _ := json.Marshal(&natsGetMsgRequest{
		LastBySubject: subject,
	})

	msg, err := s.nc.RequestWithContext(ctx, rsubject, data)
	if err != nil {
		return nil, err
	}

	var rep natsGetMsgResponse
	err = json.Unmarshal(msg.Data, &rep)
	if err != nil {
		return nil, err
	}

	if rep.Error != nil {
		if rep.Error.Code == 404 {
			return &natsStoredMsg{}, nil
		}
		return nil, fmt.Errorf("%s (%d)", rep.Error.Description, rep.Error.Code)
	}

	return rep.Message, nil
}

func (s *eventStore) Load(ctx context.Context, subject string, opts ...LoadOption) ([]*Event, uint64, error) {
	// Configure opts.
	var o loadOpts
	for _, opt := range opts {
		if err := opt.loadOpt(&o); err != nil {
			return nil, 0, err
		}
	}

	lastMsg, err := s.lastMsgForSubject(ctx, subject)
	if err != nil {
		return nil, 0, err
	}

	if lastMsg.Sequence == 0 {
		return nil, 0, nil
	}

	// Ephemeral ordered consumer.. read as fast as possible with least overhead.
	sopts := []nats.SubOpt{
		nats.OrderedConsumer(),
	}

	// Don't bother creating the consumer if the last seq is smaller than start.
	if o.afterSeq != nil {
		if lastMsg.Sequence <= *o.afterSeq {
			return nil, 0, nil
		}
		sopts = append(sopts, nats.StartSequence(*o.afterSeq))
	} else {
		sopts = append(sopts, nats.DeliverAll())
	}

	sub, err := s.js.SubscribeSync(subject, sopts...)
	if err != nil {
		return nil, 0, err
	}

	// Skip first.
	if o.afterSeq != nil {
		_, err := sub.NextMsgWithContext(ctx)
		if err != nil {
			return nil, 0, err
		}
	}

	var events []*Event
	for {
		msg, err := sub.NextMsgWithContext(ctx)
		if err != nil {
			return nil, 0, err
		}

		event, err := unpackEvent(msg)
		if err != nil {
			return nil, 0, err
		}

		events = append(events, event)

		if event.Seq == lastMsg.Sequence {
			break
		}
	}

	return events, lastMsg.Sequence, nil
}

func (s *eventStore) Append(ctx context.Context, subject string, event *Event, opts ...AppendOption) (uint64, error) {
	// Configure opts.
	var o appendOpts
	for _, opt := range opts {
		if err := opt.appendOpt(&o); err != nil {
			return 0, err
		}
	}

	popts := []nats.PubOpt{
		nats.Context(ctx),
		nats.ExpectStream(s.stream),
	}

	if o.expSeq != nil {
		popts = append(popts, nats.ExpectLastSequencePerSubject(*o.expSeq))
	}

	if event.ID == "" {
		return 0, ErrEventIDRequired
	}

	if event.Type == "" {
		return 0, ErrEventTypeRequired
	}

	eventTime := event.Time
	if eventTime.IsZero() {
		eventTime = time.Now().Local()
	}

	msg := packEvent(subject, event, eventTime)

	ack, err := s.js.PublishMsg(msg, popts...)
	if err != nil {
		if strings.Contains(err.Error(), "wrong last sequence") {
			return 0, ErrSequenceConflict
		}
		return 0, err
	}

	return ack.Sequence, nil
}

type EventStoreManager interface {
	EventStore(name string) EventStore
	CreateEventStore(config *EventStoreConfig) (EventStore, error)
	UpdateEventStore(config *EventStoreConfig) error
	DeleteEventStore(name string) error
}

// EventStoreConfig is a subset of the nats.StreamConfig for the purpose of creating
// purpose-built streams for an event store.
type EventStoreConfig struct {
	// Name of the event store and underlying stream.
	Name string
	// Description associated with the event store.
	Description string
	// Subjects to associated with the stream. If not specified, it will default to
	// the name plus the variadic wildcard, e.g. "orders.>"
	Subjects []string
	// Storage for the stream.
	Storage nats.StorageType
	// Replicas of the stream.
	Replicas int
	// Placement of the stream replicas.
	Placement *nats.Placement
}

type eventStoreManager struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func (m *eventStoreManager) EventStore(name string) EventStore {
	return &eventStore{
		stream: name,
		nc:     m.nc,
		js:     m.js,
	}
}

func (m *eventStoreManager) CreateEventStore(config *EventStoreConfig) (EventStore, error) {
	subjects := config.Subjects
	if len(subjects) == 0 {
		subjects = []string{fmt.Sprintf("%s.>", config.Name)}
	}

	_, err := m.js.AddStream(&nats.StreamConfig{
		Name:        config.Name,
		Description: config.Description,
		Subjects:    subjects,
		Storage:     config.Storage,
		Replicas:    config.Replicas,
		Placement:   config.Placement,
		DenyDelete:  true,
		DenyPurge:   true,
	})
	if err != nil {
		return nil, err
	}

	return m.EventStore(config.Name), nil
}

func (m *eventStoreManager) UpdateEventStore(config *EventStoreConfig) error {
	_, err := m.js.UpdateStream(&nats.StreamConfig{
		Name:        config.Name,
		Description: config.Description,
		Subjects:    config.Subjects,
		Storage:     config.Storage,
		Replicas:    config.Replicas,
		Placement:   config.Placement,
		DenyDelete:  true,
		DenyPurge:   true,
	})
	return err
}

func (m *eventStoreManager) DeleteEventStore(name string) error {
	return m.js.DeleteStream(name)
}
