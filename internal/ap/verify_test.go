package ap_test

import (
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/rizesql/kerberos/internal/ap"
	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/clock"
	"github.com/rizesql/kerberos/internal/kdc/shared"
	"github.com/rizesql/kerberos/internal/protocol"
	"github.com/rizesql/kerberos/internal/replay"
)

func TestVerifier(t *testing.T) {
	serverKeyBytes, _ := hex.DecodeString("aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899")
	sessionKeyBytes, _ := hex.DecodeString("112233445566778899aabbccddeeff00112233445566778899aabbccddeeff00")

	serverKey, _ := protocol.NewSessionKey(serverKeyBytes)
	sessionKey, _ := protocol.NewSessionKey(sessionKeyBytes)

	client, _ := protocol.NewPrincipal("alice", "", "ATHENA.MIT.EDU")
	server, _ := protocol.NewPrincipal("http", "server.athena.mit.edu", "ATHENA.MIT.EDU")
	clientAddr, _ := protocol.NewAddress(net.IPv4(127, 0, 0, 1))

	testClock := clock.NewTestClock()
	replayCache := replay.NewTestCache(testClock)

	verifier := ap.NewVerifier(serverKey, testClock, replayCache)

	createValidTicket := func(issuedAt time.Time) protocol.EncryptedData {
		ticket, _ := protocol.NewTicket(
			server,
			client,
			clientAddr,
			issuedAt,
			8*time.Hour,
			sessionKey,
		)
		enc, _ := shared.EncryptEntity(serverKey, ticket)
		return enc
	}

	createAuthenticator := func(c protocol.Principal, issuedAt time.Time) protocol.EncryptedData {
		auth, _ := protocol.NewAuthenticator(c, clientAddr, issuedAt)
		enc, _ := shared.EncryptEntity(sessionKey, auth)
		return enc
	}

	t.Run("Success", func(t *testing.T) {
		now := testClock.Now()
		authTime := now.Add(10 * time.Millisecond)

		ticket := createValidTicket(now)
		auth := createAuthenticator(client, authTime)
		req, _ := protocol.NewAPReq(ticket, auth)

		result, err := verifier.Verify(req)
		assert.Err(t, err, nil)
		assert.Equal(t, result.Client.String(), client.String())
	})

	t.Run("InvalidTicket", func(t *testing.T) {
		now := testClock.Now()
		authTime := now.Add(20 * time.Millisecond)

		// Encrypt ticket with wrong key
		wrongKey, _ := protocol.NewSessionKey(sessionKeyBytes)
		wrongTicket, _ := protocol.NewTicket(server, client, clientAddr, now, 8*time.Hour, sessionKey)
		badEnc, _ := shared.EncryptEntity(wrongKey, wrongTicket)

		auth := createAuthenticator(client, authTime)
		req, _ := protocol.NewAPReq(badEnc, auth)

		_, err := verifier.Verify(req)
		assert.Err(t, err, ap.ErrInvalidTicket)
	})

	t.Run("ClientMismatch", func(t *testing.T) {
		now := testClock.Now()
		authTime := now.Add(30 * time.Millisecond)

		ticket := createValidTicket(now)

		differentClient, _ := protocol.NewPrincipal("bob", "", "ATHENA.MIT.EDU")
		auth := createAuthenticator(differentClient, authTime)
		req, _ := protocol.NewAPReq(ticket, auth)

		_, err := verifier.Verify(req)
		assert.Err(t, err, "client mismatch")
	})

	t.Run("ClockSkewTooGreat", func(t *testing.T) {
		now := testClock.Now()
		oldTime := now.Add(-10 * time.Minute)

		ticket := createValidTicket(now)
		auth := createAuthenticator(client, oldTime)
		req, _ := protocol.NewAPReq(ticket, auth)

		_, err := verifier.Verify(req)
		assert.Err(t, err, ap.ErrClockSkewTooGreat)
	})

	t.Run("TicketExpired", func(t *testing.T) {
		now := testClock.Now()
		authTime := now.Add(40 * time.Millisecond)

		issuedAt := now.Add(-10 * time.Hour)
		ticket := createValidTicket(issuedAt)
		auth := createAuthenticator(client, authTime)
		req, _ := protocol.NewAPReq(ticket, auth)

		_, err := verifier.Verify(req)
		assert.Err(t, err, ap.ErrTicketExpired)
	})

	t.Run("ReplayDetected", func(t *testing.T) {
		now := testClock.Now()
		authTime := now.Add(50 * time.Millisecond)

		ticket := createValidTicket(now)
		auth := createAuthenticator(client, authTime)
		req, _ := protocol.NewAPReq(ticket, auth)

		_, err := verifier.Verify(req)
		assert.Err(t, err, nil)

		// Replay same authenticator
		_, err = verifier.Verify(req)
		assert.Err(t, err, replay.ErrReplayDetected)
	})
}
