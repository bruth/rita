# Rita


Rita is a toolkit of various event-based and reactive abstractions build on top of NATS.

**NOTE: This package is under development, so breaking changes may be introduced. Feedback is welcome on design suggestions and scope. Please open an issue if you have something to share!**


[![GoDoc](GoDoc-Image)](GoDoc-URL) [![ReportCard](ReportCard-Image)](ReportCard-URL) [![GitHub Actions](GitHubActions-Image)](GitHubActions-URL)

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

### Setup

Initialize a Rita instance by passing the NATS connection as the only required argument.

```go
r := rita.New(nc)
```

### EventStore

Create an event store for orders. This will default to a subject of "orders.>"

```go
es, err := r.CreateEventStore(&rita.EventStoreConfig{
  Name: "orders",
  Replicas: 3,
})
```

Append a couple events to the "orders.1" subject on the stream. If ID or Time are not supplied
a unique ID and the current time will be used. "ExpectSequence" of zero means no other
events should be associated with this subject yet.

```go
seq1, err := es.Append("orders.1", &rita.Event{
  Type: "order-created",
  Data: []byte("..."),
}, rita.ExpectSequence(0))
```

Example of setting the ID and Time explicitly for demonstration. Notice the next "ExpectSequence" uses seq1 from the previous append.

```go
seq2, err = es.Append("orders.1", &rita.Event{
  ID: nuid.Next(),
  Type: "order-shipped",
  Time: time.Now().Local(),
  Data: []byte("..."),
}, rita.ExpectSequence(seq1))
```

Load the events for the subject.

```go
events, lastSeq, err := es.Load("orders.1")
```

## Roadmap

- [x] type registry (in progress)
  - transparent mapping from string to type
  - support for labeling types, event, state, command, etc.
  - encoder to/decoder from nats message
- [x] event store (in progress)
  - layer on JetStream for event store
  - simple api with event store semantics
  - each store maps to one stream
- [x] event-sourced state (in progress)
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

