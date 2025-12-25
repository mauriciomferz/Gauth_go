---
title: Crypto Vectors
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

## Crypto Interface & Test Vectors (Phase 1)

This document defines the unified `Signer` interface, encoding rules, and canonical test vector guidance for Ed25519, ECDSA (P‑256), and BLS12‑381 used in GAuth.

### 1. Signer Interface

```go
type Signer interface {
    Algo() string              // Algorithm identifier ("Ed25519", "ECDSA-P256", "BLS12-381")
    Public() []byte            // Raw public key bytes (format depends on algo)
    Sign(msg []byte) ([]byte, error) // Deterministic or randomized signature (algo-specific)
    Verify(msg, sig []byte) bool     // Constant-time verification (best-effort)
}
```

Adapters implemented in `internal/crypto/signer.go`:
- `Ed25519Signer`: Pure EdDSA; public key canonical 32 bytes.
- `ECDSASigner` (P‑256): Uncompressed public key: `0x04 || X || Y` (65 bytes). Signatures encoded DER (SEQUENCE of r,s).
- `BLSSigner`: Public key serialized via library (`pk.Serialize()`); signature raw group element serialization.

### 2. Encoding Rules

| Algorithm | Public Key Encoding | Signature Encoding | Notes |
|-----------|---------------------|--------------------|-------|
| Ed25519 | 32 raw bytes | 64 raw bytes | Deterministic per RFC 8032 |
| ECDSA-P256 | Uncompressed SEC1 (0x04 + X + Y) | DER: `30 len 02 r 02 s` | Low-S enforced during signing; high-S rejected during verify |
| BLS12-381 | Library serialization (compressed form) | Library serialization | Aggregation supported (same message only) |

#### DER Signature (ECDSA)
During signing we normalize `s` to low-S: if `s > N/2`, replace with `N - s`. This removes malleable high-S variants. Verification rejects high-S outright to prevent accepting non-normalized signatures.

### 3. Test Vector Structure

`internal/crypto/test_vectors.go`
```go
type TestVector struct {
    Alg      string
    Curve    string      // For ECDSA variants
    Message  []byte
    Private  []byte      // Optional for verify-only vectors
    Public   []byte
    Signature []byte
    Valid    bool
}
```

Guidelines:
1. Provide at least one positive and one negative vector per algorithm.
2. Negative vectors: mutated first byte of signature, truncated signature, wrong curve point (ECDSA), invalid subgroup (BLS deserialize fail).
3. Message byte sequences should avoid trailing zeros (simplify cross-language hex handling).
4. For ECDSA negative high-S: craft signature then flip to high-S form prior to re-encoding; expect verification failure.

### 4. BLS Aggregation

`BLSSimpleAggregator` (in `internal/crypto/aggregator.go`):
```go
agg := NewBLSSimpleAggregator(msg)
agg.Add(pubKeyBytes, sigBytes) // verifies individual signature before inclusion
aggSig, _ := agg.Aggregate()
ok := agg.Verify(msg, aggSig, [][]byte{pub1, pub2, ...})
```
Metrics (when constructed via `NewBLSSimpleAggregatorWithMetrics`):
| Metric | Description |
|--------|-------------|
| `multi_signature_verifications_total` | Successful aggregate verifications |
| `multi_signature_verification_failures_total` | Failed aggregate verifications |
| `multi_signature_aggregate_latency_seconds` | Aggregation latency histogram |
| `multi_signature_batch_size` | Distribution of signature batch sizes |

### 5. Detached Signature Enforcement

Hook: `EnforceDetachedSignature(payload, sig []byte)` returns error if `COMPLIANCE_REQUIRE_SIGNATURE=1` (or `GAUTH_REQUIRE_DETACHED_SIGNATURE=1`) and `sig` is empty.

Metric implemented (Prometheus): `gauth_crypto_signature_missing_total` counting enforcement failures where a required detached signature was absent. This enables dashboards & alerting on systematic signature omission.

