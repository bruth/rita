# Rita


Rita is a toolkit of various event-centric and reactive abstractions build on top of NATS.

**NOTE: This package is under heavy development, so breaking changes will likely be introduced. Feedback is welcome on API design and scope. Please open an issue if you have something to share!**


[![GoDoc][GoDoc-Image]][GoDoc-URL] [![ReportCard][ReportCard-Image]][ReportCard-URL] [![GitHub Actions][GitHubActions-Image]][GitHubActions-URL]

[GoDoc-Image]: https://pkg.go.dev/badge/github.com/bruth/rita
[GoDoc-URL]: https://pkg.go.dev/github.com/bruth/rita
[ReportCard-Image]: https://goreportcard.com/report/github.com/bruth/rita
[ReportCard-URL]: https://goreportcard.com/report/github.com/bruth/rita
[GitHubActions-Image]: https://github.com/bruth/rita/actions/workflows/ci.yaml/badge.svg?branch=main
[GitHubActions-URL]: https://github.com/bruth/rita/actions?query=branch%3Amain

## Install

Requires Go 1.18+

```
go get github.com/bruth/rita
```

## Usage

### Type Registry

Rita comes with an opt-in, but recommended, type registry which provides [de]serialization transparency as well as providing consistency of usage. Fundamentally, we are defining names and mapping them to concrete types.

Given a couple of types in a domain model:

```go
type OrderPlaced struct {}
type OrderShipped struct {}
```

We can create a registry, associating names. If the names are intended to be shared and portable across other languages, then some thought should be put into the naming structure. For example, the [CloudEvents spec](https://github.com/cloudevents/spec/blob/v1.0.1/spec.md#type) suggest using the reverse DNS name.

```go
tr, err := rita.NewTypeRegistry(map[string]*rita.Type{
  "com.example.order-placed": {
    Init: func() any { return &OrderPlaced{} },
  },
  "com.example.order-shipped": {
    Init: func() any { return &OrderShipped{} },
  },
})
```

Each type currently supports a `Init` function to allocate a new value with any default fields set. The registry also accepts an optional `rita.Codec()` option for overriding the default codec for [de]serialization.

Once the set of types that will be worked with are reigstered, we can initialize a Rita instance by passing the NATS connection and the registry as an option.

```go
r, err := rita.New(nc, rita.TypeRegistry(tr))
```

> **What is the behavior without a type registry?** The expectation is that the type name is explicitly provided and the data is a `[]byte`.

### EventStore

Create an event store for orders. This will default to a subject of `orders.>`. *Note, other than a few different defaults to follow the semantics of an event store, the stream is just an vanilla JetStream and can be managed as such.*

```go
es, err := r.CreateEventStore(&rita.EventStoreConfig{
  Name: "orders",
  Replicas: 3,
})
```

Append an event to the `orders.1` subject on the stream. The _event_ can be a registered value or an `*Event` value. `ExpectSequence` can be set to zero means no other
events should be associated with this subject yet.

```go
seq1, err := es.Append("orders.1", &OrderPlaced{}, rita.ExpectSequence(0))
```

Append an another event, using the previously returned sequence.

```go
seq2, err := es.Append("orders.1", &OrderShipped{}, rita.ExpectSequence(seq1))
```

Load the events for the subject. This returns a slice of `*Event` values where the `Data` value for each event has been pre-allocated based on the `Type` using the registry.

```go
events, lastSeq, err := es.Load("orders.1")
```

The `lastSeq` value indicates the sequence of the last event appended for this subject. If a new event needs to be appended, this should be used with `ExpectSequence`.

### Views

Although event sourcing involves modeling and persisting the state transitions as events, we still need to compute _state_ in order to make decisions when commands are received.

Fundamentally, we have a model of state, and then we need to evolve the state given each event. We can model this as an interface in Go.

```go
type View interface {
  Evolve(event any) error
}
```

An implementation would then look like:

```go
type Order struct {
  // fields..
}

func (o *Order) Evolve(event any) error {
  // Switch on the event type and evolve the state.
  switch e := event.(type) {
  }
}
```

Given this model, we can use the `View` method on `EventStore` for convenience.

```go
var order Order
lastSeq, err = es.View("orders.1", &order)
```

This also works for a cross-cutting view (all orders) using a subject wildcard.

```go
var orderList OrderList
lastSeq, err = es.View("orders.*", &orderlist)
```

## Planned Features

- [x] type registry (beta)
  - transparent mapping from string to type
  - support for labeling types, event, state, command, etc.
  - encoder to/decoder from nats message
- [x] event store (alpha)
  - layer on JetStream for event store
  - simple api with event store semantics
  - each store maps to one stream
- [ ] event-sourced state (in progress)
  - model for event-sourced state representations
  - interface for user-implemented type
  - maps to a subject
  - snapshot or state up to some sequence/time or live-updating
- [ ] command deciders (next)
  - model for handling commands and emitting events
  - provide consistency boundary for state transitions
  - state is event-sourced, single subject or wildcard (with concurrency detection)
- [ ] state store (next)
  - leverage KV or Object Store for state snapshots
  - store snapshot of an event-sourced state as of a sequence
  - requires a key, subject might be sufficient
- [ ] timers and tickers
  - set a schedule or a time that will publish a message on a subject
  - use stream with max messages per subject + interest policy
  - each timer or ticker maps to a subject
  - use expected sequence on subject to prevent duplicate instance firing

