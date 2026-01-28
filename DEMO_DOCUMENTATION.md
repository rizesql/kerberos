# Kerberos Demo - Complete Walkthrough

## Table of Contents
1. [Architecture Overview](#architecture-overview)
2. [System Components](#system-components)
3. [Authentication Flow](#authentication-flow)
4. [Running the Demo](#running-the-demo)
5. [Detailed Step-by-Step Walkthrough](#detailed-step-by-step-walkthrough)
6. [Security Features](#security-features)
7. [API Endpoint Reference](#api-endpoint-reference)
8. [Troubleshooting](#troubleshooting)

---

## Architecture Overview

The Kerberos demo implements a **distributed authentication system** with three independent components that work together:

```
┌─────────────────────────────────────────────────────────────────────┐
│                         KERBEROS SYSTEM                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   ┌──────────────┐       ┌─────────────────┐       ┌─────────────┐ │
│   │   CLIENT     │       │       KDC       │       │ API SERVER  │ │
│   │   :3000      │◄─────►│     :8080       │       │   :9090     │ │
│   │              │       │                 │       │             │ │
│   │ • Web UI     │       │ • Database      │       │ • Protected │ │
│   │ • Login      │       │ • AS Exchange   │       │   endpoints │ │
│   │ • TGS req    │       │ • TGS Exchange  │       │ • Verifier  │ │
│   │ • API calls  │       │                 │       │   (offline) │ │
│   └──────────────┘       └─────────────────┘       └─────────────┘ │
│        ▲                        ▲                        ▲            │
│        │                        │                        │            │
│        └────────── No direct connection ─────────────────┘            │
│                                                                     │
│  Key: Client never contacts KDC during verification phase.         │
│       API Server verifies tickets offline using its secret key.   │
└─────────────────────────────────────────────────────────────────────┘
```

### Key Principle

**The API Server is completely independent from the KDC.** It never contacts the KDC to verify tickets. Instead:

1. During setup, the API server receives a secret key
2. When receiving a ticket, it verifies the signature cryptographically
3. If the signature is valid, the ticket must have come from the KDC

This is the **core strength** of Kerberos: once a service has the right key, it can verify tickets offline.

---

## System Components

### 1. Key Distribution Center (KDC) - `cmd/kdc/`

**Purpose:** Trusted third party that authenticates users and issues tickets

**Features:**
- **Authentication Server (AS)**: Verifies passwords and issues TGTs
- **Ticket Granting Server (TGS)**: Issues service tickets
- **Persistent Database**: Stores user credentials and service keys

**Database Schema:**
```
Principals Table
├── Primary Name (e.g., "alice")
├── Instance (e.g., "", "api-server")
├── Realm (e.g., "ATHENA.MIT.EDU")
├── Encrypted Key
└── Key Version Number (KVNO)
```

**Port:** `:8080` (configurable with `--port`)

**Key Endpoints:**
- `POST /as/exchange` - AS Exchange (login)
- `POST /tgs/exchange` - TGS Exchange (get service ticket)

---

### 2. API Server - `cmd/api/`

**Purpose:** Protected resource server

**Features:**
- **Offline Verification**: No database access, verifies tickets using crypto
- **Replay Cache**: Prevents attack replay with 5-minute window
- **Middleware Protection**: All endpoints require Kerberos authentication

**Authentication Flow:**
1. Client sends request with `Authorization: Kerberos <ticket>` header
2. Middleware extracts and decrypts ticket
3. Verifies timestamp (not expired, not too far in past)
4. Checks replay cache (not seen before)
5. If valid, extracts authenticated user and calls route handler

**Port:** `:9090` (configurable with `--port`)

**Protected Endpoints:**
- `GET /api/whoami` - Returns authenticated principal
- `GET /api/secret` - Returns secret message for authenticated user

**Startup:**
```bash
./api.exe start --key <64-char-hex-key>
```

The key must match what's registered for `http/api-server` in the KDC database.

---

### 3. Client - `cmd/client/`

**Purpose:** User-facing application with web UI

**Features:**
- **Ticket Caching**: In-memory cache stores TGT and service tickets
- **Web UI**: Beautiful three-step interface
- **Authentication Flow**: Implements full AS→TGS→AP exchange

**Architecture:**
- **Backend Routes:**
  - `POST /api/login` - Calls KDC AS Exchange
  - `POST /api/ticket` - Calls KDC TGS Exchange
  - `POST /api/call` - Calls API server with ticket

- **Frontend:** HTML/JavaScript UI served from `cmd/client/web/index.html`

**Port:** `:3000` (configurable with `--port`)

**Startup:**
```bash
./client.exe start [--port :3000] [--kdc http://localhost:8080] [--web ./cmd/client/web]
```

**Internal Ticket Cache:**
```
TicketCache
├── TGT (Ticket Granting Ticket)
└── Service Tickets
    ├── "http/api-server" → Ticket
    └── ...
```

---

## Authentication Flow

### The Three Exchanges

```
Step 1: AS Exchange (Login)
┌────────────────────────────────────────────────────────────────┐
│ Client                        KDC                              │
│   │ AS-REQ                      │                              │
│   │ (username, password, TGS)   │                              │
│   ├────────────────────────────►│                              │
│   │                  AS-REP      │                              │
│   │          (TGT + Session Key) │                              │
│   │◄────────────────────────────┤                              │
│   │                              │                              │
│   └──► Store TGT in Cache                                      │
└────────────────────────────────────────────────────────────────┘

Step 2: TGS Exchange (Get Service Ticket)
┌────────────────────────────────────────────────────────────────┐
│ Client                        KDC                              │
│   │ TGS-REQ                      │                              │
│   │ (TGT + Authenticator)        │                              │
│   │ (requesting: http/api-server)│                              │
│   ├────────────────────────────►│                              │
│   │                TGS-REP        │                              │
│   │     (Service Ticket)          │                              │
│   │◄────────────────────────────┤                              │
│   │                              │                              │
│   └──► Store Service Ticket in Cache                           │
└────────────────────────────────────────────────────────────────┘

Step 3: AP Exchange (Access Protected Resource)
┌────────────────────────────────────────────────────────────────┐
│ Client              API Server (http/api-server)              │
│   │ AP-REQ                   │                                │
│   │ (Service Ticket +        │                                │
│   │  Authenticator)          │                                │
│   ├────────────────────────►│                                │
│   │   AP-REP (or Error)       │                                │
│   │   (authenticated_as: ...) │                                │
│   │◄────────────────────────┤                                │
└────────────────────────────────────────────────────────────────┘
```

### What Each Exchange Accomplishes

| Exchange | What Happens | Why It's Safe |
|----------|--------------|---------------|
| **AS** | Client proves identity with password | Password never sent to network; only KDC knows password |
| **TGS** | TGT proves authentication to TGS | TGT is encrypted with TGS key; only KDC can read it |
| **AP** | Service Ticket proves client identity to service | Ticket encrypted with service key; only API server can decrypt it |

---

## Running the Demo

### Prerequisites

- Go 1.21+
- Windows (PowerShell) or Linux/macOS (Bash)
- SQLite (included with Go)

### Setup: Create Database and Principals

```bash
# 1. Initialize KDC database
./kdc setup --db kdc.db --realm ATHENA.MIT.EDU --secret "kdc-master-secret"

# 2. Add test user
./kadmin add --db kdc.db --principal alice --realm ATHENA.MIT.EDU --password secret123

# 3. Add API server service
./kadmin add --db kdc.db --principal http --instance api-server --realm ATHENA.MIT.EDU --password api-secret

# 4. Get the API server's key (needed for ./api start)
./kadmin get-key --db kdc.db --realm ATHENA.MIT.EDU http/api-server
# Output: 0011223344556677889...
```

### Building

```bash
# Build all three components
go build -o kdc.exe ./cmd/kdc
go build -o api.exe ./cmd/api
go build -o client.exe ./cmd/client
go build -o kadmin.exe ./cmd/kadmin
```

Or use the Makefile:
```bash
make build  # Builds all
```

### Running in Three Terminals

**Terminal 1: Start KDC**
```bash
./kdc.exe start --db kdc.db --realm ATHENA.MIT.EDU
```

Output:
```
{"level":"INFO","msg":"listening","srv":"http","addr":"[::]:8080"}
```

**Terminal 2: Start API Server**
```bash
./api.exe start --key 0011223344556677889900aabbccddee...
```

Output:
```
✓ API Server listening on :9090
{"level":"INFO","msg":"Api Server running","port":":9090"}
```

**Terminal 3: Start Client**
```bash
./client.exe start
```

Output:
```
✓ Client running on http://localhost:3000
{"level":"INFO","msg":"Client running","port":":3000"}
```

**Browser: Open the Web UI**
```
http://localhost:3000
```

You should see the beautiful three-step interface!

---

## Detailed Step-by-Step Walkthrough

### Step 1: Login (AS Exchange)

**What You Do:**
1. Open http://localhost:3000
2. Enter username: `alice`
3. Enter password: `secret123`
4. Click "Login"

**What Happens Behind the Scenes:**

```
Client Browser
    │
    └─► JavaScript fetch() to http://localhost:3000/api/login
        │
        ├─ POST body: { "username": "alice", "password": "secret123" }
        │
        └──► Client Backend (Go)
            │
            ├─ Create Principal: alice@ATHENA.MIT.EDU
            ├─ Create Nonce: random int32
            ├─ Create Address: 127.0.0.1
            │
            └─ Build AS-REQ: { client, server: krbtgt, address, nonce }
                │
                └─► HTTP POST to KDC: http://localhost:8080/as/exchange
                    │
                    └──► KDC Backend
                        │
                        ├─ Look up alice in database
                        ├─ Derive key from password "secret123"
                        ├─ Create encrypted TGT (with TGS key)
                        ├─ Create encrypted session key (with client key)
                        │
                        └─ Return AS-REP: { encrypted_tgt, encrypted_session_key }
                            │
                            └─► Client Backend receives AS-REP
                                │
                                ├─ Derive client key from password "secret123"
                                ├─ Decrypt session key (proves password is correct!)
                                ├─ Store encrypted TGT in cache
                                │
                                └─ Return to Browser: { status: "logged_in", session_key: "..." }
                                    │
                                    └──► Browser displays success message
```

**Key Insight:**
- The password is only used **locally** to derive the decryption key
- The password is **never sent to the network** (not even to KDC!)
- Both client and KDC derive the same key independently using PBKDF2
- If the derived key can decrypt the session key, the password must be correct

**Response Shows:**
```json
{
  "status": "logged_in",
  "user": "alice",
  "tgt_encrypted": true,
  "session_key": "AAECAwQ..."
}
```

---

### Step 2: Get Service Ticket (TGS Exchange)

**What You Do:**
1. Ensure Step 1 succeeded (TGT is cached)
2. Click "Get Ticket" in Step 2
3. Service field should show: `http/api-server`

**What Happens Behind the Scenes:**

```
Browser
    │
    └─► JavaScript fetch() to http://localhost:3000/api/ticket
        │
        ├─ POST body: { "service": "http/api-server" }
        │
        └──► Client Backend
            │
            ├─ Check cache: GetTGT()  ✓ Found!
            │
            ├─ Create service principal: http/api-server@ATHENA.MIT.EDU
            ├─ Create authenticator: { client, address, timestamp }
            ├─ Create nonce: random int32
            │
            └─ Build TGS-REQ: { service, tgt, authenticator, nonce }
                │
                └─► HTTP POST to KDC: http://localhost:8080/tgs/exchange
                    │
                    └──► KDC Backend
                        │
                        ├─ Decrypt TGT (using TGS key) → get session key
                        ├─ Decrypt authenticator (using session key)
                        ├─ Verify client in authenticator matches TGT
                        ├─ Verify timestamp (not expired, clock skew < 5 min)
                        ├─ Look up http/api-server in database
                        │
                        ├─ Create service ticket:
                        │  └─ Encrypted with http/api-server's key
                        │
                        └─ Return TGS-REP: { service_ticket, new_session_key }
                            │
                            └──► Client Backend
                                │
                                ├─ Store service ticket in cache
                                │   cache["http/api-server"] = service_ticket
                                │
                                └─ Return to Browser: { status: "ticket_obtained", ... }
                                    │
                                    └──► Browser displays success message
```

**Key Insight:**
- The TGT is encrypted with the TGS key (only KDC has this key)
- The authenticator is encrypted with the session key from Step 1
- The KDC decrypts both to verify the client's identity
- The new service ticket is encrypted with the **API server's** key
- Only the API server can decrypt it!

**Response Shows:**
```json
{
  "status": "ticket_obtained",
  "service": "http/api-server",
  "ticket": "encrypted_bytes_as_base64"
}
```

---

### Step 3: Access Protected Resource (AP Exchange)

**What You Do:**
1. Ensure Steps 1 & 2 succeeded (both caches filled)
2. Verify URL is: `http://localhost:9090/api/whoami`
3. Click "Call"

**What Happens Behind the Scenes:**

```
Browser
    │
    └─► JavaScript fetch() to http://localhost:3000/api/call
        │
        ├─ POST body: { "url": "http://localhost:9090/api/whoami" }
        │
        └──► Client Backend
            │
            ├─ Check cache: GetServiceTicket("http/api-server")  ✓ Found!
            │
            ├─ Create authenticator: { client, address, timestamp }
            │
            └─ Build AP-REQ: { service_ticket, authenticator }
                │
                └─ Serialize to JSON, then Base64 encode
                    │
                    └─► HTTP GET to API Server
                        │
                        ├─ Header: Authorization: Kerberos <base64_ap_req>
                        │
                        └──► API Server Backend
                            │
                            ├─ Extract AP-REQ from Authorization header
                            ├─ Base64 decode
                            ├─ Deserialize JSON
                            │
                            └─ Middleware: ap.Middleware(verifier)
                                │
                                ├─ Decrypt service_ticket (using api server's key)
                                │   └─ Extract client principal
                                │   └─ Extract session key
                                │
                                ├─ Decrypt authenticator (using session key)
                                │   └─ Extract timestamp
                                │
                                ├─ Verify timestamp:
                                │   ├─ Not expired? (timestamp < now + 5 min)
                                │   └─ Not too old? (timestamp > now - 5 min)
                                │
                                ├─ Check replay cache:
                                │   └─ Have we seen this authenticator before?
                                │
                                ├─ If valid: Store in context (ap.ClientFromContext)
                                └─ If invalid: Return 401 Unauthorized
                                    │
                                    └──► Route handler runs:
                                        │
                                        ├─ Get authenticated client from context
                                        ├─ Return protected data
                                        │   { "authenticated_as": "alice@ATHENA.MIT.EDU", ... }
                                        │
                                        └──► Middleware returns to client
                                            │
                                            └──► Client Backend receives response
                                                │
                                                └──► Browser displays:
                                                    {
                                                      "authenticated_as": "alice@ATHENA.MIT.EDU",
                                                      "message": "Welcome!"
                                                    }
```

**Key Insight:**
- The service ticket is encrypted with the API server's secret key
- Only the API server can decrypt it (using `ap.Verifier`)
- The API server **never contacts the KDC**
- It verifies the ticket entirely offline using cryptography
- This is **distributed verification** - the foundation of Kerberos

**Response Shows:**
```json
{
  "authenticated_as": "alice@ATHENA.MIT.EDU",
  "message": "Welcome to the protected resource!"
}
```

---

### Try Replay Protection!

**What You Do:**
1. Click "Call" a second time within 5 minutes

**What Happens:**
- You get a **401 Unauthorized** error
- Message: "replay detected"

**Why?**
- The API server stores the timestamp from Step 3's authenticator
- The same timestamp means it's the same request (replay attack)
- After 5 minutes, the replay cache expires and you can try again

This **proves** that the API server is protecting against attack replays!

---

## Security Features

### 1. Password Protection

| Phase | What Happens | Security |
|-------|--------------|----------|
| Transport | Password used locally only | ✅ Never sent over network |
| Authentication | KDC verifies by trying decrypt | ✅ Wrong password = fail to decrypt |
| Ticket | Sessions use derived keys | ✅ Password not stored/transmitted |

### 2. Ticket Encryption

| Ticket | Encrypted With | Only Readable By | Purpose |
|--------|----------------|------------------|---------|
| TGT | TGS Key | KDC (TGS component) | Proves client authenticated |
| Service Ticket | Service Key | API Server | Proves client authenticated to service |
| Authenticator | Session Key | Receiver | Proves client is not replayed |

### 3. Timestamp Verification

```
Server receives authenticator:
├─ Extract timestamp T
├─ Get current server time S
├─ Check: S - 5min < T < S + 5min
└─ If outside range: Reject as "clock skew too great"
```

This prevents:
- **Old tickets** being reused after expiration
- **Tokens from different time zones** being misused
- **Clock-sync attacks** (with some tolerance)

### 4. Replay Cache

```
Server receives authenticator:
├─ Hash the authenticator bytes
├─ Check if we've seen this hash before
│  ├─ First time: ✅ Cache it, allow request
│  └─ Seen before: ❌ Reject as "replay detected"
├─ Cache entries expire after 5 minutes
└─ This prevents the same request being used twice
```

### 5. Distributed Trust

```
┌─ KDC knows password
│  └─ Issues encrypted tickets
│
└─ API Server knows its own key
   └─ Verifies tickets offline
   └─ Never trusts KDC again
```

This means:
- KDC compromise only affects future tickets
- Old tickets can still be verified
- Each service is independent

### 6. No Plaintext Transmission

```
Never sent over network:
├─ Passwords
├─ Session keys (except encrypted)
├─ Service keys
├─ Unencrypted tickets

Always encrypted:
├─ TGT (with TGS key)
├─ Service tickets (with service key)
├─ Authenticators (with session key)
└─ Session keys (with user/service key)
```

---

## API Endpoint Reference

### KDC Endpoints

#### `POST /as/exchange` - Authentication Server Exchange

**Request (AS-REQ):**
```json
{
  "client": {
    "primary": "alice",
    "instance": "",
    "realm": "ATHENA.MIT.EDU"
  },
  "service": {
    "primary": "krbtgt",
    "instance": "ATHENA.MIT.EDU",
    "realm": "ATHENA.MIT.EDU"
  },
  "client_addr": {
    "address": "127.0.0.1"
  },
  "nonce": 12345
}
```

**Response (AS-REP):**
```json
{
  "ticket": {
    "ciphertext": "encrypted_bytes_base64"
  },
  "secret_part": {
    "ciphertext": "encrypted_session_key_base64"
  }
}
```

**Decryption:**
- Ticket: Decrypted by KDC's TGS key (contains TGT info)
- SecretPart: Decrypted by client's key (contains session key)

---

#### `POST /tgs/exchange` - Ticket Granting Server Exchange

**Request (TGS-REQ):**
```json
{
  "service": {
    "primary": "http",
    "instance": "api-server",
    "realm": "ATHENA.MIT.EDU"
  },
  "tgt": {
    "ciphertext": "encrypted_tgt_from_as_rep"
  },
  "authenticator": {
    "ciphertext": "encrypted_authenticator"
  },
  "nonce": 67890
}
```

**Response (TGS-REP):**
```json
{
  "ticket": {
    "ciphertext": "service_ticket_encrypted_with_api_server_key"
  },
  "secret_part": {
    "ciphertext": "new_session_key"
  }
}
```

---

### API Server Endpoints

#### `GET /api/whoami` (Protected)

**Request:**
```
GET /api/whoami HTTP/1.1
Host: localhost:9090
Authorization: Kerberos <base64_ap_req>
```

**Response (Success):**
```json
{
  "authenticated_as": "alice@ATHENA.MIT.EDU",
  "message": "Welcome to the protected resource!"
}
```

**Response (Unauthorized):**
```
HTTP/1.1 401 Unauthorized

{
  "error": "missing Authorization header"
}
```

**Response (Replay Attack):**
```
HTTP/1.1 401 Unauthorized

{
  "error": "replay detected"
}
```

---

#### `GET /api/secret` (Protected)

**Request:**
```
GET /api/secret HTTP/1.1
Host: localhost:9090
Authorization: Kerberos <base64_ap_req>
```

**Response:**
```json
{
  "secret": "secret!",
  "for": "alice@ATHENA.MIT.EDU"
}
```

---

### Client Endpoints (Backend Routes)

#### `POST /api/login` - AS Exchange

**Request:**
```json
{
  "username": "alice",
  "password": "secret123"
}
```

**Response (Success):**
```json
{
  "status": "logged_in",
  "user": "alice",
  "tgt_encrypted": true,
  "session_key": "base64_encoded_key"
}
```

**Response (Failure):**
```json
{
  "error": "kdc error: principal not found"
}
```

---

#### `POST /api/ticket` - TGS Exchange

**Request:**
```json
{
  "service": "http/api-server"
}
```

**Response (Success):**
```json
{
  "status": "ticket_obtained",
  "service": "http/api-server",
  "ticket": "base64_encoded_ticket"
}
```

**Response (Failure - Not Logged In):**
```json
{
  "error": "not logged in"
}
```

---

#### `POST /api/call` - AP Exchange

**Request:**
```json
{
  "url": "http://localhost:9090/api/whoami"
}
```

**Response (Success):**
```json
{
  "authenticated_as": "alice@ATHENA.MIT.EDU",
  "message": "Welcome to the protected resource!"
}
```

**Response (Replay Attack):**
```json
{
  "error": "replay detected"
}
```

---

## Troubleshooting

### "Principal not found"

**Cause:** User doesn't exist in KDC database

**Fix:**
```bash
./kadmin add --db kdc.db --principal alice --realm ATHENA.MIT.EDU --password secret123
```

---

### "Invalid TGT"

**Cause:** TGT was encrypted with wrong key or expired

**Fix:**
1. Ensure TGS key in database matches what KDC is using
2. Check database isn't corrupted: `./kdc setup --db kdc.db ...`

---

### "Clock skew too great"

**Cause:** Client and server clocks are more than 5 minutes apart

**Fix:**
```bash
# On Windows
w32tm /resync

# On Linux/macOS
sudo ntpdate -s time.google.com
```

---

### "Replay detected" (when shouldn't be)

**Cause:** Replay cache entry hasn't expired yet (5-minute window)

**Fix:**
- Wait 5 minutes and try again, OR
- Restart API server to clear cache (not recommended for production)

---

### API Server won't start

**Cause:** Invalid hex key format

**Fix:**
- Key must be 64 hex characters (32 bytes)
- Get correct key:
```bash
./kadmin get-key --db kdc.db --realm ATHENA.MIT.EDU http/api-server
```

---

### Client can't reach KDC

**Cause:** KDC not running or wrong address

**Fix:**
```bash
# Check KDC is running
netstat -an | findstr :8080  # Windows
netstat -an | grep :8080      # Linux/macOS

# If not running:
./kdc start --db kdc.db --realm ATHENA.MIT.EDU
```

---

## Key Takeaways

1. **Single Sign-On**: One login (TGT) → multiple services (service tickets)
2. **Distributed Trust**: Each service can verify independently without KDC
3. **Offline Verification**: API server never contacts KDC during request processing
4. **Replay Protection**: Timestamps and caches prevent request reuse
5. **Secure Password Handling**: Passwords used only locally, never transmitted
6. **Encrypted Everything**: All sensitive data encrypted in transit

This is the foundation of enterprise authentication in Windows domains (Kerberos), and the same principles apply here in our demo!
