package rita

import (
	"time"
)

// Validator can be optionally implemented by user-defined types and will be
// validated in different contexts, such as before appending an event to a stream.
type Validator interface {
	Validate() error
}

// Event is a wrapper type for encoded events.
type Event struct {
	// ID of the event. NUID, ULID, XID, or UUID are recommended.
	// This will be used as the NATS msg ID for de-duplication.
	ID string

	// Type is a unique name for the event itself. This can be ommitted
	// if a type registry is being used, otherwise it must be set explicitly
	// to identity the encoded data.
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

	// Subject is the the subject the event was appended to.
	Subject string

	// Sequence is the sequence where this event exists in the stream.
	Sequence uint64
}
