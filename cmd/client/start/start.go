package start

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"

	"github.com/rizesql/kerberos/internal/o11y/logging"
	"github.com/rizesql/kerberos/internal/server"
	"github.com/rizesql/kerberos/internal/shutdown"
)

func Run(ctx context.Context, cfg Config) error {
	logger := logging.New()
	shutdowns := shutdown.New()

	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic",
				"panic", r,
				"stack", string(debug.Stack()),
			)
		}
	}()

	// Create ticket cache (stores TGT and service tickets in memory)
	ticketCache := NewTicketCache()

	// Create server
	srv := server.New(logger)
	shutdowns.RegisterCtx(srv.Shutdown)

	// Serve static frontend files
	srv.Mux().Handle("/", http.FileServer(http.Dir(cfg.WebDir)))

	// Register API routes for auth flow
	srv.Register(&LoginRoute{kdcAddr: cfg.KDCAddr, cache: ticketCache})
	srv.Register(&GetTicketRoute{kdcAddr: cfg.KDCAddr, cache: ticketCache})
	srv.Register(&CallServiceRoute{cache: ticketCache})

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

	logger.Info("Client running", "port", cfg.Port, "frontend", cfg.WebDir)
	fmt.Printf("✓ Client running on http://localhost%s\n", cfg.Port)
	logger.Info("Press Ctrl+C to shut down")

	if err := shutdowns.WaitForSignal(ctx); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	logger.Info("Server shutdown complete")
	return nil
}
