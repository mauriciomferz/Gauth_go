---
title: ADR Key Rotation Scheduler & Secure Secret Storage Integration
category: adr
status: proposed
lastUpdated: 2025-11-12
owners: architecture-team
source: internal
refreshCadence: on-change
---
# ADR: Key Rotation Scheduler & Secure Secret Storage (Vault/KMS)

Status: Proposed
Date: 2025-10-20
Drivers:

Summary: Proposes automated key rotation scheduler, secure secret storage integration (Vault/KMS), rotation log, metrics, and discovery endpoint extensions. Implementation in beta-refactor branch.

References: See GAP_MATRIX Section 14, implementation in `internal/crypto/rotation_manager.go`, tests in `internal/crypto/rotation_manager_test.go`.

## Goals
1. Introduce automated rotation scheduler (configurable interval + jitter window).
2. Support secure storage backends: HashiCorp Vault (KV + Transit), Cloud KMS (abstract interface).
3. Persist rotation events with signed JSONL + ledger receipt anchoring + optional external notarization.
4. Expose rotation metadata via discovery & metrics.

## Non-Goals
- Multi-tenant segregation in first iteration (documented as follow-up).
- Complex threshold cryptography for rotation authorization.

## Architecture
Components:
- `RotationManager` goroutine: ticks at interval; triggers key generation -> activation -> archival.
- `KeyStore` interface: methods `Generate()`, `Activate(kid)`, `Archive(kid)`, `FetchActive()`, `List()`. Implementations: `FileKeyStore`, `VaultKeyStore`, `KMSKeyStore`.
- `RotationLog` (JSONL) augmented with signature over entry digest; entry also written to Bolt ledger; external notarization optional.

### Configuration
Env vars:
```
AGENTAUTH_KEY_ROTATION_ENABLED=1
AGENTAUTH_KEY_ROTATION_INTERVAL=24h
AGENTAUTH_KEY_ROTATION_JITTER=15m
AGENTAUTH_KEY_BACKEND=vault|kms|file
AGENTAUTH_VAULT_ADDR=https://vault.example
AGENTAUTH_VAULT_TOKEN=... (or injected via file ref)
AGENTAUTH_KMS_PROVIDER=gcp|azure|aws
AGENTAUTH_KMS_KEYRING=projects/.../locations/.../keyRings/.../cryptoKeys/...
```

### Rotation Flow
1. Scheduler triggers generation: new key pair via backend (Vault transit or local Ed25519).
2. Key stored, metadata recorded (kid, created_at, algorithm, curve, backend).
3. Update active pointer (atomic swap) -> old active becomes previous.
4. Emit rotation log entry: `{kid, prev_kid, timestamp, backend, hash}` hashed & optionally signed.
5. Persist entry to ledger + optional external notarization.

### Metrics
- `agentauth_key_rotation_total`
- `agentauth_key_rotation_failures_total`
- `agentauth_key_active_age_seconds` (gauge)
- `agentauth_key_backend_info` (constant labels via build info metric or gauge 1)

### Discovery Endpoint Extensions
Add to `/.well-known/agentauth-configuration`:
```
"key_rotation": {
  "enabled": true,
  "interval_seconds": 86400,
  "backend": "vault",
  "next_rotation_eta_seconds": 12345,
  "active_kid": "..."
}
```

### Security Considerations
- Vault token retrieval via file path or process environment; avoid logging tokens.
- Rotation atomicity ensures no window with absent active key.
- Failure handling: on generation failure, increment failure counter, alert, retry with backoff.

### Data Persistence
- Keys: stored encrypted (Vault transit or KMS encryption). Local fallback: sealed file with passphrase env var (documented risk).
- Rotation log: append-only + hash chain; chain tip periodically externally anchored.

### Testing Strategy
- Unit tests: scheduler timing (mock clock), key activation transitions, rotation log hash chain integrity.
- Property tests: rotation log digest stability ignoring non-semantic fields (e.g., ordering).
- Integration tests: Vault backend mock, simulated KMS provider.

### Migration Plan
1. Introduce interfaces & file backend for parity.
2. Add scheduler w/ interval parsing & jitter.
3. Integrate Vault backend (feature flag gating).
4. Add metrics & discovery fields.
5. Add external notarization hook (depend on notarization ADR implementation).
6. Update GAP_MATRIX & documentation.

### Risks
- Misconfiguration (too frequent rotation) causing performance churn.
- Vault/KMS latency; mitigate with caching active key in memory.

### Alternatives
- Manual rotation via CLI only (slower to respond to compromise).
- Using a single monolithic secrets manager (reduces portability).

### Success Criteria
- Automatic rotation occurs at configured interval + jitter.
- Discovery endpoint reports accurate upcoming rotation ETA.
- Ledger contains signed rotation entries with verifiable hash chain.
- Metrics reflect rotations & failures.
- GAP_MATRIX entries updated to reflect improved status.

### Follow-Up
- Multi-tenant key segregation & per-tenant rotation schedules.
- Threshold approval workflow for rotation authorization.
