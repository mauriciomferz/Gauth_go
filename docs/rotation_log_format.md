---
title: Rotation Log Format
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# AgentAuth Key Rotation Log Format and Verification

## Format
Each key rotation event is recorded as a JSON line in the rotation log file. Example entry:

```json
{
  "ts": "2025-10-24T12:34:56.789Z",
  "event": "eddsa_key_rotated",
  "new_kid": "AbCdEfGh",
  "ttl_hours": 24,
  "history_size": 3,
  "prev_kid": "XyZ12345",
  "prev_hash": "...",
  "hash": "base64-encoded-sha256",
  "signature": "base64-encoded-ed25519",
  "public_key": "base64-encoded-ed25519-public"
}
```

- `ts`: Timestamp of rotation (RFC3339Nano)
- `event`: Event type
- `new_kid`: New key ID
- `ttl_hours`: Key lifetime in hours
- `history_size`: Number of retained previous keys
- `prev_kid`: Previous key ID (if any)
- `prev_hash`: Hash of previous log entry (if any)
- `hash`: SHA-256 hash of canonical JSON (excluding `hash`, `signature`, `public_key`)
- `signature`: Ed25519 signature over canonical JSON (excluding `hash`, `signature`, `public_key`)
- `public_key`: Ed25519 public key used for signature

## Verification
To verify a rotation log entry:
1. Remove `hash`, `signature`, and `public_key` fields from the entry.
2. Serialize the remaining fields to canonical JSON.
3. Compute SHA-256 hash and compare to the `hash` field.
4. Decode `signature` and `public_key` from base64.
5. Verify the Ed25519 signature over the canonical JSON using the public key.

## Example (Go)
See `internal/crypto/rotation_log_test.go` for a full test verifying log signature and hash integrity.

## Compliance
This format ensures each rotation event is:
- Tamper-evident (hash chain)
- Cryptographically signed (Ed25519)
- Verifiable by any party with access to the log

## References
- [Ed25519 Signature Scheme](https://ed25519.cr.yp.to/)
- [RFC 3339 Timestamp Format](https://datatracker.ietf.org/doc/html/rfc3339)
- [SHA-256 Hash Function](https://en.wikipedia.org/wiki/SHA-2)
