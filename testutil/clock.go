package testutil

import "time"

var (
	defaultStartTime = time.Date(2019, 9, 20, 14, 0, 0, 0, time.UTC)
)

// Clock implements clock.Clock, but each call to Now() will increment
// a counter. The subsequent call will be the start time incremented
// by the specified unit. For example, increment by one minute at a
// time.
type Clock struct {
	Start time.Time
	unit  time.Duration
	last  time.Time
}

// Now implements clock.Clock.
func (c *Clock) Now() time.Time {
	if c.last.IsZero() {
		c.last = c.Start
	} else {
		c.last = c.last.Add(c.unit)
	}
	return c.last
}

func (c *Clock) Add(d time.Duration) time.Time {
	if c.last.IsZero() {
		c.last = c.Start
	}
	c.last = c.last.Add(d)
	return c.last
}

// Last returns the last time that was used.
func (c *Clock) Last() time.Time {
	if c.last.IsZero() {
		c.last = c.Start
	}
	return c.last
}

func NewClock(unit time.Duration) *Clock {
	return &Clock{
		Start: defaultStartTime,
		unit:  unit,
	}
}
