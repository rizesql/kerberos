package login

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

// LoginRoute - POST /api/login
// Calls KDC AS Exchange, stores TGT
func NewHandler(platform *platform.Platform) *handler {
	return &handler{
		sdk:   platform.Sdk,
		cache: platform.Cache,
	}
}

func (*handler) Method() string { return http.MethodPost }
func (*handler) Path() string   { return "/api/login" }

type request struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type response struct {
	Status       string `json:"status"`
	User         string `json:"user"`
	TGTEncrypted bool   `json:"tgt_encrypted"`
	SessionKey   string `json:"session_key"`
}

func (h *handler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, err := server.Decode[request](req)
		if err != nil {
			server.EncodeError(w, http.StatusBadRequest, err)
			return
		}

		res, err := h.login(req.Context(), body)
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

func (h *handler) login(ctx context.Context, req request) (*response, error) {
	// 1. Build AS-REQ
	client, err := protocol.NewPrincipal(protocol.Primary(req.Username), "", "ATHENA.MIT.EDU")
	if err != nil {
		return nil, fmt.Errorf("invalid username: %w", err)
	}

	tgsPrincipal, err := protocol.NewKrbtgt("ATHENA.MIT.EDU")
	if err != nil {
		return nil, fmt.Errorf("invalid krbtgt: %w", err)
	}

	nonce, err := protocol.NewNonce(int32(time.Now().UnixNano()%100000 + 1))
	if err != nil {
		return nil, err
	}

	addr, err := protocol.NewAddress(net.IPv4(127, 0, 0, 1))
	if err != nil {
		return nil, err
	}

	asReq, err := protocol.NewASReq(client, tgsPrincipal, addr, nonce)
	if err != nil {
		return nil, err
	}

	asRep, err := h.sdk.Kdc.PostAS(ctx, asReq)
	if err != nil {
		return nil, fmt.Errorf("invalid kdc response: %w", err)
	}

	// 2. Derive key from password using correct salt (realm + primary + instance)
	salt := "ATHENA.MIT.EDU" + req.Username // realm + primary + instance (empty for alice)
	clientKey, err := crypto.DeriveKey(req.Password, salt)
	if err != nil {
		return nil, fmt.Errorf("key derivation failed: %w", err)
	}

	// 3. Decrypt SecretPart to get session key
	secretPartBytes, err := crypto.Decrypt(clientKey, asRep.SecretPart().Ciphertext())
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt session key (wrong password?): %w", err)
	}

	var encRepPart protocol.EncKDCRepPart
	if err := json.Unmarshal(secretPartBytes, &encRepPart); err != nil {
		return nil, fmt.Errorf("invalid secret part format: %w", err)
	}

	sessionKey := encRepPart.SessionKey()

	// 4. Store TGT with session key and client principal in cache
	h.cache.StoreTGTWithSession(asRep.Ticket(), sessionKey, client)

	return &response{
		Status:       "logged_in",
		User:         req.Username,
		TGTEncrypted: true,
		SessionKey:   base64.StdEncoding.EncodeToString(sessionKey.Expose()),
	}, nil
}
