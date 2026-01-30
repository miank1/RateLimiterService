package clock

import "time"

// Clock interface for time operations
type Clock interface {
	Now() time.Time
}

// RealClock implements Clock using the system clock
type RealClock struct{}

func (c RealClock) Now() time.Time {
	return time.Now()
}