package start

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"runtime/debug"

	"github.com/rizesql/kerberos/internal/ap"
	"github.com/rizesql/kerberos/internal/clock"
	"github.com/rizesql/kerberos/internal/o11y/logging"
	"github.com/rizesql/kerberos/internal/protocol"
	"github.com/rizesql/kerberos/internal/replay"
	"github.com/rizesql/kerberos/internal/server"
	"github.com/rizesql/kerberos/internal/shutdown"
)

func Run(ctx context.Context, cfg Config) error {
	logger := logging.New()
	clk := clock.New()
	shutdowns := shutdown.New()

	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic",
				"panic", r,
				"stack", string(debug.Stack()),
			)
		}
	}()

	// Server's secret key (must match what's in KDC database for http/api-server)
	serverKeyBytes, err := hex.DecodeString(cfg.ServerKeyHex)
	if err != nil {
		return fmt.Errorf("invalid server key: %w", err)
	}
	serverKey, err := protocol.NewSessionKey(serverKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to create session key: %w", err)
	}

	// Create replay cache and verifier
	cache := replay.NewInMemoryCache(cfg.ReplayWindow, clk)
	verifier := ap.NewVerifier(serverKey, clk, cache)

	// Create server
	srv := server.New(logger)
	shutdowns.RegisterCtx(srv.Shutdown)

	// Register protected routes with Kerberos middleware
	srv.Register(&WhoAmIRoute{}, ap.Middleware(verifier))
	srv.Register(&SecretRoute{}, ap.Middleware(verifier))

	// Start listening
	ln, err := net.Listen("tcp", cfg.Port)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		if err := srv.Listen(ctx, ln); err != nil {
			panic(err)
		}
	}()

	logger.Info("Api Server running", "port", cfg.Port)
	logger.Info("Press Ctrl+C to shut down")
	fmt.Printf("✓ API Server listening on %s\n", cfg.Port)

	if err := shutdowns.WaitForSignal(ctx); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	logger.Info("Server shutdown complete")
	return nil
}
