/*
Package rita is toolkit of abstractions build on top of NATS.

Setup

Initialize a Rita instance by passing the NATS connection as the only required argument.

	r := rita.New(nc)

EventStore

Create an event store for orders. This will default to a subject of "orders.>"

	es, err := r.CreateEventStore(&rita.EventStoreConfig{
		Name: "orders",
		Replicas: 3,
	})

Append a couple events to the "orders.1" subject on the stream. If ID or Time are not supplied
a unique ID and the current time will be used. "ExpectSequence" of zero means no other
events should be associated with this subject yet.

	seq1, err := es.Append("orders.1", &rita.Event{
		Type: "order-created",
		Data: []byte("..."),
	}, rita.ExpectSequence(0))

Example of setting the ID and Time explicitly for demonstration. Notice the next "ExpectSequence"
uses seq1 from the previous append.

	seq2, err = es.Append("orders.1", &rita.Event{
		ID: nuid.Next(),
		Type: "order-shipped",
		Time: time.Now().Local(),
		Data: []byte("..."),
	}, rita.ExpectSequence(seq1))

Load the events for the subject.

	events, lastSeq, err := es.Load("orders.1")

Type Registry
*/
package rita
