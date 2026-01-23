package clock

import "time"

type SystemClock struct{}

func New() *SystemClock {
	return &SystemClock{}
}

var _ Clock = &SystemClock{}

func (c *SystemClock) Now() time.Time { return time.Now() }
