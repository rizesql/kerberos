# Kerberos Demo Guide

---

## 1. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         KERBEROS SYSTEM                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   ┌──────────┐         ┌─────────────────┐         ┌──────────────┐ │
│   │  CLIENT  │◄───────►│       KDC       │         │  API SERVER  │ │
│   │ :3000    │         │     :8080       │         │    :9090     │ │
│   │          │         │                 │         │              │ │
│   │ • Web UI │         │ • AS Exchange   │         │ • Protected  │ │
│   │ • Ticket │         │ • TGS Exchange  │         │   endpoints  │ │
│   │   cache  │◄────────┼─────────────────┼────────►│ • Uses       │ │
│   │          │         │                 │         │   ap.Verify  │ │
│   └──────────┘         └─────────────────┘         └──────────────┘ │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Key Insight:** The Api Server NEVER contacts the KDC. It verifies tickets offline using its own secret key.

---

## 2. What's Already Built

| Component                         | Location                    | Status  |
| --------------------------------- | --------------------------- | ------- |
| AS Exchange (login)               | `internal/kdc/as/`          | ✅ Done |
| TGS Exchange (get service ticket) | `internal/kdc/tgs/`         | ✅ Done |
| AP Verification                   | `internal/ap/verify.go`     | ✅ Done |
| AP Middleware                     | `internal/ap/middleware.go` | ✅ Done |
| Replay Cache                      | `internal/replay/cache.go`  | ✅ Done |
| Protocol Types                    | `internal/protocol/`        | ✅ Done |
| Server Framework                  | `internal/server/`          | ✅ Done |
| KDC Server                        | `cmd/kdc/`                  | ✅ Done |

---

## 3. What YOU Need to Build

### 3.1 Api Server (`cmd/api/start/start.go`)

```go
package start

import (
    "context"
    "fmt"
    "net"
    "runtime/debug"
    "time"

    "github.com/rizesql/kerberos/internal/ap"
    "github.com/rizesql/kerberos/internal/clock"
    "github.com/rizesql/kerberos/internal/o11y/logging"
    "github.com/rizesql/kerberos/internal/protocol"
    "github.com/rizesql/kerberos/internal/replay"
    "github.com/rizesql/kerberos/internal/server"
    "github.com/rizesql/kerberos/internal/shutdown"
)

type Config struct {
    Port         string
    ServerKeyHex string // 32-byte hex-encoded key (must match KDC database)
}

func Run(ctx context.Context, cfg Config) error {
    logger := logging.New()
    clk := clock.New()
    shutdowns := shutdown.New()

    defer func() {
        if r := recover(); r != nil {
            logger.Error("panic", "panic", r, "stack", string(debug.Stack()))
        }
    }()

    // Server's secret key (must match what's in KDC database for http/api-server)
    serverKeyBytes, err := hex.DecodeString(cfg.ServerKeyHex)
    if err != nil {
        return fmt.Errorf("invalid server key: %w", err)
    }
    serverKey, err := protocol.NewSessionKey(serverKeyBytes)
    if err != nil {
        return fmt.Errorf("failed to create session key: %w", err)
    }

    // Create replay cache and verifier
    cache := replay.NewInMemoryCache(10*time.Minute, clk)
    verifier := ap.NewVerifier(serverKey, clk, cache)

    // Create server
    srv := server.New(logger)
    shutdowns.RegisterCtx(srv.Shutdown)

    // Register protected routes with Kerberos middleware
    srv.Register(&WhoAmIRoute{}, ap.Middleware(verifier))
    srv.Register(&SecretRoute{}, ap.Middleware(verifier))

    // Start listening
    ln, err := net.Listen("tcp", cfg.Port)
    if err != nil {
        return fmt.Errorf("failed to listen: %w", err)
    }

    go func() {
        if err := srv.Listen(ctx, ln); err != nil {
            panic(err)
        }
    }()

    logger.Info("Api Server running", "port", cfg.Port)
    logger.Info("Press Ctrl+C to shut down")

    if err := shutdowns.WaitForSignal(ctx); err != nil {
        return fmt.Errorf("shutdown failed: %w", err)
    }

    logger.Info("Server shutdown complete")
    return nil
}
```

#### Routes (`cmd/api/routes.go`)

