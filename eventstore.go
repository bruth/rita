package rita

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bruth/rita/codec"
	"github.com/nats-io/nats.go"
)

const (
	eventTypeHdr       = "rita-type"
	eventTimeHdr       = "rita-time"
	eventCodecHdr      = "rita-codec"
	eventMetaPrefixHdr = "rita-meta-"
	eventTimeFormat    = time.RFC3339Nano
)

var (
	ErrSequenceConflict  = errors.New("rita: sequence conflict")
	ErrEventIDRequired   = errors.New("rita: event id required")
	ErrEventTypeRequired = errors.New("rita: event type required")
)

// Validator can be optionally implemented by user-defined types and will be
// validated in different contexts, such as before appending an event to a stream.
type validator interface {
	Validate() error
}

type Evolver interface {
	Evolve(event *Event) error
}

// Event is a wrapper for application-defined events.
type Event struct {
	// ID of the event. This will be used as the NATS msg ID
	// for de-duplication.
	ID string

	// Time is the time of when the event occurred which may be different
	// from the time the event is appended to the store. If no time is provided,
	// the current local time will be used.
	Time time.Time

	// Type is a unique name for the event itself. This can be ommitted
	// if a type registry is being used, otherwise it must be set explicitly
	// to identity the encoded data.
	Type string

	// Data is the event data. This must be a byte slice (pre-encoded) or a value
	// of a type registered in the type registry.
	Data any

	// Metadata is application-defined metadata about the event.
	Meta map[string]string

	// Subject is the the subject the event is associated with. Read-only.
	Subject string

	// Sequence is the sequence where this event exists in the stream. Read-only.
	Sequence uint64
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

// EventStoreConfig is a subset of the nats.StreamConfig for the purpose of creating
// purpose-built streams for an event store.
type EventStoreConfig struct {
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

// EventStore provides event store semantics over a NATS stream.
type EventStore struct {
	name string
	rt   *Rita
}

// wrapEvent wraps a user-defined event into the Event envelope. It performs
// validation to ensure all the properties are either defined or defaults are set.
func (s *EventStore) wrapEvent(event *Event) (*Event, error) {
	if event.Data == nil {
		return nil, fmt.Errorf("event data is nil")
	}

	if s.rt.types == nil {
		if event.Type == "" {
			return nil, errors.New("event type is not defined")
		}
	} else {
		t, err := s.rt.types.Lookup(event.Data)
		if err != nil {
			return nil, err
		}

		if event.Type == "" {
			event.Type = t
		} else if event.Type != t {
			return nil, fmt.Errorf("wrong type for event data: %s", event.Type)
		}
	}

	if v, ok := event.Data.(validator); ok {
		if err := v.Validate(); err != nil {
			return nil, err
		}
	}

	// Set ID if empty.
	if event.ID == "" {
		event.ID = s.rt.id.New()
	}

	// Set time if empty.
	if event.Time.IsZero() {
		event.Time = s.rt.clock.Now().Local()
	}

	return event, nil
}

// packEvent pack an event into a NATS message. The advantage of using NATS headers
// is that the server supports creating a consumer that _only_ gets the headers
// without the data as an optimization for some use cases.
func (s *EventStore) packEvent(subject string, event *Event) (*nats.Msg, error) {
	// Marshal the data.
	var (
		data      []byte
		err       error
		codecName string
	)

	if s.rt.types == nil {
		data, err = codec.Binary.Marshal(event.Data)
		codecName = codec.Binary.Name()
	} else {
		data, err = s.rt.types.Marshal(event.Data)
		codecName = s.rt.types.Codec().Name()
	}
	if err != nil {
		return nil, err
	}

	msg := nats.NewMsg(subject)
	msg.Data = data

	// Map event envelope to NATS header.
	msg.Header.Set(nats.MsgIdHdr, event.ID)
	msg.Header.Set(eventTypeHdr, event.Type)
	msg.Header.Set(eventTimeHdr, event.Time.Format(eventTimeFormat))
	msg.Header.Set(eventCodecHdr, codecName)

	for k, v := range event.Meta {
		msg.Header.Set(fmt.Sprintf("%s%s", eventMetaPrefixHdr, k), v)
	}

	return msg, nil
}

// unpackEvent unpacks an Event from a NATS message.
func (s *EventStore) unpackEvent(msg *nats.Msg) (*Event, error) {
	eventType := msg.Header.Get(eventTypeHdr)
	codecName := msg.Header.Get(eventCodecHdr)

	var (
		data interface{}
		err  error
	)

	c, ok := codec.Codecs[codecName]
	if !ok {
		return nil, fmt.Errorf("%w: %s", codec.ErrCodecNotRegistered, codecName)
	}

	// No type registry, so assume byte slice.
	if s.rt.types == nil {
		var b []byte
		err = c.Unmarshal(msg.Data, &b)
		data = b
	} else {
		v, err := s.rt.types.Init(eventType)
		if err != nil {
			return nil, err
		}
		err = c.Unmarshal(msg.Data, v)
		data = v
	}
	if err != nil {
		return nil, err
	}

	md, err := msg.Metadata()
	if err != nil {
		return nil, fmt.Errorf("unpack: failed to get metadata: %s", err)
	}

	eventTime, err := time.Parse(eventTimeFormat, msg.Header.Get(eventTimeHdr))
	if err != nil {
		return nil, fmt.Errorf("unpack: failed to parse event time: %s", err)
	}

	meta := make(map[string]string)

	for h := range msg.Header {
		if strings.HasPrefix(h, eventMetaPrefixHdr) {
			key := h[len(eventMetaPrefixHdr):]
			meta[key] = msg.Header.Get(h)
		}
	}

	return &Event{
		ID:       msg.Header.Get(nats.MsgIdHdr),
		Type:     msg.Header.Get(eventTypeHdr),
		Time:     eventTime,
		Data:     data,
		Meta:     meta,
		Sequence: md.Sequence.Stream,
		Subject:  msg.Subject,
	}, nil
}

// lastSeqForSubject queries the JS API to identify the current latest sequence for a subject.
// This is used as an best-guess indicator of the current end of the even history.
func (s *EventStore) lastMsgForSubject(ctx context.Context, subject string) (*natsStoredMsg, error) {
	rsubject := fmt.Sprintf("$JS.API.STREAM.MSG.GET.%s", s.name)

	data, _ := json.Marshal(&natsGetMsgRequest{
		LastBySubject: subject,
	})

	msg, err := s.rt.nc.RequestWithContext(ctx, rsubject, data)
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

// Load fetches all events for a specific subject. The primary use case
// is to use a concrete subject, e.g. "orders.1" corresponding to an
// aggregate/entity identifier. The second use case is to load events for
// a cross-cutting view which can use subject wildcards.
func (s *EventStore) Load(ctx context.Context, subject string, opts ...LoadOption) ([]*Event, uint64, error) {
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

	sub, err := s.rt.js.SubscribeSync(subject, sopts...)
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

		event, err := s.unpackEvent(msg)
		if err != nil {
			return nil, 0, err
		}

		events = append(events, event)

		if event.Sequence == lastMsg.Sequence {
			break
		}
	}

	return events, lastMsg.Sequence, nil
}

// Append appends a one or more events to the subject's event sequence.
// It returns the resulting sequence number of the last appended event.
func (s *EventStore) Append(ctx context.Context, subject string, events []*Event, opts ...AppendOption) (uint64, error) {
	// Configure opts.
	var o appendOpts
	for _, opt := range opts {
		if err := opt.appendOpt(&o); err != nil {
			return 0, err
		}
	}

	var ack *nats.PubAck

	for i, event := range events {
		popts := []nats.PubOpt{
			nats.Context(ctx),
			nats.ExpectStream(s.name),
		}

		if i == 0 && o.expSeq != nil {
			popts = append(popts, nats.ExpectLastSequencePerSubject(*o.expSeq))
		}

		e, err := s.wrapEvent(event)
		if err != nil {
			return 0, err
		}

		msg, err := s.packEvent(subject, e)
		if err != nil {
			return 0, err
		}

		// TODO: add retry logic in case of intermittent errors?
		ack, err = s.rt.js.PublishMsg(msg, popts...)
		if err != nil {
			if strings.Contains(err.Error(), "wrong last sequence") {
				return 0, ErrSequenceConflict
			}
			return 0, err
		}
	}

	return ack.Sequence, nil
}

// Evolve loads events and evolves a model of state. The sequence of the
// last event that evolved the state is returned, including when an error
// occurs.
func (s *EventStore) Evolve(ctx context.Context, subject string, model Evolver, opts ...LoadOption) (uint64, error) {
	events, _, err := s.Load(ctx, subject, opts...)
	if err != nil {
		return 0, err
	}

	var lastSeq uint64
	for _, e := range events {
		if err := model.Evolve(e); err != nil {
			return lastSeq, err
		}
		lastSeq = e.Sequence
	}

	return lastSeq, nil
}

// Create creates the event store given the configuration. The stream
// name is the name of the store and the subjects default to "{name}}.>".
func (s *EventStore) Create(config *EventStoreConfig) error {
	if config == nil {
		config = &EventStoreConfig{}
	}

	subjects := config.Subjects
	if len(subjects) == 0 {
		subjects = []string{fmt.Sprintf("%s.>", s.name)}
	}

	_, err := s.rt.js.AddStream(&nats.StreamConfig{
		Name:        s.name,
		Description: config.Description,
		Subjects:    subjects,
		Storage:     config.Storage,
		Replicas:    config.Replicas,
		Placement:   config.Placement,
		DenyDelete:  true,
		DenyPurge:   true,
	})
	return err
}

// Update updates the event store configuration.
func (s *EventStore) Update(config *EventStoreConfig) error {
	_, err := s.rt.js.UpdateStream(&nats.StreamConfig{
		Name:        s.name,
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

// Delete deletes the event store.
func (s *EventStore) Delete() error {
	return s.rt.js.DeleteStream(s.name)
}
