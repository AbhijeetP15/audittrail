# AuditTrail (starter repo)

An early skeleton for **AuditTrail** — a permissioned, blockchain‑based audit ledger that records who accessed or verified your digital credentials and when. The goal is to give users **visibility** and **control** over access events for verifiable credentials (e.g., diploma, license).

> This repo includes a draft smart‑contract (Hyperledger Fabric chaincode in Go), a stub HTTP API (Node/Express), and docs to get started. Details are intentionally light at this stage.

## Description of the project
- **What it does:** Records *access events* for user credentials on a permissioned ledger and exposes simple APIs to issue, verify, revoke credentials, and query audit trails.
- **Why Fabric:** Permissioned access, privacy controls, and modular chaincode fit audit requirements better than public chains at this stage.
- **Scope (MVP):**
  - Issue a credential (metadata only; no PII on‑chain — store only hashes/IDs)
  - Record verify/access events
  - Revoke a credential
  - Query audit trail per holder DID

## Dependencies / setup (draft)
- **Go** ≥ 1.21
- **Node.js** ≥ 18 (for the API)
- **Hyperledger Fabric** 2.5 toolchain and samples (development only)
  - Docker & Docker Compose
  - `fabric-samples` devnet (recommended for local testing)

> Tip: If Fabric is heavy for now, you can still iterate purely on function signatures and unit tests. The chaincode file compiles independently.

### Quick local setup (very rough)
1. Install Go and Node.
2. From the repo root:
   ```bash
   # API deps
   cd api && npm install && cd ..
   # (Optional) Go build check
   cd contracts && go mod tidy && go build ./... && cd ..
   ```
3. To stand up a local Fabric devnet later, follow the official `fabric-samples` docs and point the chaincode package to `./contracts`.

## How to run (draft / incomplete)
- **API (stub only):**
  ```bash
  cd api
  npm start
  # POST/GET the endpoints below (they are mock handlers for now)
  ```

- **Chaincode (draft):** export as a chaincode package and deploy via Fabric lifecycle when a devnet is available. Current file contains signatures & comments for the MVP.

## Draft Contract/Code
- Location: [`contracts/chaincode.go`](contracts/chaincode.go)
- Key functions (signatures can evolve):
  - `IssueCreds(ctx, credID, holderDID, credType, hashedData, issuerID) error`
  - `VerifyCreds(ctx, credID, verifierID) (*VerificationResult, error)`
  - `RevokeCreds(ctx, credID, reason, revokerID) error`
  - `QueryAuditTrail(ctx, holderDID, pageSize, bookmark) ([]AccessEvent, string, error)`

> See inline comments for data model and invariants.

## API (stub)
- Location: [`api/server.js`](api/server.js)
- Endpoints (mock):
  - `POST /api/issue`
  - `POST /api/verify`
  - `POST /api/revoke`
  - `GET  /api/audit?holderDid=...`

## Roadmap (short)
- Hook API to Fabric SDK (Node or Go)
- E2E flow in local devnet (issue → verify → revoke → query)
- AuthN/Z for API (issuer/verifier roles)
- Basic dashboard UI (later)

## Contributions
- Keep README updated as interfaces stabilize.

