package tgs_test

import (
	"encoding/hex"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/crypto"
	"github.com/rizesql/kerberos/internal/kdb"
	"github.com/rizesql/kerberos/internal/kdc"
	"github.com/rizesql/kerberos/internal/kdc/shared"
	"github.com/rizesql/kerberos/internal/kdc/tgs"
	"github.com/rizesql/kerberos/internal/protocol"
	"github.com/rizesql/kerberos/internal/testkit"
)

func TestHandler(t *testing.T) {
	h := testkit.NewHarness(t)

	// Keys
	clientKeyBytes, _ := hex.DecodeString("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	tgsKeyBytes, _ := hex.DecodeString("ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100")
	serviceKeyBytes, _ := hex.DecodeString("aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899")

	tgtSessionKeyBytes, _ := hex.DecodeString("1122334455667788990011223344556677889900112233445566778899001122")
	tgtSessionKey, _ := protocol.NewSessionKey(tgtSessionKeyBytes)

	expectedNewSessionKeyBytes, _ := hex.DecodeString("112233445566778899aabbccddeeff00112233445566778899aabbccddeeff00")
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

	srv := h.NewServer()
	tgs := tgs.NewHandler(h.NewKDCPlatform(), kdc.Config{
		Realm:          "ATHENA.MIT.EDU",
		TicketLifetime: 8 * time.Hour,
	})
	srv.Register(tgs)

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

	createValidAuthenticator := func(issuedAt time.Time) protocol.EncryptedData {
		auth, _ := protocol.NewAuthenticator(client, clientAddr, issuedAt)
		encAuth, _ := shared.EncryptEntity(tgtSessionKey, auth)
		return encAuth
	}

	now := h.Clock.Now()
	encTGT := createValidTGT(now, 8*time.Hour)
	encAuth := createValidAuthenticator(now)
	nonce, _ := protocol.NewNonce(12345)

	req, _ := protocol.NewTGSReq(servicePrincipal, encTGT, encAuth, nonce)

	// Call
	res := testkit.Call[protocol.TGSReq, protocol.TGSRep](t, srv, tgs, nil, req)
	assert.Equal(t, res.Status, http.StatusOK)
	if res.Body == nil {
		t.Fatal("response body is nil")
	}

	secretPartBytes, err := crypto.Decrypt(tgtSessionKey, res.Body.SecretPart().Ciphertext())
	assert.Err(t, err, nil)

	var encPart protocol.EncKDCRepPart
	err = encPart.UnmarshalJSON(secretPartBytes)
	assert.Err(t, err, nil)

	assert.Equal(t, encPart.Nonce(), nonce)
	assert.Equal(t, encPart.Server().String(), servicePrincipal.String())
	assert.Equal(t, encPart.IssuedAt().Equal(now), true)
	assert.Equal(t, string(encPart.SessionKey().Expose()), string(expectedNewSessionKey.Expose()))

	serviceKey, _ := protocol.NewSessionKey(serviceKeyBytes)
	ticketBytes, err := crypto.Decrypt(serviceKey, res.Body.Ticket().Ciphertext())
	assert.Err(t, err, nil)

	var ticket protocol.Ticket
	err = ticket.UnmarshalJSON(ticketBytes)
	assert.Err(t, err, nil)

	assert.Equal(t, ticket.Client().String(), client.String())
	assert.Equal(t, ticket.Server().String(), servicePrincipal.String())
	assert.Equal(t, ticket.IssuedAt().Equal(now), true)
	assert.Equal(t, string(ticket.SessionKey().Expose()), string(expectedNewSessionKey.Expose()))
}
