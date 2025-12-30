---
title: Auditor Cli
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Auditor CLI

The Auditor CLI provides offline verification of core transparency and authorization artifacts emitted by a running AgentAuth server.

## Features (current)
## Features (current)
- Rotation Summary Verification: Fetches `/api/v1/rotation/summary` and verifies every collected EdDSA signature.
- PoA Verification (Remote): Fetches a PoA document (placeholder endpoint `/api/v1/poa/{id}`) and verifies canonical digest + multi-signature threshold.
- PoA Verification (Local): Reads a JSON PoA file and performs digest + signature threshold validation.
- Attestation Verification (Local & Remote): Verifies Ed25519 signature over domain-separated unsigned attestation subset and recomputes combined hash triple. Local mode additionally tracks replay of nonces across a CLI session.

# Verify local attestation JSON
./auditor --mode attestation-file --attestation-file ./att.json

# Verify remote attestation (assuming endpoint exists)
./auditor --mode attestation-remote --base-url http://localhost:8080
- Revocation Chain & Consistency Proof Verification: Recompute Merkle roots and verify consistency proofs.

# Verify local PoA JSON file
./auditor --mode poa-file --poa-file ./poa_sample.json

# Verify local attestation JSON
./auditor --mode attestation-file --attestation-file ./att.json

# Verify remote attestation (assuming endpoint exists)
./auditor --mode attestation-remote --base-url http://localhost:8080
Build from source:
- Attestation Remote Replay Inspection: Future enhancement to query server for nonce replay status history.
- Rotation integrity (multi-signature governance)
- Deterministic PoA digests and threshold signatures
- Attestation integrity & combined hash linkage to audit / anchor artifacts
The resulting binary `auditor` can be placed in your PATH.

## Usage
```bash
# Verify rotation summary
./auditor --mode rotation --base-url http://localhost:8080

# Verify remote PoA (assuming endpoint exists)
./auditor --mode poa-verify --base-url http://localhost:8080 --poa-id poa_123

# Verify local PoA JSON file
./auditor --mode poa-file --poa-file ./poa_sample.json
```
Add `-v` for verbose timing output.

## JSON Output Contract
Each run prints a JSON object:
```json
{
  "success": true,
  "mode": "rotation",
  "detail": {"signatures_valid": 3, "signatures_total": 3},
  "latency_ms": 42
}
```
On failure:
```json
{
  "success": false,
  "mode": "poa-file",
  "error": "audit_failed",
  "reason": "poa-file required",
  "latency_ms": 0
}
```

## Exit Codes
- 0: Success
- 1: Verification failure or invalid invocation

## Security Notes
- Verification uses in-process `GlobalEdDSARegistry`; ensure it is initialized identically to the server's keyset (shared config or exported fixtures).
- No network calls are made in local PoA verification mode beyond reading the file.
- Future attestation mode will perform replay nonce checks without modifying server state.

## Extensibility
Add new modes by extending the `switch` in `cmd/auditor/main.go` and implementing a helper function returning a structured map. Keep output backward compatible.

## Roadmap Alignment
This tool supports external auditor workflows required for RFC0111 / RFC0115 compliance narratives by enabling independent verification of:
- Rotation integrity (multi-signature governance)
- Deterministic PoA digests and threshold signatures

Attestation & revocation verification will complete transparency coverage in forthcoming iterations.

---
_Last updated: 2025-10-26_
