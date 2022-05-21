package rita

import "time"

type Command struct {
	ID   string
	Time time.Time
	Type string
	Data any
}

type Decider interface {
	Decide(command *Command) ([]*Event, error)
}
