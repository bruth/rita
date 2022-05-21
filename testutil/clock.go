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
	Unit  time.Duration
	last  time.Time
	count int
}

// Now implements clock.Clock.
func (c *Clock) Now() time.Time {
	c.last = c.Start.Add(time.Duration(c.count) * c.Unit)
	c.count++
	return c.last
}

// Last returns the last time that was used.
func (c *Clock) Last() time.Time {
	return c.last
}

func NewClock(unit time.Duration) *Clock {
	return &Clock{
		Start: defaultStartTime,
		Unit:  unit,
		last:  defaultStartTime,
	}
}
