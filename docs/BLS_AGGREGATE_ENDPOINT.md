---
title: Bls Aggregate Endpoint
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Aggregated BLS Endpoint (`/api/v1/crypto/bls/aggregate`)

Prototype endpoint enabling compressed multi-signer issuance & verification using BLS12-381 aggregated signatures.

## Modes
- issue: Generate aggregated signature over identical canonical `message_b64` bytes for either ephemeral participants or provided `private_keys_b64[]`.
- verify: Verify a previously produced aggregated signature using participant public keys and the original message.

## Request / Response Schemas
### Issue Request
```jsonc
{
  "mode": "issue",
  "message_b64": "aGVsbG8gd29ybGQ=",
  "participants": 3
  // OR instead of participants: "private_keys_b64": ["...","...",...] (SecretKey serialized, base64)
}
```

### Issue Response
```jsonc
{
  "success": true,
  "mode": "issue",
  "participant_count": 3,
  "aggregated_signature_b64": "...", // Serialized aggregated signature
  "public_keys_b64": ["...","...","..."], // Serialized PublicKey per participant
  "key_ids": ["9f3ab1...","b821c0...","0d54aa..."], // First 12 hex chars of sha256(pubkey)
  "latency_ms": 1
}
```

### Verify Request
```jsonc
{
  "mode": "verify",
  "message_b64": "aGVsbG8gd29ybGQ=",
  "aggregated_signature_b64": "...",
  "public_keys_b64": ["...","...","..."]
}
```

### Verify Response
```jsonc
{
  "success": true,
  "mode": "verify",
  "participant_count": 3,
  "valid": true,
  "latency_ms": 1
}
```

## Error Codes (HTTP 400 Unless Otherwise Noted)
| Error | Meaning |
|-------|---------|
| invalid_mode | `mode` not `issue` or `verify` |
| missing_message | `message_b64` absent |
| message_decode_failed | Base64 decode failed for `message_b64` |
| participants_too_large | `participants` > 64 (guardrail) |
| no_private_keys_or_participants | Neither `participants` nor `private_keys_b64[]` provided |
| private_key_decode_failed | Base64 decode failure for a provided private key |
| private_key_deserialize_failed | BLS deserialization failure for private key bytes |
| aggregated_signature_decode_failed | Base64 decode failure for `aggregated_signature_b64` |
| aggregated_signature_deserialize_failed | BLS signature deserialization failure |
| public_key_decode_failed | Base64 decode failure for an entry in `public_keys_b64[]` |
| public_key_deserialize_failed | BLS public key deserialization failure |
| missing_signature_or_keys | Verify mode missing aggregated signature or key list |
| bls_init_failed (500) | Library initialization failure |

## Metrics
Recorded via Metrics interface:
- `multi_signature_batch_size` histogram (issue + verify paths)
- `multi_signature_aggregate_latency_seconds` histogram (issue + verify paths)
- `multi_signature_verifications_total` (successful verify)
- `multi_signature_verification_failures_total` (failed verify)
- `multi_signature_verification_latency_seconds` (verify latency only)

## Security Notes
- All participants MUST sign identical canonical bytes supplied in `message_b64`.
- Ephemeral issuance mode does not persist secret keys; caller must retain returned public keys.
- Rogue key / proof-of-possession mitigation planned but not yet enforced in prototype.
- Aggregated signature verification currently assumes honest key setup; future updates will introduce PoP checks.

## Operational Guidance
1. Use `participants` for quick prototyping; switch to `private_keys_b64[]` when integrating stored signer material.
2. Persist `public_keys_b64[]` and `aggregated_signature_b64` alongside message canonical form in audit storage.
3. Monitor batch size distribution; abnormal spikes may indicate misuse or unexpected aggregation strategy shifts.
4. Track per-algorithm anchor emissions (`capability_anchor_algorithm_emitted_total{algorithm="bls12-381-agg"}`) to measure adoption.

## Example (Go)
```go
issueBody := map[string]interface{}{"mode":"issue","message_b64":base64.StdEncoding.EncodeToString([]byte("hello")),"participants":3}
// POST to /api/v1/crypto/bls/aggregate -> parse aggregated_signature_b64 & public_keys_b64[]
verifyBody := map[string]interface{}{"mode":"verify","message_b64":issueBody["message_b64"],"aggregated_signature_b64":aggSig,"public_keys_b64":pubKeys}
// POST verify -> expect valid=true
```

## Planned Enhancements
- Proof-of-possession requirement for public keys.
- Weight mapping + heterogeneous aggregated threshold support.
- Persistent aggregated signature audit ledger + external anchoring.
- Error detail standardization (structured codes + correlation IDs).

## Troubleshooting
| Symptom | Suggestion |
|---------|------------|
| Frequent `aggregated_signature_deserialize_failed` | Verify signature bytes unmodified; check transport encoding |
| `public_key_decode_failed` at index N | Confirm base64 integrity; inspect upstream copy/paste or truncation |
| Latency spikes >100ms | Enable pprof; check CPU contention; confirm participants <=64 |
| `bls_init_failed` | Ensure library loaded correctly; version mismatch or memory pressure |

---
Prototype status: interface may evolve; monitor release notes for breaking changes prior to production reliance.
