package testkit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/clock"
	"github.com/rizesql/kerberos/internal/crypto"
	"github.com/rizesql/kerberos/internal/kdb"
	"github.com/rizesql/kerberos/internal/kdc"
	"github.com/rizesql/kerberos/internal/o11y/logging"
	"github.com/rizesql/kerberos/internal/replay"
	"github.com/rizesql/kerberos/internal/server"
)

type Harness struct {
	t *testing.T

	// Shared Infrastructure
	DB           kdb.Database
	Clock        *clock.TestClock
	KeyGenerator *crypto.TestKeyGenerator
	ReplayCache  *replay.TestCache
	Logger       *logging.Logger
}

func NewHarness(t *testing.T) *Harness {
	t.Helper()

	logger := logging.Noop()

	db, err := kdb.New(kdb.Config{
		DSN:    ":memory:",
		Logger: logger,
	})
	assert.Err(t, err, nil)

	err = db.Migrate()
	assert.Err(t, err, nil)

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})

	testClock := clock.NewTestClock()

	return &Harness{
		t:            t,
		DB:           db,
		Clock:        testClock,
		KeyGenerator: crypto.NewTestKeyGenerator(),
		ReplayCache:  replay.NewTestCache(testClock),
		Logger:       logger,
	}
}

// --- Platform Factories ---

// NewKDCPlatform creates a KDC platform connected to the shared environment.
func (h *Harness) NewKDCPlatform() *kdc.Platform {
	return &kdc.Platform{
		Clock:        h.Clock,
		KeyGenerator: h.KeyGenerator,
		Database:     h.DB,
		Logger:       h.Logger,
		ReplayCache:  h.ReplayCache,
	}
}

// --- Server Helpers ---
func (h *Harness) NewServer() *server.Server {
	return server.New(h.Logger)
}

func Call[Req, Res any](t *testing.T, srv *server.Server, r server.Route, headers http.Header, req Req) TestResponse[Res] {
	t.Helper()

	rr := httptest.NewRecorder()

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(req)
	assert.Err(t, err, nil)

	httpReq := httptest.NewRequest(r.Method(), r.Path(), body)
	if headers != nil {
		httpReq.Header = headers
	}

	srv.Mux().ServeHTTP(rr, httpReq)

	rawBody := rr.Body.Bytes()
	res := TestResponse[Res]{
		Status:  rr.Code,
		Headers: rr.Header(),
		RawBody: string(rawBody),
	}

	if len(rawBody) > 0 {
		var responseBody Res
		if err := json.Unmarshal(rawBody, &responseBody); err == nil {
			res.Body = &responseBody
		}
	}

	return res
}

type TestResponse[TBody any] struct {
	Status  int
	Headers http.Header
	Body    *TBody
	RawBody string
}

// --- Data Helpers ---

func (h *Harness) CreatePrincipal(ctx context.Context, params kdb.CreatePrincipalParams) kdb.Principal {
	h.t.Helper()
	p, err := kdb.Query.CreatePrincipal(ctx, h.DB, params)
	assert.Err(h.t, err, nil)
	return p
}
