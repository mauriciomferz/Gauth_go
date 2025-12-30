---
title: Milestone 2b Plan
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Milestone 2B Plan: Cryptographic Authenticity & Advanced Revocation

Status: In Progress (authenticity core implemented)
Actual Start: 2025-10-15
Target Complete: 2025-10-28 (10 calendar days)
Owner: Core Maintainers

## 1. Objective
Introduce verifiable cryptographic authenticity and strengthen delegation / revocation guarantees while expanding RFC citation coverage. Provide a foundation that is production-hardening friendly (plug‑in key management, external anchoring) without prematurely optimizing.

## 2. Scope (In)
- Signature abstraction for delegation (POA) objects and issued envelopes (NOT full algorithm zoo; start with Ed25519).
- Key material lifecycle: in‑memory key ring + pluggable provider interface (file, env, external HSM/KMS adapter stub).
- Signed delegation issuance flow: sign canonical POA representation + include signature & key ID in token envelope.
- Validation path: verify signature chain (grantor signature validity + non‑revoked + expiry + scope narrowing rules already present).
- Revocation enhancements: reason codes taxonomy + timestamp normalization + chain compact integrity proof (hash-of-hash summary).
- RFC citation matrix expansion referencing new sections (cryptographic steps, revocation semantics, chain tamper detection).
- Observability: counters for signature verify success/failure, revocation lookups, chain integrity failures.
- Skipped/placeholder tests that outline multi-sig & threshold delegation for future milestone.

## 3. Scope (Out / Deferred)
- Multi-signature & threshold (aggregate BLS / MuSig style) – plan only.
- External transparency log anchoring (submit chain tips) – stub interface only.
- Production persistence (DB-backed) for revocation events.
- Key rotation automation policies beyond manual trigger.

## 4. Deliverables
| Category | Deliverable | Acceptance Criteria |
|----------|-------------|---------------------|
| Crypto | `pkg/crypto/signature.go` Ed25519 signer + key provider | Implemented & tested (round trip, rotation, invalid sig) |
| Abstraction | Signing interfaces (`Signer`, `KeyProvider`) | Implemented; supports future pluggable providers |
| Delegation | POA issuance adds signature + key ID | Implemented; metrics increment on success/failure |
| Validation | Signature verification integrated | Implemented; success & failure counters increment; digest mismatch handled |
| Revocation | Enhanced `RevocationEvent` with Reason enum | Implemented (taxonomy + fallback); tests validate reason normalization |
| Integrity | Compact chain proof (hash of entire chain) | Implemented `AggregateHash()` with deterministic sequence + set components; tests added |
| Metrics | Signature issue/verify counters + latency sampling | Implemented in-memory; Prometheus adapter pending |
| Docs | Updated `COMPLIANCE_IMPLEMENTATION.md` & matrix | Pending update to reflect authenticity & metrics |
| Benchmarks | `BenchmarkSignCanonicalPOA`, `BenchmarkVerifyCanonicalPOA` | Baseline numbers captured (PERFORMANCE.md update pending) |

## 5. Data Structures (Draft)
```go
// Signature metadata embedded in envelopes / POA
type SignatureMeta struct {
    KeyID     string `json:"key_id"`
    Algo      string `json:"algo"` // ed25519
    Signature []byte `json:"sig"`
}
```

## 6. Interfaces (Draft)
```go
type Signer interface {
    KeyID() string
    Algorithm() string
    Sign(msg []byte) ([]byte, error)
}

type Verifier interface {
    Algorithm() string
    Verify(msg, sig []byte, keyID string) error
}

// KeyProvider returns active + previous public keys for verification.
type KeyProvider interface {
    ActiveSigner() (Signer, error)
    PublicKey(keyID string) ([]byte, string, error) // returns key bytes + algo
}
```

## 7. Canonicalization
Canonical POA signing: Implemented deterministic canonical JSON (stable ordering of arrays/maps, RFC3339 timestamps) with domain separation prefix `AgentAuthPOA_v1:` hashed via SHA-256; golden test added.

## 8. Metrics Additions
Implemented (in-memory): signatures_issued_total, signature_issue_failures_total, signature_verifications_total, signature_verification_failures_total, revocation_integrity_fail_total.

## 9. Risks & Mitigations
| Risk | Impact | Mitigation |
|------|--------|-----------|
| Canonicalization drift | Invalid signatures after code changes | Centralize helper + golden tests |
| Key rotation race | Validation using stale key set | Snapshot keys atomically; include grace window tests |
| Performance regression | Increased validation latency | Benchmarks; track ns/op delta (< +15%) |
| Reservoir percentile distortion | Misleading latency insight | Document approximation; allow disabling |

## 10. Success Metrics
- All new tests (unit + integration) pass.
- Signature verification adds <15% overhead vs pre-signing validation baseline.
- Zero flaky golden canonicalization tests across 10 CI runs.
- Documentation matrix updated with >=5 new RFC citation entries.

## 11. Task Breakdown (High Level)
1. (Done) Crypto interfaces + Ed25519 implementation.
2. (Done) In-memory key provider (rotation) integrated.
3. (Done) Canonical POA digest helper + golden test.
4. (Done) Signature issuance integration + metrics.
5. (Done) Verification in validation path + metrics.
6. (Done) Revocation event reason taxonomy & aggregate chain hash.
7. (Done) Metrics counters including integrity failure counter.
8. (Done initial) Micro benchmarks captured; integrate into baseline doc.
9. (In progress) Docs & compliance matrix expansion.
10. (Planned) Placeholder multi-sig tests & README stub.

## 12. Out-of-Scope Placeholders
Add `internal/crypto/multisig/README.md` describing future threshold design (not implemented).

## 13. Open Questions
- Do we need deterministic JSON or CBOR canonicalization? (Leaning JSON first.)
- Should we pre-hash with domain separation tag? (Likely: `AgentAuthPOA_v1:` prefix.)

---
Draft – iterate via PR before locking dates.