```go
package start

import (
    "net/http"

    "github.com/rizesql/kerberos/internal/ap"
    "github.com/rizesql/kerberos/internal/server"
)

// WhoAmIRoute - returns the authenticated principal
type WhoAmIRoute struct{}

func (r *WhoAmIRoute) Method() string { return http.MethodGet }
func (r *WhoAmIRoute) Path() string   { return "/api/whoami" }
func (r *WhoAmIRoute) Handle() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        client, ok := ap.ClientFromContext(r.Context())
        if !ok {
            server.EncodeError(w, http.StatusUnauthorized, ap.ErrMissingAuthHeader)
            return
        }

        server.Encode(w, http.StatusOK, map[string]string{
            "authenticated_as": client.String(),
            "message":          "Welcome to the protected resource!",
        })
    }
}

// SecretRoute - returns a secret message
type SecretRoute struct{}

func (r *SecretRoute) Method() string { return http.MethodGet }
func (r *SecretRoute) Path() string   { return "/api/secret" }
func (r *SecretRoute) Handle() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        client, _ := ap.ClientFromContext(r.Context())

        server.Encode(w, http.StatusOK, map[string]string{
            "secret": "secret!",
            "for":    client.String(),
        })
    }
}
```

---

### 3.2 Client with Web Frontend (`cmd/client/start/start.go`)

```go
package start

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "runtime/debug"

    "github.com/rizesql/kerberos/internal/o11y/logging"
    "github.com/rizesql/kerberos/internal/server"
    "github.com/rizesql/kerberos/internal/shutdown"
)

type Config struct {
    Port    string
    KDCAddr string // e.g., "http://localhost:8080"
    WebDir  string // e.g., "./web"
}

func Run(ctx context.Context, cfg Config) error {
    logger := logging.New()
    shutdowns := shutdown.New()

    defer func() {
        if r := recover(); r != nil {
            logger.Error("panic", "panic", r, "stack", string(debug.Stack()))
        }
    }()

    // Create ticket cache (stores TGT and service tickets in memory)
    ticketCache := NewTicketCache()

    // Create server
    srv := server.New(logger)
    shutdowns.RegisterCtx(srv.Shutdown)

    // Serve static frontend files
    srv.Mux().Handle("/", http.FileServer(http.Dir(cfg.WebDir)))

    // Register API routes for auth flow
    srv.Register(&LoginRoute{kdcAddr: cfg.KDCAddr, cache: ticketCache})
    srv.Register(&GetTicketRoute{kdcAddr: cfg.KDCAddr, cache: ticketCache})
    srv.Register(&CallServiceRoute{cache: ticketCache})

    // Start listening
    ln, err := net.Listen("tcp", cfg.Port)
    if err != nil {
        return fmt.Errorf("failed to listen: %w", err)
    }

    go func() {
        if err := srv.Listen(ctx, ln); err != nil {
            panic(err)
        }
    }()

    logger.Info("Client running", "port", cfg.Port, "frontend", cfg.WebDir)
    logger.Info("Open http://localhost" + cfg.Port + " in your browser")
    logger.Info("Press Ctrl+C to shut down")

    if err := shutdowns.WaitForSignal(ctx); err != nil {
        return fmt.Errorf("shutdown failed: %w", err)
    }

    return nil
}
```

#### Client Routes (`cmd/client/routes.go`)

```go
package start

import (
    "encoding/base64"
    "encoding/json"
    "net/http"
    "time"

    "github.com/rizesql/kerberos/internal/crypto"
    "github.com/rizesql/kerberos/internal/kdc/shared"
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
        client, _ := protocol.NewPrincipal(body.Username, "", "ATHENA.MIT.EDU")
        tgsPrincipal, _ := protocol.NewKrbtgt("ATHENA.MIT.EDU")
        // ... build request and POST to KDC ...

        // 2. Derive key from password
        clientKey := crypto.DeriveKey(body.Password)

        // 3. Decrypt AS-REP and store TGT
        // ... decrypt and cache ...

        server.Encode(w, http.StatusOK, map[string]string{
            "status": "logged_in",
            "user":   body.Username,
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
        // ... build TGS-REQ using cached TGT and POST to KDC ...
        // ... store service ticket in cache ...

        server.Encode(w, http.StatusOK, map[string]string{
            "status":  "ticket_obtained",
            "service": body.Service,
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
        // ... build AP-REQ using cached service ticket ...
        // ... call Api Server and return response ...
    }
}
```

#### Simple Frontend (`cmd/client/web/index.html`)

