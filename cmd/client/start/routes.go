package start

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/rizesql/kerberos/internal/crypto"
	"github.com/rizesql/kerberos/internal/protocol"
	"github.com/rizesql/kerberos/internal/server"
)

// LoginRoute - POST /api/login
// Calls KDC AS Exchange, stores TGT
type LoginRoute struct {
	kdcAddr string
	cache   *TicketCache
}

func (r *LoginRoute) Method() string { return http.MethodPost }
func (r *LoginRoute) Path() string   { return "/api/login" }
func (r *LoginRoute) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			server.EncodeError(w, http.StatusBadRequest, err)
			return
		}

		// 1. Build AS-REQ
		client, err := protocol.NewPrincipal(protocol.Primary(body.Username), "", "ATHENA.MIT.EDU")
		if err != nil {
			server.EncodeError(w, http.StatusBadRequest, fmt.Errorf("invalid username: %w", err))
			return
		}

		tgsPrincipal, err := protocol.NewKrbtgt("ATHENA.MIT.EDU")
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, fmt.Errorf("invalid krbtgt: %w", err))
			return
		}

		nonce, err := protocol.NewNonce(int32(time.Now().UnixNano()%100000 + 1))
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		addr, err := protocol.NewAddress(net.IPv4(127, 0, 0, 1))
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		asReq, err := protocol.NewASReq(client, tgsPrincipal, addr, nonce)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		// 2. POST to KDC AS Exchange
		reqBody, err := json.Marshal(asReq)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		resp, err := http.Post(r.kdcAddr+"/as", "application/json", bytes.NewReader(reqBody))
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, fmt.Errorf("kdc unreachable: %w", err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			server.EncodeError(w, resp.StatusCode, fmt.Errorf("kdc error: %s", string(bodyBytes)))
			return
		}

		var asRep protocol.ASRep
		if err := json.NewDecoder(resp.Body).Decode(&asRep); err != nil {
			server.EncodeError(w, http.StatusInternalServerError, fmt.Errorf("invalid kdc response: %w", err))
			return
		}

		// 3. Derive key from password using correct salt (realm + primary + instance)
		salt := "ATHENA.MIT.EDU" + body.Username // realm + primary + instance (empty for alice)
		clientKey, err := crypto.DeriveKey(body.Password, salt)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, fmt.Errorf("key derivation failed: %w", err))
			return
		}

		// 4. Decrypt SecretPart to get session key
		secretPartBytes, err := crypto.Decrypt(clientKey, asRep.SecretPart().Ciphertext())
		if err != nil {
			server.EncodeError(w, http.StatusUnauthorized, fmt.Errorf("failed to decrypt session key (wrong password?): %w", err))
			return
		}

		var encRepPart protocol.EncKDCRepPart
		if err := json.Unmarshal(secretPartBytes, &encRepPart); err != nil {
			server.EncodeError(w, http.StatusInternalServerError, fmt.Errorf("invalid secret part format: %w", err))
			return
		}

		sessionKey := encRepPart.SessionKey()

		// 5. Store TGT with session key and client principal in cache
		r.cache.StoreTGTWithSession(asRep.Ticket(), sessionKey, client)

		server.Encode(w, http.StatusOK, map[string]interface{}{
			"status":        "logged_in",
			"user":          body.Username,
			"tgt_encrypted": true,
			"session_key":   base64.StdEncoding.EncodeToString(sessionKey.Expose()),
		})
	}
}

// GetTicketRoute - POST /api/ticket
// Calls KDC TGS Exchange, stores service ticket
type GetTicketRoute struct {
	kdcAddr string
	cache   *TicketCache
}

