package shutdown

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"
)

type ShutdownCtx func(ctx context.Context) error

type Shutdown func() error

type ShutdownError struct {
	Errors []error
}

func (se *ShutdownError) Error() string {
	if len(se.Errors) == 1 {
		return fmt.Sprintf("shutdown err: %v", se.Errors[0])
	}

	return fmt.Sprintf("shutdown errors (%d): %v", len(se.Errors), se.Errors)
}

type shutdownState bool

const (
	shutdownStateIdle    shutdownState = false
	shutdownStateRunning shutdownState = true
)

type Shutdowns struct {
	mu        sync.RWMutex
	callbacks []ShutdownCtx
	state     shutdownState
}

func New() *Shutdowns {
	return &Shutdowns{
		mu:        sync.RWMutex{},
		callbacks: []ShutdownCtx{},
		state:     shutdownStateIdle,
	}
}

func (s *Shutdowns) Register(cbs ...Shutdown) {
	if len(cbs) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == shutdownStateRunning {
		return
	}

	for _, cb := range cbs {
		fn := cb
		s.callbacks = append(s.callbacks, func(context.Context) error {
			return fn()
		})
	}
}

func (s *Shutdowns) RegisterCtx(cbs ...ShutdownCtx) {
	if len(cbs) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == shutdownStateRunning {
		return
	}

	s.callbacks = append(s.callbacks, cbs...)
}

func (s *Shutdowns) Shutdown(ctx context.Context) []error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == shutdownStateRunning {
		return []error{}
	}

	s.state = shutdownStateRunning

	if len(s.callbacks) == 0 {
		return []error{}
	}

	var shutdownErrs []error
	for _, cb := range slices.Backward(s.callbacks) {
		if err := cb(ctx); err != nil {
			shutdownErrs = append(shutdownErrs, err)
		}
	}

	if len(shutdownErrs) > 0 {
		return shutdownErrs
	}

	return []error{}
}

func (s *Shutdowns) WaitForSignal(ctx context.Context, timeouts ...time.Duration) error {
	timeout := 30 * time.Second
	if len(timeouts) > 0 && timeouts[0] > 0 {
		timeout = timeouts[0]
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)

	select {
	case <-sig:
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if errs := s.Shutdown(shutdownCtx); len(errs) > 0 {
		return &ShutdownError{Errors: errs}
	}
	return nil
}
