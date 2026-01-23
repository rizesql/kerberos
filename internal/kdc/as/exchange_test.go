package as_test

import (
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/crypto"
	"github.com/rizesql/kerberos/internal/kdb"
	"github.com/rizesql/kerberos/internal/kdc"
	"github.com/rizesql/kerberos/internal/kdc/as"
	"github.com/rizesql/kerberos/internal/kdc/shared"
	"github.com/rizesql/kerberos/internal/protocol"
	"github.com/rizesql/kerberos/internal/testkit"
)

func TestExchange(t *testing.T) {
	h := testkit.NewHarness(t)

	// Keys
	clientKeyBytes, _ := hex.DecodeString("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	serviceKeyBytes, _ := hex.DecodeString("ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100")
	expectedSessionKeyBytes, _ := hex.DecodeString("112233445566778899aabbccddeeff00112233445566778899aabbccddeeff00")
	expectedSessionKey, _ := protocol.NewSessionKey(expectedSessionKeyBytes)

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
		KeyBytes:    serviceKeyBytes,
		Kvno:        1,
	})

	exchange := as.NewExchange(h.NewKDCPlatform(), kdc.Config{
		Realm:          "ATHENA.MIT.EDU",
		TicketLifetime: 8 * time.Hour,
	})

	// Inputs
	client, _ := protocol.NewPrincipal("alice", "", "ATHENA.MIT.EDU")
	service, _ := protocol.NewPrincipal("krbtgt", "ATHENA.MIT.EDU", "ATHENA.MIT.EDU")
	addr, _ := protocol.NewAddress(net.IPv4(127, 0, 0, 1))
	nonce, _ := protocol.NewNonce(999)
	req, _ := protocol.NewASReq(client, service, addr, nonce)

	// --- 1. Success Case ---
	rep, err := exchange.Handle(t.Context(), req)
	assert.Err(t, err, nil)

	// Verify Secret Part (encrypted with Client Key)
	clientKey, _ := protocol.NewSessionKey(clientKeyBytes)
	secretPartBytes, err := crypto.Decrypt(clientKey, rep.SecretPart().Ciphertext())
	assert.Err(t, err, nil)

	var encPart protocol.EncKDCRepPart
	err = encPart.UnmarshalJSON(secretPartBytes)
	assert.Err(t, err, nil)

	assert.Equal(t, encPart.Nonce(), nonce)
	assert.Equal(t, encPart.Server().String(), service.String())
	assert.Equal(t, encPart.IssuedAt().Equal(h.Clock.Now()), true)
	assert.Equal(t, string(encPart.SessionKey().Expose()), string(expectedSessionKey.Expose()))

	// Verify Ticket (encrypted with Service Key)
	serviceKey, _ := protocol.NewSessionKey(serviceKeyBytes)
	ticketBytes, err := crypto.Decrypt(serviceKey, rep.Ticket().Ciphertext())
	assert.Err(t, err, nil)

	var ticket protocol.Ticket
	err = ticket.UnmarshalJSON(ticketBytes)
	assert.Err(t, err, nil)

	assert.Equal(t, ticket.Client().String(), client.String())
	assert.Equal(t, ticket.Server().String(), service.String())
	assert.Equal(t, ticket.IssuedAt().Equal(h.Clock.Now()), true)
	assert.Equal(t, string(ticket.SessionKey().Expose()), string(expectedSessionKey.Expose()))

	// --- 2. Wrong Realm ---
	wrongClient, _ := protocol.NewPrincipal("alice", "", "OTHER.REALM")
	wrongNonce, _ := protocol.NewNonce(999)
	wrongReq, _ := protocol.NewASReq(wrongClient, service, addr, wrongNonce)
	_, err = exchange.Handle(t.Context(), wrongReq)
	assert.Err(t, err, shared.ErrWrongRealm)

	// --- 3. Client Not Found ---
	unknownClient, _ := protocol.NewPrincipal("bob", "", "ATHENA.MIT.EDU")
	notFoundNonce, _ := protocol.NewNonce(999)
	notFoundReq, _ := protocol.NewASReq(unknownClient, service, addr, notFoundNonce)
	_, err = exchange.Handle(t.Context(), notFoundReq)
	assert.Err(t, err, shared.ErrPrincipalNotFound)

	// --- 4. Service Not Found ---
	unknownService, _ := protocol.NewPrincipal("http", "unknown", "ATHENA.MIT.EDU")
	svcNotFoundNonce, _ := protocol.NewNonce(999)
	svcNotFoundReq, _ := protocol.NewASReq(client, unknownService, addr, svcNotFoundNonce)
	_, err = exchange.Handle(t.Context(), svcNotFoundReq)
	assert.Err(t, err, shared.ErrPrincipalNotFound)
}
