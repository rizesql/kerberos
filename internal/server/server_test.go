package server_test

import (
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/o11y/logging"
	"github.com/rizesql/kerberos/internal/server"
)

type mockRoute struct {
	method string
	path   string
	handle http.HandlerFunc
}

func (m mockRoute) Method() string           { return m.method }
func (m mockRoute) Path() string             { return m.path }
func (m mockRoute) Handle() http.HandlerFunc { return m.handle }

func TestServer_Register(t *testing.T) {
	logger := logging.Noop()
	srv := server.New(logger)

	called := false
	route := mockRoute{
		method: http.MethodGet,
		path:   "/test",
		handle: func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		},
	}

	srv.Register(route)

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	rr := newResponseRecorder()
	srv.Mux().ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusOK)
	assert.Equal(t, called, true)
}

func TestServer_Middleware(t *testing.T) {
	logger := logging.Noop()
	srv := server.New(logger)

	middlewareCalled := false
	mw := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next(w, r)
		}
	}

	route := mockRoute{
		method: http.MethodGet,
		path:   "/mw",
		handle: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	}

	srv.Register(route, mw)

	req, _ := http.NewRequest(http.MethodGet, "/mw", nil)
	rr := newResponseRecorder()
	srv.Mux().ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusOK)
	assert.Equal(t, middlewareCalled, true)
}

func TestServer_ListenState(t *testing.T) {
	logger := logging.Noop()
	srv := server.New(logger)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.Err(t, err, nil)
	t.Cleanup(func() {
		if err := ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Error(err)
		}
	})

	// Start listening in a goroutine
	errChan := make(chan error)
	go func() {
		errChan <- srv.Listen(t.Context(), ln)
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Try to listen again (should return nil and log warning, but not error)
	// The implementation says:
	// if s.state == ServerStateListening { return nil }
	err = srv.Listen(t.Context(), ln)
	assert.Err(t, err, nil)

	// Shutdown
	err = srv.Shutdown(t.Context())
	assert.Err(t, err, nil)

	// Check if initial listen returned (it should have returned http.ErrServerClosed or nil)
	// Since Shutdown was called, Listen should return.
	select {
	case err := <-errChan:
		// Listen returns nil on graceful shutdown if it ignored ErrServerClosed
		assert.Err(t, err, nil)
	case <-time.After(1 * time.Second):
		t.Fatal("Listen did not return after Shutdown")
	}
}

// Helpers
type responseRecorder struct {
	Code      int
	HeaderMap http.Header
	Body      *bytesBuffer
}

func newResponseRecorder() *responseRecorder {
	return &responseRecorder{
		Code:      http.StatusOK,
		HeaderMap: make(http.Header),
		Body:      &bytesBuffer{},
	}
}

func (rw *responseRecorder) Header() http.Header {
	return rw.HeaderMap
}

func (rw *responseRecorder) Write(buf []byte) (int, error) {
	return rw.Body.Write(buf)
}

func (rw *responseRecorder) WriteHeader(code int) {
	rw.Code = code
}

type bytesBuffer struct {
	data []byte
}

func (b *bytesBuffer) Write(p []byte) (n int, err error) {
	b.data = append(b.data, p...)
	return len(p), nil
}