### 6. Future Work (Phase 2+)
| Item | Description |
|------|-------------|
| RFC6979 Deterministic ECDSA | Replace rand-based signing with deterministic nonce derivation |
| Threshold Signatures | Introduce share-based or MPC threshold adapter |
| Domain Separation for BLS | Prefix message with context string to avoid cross-protocol collisions |
| Batch Verification | Optimize Ed25519/ECDSA multi-verify using curve / library primitives |
| Verify-Only Mode Public Wrapper | Support constructing signers from raw public key bytes for remote key ingestion |
| Algorithm Metadata Endpoint | Expose `/api/v1/crypto/algorithms` returning enabled algos & parameters |

### 7. Acceptance Criteria (Phase 1 Complete)
Checklist:
- [x] Unified interface compiled and adopted by tests.
- [x] Low-S normalization enforced and tested (high-S signatures rejected).
- [x] BLS aggregation tests pass for 3 participants.
- [x] Negative signature corruption tests fail verification.
- [x] Detached signature missing counter exposed & documented.
- [ ] RFC6979 adoption (deferred).

### 8. Cross-Language JSON Fixture Set

Location: `internal/crypto/fixtures/crypto_vectors.json`

Schema (each element):
```json
{
  "alg": "Ed25519 | ECDSA-P256 | BLS12-381",
  "message_hex": "<hex-encoded message bytes>",
  "public_hex": "<hex-encoded public key bytes>",
  "signature_hex": "<hex-encoded signature bytes>",
  "valid": true,
  "note": "optional context for negative vectors"
}
```

Generation command:
```bash
go run ./cmd/gen-crypto-vectors > internal/crypto/fixtures/crypto_vectors.json
```

Determinism:
- Ed25519: deterministic (seeded) — stable across regenerations.
- ECDSA-P256: deterministic private key & deterministic nonce stream; produces canonical low-S DER.
- BLS12-381: current library uses CSPRNG only; entries marked with note `non-deterministic key (library CSPRNG)` may change on regeneration.

Negative Cases Included:
- Ed25519: mutated first byte (fails verify).
- ECDSA-P256: high-S malleable variant (rejected), truncated DER (parse fail), mutated variants.
- BLS12-381: corrupted first byte (deserialize or verify failure).

Validation Test: `internal/crypto/fixtures/crypto_vectors_test.go` loads the JSON and enforces:
- DER parse succeeds for valid ECDSA vectors.
- High-S ECDSA signatures are always treated invalid even if curve math would ordinarily pass.
- BLS signatures verify byte-for-byte; corrupted variants fail.

When adding a new algorithm:
1. Extend generator in `cmd/gen-crypto-vectors/main.go`.
2. Add positive + at least one structural corruption + one semantic malleability case (if applicable).
3. Update this document (Sections 2 & 8) with encoding & determinism notes.
4. Extend fixture validation test to cover new algorithm semantics.

Cross-language consumption tips:
- Treat all hex as lowercase; do not normalize or trim leading zeros.
- For ECDSA DER, ensure your decoder enforces SEQUENCE length consistency and rejects high-S if you want canonical equivalence.
- For BLS, confirm library expects compressed public keys & signatures matching serialization used here (herumi bls-eth-go-binary format).

### 9. Security Considerations
- Enforce low-S ensures uniqueness of ECDSA signatures (mitigates malleability).
- BLS aggregation assumes identical message; heterogeneous message aggregation out-of-scope Phase 1.
- Ed25519 always deterministic; side-channel mitigations rely on upstream Go implementation.
- High-S rejection prevents accepting signatures produced by non-normalizing implementations.

### 10. Error Codes (Preliminary)
| Code | Meaning |
|------|---------|
| `ed25519_signer_no_private` | Attempted to sign without private key material |
| `ecdsa_signer_no_private` | Missing private key for signing |
| `invalid_der_prefix` | Malformed DER signature (wrong SEQUENCE tag) |
| `r_length_out_of_bounds` | Declared r length exceeds buffer |
| `s_length_out_of_bounds` | Declared s length exceeds buffer |
| `bls_signer_no_private` | Signing attempted without BLS secret key |
| `ecdsa_high_s_rejected` | High-S signature rejected as non-canonical |

---
Maintainers: Update this doc when adding algorithms or modifying encoding. All new signature formats MUST specify canonical encoding, malleability handling, and deterministic vs randomized nonce strategy.