```html
<!DOCTYPE html>
<html>
  <head>
    <title>Kerberos Demo</title>
    <style>
      body {
        font-family: sans-serif;
        max-width: 600px;
        margin: 50px auto;
      }
      .step {
        margin: 20px 0;
        padding: 15px;
        border: 1px solid #ccc;
        border-radius: 8px;
      }
      button {
        padding: 10px 20px;
        cursor: pointer;
      }
      pre {
        background: #f5f5f5;
        padding: 10px;
        overflow-x: auto;
      }
    </style>
  </head>
  <body>
    <h1>🔐 Kerberos Demo</h1>

    <div class="step">
      <h3>Step 1: Login (AS Exchange)</h3>
      <input id="username" placeholder="Username" value="alice" />
      <input id="password" type="password" placeholder="Password" />
      <button onclick="login()">Login</button>
      <pre id="login-result"></pre>
    </div>

    <div class="step">
      <h3>Step 2: Get Service Ticket (TGS Exchange)</h3>
      <input id="service" placeholder="Service" value="http/api-server" />
      <button onclick="getTicket()">Get Ticket</button>
      <pre id="ticket-result"></pre>
    </div>

    <div class="step">
      <h3>Step 3: Access Protected Resource (AP Exchange)</h3>
      <input
        id="url"
        placeholder="URL"
        value="http://localhost:9090/api/whoami"
        style="width:300px"
      />
      <button onclick="callService()">Call</button>
      <pre id="call-result"></pre>
    </div>

    <script>
      async function login() {
        const res = await fetch("/api/login", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            username: document.getElementById("username").value,
            password: document.getElementById("password").value,
          }),
        });
        document.getElementById("login-result").textContent = await res.text();
      }

      async function getTicket() {
        const res = await fetch("/api/ticket", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            service: document.getElementById("service").value,
          }),
        });
        document.getElementById("ticket-result").textContent = await res.text();
      }

      async function callService() {
        const res = await fetch("/api/call", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            url: document.getElementById("url").value,
          }),
        });
        document.getElementById("call-result").textContent = await res.text();
      }
    </script>
  </body>
</html>
```

---

## 4. Database Setup

### 4.1 Initialize KDC (already built: `cmd/kdc/setup`)

```bash
# Creates database, applies schema, creates krbtgt principal
./kdc setup --db kdc.db --realm ATHENA.MIT.EDU --secret "master-secret"
```

This uses `crypto.DeriveKey()` to derive the krbtgt key from the secret.

---

### 4.2 Add Principals (`cmd/kadmin/add` - YOU NEED TO BUILD THIS)

**Usage:**
```bash
# Add a user (derives key from password)
./kadmin add --db kdc.db --principal alice --realm ATHENA.MIT.EDU --password secret123

# Add a service (uses hex key directly)
./kadmin add --db kdc.db --principal http/api-server --realm ATHENA.MIT.EDU --key <32-byte-hex>
```

**Implementation (~30 lines)** - follow the pattern in `cmd/kdc/setup/setup.go`:

```go
package add

import (
    "context"
    "fmt"

    "github.com/rizesql/kerberos/internal/crypto"
    "github.com/rizesql/kerberos/internal/kdb"
    "github.com/rizesql/kerberos/internal/o11y/logging"
    "github.com/rizesql/kerberos/internal/protocol"
)

type Config struct {
    DBPath    string
    Principal string // e.g., "alice" or "http/api-server"
    Instance  string // e.g., "" or "api-server"
    Realm     string
    Password  string // for users
    KeyHex    string // for services (alternative to password)
}

func Run(ctx context.Context, cfg Config) error {
    logger := logging.New()

    db, err := kdb.New(kdb.Config{DSN: cfg.DBPath, Logger: logger})
    if err != nil {
        return fmt.Errorf("failed to open db: %w", err)
    }
    defer db.Close()

    // Derive key from password OR use provided hex key
    var keyBytes []byte
    if cfg.Password != "" {
        key, err := crypto.DeriveKey(cfg.Password, cfg.Realm+cfg.Principal+cfg.Realm)
        if err != nil {
            return fmt.Errorf("failed to derive key: %w", err)
        }
        keyBytes = key.Expose()
    } else {
        keyBytes, _ = hex.DecodeString(cfg.KeyHex)
    }

    // Create principal
    _, err = kdb.Query.CreatePrincipal(ctx, db, kdb.CreatePrincipalParams{
        PrimaryName: cfg.Principal,
        Instance:    cfg.Instance,
        Realm:       cfg.Realm,
        KeyBytes:    keyBytes,
        Kvno:        1,
    })
    if err != nil {
        return fmt.Errorf("failed to create principal: %w", err)
    }

    logger.Info("Principal created", "principal", fmt.Sprintf("%s/%s@%s", cfg.Principal, cfg.Instance, cfg.Realm))
    return nil
}
```

