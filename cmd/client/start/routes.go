package start

// LoginRoute - POST /api/login
// Calls KDC AS Exchange, stores TGT
// type LoginRoute struct {
// 	sdk   *sdk.Sdk
// 	cache *platform.TicketCache
// }

// func (r *LoginRoute) Method() string { return http.MethodPost }
// func (r *LoginRoute) Path() string   { return "/api/login" }

// type loginRequest struct {
// 	Username string `json:"username"`
// 	Password string `json:"password"`
// }

// func (r *LoginRoute) Handle() http.HandlerFunc {
// 	return func(w http.ResponseWriter, req *http.Request) {
// 		body, err := server.Decode[loginRequest](req)
// 		if err != nil {
// 			server.EncodeError(w, http.StatusBadRequest, err)
// 			return
// 		}

// 		res, err := r.login(req.Context(), body)
// 		if err != nil {
// 			// In a real app, we'd map domain errors to specific HTTP codes here
// 			server.EncodeError(w, http.StatusInternalServerError, err)
// 			return
// 		}

// 		if err := server.Encode(w, http.StatusOK, res); err != nil {
// 			server.EncodeError(w, http.StatusInternalServerError, err)
// 			return
// 		}
// 	}
// }

// func (r *LoginRoute) login(ctx context.Context, body loginRequest) (map[string]any, error) {
// 	// 1. Build AS-REQ
// 	client, err := protocol.NewPrincipal(protocol.Primary(body.Username), "", "ATHENA.MIT.EDU")
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid username: %w", err)
// 	}

// 	tgsPrincipal, err := protocol.NewKrbtgt("ATHENA.MIT.EDU")
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid krbtgt: %w", err)
// 	}

// 	nonce, err := protocol.NewNonce(int32(time.Now().UnixNano()%100000 + 1))
// 	if err != nil {
// 		return nil, err
// 	}

// 	addr, err := protocol.NewAddress(net.IPv4(127, 0, 0, 1))
// 	if err != nil {
// 		return nil, err
// 	}

// 	asReq, err := protocol.NewASReq(client, tgsPrincipal, addr, nonce)
// 	if err != nil {
// 		return nil, err
// 	}

// 	asRep, err := r.sdk.Kdc.PostAS(ctx, asReq)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid kdc response: %w", err)
// 	}

// 	// 2. Derive key from password using correct salt (realm + primary + instance)
// 	salt := "ATHENA.MIT.EDU" + body.Username // realm + primary + instance (empty for alice)
// 	clientKey, err := crypto.DeriveKey(body.Password, salt)
// 	if err != nil {
// 		return nil, fmt.Errorf("key derivation failed: %w", err)
// 	}

// 	// 3. Decrypt SecretPart to get session key
// 	secretPartBytes, err := crypto.Decrypt(clientKey, asRep.SecretPart().Ciphertext())
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decrypt session key (wrong password?): %w", err)
// 	}

// 	var encRepPart protocol.EncKDCRepPart
// 	if err := json.Unmarshal(secretPartBytes, &encRepPart); err != nil {
// 		return nil, fmt.Errorf("invalid secret part format: %w", err)
// 	}

// 	sessionKey := encRepPart.SessionKey()

// 	// 4. Store TGT with session key and client principal in cache
// 	r.cache.StoreTGTWithSession(asRep.Ticket(), sessionKey, client)

// 	return map[string]any{
// 		"status":        "logged_in",
// 		"user":          body.Username,
// 		"tgt_encrypted": true,
// 		"session_key":   base64.StdEncoding.EncodeToString(sessionKey.Expose()),
// 	}, nil
// }

// GetTicketRoute - POST /api/ticket
// Calls KDC TGS Exchange, stores service ticket
// type GetTicketRoute struct {
// 	sdk   *sdk.Sdk
// 	cache *TicketCache
// }

// func (r *GetTicketRoute) Method() string { return http.MethodPost }
// func (r *GetTicketRoute) Path() string   { return "/api/ticket" }

// type ticketRequest struct {
// 	Service string `json:"service"` // e.g., "http/api-server"
// }

// func (r *GetTicketRoute) Handle() http.HandlerFunc {
// 	return func(w http.ResponseWriter, req *http.Request) {
// 		body, err := server.Decode[ticketRequest](req)
// 		if err != nil {
// 			server.EncodeError(w, http.StatusBadRequest, err)
// 			return
// 		}

// 		res, err := r.getTicket(req.Context(), body)
// 		if err != nil {
// 			server.EncodeError(w, http.StatusInternalServerError, err)
// 			return
// 		}

// 		if err := server.Encode(w, http.StatusOK, res); err != nil {
// 			server.EncodeError(w, http.StatusInternalServerError, err)
// 			return
// 		}
// 	}
// }

// func (r *GetTicketRoute) getTicket(ctx context.Context, body ticketRequest) (map[string]any, error) {
// 	// 1. Get TGT and session key from cache
// 	tgt := r.cache.GetTGT()
// 	if tgt == nil {
// 		return nil, fmt.Errorf("not logged in")
// 	}

// 	sessionKey := r.cache.GetTGTSessionKey()
// 	if sessionKey == nil {
// 		return nil, fmt.Errorf("no session key")
// 	}

// 	clientPrincipal := r.cache.GetClientPrincipal()

// 	// 2. Parse service principal
// 	if body.Service != "http/api-server" {
// 		return nil, fmt.Errorf("unsupported service: %s", body.Service)
// 	}

