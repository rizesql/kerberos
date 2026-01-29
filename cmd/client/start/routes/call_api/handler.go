package callapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/rizesql/kerberos/cmd/client/start/platform"
	"github.com/rizesql/kerberos/internal/crypto"
	"github.com/rizesql/kerberos/internal/protocol"
	"github.com/rizesql/kerberos/internal/server"
)

type handler struct {
	cache *platform.TicketCache
}

// CallServiceRoute - POST /api/call
// Calls protected Api Server endpoint
func NewHandler(platform *platform.Platform) *handler {
	return &handler{
		cache: platform.Cache,
	}
}

func (*handler) Method() string { return http.MethodPost }
func (*handler) Path() string   { return "/api/call" }

type request struct {
	URL string `json:"url"` // e.g., "http://localhost:9090/api/whoami"
}

func (h *handler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, err := server.Decode[request](req)
		if err != nil {
			server.EncodeError(w, http.StatusBadRequest, err)
			return
		}

		respBody, statusCode, err := h.callService(req.Context(), body)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		// w.Header().Set("Content-Type", "application/json")
		// w.WriteHeader(statusCode)
		// w.Write(respBody)
		if err := server.Encode(w, statusCode, respBody); err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}
	}
}

func (h *handler) callService(ctx context.Context, req request) ([]byte, int, error) {
	// 1. Get service ticket and session key from cache
	ticket := h.cache.GetServiceTicket("http/api-server")
	if ticket == nil {
		return nil, 0, fmt.Errorf("no service ticket cached")
	}

	serviceSessionKey := h.cache.GetServiceSessionKey("http/api-server")
	if serviceSessionKey == nil {
		return nil, 0, fmt.Errorf("no service session key")
	}

	clientPrincipal := h.cache.GetClientPrincipal()

	// 2. Build AP-REQ with encrypted authenticator
	addr, err := protocol.NewAddress(net.IPv4(127, 0, 0, 1))
	if err != nil {
		return nil, 0, err
	}

	authenticator, err := protocol.NewAuthenticator(clientPrincipal, addr, time.Now())
	if err != nil {
		return nil, 0, err
	}

	// ENCRYPT the authenticator with the service session key
	authBytes, err := json.Marshal(authenticator)
	if err != nil {
		return nil, 0, err
	}

	encryptedAuthBytes, err := crypto.Encrypt(*serviceSessionKey, authBytes)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to encrypt authenticator: %w", err)
	}

	encAuth, err := protocol.NewEncryptedData(encryptedAuthBytes)
	if err != nil {
		return nil, 0, err
	}

	apReq, err := protocol.NewAPReq(*ticket, encAuth)
	if err != nil {
		return nil, 0, err
	}

	// 3. Serialize AP-REQ
	apReqBytes, err := json.Marshal(apReq)
	if err != nil {
		return nil, 0, err
	}

	// 4. Call protected endpoint
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		return nil, 0, err
	}

	httpReq.Header.Set("Authorization", "Kerberos "+base64.StdEncoding.EncodeToString(apReqBytes))

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, 0, fmt.Errorf("service unreachable: %w", err)
	}
	defer resp.Body.Close()

	// 5. Return service response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	return respBody, resp.StatusCode, nil
}