---

### 4.3 Demo Setup Commands

```bash
# 1. Initialize KDC database (creates krbtgt)
./kdc setup --db kdc.db --realm ATHENA.MIT.EDU --secret "kdc-master-secret"

# 2. Add test user
./kadmin add --db kdc.db --principal alice --realm ATHENA.MIT.EDU --password secret123

# 3. Add API server service
./kadmin add --db kdc.db --principal http --instance api-server --realm ATHENA.MIT.EDU --password api-secret

# 4. Start everything
./kdc start --db kdc.db --realm ATHENA.MIT.EDU &
./api start --key $(./kadmin get-key http/api-server) &
./client start
```

---

## 5. Demo Presentation Script (Live)

### Terminal Setup

- **Terminal 1**: KDC server (`./kdc start`)
- **Terminal 2**: Api Server (`./api start`)
- **Terminal 3**: Client (`./client start`)
- **Browser**: Open `http://localhost:3000`

### Step-by-Step Demo

| Step | Action                         | Expected Result              | Teacher Talking Point                 |
| ---- | ------------------------------ | ---------------------------- | ------------------------------------- |
| 1    | Start KDC                      | "KDC listening on :8080"     | "Central authority with database"     |
| 2    | Start Api Server               | "Api listening on :9090"     | "Independent, no DB access"           |
| 3    | Start Client                   | "Open http://localhost:3000" | "User-facing web interface"           |
| 4    | Open browser                   | See the demo UI              | "Three-step authentication flow"      |
| 5    | Click "Login"                  | Shows TGT obtained           | "Password NEVER sent over network"    |
| 6    | Click "Get Ticket"             | Shows service ticket         | "Can request tickets for any service" |
| 7    | Click "Call"                   | Shows authenticated response | "Api Server verified offline!"        |
| 8    | Click "Call" again immediately | Shows 401 replay error       | "Replay cache prevents attacks"       |

### Key Points to Emphasize

1. **Distributed**: Api Server doesn't contact KDC during verification
2. **Secure**: Password only used locally to derive decryption key
3. **Single Sign-On**: One TGT → multiple service tickets
4. **Replay Protection**: Same request rejected on retry

---

## 6. API Reference

### KDC Endpoints

#### `POST /as/exchange`

```json
// Request (AS-REQ)
{
  "client": {"primary": "alice", "instance": "", "realm": "ATHENA.MIT.EDU"},
  "server": {"primary": "krbtgt", "instance": "ATHENA.MIT.EDU", "realm": "ATHENA.MIT.EDU"},
  "address": "127.0.0.1",
  "nonce": 12345
}

// Response (AS-REP)
{
  "ticket": {...},      // TGT (encrypted with TGS key)
  "secret_part": {...}  // Session key (encrypted with client key)
}
```

#### `POST /tgs/exchange`

```json
// Request (TGS-REQ)
{
  "server": {"primary": "http", "instance": "api-server", "realm": "ATHENA.MIT.EDU"},
  "tgt": {...},
  "authenticator": {...},
  "nonce": 67890
}

// Response (TGS-REP)
{
  "ticket": {...},      // Service ticket
  "secret_part": {...}  // New session key
}
```

### Api Server Endpoints

#### `GET /api/whoami` (Protected)

```
Authorization: Kerberos <base64-encoded-AP-REQ>

// Response
{"authenticated_as": "alice@ATHENA.MIT.EDU", "message": "Welcome!"}
```

---

## 7. Troubleshooting

| Issue                  | Cause                     | Fix                      |
| ---------------------- | ------------------------- | ------------------------ |
| "principal not found"  | User not in database      | Add principal to DB      |
| "invalid TGT"          | Wrong TGS key             | Check krbtgt key matches |
| "clock skew too great" | Time difference > 5min    | Sync clocks              |
| "replay detected"      | Same authenticator reused | Generate new timestamp   |
