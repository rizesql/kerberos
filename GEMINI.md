# Gemini Context

This file serves as the primary context provider for the Gemini agent, summarizing the project requirements, architecture, implementation status, and demo instructions.

## 1. Project Requirements (from `docs/PROIECT_FINAL.md`)

**Course:** Distributed Systems (2025)
**Goal:** Implement a distributed system project based on the "Kerberos: An Authentication Service for Open Network Systems" paper.

### Deliverables:
1.  **Documentation (6p)**:
    *   Language: Romanian.
    *   Length: Minimum 5 pages.
    *   Structure: Introduction, Serial Algorithm, Distributed Algorithm (SPMD), Theoretical Analysis (Correctness, Complexity, Scalability, Topology).
2.  **Implementation (4p)**:
    *   Language: Go.
    *   Goal: Support the theoretical analysis with specific problem tests.
    *   Requirement: Mini-demo during presentation.

## 2. Architecture & Reference Paper

**Paper:** *Kerberos: An Authentication Service for Open Network Systems* (Steiner, Neuman, Schiller)

### System Components
*   **KDC (Key Distribution Center)**: Trusted third party (AS + TGS).
    *   **AS (Authentication Server)**: Authenticates client, issues TGT.
    *   **TGS (Ticket Granting Server)**: Issues service tickets.
*   **Client**: User agent (Web UI + CLI).
*   **API Server**: Protected resource server (verifies tickets offline).

### Protocol Flow
1.  **AS Exchange**: Client -> AS (Login) -> TGT.
2.  **TGS Exchange**: Client + TGT -> TGS -> Service Ticket.
3.  **CS/AP Exchange**: Client + Service Ticket -> API Server -> Access.

## 3. Repository Structure & Implementation Status

**Language:** Go
**Database:** SQLite (via `sqlc`)

### Directory Structure
*   `cmd/`
    *   `kdc/`: Key Distribution Center server (AS + TGS).
    *   `api/`: **[TODO]** Protected API Server.
    *   `client/`: **[TODO]** Client application with Web UI.
    *   `kadmin/`: **[TODO]** Administration tool (add principals).
*   `internal/`
    *   `protocol/`: Kerberos protocol structs (Ticket, Authenticator, etc.).
    *   `crypto/`: Encryption utilities (DES/AES, KDF).
    *   `kdb/`: Database interface and SQL queries.
    *   `kdc/`: KDC server logic (AS and TGS handlers).
    *   `ap/`: Application Protocol (verification logic).
    *   `replay/`: Replay cache implementation.
    *   `server/`: HTTP server boilerplate.
*   `docs/`: Documentation and LaTeX source files.

### Development Status (from `DEMO_GUIDE.md`)

| Component                         | Location                    | Status  |
| --------------------------------- | --------------------------- | ------- |
| **Core Protocol**                 |                             |         |
| Protocol Types                    | `internal/protocol/`        | ✅ Done |
| Crypto/KDF                        | `internal/crypto/`          | ✅ Done |
| Database/SQL                      | `internal/kdb/`             | ✅ Done |
| **KDC**                           |                             |         |
| AS Exchange (login)               | `internal/kdc/as/`          | ✅ Done |
| TGS Exchange (service ticket)     | `internal/kdc/tgs/`         | ✅ Done |
| KDC Server Entrypoint             | `cmd/kdc/`                  | ✅ Done |
| **Application Protocol (AP)**     |                             |         |
| AP Verification                   | `internal/ap/verify.go`     | ✅ Done |
| AP Middleware                     | `internal/ap/middleware.go` | ✅ Done |
| Replay Cache                      | `internal/replay/cache.go`  | ✅ Done |
| **Missing Components (TODO)**     |                             |         |
| **API Server**                    | `cmd/api/`                  | ✅ Done |
| **Client App (Web UI)**           | `cmd/client/`               | ✅ Done |
| **KAdmin Tool**                   | `cmd/kadmin/`               | ✅ Done |

## 4. Next Steps (Action Plan)

Based on `DEMO_GUIDE.md`, the immediate tasks are:

1.  **Implement API Server** (`cmd/api/`):
    *   Setup server with `ap.Verifier` and `ap.Middleware`.
    *   Implement routes: `/api/whoami` and `/api/secret`.
2.  **Implement Client** (`cmd/client/`):
    *   Setup server to serve static files (`web/`).
    *   Implement API routes: `/api/login`, `/api/ticket`, `/api/call`.
    *   Create simple HTML frontend.
3.  **Implement KAdmin** (`cmd/kadmin/`):
    *   CLI tool to add principals (users and services) to the DB.

## 5. Demo Instructions

Refer to `DEMO_GUIDE.md` for detailed demo commands.

**Quick Start:**
1.  **Setup KDC:** `./kdc setup --db kdc.db ...`
2.  **Add Principals:** Use `kadmin` (to be built).
3.  **Run Services:**
    *   `./kdc start`
    *   `./api start`
    *   `./client start`
