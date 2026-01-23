package kdc

import (
	"time"

	"github.com/rizesql/kerberos/internal/protocol"
)

type Config struct {
	Realm          protocol.Realm
	TicketLifetime time.Duration
}
