package clock

import (
	"sync"
	"time"
)

type TestClock struct {
	mu  sync.RWMutex
	now time.Time
}

func NewTestClock(now ...time.Time) *TestClock {
	if len(now) == 0 {
		now = append(now, time.Now())
	}
	return &TestClock{mu: sync.RWMutex{}, now: now[0]}
}

var _ Clock = (*TestClock)(nil)

func (c *TestClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

func (c *TestClock) Tick(d time.Duration) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)

	return c.now
}

func (c *TestClock) Set(t time.Time) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t

	return c.now
}