// 	servicePrincipal, err := protocol.NewPrincipal("http", "api-server", "ATHENA.MIT.EDU")
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid service: %w", err)
// 	}

// 	// 3. Create authenticator with REAL client principal
// 	addr, err := protocol.NewAddress(net.IPv4(127, 0, 0, 1))
// 	if err != nil {
// 		return nil, err
// 	}

// 	timestamp := time.Now()

// 	authenticator, err := protocol.NewAuthenticator(clientPrincipal, addr, timestamp)
// 	if err != nil {
// 		return nil, err
// 	}

// 	nonce, err := protocol.NewNonce(int32(time.Now().UnixNano()%100000 + 1))
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 4. ENCRYPT authenticator with session key from AS-REP
// 	authBytes, err := json.Marshal(authenticator)
// 	if err != nil {
// 		return nil, err
// 	}

// 	encryptedAuthBytes, err := crypto.Encrypt(*sessionKey, authBytes)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to encrypt authenticator: %w", err)
// 	}

// 	encAuth, err := protocol.NewEncryptedData(encryptedAuthBytes)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 5. Build TGS-REQ
// 	tgsReq, err := protocol.NewTGSReq(servicePrincipal, *tgt, encAuth, nonce)
// 	if err != nil {
// 		return nil, err
// 	}

// 	tgsRep, err := r.sdk.Kdc.PostTGS(ctx, tgsReq)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid kdc response: %w", err)
// 	}

// 	// 6. Decrypt SecretPart to get service session key
// 	serviceSecretBytes, err := crypto.Decrypt(*sessionKey, tgsRep.SecretPart().Ciphertext())
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decrypt service session key: %w", err)
// 	}

// 	var serviceEncRepPart protocol.EncKDCRepPart
// 	if err := json.Unmarshal(serviceSecretBytes, &serviceEncRepPart); err != nil {
// 		return nil, fmt.Errorf("invalid service secret part format: %w", err)
// 	}

// 	serviceSessionKey := serviceEncRepPart.SessionKey()

// 	// 7. Store service ticket WITH session key in cache
// 	r.cache.StoreServiceTicketWithSession(body.Service, tgsRep.Ticket(), serviceSessionKey)

// 	return map[string]any{
// 		"status":  "ticket_obtained",
// 		"service": body.Service,
// 		"ticket":  base64.StdEncoding.EncodeToString(tgsRep.Ticket().Ciphertext()),
// 	}, nil
// }

// CallServiceRoute - POST /api/call
// Calls protected Api Server endpoint
// type CallServiceRoute struct {
// 	cache *TicketCache
// }

// func (r *CallServiceRoute) Method() string { return http.MethodPost }
// func (r *CallServiceRoute) Path() string   { return "/api/call" }

// type callRequest struct {
// 	URL string `json:"url"` // e.g., "http://localhost:9090/api/whoami"
// }

// func (r *CallServiceRoute) Handle() http.HandlerFunc {
// 	return func(w http.ResponseWriter, req *http.Request) {
// 		body, err := server.Decode[callRequest](req)
// 		if err != nil {
// 			server.EncodeError(w, http.StatusBadRequest, err)
// 			return
// 		}

// 		respBody, statusCode, err := r.callService(req.Context(), body)
// 		if err != nil {
// 			server.EncodeError(w, http.StatusInternalServerError, err)
// 			return
// 		}

// 		w.Header().Set("Content-Type", "application/json")
// 		w.WriteHeader(statusCode)
// 		w.Write(respBody)
// 	}
// }

// func (r *CallServiceRoute) callService(ctx context.Context, body callRequest) ([]byte, int, error) {
// 	// 1. Get service ticket and session key from cache
// 	ticket := r.cache.GetServiceTicket("http/api-server")
// 	if ticket == nil {
// 		return nil, 0, fmt.Errorf("no service ticket cached")
// 	}

// 	serviceSessionKey := r.cache.GetServiceSessionKey("http/api-server")
// 	if serviceSessionKey == nil {
// 		return nil, 0, fmt.Errorf("no service session key")
// 	}

// 	clientPrincipal := r.cache.GetClientPrincipal()

// 	// 2. Build AP-REQ with encrypted authenticator
// 	addr, err := protocol.NewAddress(net.IPv4(127, 0, 0, 1))
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	authenticator, err := protocol.NewAuthenticator(clientPrincipal, addr, time.Now())
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	// ENCRYPT the authenticator with the service session key
// 	authBytes, err := json.Marshal(authenticator)
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	encryptedAuthBytes, err := crypto.Encrypt(*serviceSessionKey, authBytes)
// 	if err != nil {
// 		return nil, 0, fmt.Errorf("failed to encrypt authenticator: %w", err)
// 	}

// 	encAuth, err := protocol.NewEncryptedData(encryptedAuthBytes)
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	apReq, err := protocol.NewAPReq(*ticket, encAuth)
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	// 3. Serialize AP-REQ
// 	apReqBytes, err := json.Marshal(apReq)
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	// 4. Call protected endpoint
// 	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, body.URL, nil)
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	httpReq.Header.Set("Authorization", "Kerberos "+base64.StdEncoding.EncodeToString(apReqBytes))

// 	client := &http.Client{}
// 	resp, err := client.Do(httpReq)
// 	if err != nil {
// 		return nil, 0, fmt.Errorf("service unreachable: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	// 5. Return service response
// 	respBody, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	return respBody, resp.StatusCode, nil
// }
