package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/rizesql/kerberos/internal/o11y/logging"
)

type ServerState int

const (
	ServerStateClosed ServerState = iota
	ServerStateListening
)

type Server struct {
	mu    sync.Mutex
	state ServerState

	logger *logging.Logger
	mux    *http.ServeMux
	srv    *http.Server
}

func New(logger *logging.Logger) *Server {
	mux := http.NewServeMux()

	srv := &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	return &Server{
		mu:     sync.Mutex{},
		logger: logger,
		mux:    mux,
		srv:    srv,
	}
}

func (s *Server) Mux() *http.ServeMux {
	return s.mux
}

func (s *Server) Listen(ctx context.Context, ln net.Listener) error {
	s.mu.Lock()
	if s.state == ServerStateListening {
		s.logger.Warn("Server is already listening")
		s.mu.Unlock()
		return nil
	}
	s.state = ServerStateListening
	s.mu.Unlock()

	s.logger.Info("listening",
		"srv", "http",
		"addr", ln.Addr().String(),
	)

	if err := s.srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Register(r Route, mws ...Middleware) {
	s.logger.Debug("registering",
		"method", r.Method(),
		"path", r.Path(),
	)

	handler := r.Handle()
	for _, mw := range slices.Backward(mws) {
		handler = mw(handler)
	}

	s.mux.HandleFunc(fmt.Sprintf("%s %s", r.Method(), r.Path()), handler)
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.state = ServerStateClosed
	s.mu.Unlock()

	if err := s.srv.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}
