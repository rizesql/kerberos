package ap_test

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/rizesql/kerberos/internal/ap"
	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/kdc/shared"
	"github.com/rizesql/kerberos/internal/protocol"
	"github.com/rizesql/kerberos/internal/server"
	"github.com/rizesql/kerberos/internal/testkit"
)

type protected struct{}

var _ server.Route = (*protected)(nil)

func (p *protected) Method() string { return http.MethodGet }
func (p *protected) Path() string   { return "/protected" }

func (p *protected) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok := ap.ClientFromContext(r.Context())
		if !ok {
			server.EncodeError(w, http.StatusUnauthorized, fmt.Errorf("client not in context"))
			return
		}

		if err := server.Encode(w, http.StatusOK, c); err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}
	}
}

func TestMiddleware(t *testing.T) {
	h := testkit.NewHarness(t)

	serverKeyBytes, _ := hex.DecodeString("aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899")
	sessionKeyBytes, _ := hex.DecodeString("112233445566778899aabbccddeeff00112233445566778899aabbccddeeff00")

	serverKey, _ := protocol.NewSessionKey(serverKeyBytes)
	sessionKey, _ := protocol.NewSessionKey(sessionKeyBytes)

	client, _ := protocol.NewPrincipal("alice", "", "ATHENA.MIT.EDU")
	server, _ := protocol.NewPrincipal("http", "server.athena.mit.edu", "ATHENA.MIT.EDU")
	clientAddr, _ := protocol.NewAddress(net.IPv4(127, 0, 0, 1))

	verifier := ap.NewVerifier(serverKey, h.Clock, h.ReplayCache)

	srv := h.NewServer()
	handler := protected{}
	srv.Register(&handler, ap.Middleware(verifier))

	createAPReq := func(authTimeOffset time.Duration) string {
		now := h.Clock.Now()
		authTime := now.Add(authTimeOffset)

		ticket, _ := protocol.NewTicket(server, client, clientAddr, now, 8*time.Hour, sessionKey)
		encTicket, _ := shared.EncryptEntity(serverKey, ticket)

		auth, _ := protocol.NewAuthenticator(client, clientAddr, authTime)
		encAuth, _ := shared.EncryptEntity(sessionKey, auth)

		apReq, _ := protocol.NewAPReq(encTicket, encAuth)
		data, _ := json.Marshal(apReq)
		return base64.StdEncoding.EncodeToString(data)
	}

	t.Run("ValidAPReq", func(t *testing.T) {
		req := createAPReq(100 * time.Millisecond)
		headers := http.Header{}
		headers.Set("Authorization", "Kerberos "+req)

		res := testkit.Call[string, protocol.Principal](t, srv, &handler, headers, req)

		assert.Equal(t, res.Status, http.StatusOK)
		if res.Body == nil {
			t.Fatal("response body is nil")
		}

		assert.Equal(t, res.Body.String(), client.String())
	})

	t.Run("MissingAuthorizationHeader", func(t *testing.T) {
		res := testkit.Call[any, protocol.Principal](t, srv, &handler, nil, nil)
		assert.Equal(t, res.Status, http.StatusUnauthorized)
	})

	t.Run("InvalidScheme", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer sometoken")

		res := testkit.Call[any, protocol.Principal](t, srv, &handler, nil, nil)
		assert.Equal(t, res.Status, http.StatusUnauthorized)
	})

	t.Run("InvalidBase64", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Kerberos !!!notbase64!!!")

		res := testkit.Call[any, protocol.Principal](t, srv, &handler, nil, nil)
		assert.Equal(t, res.Status, http.StatusUnauthorized)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		invalidJSON := base64.StdEncoding.EncodeToString([]byte("not json"))
		headers := http.Header{}
		headers.Set("Authorization", "Kerberos "+invalidJSON)

		res := testkit.Call[any, protocol.Principal](t, srv, &handler, nil, nil)
		assert.Equal(t, res.Status, http.StatusUnauthorized)
	})

	t.Run("InvalidTicket", func(t *testing.T) {
		now := h.Clock.Now()
		authTime := now.Add(200 * time.Millisecond)

		wrongKey, _ := protocol.NewSessionKey(sessionKeyBytes)
		ticket, _ := protocol.NewTicket(server, client, clientAddr, now, 8*time.Hour, sessionKey)
		encTicket, _ := shared.EncryptEntity(wrongKey, ticket)

		auth, _ := protocol.NewAuthenticator(client, clientAddr, authTime)
		encAuth, _ := shared.EncryptEntity(sessionKey, auth)

		apReq, _ := protocol.NewAPReq(encTicket, encAuth)
		data, _ := json.Marshal(apReq)
		encoded := base64.StdEncoding.EncodeToString(data)

		headers := http.Header{}
		headers.Set("Authorization", "Kerberos "+encoded)

		res := testkit.Call[any, protocol.Principal](t, srv, &handler, nil, nil)
		assert.Equal(t, res.Status, http.StatusUnauthorized)
	})
}
