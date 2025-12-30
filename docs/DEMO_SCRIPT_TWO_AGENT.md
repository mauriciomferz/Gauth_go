---
title: Demo Script Two Agent
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Two-Agent AgentAuth Demo Script (Rotation, PoA, Attestation, Throttle)

Objective: Demonstrate governance transparency and reactive controls as two autonomous agents negotiate authorization, verify integrity artifacts, and handle anomaly-induced throttling.

## Cast
| Actor | Role |
|-------|------|
| Agent A | Requests transaction delegation (consumer) |
| Agent B | Issues PoA & performs governance attestations (issuer) |

## Pre-Demo Setup
1. Export environment flags:
   - `GAUTH_ROTATIONS_SIGN=1` `GAUTH_ROTATIONS_MULTISIG=1` `GAUTH_ROTATIONS_THRESHOLD=2`
   - `GAUTH_MODEL_LIMIT_ATTEST_SIGN=1` `GAUTH_ATTEST_STREAM_ENABLE=1`
   - `GAUTH_SEMANTIC_ANOMALY_Z_THRESHOLD=2.5`
2. Ensure at least 2 active Ed25519 rotation keys (trigger rotation if needed).
3. Warm revocation/Merkle chain (if endpoint implemented later).
4. Start server: `go run cmd/web-server/main.go`.

## Phase 1: Rotation Transparency
1. Agent B fetches `GET /api/v1/beta/rotations/summary`.
2. Verifies each signature and threshold.
3. Extracts `aggregate_hash` for future anchoring narrative.

Talking Point: “Multi-signature threshold met with SatisfiedWeight >= Threshold; each signature bound to canonical payload prefixed.”

## Phase 2: PoA Issuance & Validation
1. Agent B constructs PoA Definition (scopes: `transaction:execute`, requirements: model policy / quotas).
2. (Future) Applies weighted multi-signature scheme (currently single or equal weight placeholder).
3. Agent B issues PoA to Agent A.
4. Agent A submits transaction using PoA.
5. Server validates semantic + structural constraints.

Outcome: Authorized transaction accepted.

## Phase 3: Governance Attestation
1. Agent A requests `GET /api/v1/model/limits/attestation`.
2. Verifies Ed25519 signature with prefix `GAUTH_MODEL_LIMIT_ATTEST:`.
3. Computes expected `combined_hash` locally.
4. (If notarization enabled) Confirms notarization success field and timestamp.

Talking Point: “Domain-separated signatures prevent replay across artifact classes.”

## Phase 4: Semantic Anomaly Trigger
1. Rapidly send a burst of invalid PoA uses (simulate misuse) to inflate semantic rejection counters.
2. Observe z-score exceed threshold activating throttle.
3. Agent A receives `429 semantic_throttle_active` on subsequent requests.
4. Metrics scrape (`/api/v1/metrics/prometheus/semantic`) shows elevated anomaly scores.

Talking Point: “Reactive control engages pre-emptively based on standardized anomaly z-score.”

## Phase 5: Revocation Proof (Future Extension)
1. Revoke PoA or token; capture event hash.
2. Fetch `GET /api/v1/revocation/proof/:hash` (after implementation) and recompute Merkle root.
3. Validate match vs server-returned latest root.

## Phase 6: Rotation Continuity Negative Case (Optional)
1. Introduce artificial gap (test fixture) – expect `rotation_continuity_gap` error.
2. Emphasize fail-closed integrity stance.

## Wrap-Up & Roadmap Callouts
- Weight-based multi-signature PoA enforcement upcoming.
- Replay nonce addition for attestation.
- External anchoring & auditor CLI.
- Algorithm agility (BLS aggregate) for signature compression.

## Script Timing (Approx.)
| Segment | Duration |
|---------|----------|
| Setup | 2 min |
| Rotation & PoA | 4 min |
| Attestation | 2 min |
| Anomaly Trigger | 3 min |
| Revocation Proof | 2 min |
| Continuity Gap | 2 min |
| Roadmap | 1 min |

## Quick Verification Snippets
Rotation Summary Verification (Go pseudocode):
```go
sum := fetchRotationSummary()
if err := verification.VerifyRotationSummarySignature(sum.Summary); err != nil { panic(err) }
```

Attestation Verification (Go pseudocode):
```go
att := fetchModelLimitsAttestation()
raw := rebuildUnsigned(att)
msg := append([]byte("GAUTH_MODEL_LIMIT_ATTEST:"), raw...)
ok := ed25519.Verify(pubKey, msg, sigBytes)
```

Semantic Throttle Observation:
```bash
curl -s localhost:8080/api/v1/metrics/prometheus/semantic | grep anomaly_score
```

---
Maintain this script as features land; align with `RFC_COMPLIANCE_GAPS.md` remediation status.
