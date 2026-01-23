package kdc

import (
	"github.com/rizesql/kerberos/internal/clock"
	"github.com/rizesql/kerberos/internal/crypto"
	"github.com/rizesql/kerberos/internal/kdb"
	"github.com/rizesql/kerberos/internal/o11y/logging"
	"github.com/rizesql/kerberos/internal/replay"
)

type Platform struct {
	Clock        clock.Clock
	KeyGenerator crypto.KeyGenerator
	Database     kdb.Database
	Logger       *logging.Logger
	ReplayCache  replay.Cache
}

func NewPlatform(
	db kdb.Database,
	logger *logging.Logger,
	clk clock.Clock,
	keygen crypto.KeyGenerator,
	replayCache replay.Cache,
) *Platform {
	return &Platform{
		Clock:        clk,
		KeyGenerator: keygen,
		Database:     db,
		Logger:       logger,
		ReplayCache:  replayCache,
	}
}
