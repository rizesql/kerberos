package replay

import (
	"time"

	"github.com/rizesql/kerberos/internal/clock"
)

type TestCache struct {
	*InMemoryCache
}

func NewTestCache(clock clock.Clock) *TestCache {
	return &TestCache{
		InMemoryCache: NewInMemoryCache(10*time.Minute, clock),
	}
}
