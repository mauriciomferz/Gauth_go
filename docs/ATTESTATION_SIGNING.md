---
title: Attestation Signing
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Model Limits Attestation: Notarization & Signing Flow

This document summarizes the canonical construction, notarization, and signing order for
the model limits attestation artifacts exposed at:

Endpoint: `/api/v1/model/limits/attestation`
Streaming: `/api/v1/model/limits/attestation/stream` (SSE)

## Construction Order

1. Build unsigned base struct (snapshot hash, audit/anchor heads, strict flag, nonce).
2. Optionally augment with surge stats (recent limit exceed surge detection) if a surge
   trigger occurred within the last 5 seconds.
3. If AGENTAUTH_MODEL_LIMIT_ATTEST_NOTARIZE=1 and a notarizer is configured, compute the
   combined hash:

   ```
   combined = sha256( "attest|" + snapshot.hash + "|" + audit.head + "|" + anchor.head )
   represented as: "sha256:<hex>"
   ```

   Submit to the notarizer; attach receipt (provider, timestamp, latency, success)
   to the unsigned struct *before* signing so the receipt itself is covered by the
   signature.
4. Marshal the unsigned attestation (excluding all signature-bearing fields).
5. Sign primary signature over: `AttestationDomainPrefix + unsignedJSON` where
   `AttestationDomainPrefix` is the constant `"AGENTAUTH_MODEL_LIMIT_ATTEST:"`.
6. If `AGENTAUTH_ATTEST_DOMAIN_PREFIX` environment variable is set, also produce a dual
   domain signature over: `<env-prefix> + unsignedJSON` (enables migration / agility).

## Signature Fields

| Field | Meaning |
|-------|---------|
| `signature` | Base64 raw URL primary Ed25519 signature over domain-prefixed unsigned payload |
| `sig_kid` | Key identifier of active Ed25519 key in global registry |
| `sig_mode` | Algorithm label (`eddsa`) |
| `domain_signature` | Optional dual domain signature when env prefix set |
| `domain_prefix` | The env-provided domain prefix corresponding to `domain_signature` |

Both `domain_signature` and `domain_prefix` use `omitempty` and are absent for legacy
consumers when not configured.

## Notarization Coverage

Because notarization (receipt) is attached prior to signing, verifiers validating the
signature inherently validate inclusion of (provider, timestamp, latency_seconds, success)
in the signed payload.

### Example (Dual Domain + Notarization)

```json
{
   "success": true,
   "configured": true,
   "nonce": "f8f4c3f2-2d3b-4e9d-b2a7-9c2ad1e0b1d1",
   "snapshot": {"hash": "sha256:1d3ea9c8...", "generated_at": "2025-10-27T21:40:11Z"},
   "audit": {"head_hash": "sha256:9ab41...", "entries": 42},
   "anchor": {"latest_hash": "sha256:7fee1...", "entries": 10, "interval": 1},
   "strict_unknown": true,
   "notarization": {"provider": "memory", "timestamp": "2025-10-27T21:40:11.123Z", "latency_seconds": 0.0023, "success": true},
   "signature": "1lX6qY9...rawurl...",
   "sig_kid": "active-key-1",
   "sig_mode": "eddsa",
   "domain_signature": "b8s7Qp2...rawurl...",
   "domain_prefix": "EXTRA_ATTEST:"
}
```

Primary and domain signing messages:

```
primary_msg = "AGENTAUTH_MODEL_LIMIT_ATTEST:" + unsignedJSON
domain_msg  = "EXTRA_ATTEST:" + unsignedJSON
```

### Soft Invalid Semantics Clarification

Any signature failure (primary or optional domain) results in a "soft invalid" response:

* HTTP 200
* `success: true`
* `valid: false`
* `error` one of: `signature_invalid`, `domain_signature_invalid`, `domain_signature_prefix_missing`, `domain_signature_base64_invalid`

Primary signature failure short-circuits domain signature checks. When the primary is valid but the domain signature fails, the overall attestation is still treated as invalid (soft) to ensure clients depending on dual signature integrity are protected.


## Replay Protection

`nonce` is included in the unsigned payload and therefore covered by the signature. The
verification path records nonces to defend against replay (configurable TTL, in-memory or
durable store). A reused nonce triggers a soft or hard failure depending on policy.

Soft vs Hard outcome summary:

| Condition | HTTP | success | valid | error | SoftInvalid |
|-----------|------|---------|-------|-------|------------|
| Signature mismatch (tamper) | 200 | true | false | signature_invalid | true |
| Nonce replay | 409 | true | false | attestation_nonce_replay | false |
| Missing signature fields | 400 | false | false | attestation_signature_fields_missing | false |
| Unknown key (kid) | 404 | false | false | attestation_unknown_kid | false |

## Canonicalization

Canonical JSON currently relies on Go struct field ordering (stable) and excludes all
signature-bearing fields. The helper `AttestationService.CanonicalizeModelLimitsUnsigned`
normalizes a JSON blob by removing `signature`, `sig_kid`, `sig_mode`, `domain_signature`,
and `domain_prefix` keys. Future migrations can plug in stricter canonicalization without
changing callers.

