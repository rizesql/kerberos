package start

import (
	"context"
	"fmt"
	"net"
	"runtime/debug"

	"github.com/rizesql/kerberos/internal/clock"
	"github.com/rizesql/kerberos/internal/crypto"
	"github.com/rizesql/kerberos/internal/kdb"
	"github.com/rizesql/kerberos/internal/kdc"
	"github.com/rizesql/kerberos/internal/kdc/as"
	kdc_http "github.com/rizesql/kerberos/internal/kdc/http"
	"github.com/rizesql/kerberos/internal/o11y/logging"
	"github.com/rizesql/kerberos/internal/server"
	"github.com/rizesql/kerberos/internal/shutdown"
)

func Run(ctx context.Context, cfg Config) error {
	logger := logging.New()

	clock := clock.New()
	keygen := crypto.NewKeyGenerator()

	shutdowns := shutdown.New()

	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic",
				"panic", r,
				"stack", string(debug.Stack()),
			)
		}
	}()

	db, err := kdb.New(kdb.Config{
		DSN:    cfg.DBPath,
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}
	shutdowns.Register(db.Close)

	platform := kdc.NewPlatform(db, logger, clock, keygen)

	srv := server.New(
		server.Dependencies{Logger: logger},
		server.Config{},
	)
	shutdowns.RegisterCtx(srv.Shutdown)

	kdc_http.Register(srv, platform, as.Config{
		Realm:      cfg.Realm,
		TicketLife: cfg.TicketLife,
	})

	ln, err := net.Listen("tcp", cfg.Port)
	if err != nil {
		logger.Error("failed to listen on port",
			"error", err,
		)
		return err
	}

	go func() {
		if err := srv.Listen(ctx, ln); err != nil {
			panic(err)
		}
	}()

	logger.Info("Press Ctrl+C to shut down")
	if err := shutdowns.WaitForSignal(ctx); err != nil {
		logger.Error("shutdown failed", "error", err)
		return fmt.Errorf("shutdown failed: %w", err)
	}

	logger.Info("Server shutdown complete")
	return nil
}
