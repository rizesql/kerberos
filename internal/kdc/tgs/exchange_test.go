package tgs_test

import (
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/crypto"
	"github.com/rizesql/kerberos/internal/kdb"
	"github.com/rizesql/kerberos/internal/kdc"
	"github.com/rizesql/kerberos/internal/kdc/shared"
	"github.com/rizesql/kerberos/internal/kdc/tgs"
	"github.com/rizesql/kerberos/internal/protocol"
	"github.com/rizesql/kerberos/internal/replay"
	"github.com/rizesql/kerberos/internal/testkit"
)

func TestExchange(t *testing.T) {
	h := testkit.NewHarness(t)

	// Keys
	clientKeyBytes, _ := hex.DecodeString("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	tgsKeyBytes, _ := hex.DecodeString("ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100")
	serviceKeyBytes, _ := hex.DecodeString("aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899")
	tgtSessionKeyBytes, _ := hex.DecodeString("1122334455667788990011223344556677889900112233445566778899001122")
	expectedNewSessionKeyBytes, _ := hex.DecodeString("112233445566778899aabbccddeeff00112233445566778899aabbccddeeff00")

	tgtSessionKey, _ := protocol.NewSessionKey(tgtSessionKeyBytes)
	expectedNewSessionKey, _ := protocol.NewSessionKey(expectedNewSessionKeyBytes)

	// Principals
	client, _ := protocol.NewPrincipal("alice", "", "ATHENA.MIT.EDU")
	tgsPrincipal, _ := protocol.NewPrincipal("krbtgt", "ATHENA.MIT.EDU", "ATHENA.MIT.EDU")
	servicePrincipal, _ := protocol.NewPrincipal("http", "server.athena.mit.edu", "ATHENA.MIT.EDU")
	clientAddr, _ := protocol.NewAddress(net.IPv4(127, 0, 0, 1))

	// Seed DB
	h.CreatePrincipal(t.Context(), kdb.CreatePrincipalParams{
		PrimaryName: "alice",
		Instance:    "",
		Realm:       "ATHENA.MIT.EDU",
		KeyBytes:    clientKeyBytes,
		Kvno:        1,
	})
	h.CreatePrincipal(t.Context(), kdb.CreatePrincipalParams{
		PrimaryName: "krbtgt",
		Instance:    "ATHENA.MIT.EDU",
		Realm:       "ATHENA.MIT.EDU",
		KeyBytes:    tgsKeyBytes,
		Kvno:        1,
	})
	h.CreatePrincipal(t.Context(), kdb.CreatePrincipalParams{
		PrimaryName: "http",
		Instance:    "server.athena.mit.edu",
		Realm:       "ATHENA.MIT.EDU",
		KeyBytes:    serviceKeyBytes,
		Kvno:        1,
	})

	exchange := tgs.NewExchange(h.NewKDCPlatform(), kdc.Config{
		Realm:          "ATHENA.MIT.EDU",
		TicketLifetime: 8 * time.Hour,
	})

	// Helper to create a valid TGT encrypted with TGS key.
	createValidTGT := func(issuedAt time.Time, lifetime time.Duration) protocol.EncryptedData {
		tgt, _ := protocol.NewTicket(
			tgsPrincipal,
			client,
			clientAddr,
			issuedAt,
			lifetime,
			tgtSessionKey,
		)
		tgsKey, _ := protocol.NewSessionKey(tgsKeyBytes)
		encTGT, _ := shared.EncryptEntity(tgsKey, tgt)
		return encTGT
	}

	// Helper to create a valid authenticator encrypted with TGT session key.
	createValidAuthenticator := func(issuedAt time.Time) protocol.EncryptedData {
		auth, _ := protocol.NewAuthenticator(client, clientAddr, issuedAt)
		encAuth, _ := shared.EncryptEntity(tgtSessionKey, auth)
		return encAuth
	}

	// --- 1. Success Case ---
	t.Run("Success", func(t *testing.T) {
		now := h.Clock.Now()
		authTime := now.Add(100 * time.Millisecond) // Unique timestamp for this test
		encTGT := createValidTGT(now, 8*time.Hour)
		encAuth := createValidAuthenticator(authTime)
		nonce, _ := protocol.NewNonce(12345)

		req, _ := protocol.NewTGSReq(servicePrincipal, encTGT, encAuth, nonce)
		rep, err := exchange.Handle(t.Context(), req)
		assert.Err(t, err, nil)

		// Verify Secret Part (encrypted with TGT session key)
		secretPartBytes, err := crypto.Decrypt(tgtSessionKey, rep.SecretPart().Ciphertext())
		assert.Err(t, err, nil)

		var encPart protocol.EncKDCRepPart
		err = encPart.UnmarshalJSON(secretPartBytes)
		assert.Err(t, err, nil)

		assert.Equal(t, encPart.Nonce(), nonce)
		assert.Equal(t, encPart.Server().String(), servicePrincipal.String())
		assert.Equal(t, encPart.IssuedAt().Equal(now), true)
		assert.Equal(t, string(encPart.SessionKey().Expose()), string(expectedNewSessionKey.Expose()))

		// Verify Service Ticket (encrypted with Service Key)
		serviceKey, _ := protocol.NewSessionKey(serviceKeyBytes)
		ticketBytes, err := crypto.Decrypt(serviceKey, rep.Ticket().Ciphertext())
		assert.Err(t, err, nil)

		var ticket protocol.Ticket
		err = ticket.UnmarshalJSON(ticketBytes)
		assert.Err(t, err, nil)

		assert.Equal(t, ticket.Client().String(), client.String())
		assert.Equal(t, ticket.Server().String(), servicePrincipal.String())
		assert.Equal(t, ticket.IssuedAt().Equal(now), true)
		assert.Equal(t, string(ticket.SessionKey().Expose()), string(expectedNewSessionKey.Expose()))
	})

	// --- 2. Invalid TGT (wrong encryption key) ---
	t.Run("InvalidTGT", func(t *testing.T) {
		now := h.Clock.Now()
		authTime := now.Add(200 * time.Millisecond) // Unique timestamp for this test

		// Encrypt TGT with the wrong key (client key instead of TGS key)
		tgt, _ := protocol.NewTicket(
			tgsPrincipal,
			client,
			clientAddr,
			now,
			8*time.Hour,
			tgtSessionKey,
		)
		wrongKey, _ := protocol.NewSessionKey(clientKeyBytes)
		encTGT, _ := shared.EncryptEntity(wrongKey, tgt)

		encAuth := createValidAuthenticator(authTime)
		nonce, _ := protocol.NewNonce(12346)

		req, _ := protocol.NewTGSReq(servicePrincipal, encTGT, encAuth, nonce)
		_, err := exchange.Handle(t.Context(), req)

		// Should fail with "invalid TGT" error
		assert.Err(t, err, "invalid TGT")
	})

	// --- 3. Invalid Authenticator (wrong encryption key) ---
	t.Run("InvalidAuthenticator", func(t *testing.T) {
		now := h.Clock.Now()
		authTime := now.Add(300 * time.Millisecond) // Unique timestamp for this test
		encTGT := createValidTGT(now, 8*time.Hour)

		// Encrypt authenticator with wrong key
		auth, _ := protocol.NewAuthenticator(client, clientAddr, authTime)
		wrongKey, _ := protocol.NewSessionKey(clientKeyBytes)
		encAuth, _ := shared.EncryptEntity(wrongKey, auth)

		nonce, _ := protocol.NewNonce(12347)

		req, _ := protocol.NewTGSReq(servicePrincipal, encTGT, encAuth, nonce)
		_, err := exchange.Handle(t.Context(), req)

		// Should fail with "invalid authenticator" error
		assert.Err(t, err, "invalid authenticator")
	})

	// --- 4. Client Mismatch (TGT client != Authenticator client) ---
	t.Run("ClientMismatch", func(t *testing.T) {
		now := h.Clock.Now()
		authTime := now.Add(400 * time.Millisecond) // Unique timestamp for this test
		encTGT := createValidTGT(now, 8*time.Hour)

		// Create authenticator with different client
		differentClient, _ := protocol.NewPrincipal("bob", "", "ATHENA.MIT.EDU")
		auth, _ := protocol.NewAuthenticator(differentClient, clientAddr, authTime)
		encAuth, _ := shared.EncryptEntity(tgtSessionKey, auth)

		nonce, _ := protocol.NewNonce(12348)

		req, _ := protocol.NewTGSReq(servicePrincipal, encTGT, encAuth, nonce)
		_, err := exchange.Handle(t.Context(), req)

		// Should fail with client mismatch
		assert.Err(t, err, "client mismatch")
	})

	// --- 5. Clock Skew Too Great (authenticator timestamp too old) ---
	t.Run("ClockSkewTooGreat_Past", func(t *testing.T) {
		now := h.Clock.Now()
		encTGT := createValidTGT(now, 8*time.Hour)

		// Create authenticator with timestamp 10 minutes in the past
		oldTime := now.Add(-10 * time.Minute)
		auth, _ := protocol.NewAuthenticator(client, clientAddr, oldTime)
		encAuth, _ := shared.EncryptEntity(tgtSessionKey, auth)

		nonce, _ := protocol.NewNonce(12349)

		req, _ := protocol.NewTGSReq(servicePrincipal, encTGT, encAuth, nonce)
		_, err := exchange.Handle(t.Context(), req)

		assert.Err(t, err, "clock skew too great")
	})

	// --- 6. Clock Skew Too Great (authenticator timestamp too far in future) ---
	t.Run("ClockSkewTooGreat_Future", func(t *testing.T) {
		now := h.Clock.Now()
		encTGT := createValidTGT(now, 8*time.Hour)

		// Create authenticator with timestamp 10 minutes in the future
		futureTime := now.Add(10 * time.Minute)
		auth, _ := protocol.NewAuthenticator(client, clientAddr, futureTime)
		encAuth, _ := shared.EncryptEntity(tgtSessionKey, auth)

		nonce, _ := protocol.NewNonce(12350)

		req, _ := protocol.NewTGSReq(servicePrincipal, encTGT, encAuth, nonce)
		_, err := exchange.Handle(t.Context(), req)

		assert.Err(t, err, "clock skew too great")
	})

	// --- 7. TGT Expired ---
	t.Run("TGTExpired", func(t *testing.T) {
		now := h.Clock.Now()
		authTime := now.Add(700 * time.Millisecond) // Unique timestamp for this test

		// Create TGT that was issued 10 hours ago with 8-hour lifetime (expired 2 hours ago)
		issuedAt := now.Add(-10 * time.Hour)
		encTGT := createValidTGT(issuedAt, 8*time.Hour)

		encAuth := createValidAuthenticator(authTime)
		nonce, _ := protocol.NewNonce(12351)

		req, _ := protocol.NewTGSReq(servicePrincipal, encTGT, encAuth, nonce)
		_, err := exchange.Handle(t.Context(), req)

		assert.Err(t, err, "TGT expired")
	})

	// --- 8. Service Not Found ---
	t.Run("ServiceNotFound", func(t *testing.T) {
		now := h.Clock.Now()
		authTime := now.Add(800 * time.Millisecond) // Unique timestamp for this test
		encTGT := createValidTGT(now, 8*time.Hour)
		encAuth := createValidAuthenticator(authTime)

		unknownService, _ := protocol.NewPrincipal("ldap", "unknown.server", "ATHENA.MIT.EDU")
		nonce, _ := protocol.NewNonce(12352)

		req, _ := protocol.NewTGSReq(unknownService, encTGT, encAuth, nonce)
		_, err := exchange.Handle(t.Context(), req)

		assert.Err(t, err, shared.ErrPrincipalNotFound)
	})

	// --- 9. Replay Attack Detection ---
	t.Run("ReplayAttackDetected", func(t *testing.T) {
		now := h.Clock.Now()
		encTGT := createValidTGT(now, 8*time.Hour)

		// Create authenticator with a specific timestamp
		replayTime := now.Add(1 * time.Millisecond) // Unique timestamp for this test
		auth, _ := protocol.NewAuthenticator(client, clientAddr, replayTime)
		encAuth, _ := shared.EncryptEntity(tgtSessionKey, auth)

		nonce1, _ := protocol.NewNonce(99001)
		req1, _ := protocol.NewTGSReq(servicePrincipal, encTGT, encAuth, nonce1)

		// First request should succeed
		_, err := exchange.Handle(t.Context(), req1)
		assert.Err(t, err, nil)

		// Replay: same authenticator (same client + timestamp) should be rejected
		nonce2, _ := protocol.NewNonce(99002)
		req2, _ := protocol.NewTGSReq(servicePrincipal, encTGT, encAuth, nonce2)

		_, err = exchange.Handle(t.Context(), req2)
		assert.Err(t, err, replay.ErrReplayDetected)
	})

	// --- 10. Different Timestamps Are Not Replays ---
	t.Run("DifferentTimestampsAllowed", func(t *testing.T) {
		now := h.Clock.Now()
		encTGT := createValidTGT(now, 8*time.Hour)

		// First request with timestamp T1
		t1 := now.Add(2 * time.Millisecond)
		auth1, _ := protocol.NewAuthenticator(client, clientAddr, t1)
		encAuth1, _ := shared.EncryptEntity(tgtSessionKey, auth1)
		nonce1, _ := protocol.NewNonce(99010)
		req1, _ := protocol.NewTGSReq(servicePrincipal, encTGT, encAuth1, nonce1)

		_, err := exchange.Handle(t.Context(), req1)
		assert.Err(t, err, nil)

		// Second request with different timestamp T2 should also succeed
		t2 := now.Add(3 * time.Millisecond)
		auth2, _ := protocol.NewAuthenticator(client, clientAddr, t2)
		encAuth2, _ := shared.EncryptEntity(tgtSessionKey, auth2)
		nonce2, _ := protocol.NewNonce(99011)
		req2, _ := protocol.NewTGSReq(servicePrincipal, encTGT, encAuth2, nonce2)

		_, err = exchange.Handle(t.Context(), req2)
		assert.Err(t, err, nil)
	})
}
