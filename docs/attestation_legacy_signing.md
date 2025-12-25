---
title: Attestation Legacy Signing
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Attestation Legacy Signing Provenance

This document archives the legacy (pre-`AttestationService`) model limits attestation signing + notarization flow
originally embedded in `web/server_clean.go`. It serves as a provenance reference for future migrations
(e.g., domain-separated signing enforcement, replay protection, tracing injection).

## Legacy Location
File: `web/server_clean.go` (commit `bec82e780009a8bb5676d0aeaa84dcba1f42a4bf` – "Major refactoring: Clean up and reorganize project structure").

Core functions:
- `apiModelLimitsAttestation` – built unsigned attestation then invoked `maybeAugmentAndSignAttestation` inline.
- `maybeAugmentAndSignAttestation` – added surge stats, optional notarization, and raw Ed25519 signature.

## Legacy Snippet (Signing + Notarization)
```go
if os.Getenv("GAUTH_MODEL_LIMIT_ATTEST_NOTARIZE") == "1" && s.notarizer != nil && att.Snapshot.Hash != "" {
    auditHead := ""; if att.Audit != nil { auditHead = att.Audit.HeadHash }
    anchorHead := ""; if att.Anchor != nil { anchorHead = att.Anchor.LatestHash }
    seed := fmt.Sprintf("attest|%s|%s|%s", att.Snapshot.Hash, auditHead, anchorHead)
    h := sha256.Sum256([]byte(seed))
    combinedHash := fmt.Sprintf("sha256:%x", h[:])
    if receipt, nErr := s.notarizer.Notarize(combinedHash); nErr == nil {
        att.Notarization = &struct { Provider string `json:"provider"`; Timestamp string `json:"timestamp"`; LatencySeconds float64 `json:"latency_seconds"`; Success bool `json:"success"` }{Provider: receipt.Provider, Timestamp: receipt.Timestamp, LatencySeconds: receipt.LatencySeconds, Success: receipt.Success}
    }
}
if os.Getenv("GAUTH_MODEL_LIMIT_ATTEST_SIGN") == "1" && crypto.GlobalEdDSARegistry != nil {
    if active := crypto.GlobalEdDSARegistry.Active(); active != nil && len(active.Private) == ed25519.PrivateKeySize {
        unsigned := att; unsigned.Signature = ""; unsigned.SigKid = ""; unsigned.SigMode = ""
        if raw, jerr := json.Marshal(unsigned); jerr == nil {
            sig := ed25519.Sign(active.Private, raw)
            att.Signature = base64.RawStdEncoding.EncodeToString(sig)
            att.SigKid = active.ID
            att.SigMode = sigModeEdDSA
        }
    }
}
```

Characteristics:
- Signing performed directly inside HTTP handler pipeline.
- Raw Ed25519 signature over JSON bytes (no domain prefix) for backward compatibility.
- Notarization uses combined hash `sha256(attest|snapshot|auditHead|anchorHead)` submitted to `s.notarizer`.
- Error paths largely silent (failures skip augmentation silently).
- Replay protection absent; nonce not part of signature scope.
- Surge stats also mutated inside same augmentation function.

## New Service Architecture
File: `pkg/attest/service.go`.
- `AttestationService` centralizes:
  - `BuildUnsigned` (nonce generation, caller provides canonical JSON).
  - `NotarizeAndSignModelLimits` (combined hash + optional notarization + raw signature).
  - Domain-separated signing helpers (`SignDomainSeparated`, `SignDomainSeparatedB64`).
- Explicit error surfaces: `no_active_key`, `no_private_material`.
- Separation allows:
  - Future injection of replay store.
  - Tracing wrappers.
  - Metrics without entangling HTTP handlers.

## Behavioral Parity
| Concern | Legacy | New Service |
|---------|--------|------------|
| Raw signing | ed25519 over unsigned JSON | Same (phase 1 preserved) |
| Domain separation | Not used | Available helpers (not yet enforced) |
| Notarization seed | `attest|snapshot|audit|anchor` SHA-256 | Same computation |
| Error handling | Silent skip | Returns explicit errors (best-effort otherwise) |
| Replay protection | None | Placeholder (future injection) |
| Test isolation | Global registry side effects | Dedicated unit tests with isolation helpers |

## Migration Plan (Phases)
1. (Completed) Extract logic – preserve raw signing semantics.
2. (Completed) Introduce optional domain-separated dual signature via `GAUTH_ATTEST_DOMAIN_PREFIX` (raw + domain in response).
3. Enforce domain separation once ecosystem upgraded; retain backward compat (serve both for deprecation window, then remove raw).
4. Add replay protection (nonce registry + TTL + durable store).
5. Attach tracing + metrics decorators at service boundary.
6. Remove legacy augmentation code from `web/server_clean.go` after full adoption.

## Risks & Notes
- Ensure any future domain prefix is stable and documented (`GAUTH_ATTEST_DOMAIN_PREFIX` proposed).
- Keep combined hash format unchanged for notarization receipts to maintain historical continuity.
- When enforcing domain separation, communicate cutover window and provide dual verification path for older signatures.

## Recommended Removals (Future)
- Delete `maybeAugmentAndSignAttestation` after handlers call `AttestationService` exclusively.
- Remove environment variable `GAUTH_MODEL_LIMIT_ATTEST_SIGN` in favor of service-level config object.

## Verification
Unit tests in `pkg/attest/service_test.go` cover:
- Nonce generation
- Domain-separated signing (single and base64 helpers)
- Raw + notarization path
- Error modes: missing key, truncated private
- Dual signature migration (raw + domain) when `GAUTH_ATTEST_DOMAIN_PREFIX` set
Legacy semantics validated by reproducing raw signature verification with active key.

---
Provenance archived: commit `bec82e780009a8bb5676d0aeaa84dcba1f42a4bf`. Safe to proceed with next migration phase.
