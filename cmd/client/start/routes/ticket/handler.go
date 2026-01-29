package ticket

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/rizesql/kerberos/cmd/client/start/platform"
	"github.com/rizesql/kerberos/internal/crypto"
	"github.com/rizesql/kerberos/internal/protocol"
	"github.com/rizesql/kerberos/internal/sdk"
	"github.com/rizesql/kerberos/internal/server"
)

type handler struct {
	sdk   *sdk.Sdk
	cache *platform.TicketCache
}

// GetTicketRoute - POST /api/ticket
// Calls KDC TGS Exchange, stores service ticket
func NewHandler(platform *platform.Platform) *handler {
	return &handler{
		sdk:   platform.Sdk,
		cache: platform.Cache,
	}
}

func (*handler) Method() string { return http.MethodPost }
func (*handler) Path() string   { return "/api/ticket" }

type request struct {
	Service string `json:"service"` // e.g., "http/api-server"
}

type response struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Ticket  string `json:"ticket"`
}

func (h *handler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, err := server.Decode[request](req)
		if err != nil {
			server.EncodeError(w, http.StatusBadRequest, err)
			return
		}

		res, err := h.getTicket(req.Context(), body)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		if err := server.Encode(w, http.StatusOK, res); err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}
	}
}

func (h *handler) getTicket(ctx context.Context, req request) (*response, error) {
	// 1. Get TGT and session key from cache
	tgt := h.cache.GetTGT()
	if tgt == nil {
		return nil, fmt.Errorf("not logged in")
	}

	sessionKey := h.cache.GetTGTSessionKey()
	if sessionKey == nil {
		return nil, fmt.Errorf("no session key")
	}

	clientPrincipal := h.cache.GetClientPrincipal()

	// 2. Parse service principal
	if req.Service != "http/api-server" {
		return nil, fmt.Errorf("unsupported service: %s", req.Service)
	}

	servicePrincipal, err := protocol.NewPrincipal("http", "api-server", "ATHENA.MIT.EDU")
	if err != nil {
		return nil, fmt.Errorf("invalid service: %w", err)
	}

	// 3. Create authenticator with REAL client principal
	addr, err := protocol.NewAddress(net.IPv4(127, 0, 0, 1))
	if err != nil {
		return nil, err
	}

	timestamp := time.Now()

	authenticator, err := protocol.NewAuthenticator(clientPrincipal, addr, timestamp)
	if err != nil {
		return nil, err
	}

	nonce, err := protocol.NewNonce(int32(time.Now().UnixNano()%100000 + 1))
	if err != nil {
		return nil, err
	}

	// 4. ENCRYPT authenticator with session key from AS-REP
	authBytes, err := json.Marshal(authenticator)
	if err != nil {
		return nil, err
	}

	encryptedAuthBytes, err := crypto.Encrypt(*sessionKey, authBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt authenticator: %w", err)
	}

	encAuth, err := protocol.NewEncryptedData(encryptedAuthBytes)
	if err != nil {
		return nil, err
	}

	// 5. Build TGS-REQ
	tgsReq, err := protocol.NewTGSReq(servicePrincipal, *tgt, encAuth, nonce)
	if err != nil {
		return nil, err
	}

	tgsRep, err := h.sdk.Kdc.PostTGS(ctx, tgsReq)
	if err != nil {
		return nil, fmt.Errorf("invalid kdc response: %w", err)
	}

	// 6. Decrypt SecretPart to get service session key
	serviceSecretBytes, err := crypto.Decrypt(*sessionKey, tgsRep.SecretPart().Ciphertext())
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt service session key: %w", err)
	}

	var serviceEncRepPart protocol.EncKDCRepPart
	if err := json.Unmarshal(serviceSecretBytes, &serviceEncRepPart); err != nil {
		return nil, fmt.Errorf("invalid service secret part format: %w", err)
	}

	serviceSessionKey := serviceEncRepPart.SessionKey()

	// 7. Store service ticket WITH session key in cache
	h.cache.StoreServiceTicketWithSession(req.Service, tgsRep.Ticket(), serviceSessionKey)

	return &response{
		Status:  "ticket_obtained",
		Service: req.Service,
		Ticket:  base64.StdEncoding.EncodeToString(tgsRep.Ticket().Ciphertext()),
	}, nil
}
