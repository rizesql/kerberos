package platform

import (
	"github.com/rizesql/kerberos/internal/sdk"
)

type Platform struct {
	Sdk   *sdk.Sdk
	Cache *TicketCache
}

func NewPlatform(sdk *sdk.Sdk, ticketCache *TicketCache) *Platform {
	return &Platform{
		Sdk:   sdk,
		Cache: ticketCache,
	}
}
