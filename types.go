package rita

import (
	"time"
)

// Validator can be optionally implemented by user-defined types and will be
// validated in different contexts, such as before appending an event to a stream.
type Validator interface {
	Validate() error
}

/*
type Message struct {
	ID   string
	Time time.Time
	Type string
	Data any
}
*/

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

	// Data is the event data. This must be a byte slice (pre-encoded) or a value
	// of a type registered in the type registry.
	Data any

	// Metadata associated with the event.
	Meta map[string]string

	// Seq is the sequence where this event exists in the stream.
	Seq uint64
}

/*
type Snapshot struct {
	// Type name of the serialized data.
	Type string
	// Snapshot value itself.
	Data any
	// Subject pattern that this snapshot was derived from.
	Subject string
	// Sequence of the event last applied to this snapshot.
	Sequence uint64
	// Revision of the snapshot.
	Revision uint64
}
*/
