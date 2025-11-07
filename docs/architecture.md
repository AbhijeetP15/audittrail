# Architecture (Draft)

- **Ledger:** Hyperledger Fabric (permissioned) to record credential access events.
- **Chaincode (Go):** Implements Issue, Verify, Revoke, and Audit query.
- **API (Node/Express):** Thin service that exposes HTTP endpoints and delegates to Fabric SDK (to be wired).
- **Data:** No PII on-chain. Only identifiers, hashes, and event metadata.
- **Future UI:** A small dashboard for holders to review their audit trail.

> This draft mirrors the initial project proposal. It will evolve as we test on a local Fabric devnet.