func (r *GetTicketRoute) Method() string { return http.MethodPost }
func (r *GetTicketRoute) Path() string   { return "/api/ticket" }
func (r *GetTicketRoute) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Service string `json:"service"` // e.g., "http/api-server"
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			server.EncodeError(w, http.StatusBadRequest, err)
			return
		}

		// 1. Get TGT and session key from cache
		tgt := r.cache.GetTGT()
		if tgt == nil {
			server.EncodeError(w, http.StatusUnauthorized, fmt.Errorf("not logged in"))
			return
		}

		sessionKey := r.cache.GetTGTSessionKey()
		if sessionKey == nil {
			server.EncodeError(w, http.StatusUnauthorized, fmt.Errorf("no session key"))
			return
		}

		clientPrincipal := r.cache.GetClientPrincipal()

		// 2. Parse service principal
		var servicePrincipal protocol.Principal
		if body.Service == "http/api-server" {
			sp, err := protocol.NewPrincipal("http", "api-server", "ATHENA.MIT.EDU")
			if err != nil {
				server.EncodeError(w, http.StatusBadRequest, fmt.Errorf("invalid service: %w", err))
				return
			}
			servicePrincipal = sp
		} else {
			server.EncodeError(w, http.StatusBadRequest, fmt.Errorf("unsupported service: %s", body.Service))
			return
		}

		// 3. Create authenticator with REAL client principal
		addr, err := protocol.NewAddress(net.IPv4(127, 0, 0, 1))
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		timestamp := time.Now()

		authenticator, err := protocol.NewAuthenticator(clientPrincipal, addr, timestamp)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		nonce, err := protocol.NewNonce(int32(time.Now().UnixNano()%100000 + 1))
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		// 4. ENCRYPT authenticator with session key from AS-REP
		authBytes, err := json.Marshal(authenticator)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		encryptedAuthBytes, err := crypto.Encrypt(*sessionKey, authBytes)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, fmt.Errorf("failed to encrypt authenticator: %w", err))
			return
		}

		encAuth, err := protocol.NewEncryptedData(encryptedAuthBytes)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		// 5. Build TGS-REQ

		tgsReq, err := protocol.NewTGSReq(servicePrincipal, *tgt, encAuth, nonce)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		// 5. POST to KDC TGS Exchange
		reqBody, err := json.Marshal(tgsReq)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		resp, err := http.Post(r.kdcAddr+"/tgs", "application/json", bytes.NewReader(reqBody))
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, fmt.Errorf("kdc unreachable: %w", err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			server.EncodeError(w, resp.StatusCode, fmt.Errorf("kdc error: %s", string(bodyBytes)))
			return
		}

		var tgsRep protocol.TGSRep
		if err := json.NewDecoder(resp.Body).Decode(&tgsRep); err != nil {
			server.EncodeError(w, http.StatusInternalServerError, fmt.Errorf("invalid kdc response: %w", err))
			return
		}

		// 6. Decrypt SecretPart to get service session key
		serviceSecretBytes, err := crypto.Decrypt(*sessionKey, tgsRep.SecretPart().Ciphertext())
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, fmt.Errorf("failed to decrypt service session key: %w", err))
			return
		}

		var serviceEncRepPart protocol.EncKDCRepPart
		if err := json.Unmarshal(serviceSecretBytes, &serviceEncRepPart); err != nil {
			server.EncodeError(w, http.StatusInternalServerError, fmt.Errorf("invalid service secret part format: %w", err))
			return
		}

		serviceSessionKey := serviceEncRepPart.SessionKey()

		// 7. Store service ticket WITH session key in cache
		r.cache.StoreServiceTicketWithSession(body.Service, tgsRep.Ticket(), serviceSessionKey)

		server.Encode(w, http.StatusOK, map[string]interface{}{
			"status":  "ticket_obtained",
			"service": body.Service,
			"ticket":  base64.StdEncoding.EncodeToString(tgsRep.Ticket().Ciphertext()),
		})
	}
}

// CallServiceRoute - POST /api/call
// Calls protected Api Server endpoint
type CallServiceRoute struct {
	cache *TicketCache
}

func (r *CallServiceRoute) Method() string { return http.MethodPost }
func (r *CallServiceRoute) Path() string   { return "/api/call" }
func (r *CallServiceRoute) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			URL string `json:"url"` // e.g., "http://localhost:9090/api/whoami"
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			server.EncodeError(w, http.StatusBadRequest, err)
			return
		}

		// 1. Get service ticket and session key from cache
		ticket := r.cache.GetServiceTicket("http/api-server")
		if ticket == nil {
			server.EncodeError(w, http.StatusUnauthorized, fmt.Errorf("no service ticket cached"))
			return
		}

		serviceSessionKey := r.cache.GetServiceSessionKey("http/api-server")
		if serviceSessionKey == nil {
			server.EncodeError(w, http.StatusUnauthorized, fmt.Errorf("no service session key"))
			return
		}

		clientPrincipal := r.cache.GetClientPrincipal()

		// 2. Build AP-REQ with encrypted authenticator
		addr, err := protocol.NewAddress(net.IPv4(127, 0, 0, 1))
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		timestamp := time.Now()

		authenticator, err := protocol.NewAuthenticator(clientPrincipal, addr, timestamp)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		// ENCRYPT the authenticator with the service session key
		authBytes, err := json.Marshal(authenticator)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		encryptedAuthBytes, err := crypto.Encrypt(*serviceSessionKey, authBytes)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, fmt.Errorf("failed to encrypt authenticator: %w", err))
			return
		}

		encAuth, err := protocol.NewEncryptedData(encryptedAuthBytes)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		apReq, err := protocol.NewAPReq(*ticket, encAuth)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		// 3. Serialize AP-REQ
		apReqBytes, err := json.Marshal(apReq)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		// 4. Call protected endpoint
		httpReq, err := http.NewRequest(http.MethodGet, body.URL, nil)
		if err != nil {
			server.EncodeError(w, http.StatusBadRequest, err)
			return
		}

		httpReq.Header.Set("Authorization", "Kerberos "+base64.StdEncoding.EncodeToString(apReqBytes))

		client := &http.Client{}
		resp, err := client.Do(httpReq)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, fmt.Errorf("service unreachable: %w", err))
			return
		}
		defer resp.Body.Close()

		// 5. Return service response
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			server.EncodeError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
	}
}