## Verification Summary

Verifier reconstructs unsigned JSON (dropping signature fields), prepends the constant
`AttestationDomainPrefix`, and Ed25519 verifies against the advertised public key. It also:

* Recomputes combined hash for optional notarization and returns it.
* Performs nonce replay detection via configured replay strategy.

## Environment Flags

| Variable | Purpose |
|----------|---------|
| `AGENTAUTH_MODEL_LIMIT_ATTEST_SIGN` | Enable signing (primary and optional dual) |
| `AGENTAUTH_MODEL_LIMIT_ATTEST_NOTARIZE` | Enable notarization (receipt added before signing) |
| `AGENTAUTH_ATTEST_DOMAIN_PREFIX` | Enable dual domain signature (agility / migration) |
| `AGENTAUTH_ATTEST_STREAM_ENABLE` | Enable SSE streaming endpoint |
| `AGENTAUTH_ATTEST_NONCE_TTL` | TTL (e.g. `1h`) for replay nonce eviction |

## Failure Modes

| Failure | Description | Handling |
|---------|-------------|----------|
| `no_active_key` | Global Ed25519 registry has no active key | 500 when signing requested |
| Notarizer error | Receipt acquisition failed | Receipt omitted (best-effort) |
| Replay detected | Nonce previously seen | Verification returns invalid (soft/hard) |
| `signature_invalid` | Payload mutated post-signing | 200 with success=true valid=false (soft) |
| `domain_signature_invalid` | Secondary domain signature failed verification | 200 soft invalid |
| `domain_signature_prefix_missing` | domain_signature present but domain_prefix absent | 200 soft invalid |
| `domain_signature_base64_invalid` | domain_signature not valid base64 (raw URL) | 200 soft invalid |
| `attestation_signature_base64_invalid` | Signature not valid base64 | 400 invalid |
| `attestation_signature_fields_missing` | Missing mandatory signature fields | 400 invalid |

## Future Enhancements

* Structured canonicalization (sorted key emission with stable float formatting)
* Externalized surge stats plugin interface
* Pluggable hash algorithm agility for combined hash
* Dual signature deprecation path once migration stabilizes

## RB6 Signature Agility & Metrics

RB6 adds an agile signing abstraction (`Signer` interface in `internal/crypto/signer.go`) enabling future algorithm introductions (ECDSA-P256, BLS12-381) without modifying attestation call sites. The `AttestationService.NotarizeAndSignModelLimits` now prefers `GlobalRotatingSigner()`; if it produces a signature, legacy direct Ed25519 logic is skipped. If unavailable or it fails, the service falls back to direct Ed25519 signing via the global registry.

Verification remains public-key based (`KeyFinder`) for stability; multi-algorithm verification will be introduced after sufficient fuzz & negative test coverage. Domain signature soft invalid codes remain unchanged.

### Metrics (Domain Signature Outcomes)

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `attestation_domain_signature_failures_total` | counter | `reason` | Counts soft invalid domain signature failures (`domain_signature_invalid`, `domain_signature_prefix_missing`, `domain_signature_base64_invalid`). |
| `attestation_domain_signature_success_total` | counter | none | Counts successful verification of an included domain signature. |

These metrics allow tracking adoption and integrity of dual domain signing separately from primary signature failures.

### Backward Compatibility

Legacy clients ignoring `domain_signature` and `domain_prefix` continue to operate; fields use `omitempty`. If agility signer absent, behavior matches pre-RB6 flow.

### Rationale

Separating signing agility from verification reduces blast radius for algorithm expansion and permits incremental rollout (sign first, verify later) validated by fuzz harness (`FuzzVerifyModelLimitsAttestation`).

## Persistent Replay Protection & Fuzz Coverage (RB17/RB16)

Attestation nonce replay protection now supports durability via a WAL + periodic snapshot/compact cycle (every 5 minutes, configurable with `AGENTAUTH_ATTEST_REPLAY_COMPACT_MINUTES`). The durable store ensures replay detection survives process restarts. Tests:

* `TestAttestationReplayPersistenceRestart` validates a recorded nonce remains detected after restart.
* `replay_store_*` tests cover corruption recovery and WAL rotation.

Fuzz harness seeds exercise signature base64 decoding failures, missing nonce conditions, dual signature variants, and inconsistent notarization (`Success=false`). This provides early detection of panics or incorrect soft invalid classification as the verification logic evolves.

Environment knobs:

| Variable | Purpose |
|----------|---------|
| `AGENTAUTH_ATTEST_REPLAY_COMPACT_MINUTES` | Interval in minutes for WAL snapshot+compact loop |
| `AGENTAUTH_REPLAY_CAP` | Capacity limit for in-memory nonce map (evicts oldest when exceeded) |
| `AGENTAUTH_REPLAY_WAL` or the explicit path passed to `NewReplayNonceStoreWithConfig` | Enables durability for replay store |

