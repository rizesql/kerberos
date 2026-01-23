package replay

import (
	"errors"
	"sync"
	"time"

	"github.com/rizesql/kerberos/internal/clock"
)

var ErrReplayDetected = errors.New("replay attack detected: request already processed")

type Cache interface {
	Check(client string, timestamp time.Time) error
}

type entry struct{ expiresAt time.Time }

type InMemoryCache struct {
	mu       sync.Mutex
	entries  map[string]entry
	window   time.Duration
	clock    clock.Clock
	lastGC   time.Time
	gcPeriod time.Duration
}

func NewInMemoryCache(window time.Duration, clock clock.Clock) *InMemoryCache {
	return &InMemoryCache{
		entries:  make(map[string]entry),
		window:   window,
		clock:    clock,
		gcPeriod: window / 2,
	}
}

func (c *InMemoryCache) Check(client string, timestamp time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.clock.Now()

	if now.Sub(c.lastGC) > c.gcPeriod {
		c.gc(now)
		c.lastGC = now
	}

	key := makeKey(client, timestamp)

	if e, exists := c.entries[key]; exists {
		if now.Before(e.expiresAt) {
			return ErrReplayDetected
		}
	}

	c.entries[key] = entry{
		expiresAt: now.Add(c.window),
	}

	return nil
}

func (c *InMemoryCache) gc(now time.Time) {
	for key, e := range c.entries {
		if now.After(e.expiresAt) {
			delete(c.entries, key)
		}
	}
}

func makeKey(client string, timestamp time.Time) string {
	return client + "|" + timestamp.UTC().Format(time.RFC3339Nano)
}
