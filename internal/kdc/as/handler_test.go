package as_test

import (
	"encoding/hex"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/rizesql/kerberos/internal/assert"
	"github.com/rizesql/kerberos/internal/crypto"
	"github.com/rizesql/kerberos/internal/kdb"
	"github.com/rizesql/kerberos/internal/kdc/as"
	"github.com/rizesql/kerberos/internal/protocol"
	"github.com/rizesql/kerberos/internal/testkit"
)

func TestHandler(t *testing.T) {
	h := testkit.NewHarness(t)

	// Setup keys
	clientKeyBytes, _ := hex.DecodeString("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	serviceKeyBytes, _ := hex.DecodeString("ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100")

	clientKey, _ := protocol.NewSessionKey(clientKeyBytes)

	// Seed DB
	h.CreatePrincipal(t.Context(), kdb.CreatePrincipalParams{
		PrimaryName: "client",
		Instance:    "user",
		Realm:       "TEST.REALM",
		KeyBytes:    clientKeyBytes,
		Kvno:        1,
	})

	h.CreatePrincipal(t.Context(), kdb.CreatePrincipalParams{
		PrimaryName: "service",
		Instance:    "http",
		Realm:       "TEST.REALM",
		KeyBytes:    serviceKeyBytes,
		Kvno:        1,
	})

	srv := h.NewServer()
	as := as.NewHandler(h.NewKDCPlatform(), as.Config{
		Realm:      "TEST.REALM",
		TicketLife: 1 * time.Hour,
	})
	srv.Register(as)

	client, _ := protocol.NewPrincipal("client", "user", "TEST.REALM")
	service, _ := protocol.NewPrincipal("service", "http", "TEST.REALM")
	addr, _ := protocol.NewAddress(net.IPv4(127, 0, 0, 1))
	nonce, _ := protocol.NewNonce(123456)
	req, _ := protocol.NewASReq(client, service, addr, nonce)

	// Call
	resp := testkit.Call[protocol.ASReq, protocol.ASRep](t, srv, as, nil, req)

	assert.Equal(t, resp.Status, http.StatusOK)
	if resp.Body == nil {
		t.Fatal("response body is nil")
	}

	// Decrypt SecretPart using ClientKey
	encSecretPart := resp.Body.SecretPart()
	secretPartBytes, err := crypto.Decrypt(clientKey, encSecretPart.Ciphertext())
	assert.Err(t, err, nil)

	var repPart protocol.EncKDCRepPart
	err = repPart.UnmarshalJSON(secretPartBytes)
	assert.Err(t, err, nil)

	assert.Equal(t, repPart.Nonce(), nonce)
	assert.Equal(t, repPart.Server(), service)
}

func TestHandler_WrongRealm(t *testing.T) {
	h := testkit.NewHarness(t)

	srv := h.NewServer()
	as := as.NewHandler(h.NewKDCPlatform(), as.Config{
		Realm: "TEST.REALM",
	})
	srv.Register(as)

	// Wrong Realm
	client, _ := protocol.NewPrincipal("client", "", "OTHER.REALM")
	service, _ := protocol.NewPrincipal("krbtgt", "", "TEST.REALM")
	addr, _ := protocol.NewAddress(net.IPv4(127, 0, 0, 1))
	nonce, _ := protocol.NewNonce(123)
	req, _ := protocol.NewASReq(client, service, addr, nonce)

	resp := testkit.Call[protocol.ASReq, any](t, srv, as, nil, req)
	assert.Equal(t, resp.Status, http.StatusBadRequest)
}

func TestHandler_PrincipalNotFound(t *testing.T) {
	h := testkit.NewHarness(t)

	// Seed only service, so client lookup fails
	serviceKeyBytes, _ := hex.DecodeString("ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100")
	h.CreatePrincipal(t.Context(), kdb.CreatePrincipalParams{
		PrimaryName: "service",
		Instance:    "http",
		Realm:       "TEST.REALM",
		KeyBytes:    serviceKeyBytes,
		Kvno:        1,
	})

	srv := h.NewServer()
	as := as.NewHandler(h.NewKDCPlatform(), as.Config{
		Realm: "TEST.REALM",
	})
	srv.Register(as)

	// Client Not Found
	client, _ := protocol.NewPrincipal("unknown", "user", "TEST.REALM")
	service, _ := protocol.NewPrincipal("service", "http", "TEST.REALM")
	addr, _ := protocol.NewAddress(net.IPv4(127, 0, 0, 1))
	nonce, _ := protocol.NewNonce(123)
	req, _ := protocol.NewASReq(client, service, addr, nonce)

	resp := testkit.Call[protocol.ASReq, any](t, srv, as, nil, req)
	assert.Equal(t, resp.Status, http.StatusNotFound)
}
