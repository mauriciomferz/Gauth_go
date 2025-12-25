---
title: Rotation V2 Demo
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Rotation V2 Demo Guide

This guide shows how to exercise the weighted multi‑sig Rotation V2 endpoint (`GET /api/v1/rotation/summary/v2`) locally.

## Overview
Rotation V2 produces a JSON artifact containing:
- `signers[]`: id, alg, weight, optional signature/public key
- `threshold_weight`: minimum verified weight required
- `verified_weight` + `verified_weight_by_alg`
- Continuity hashes (`previous_artifact_hash`, `canonical_digest`)

Signatures (currently Ed25519) are domain‑separated over:
```
preimage = "GAUTH_ROTATION_V2:" + canonical_digest
```

## Configuration File
Default example: `config/multisig_weights.json`:
```json
{
  "schema_version": 1,
  "active_key_set_id": "default-set",
  "threshold_weight": 100,
  "signers": [
    { "id": "hsm-a", "alg": "ED25519", "weight": 60 },
    { "id": "soft-b", "alg": "ED25519", "weight": 20 },
    { "id": "notary-c", "alg": "ED25519", "weight": 40 }
  ],
  "algorithm_suite": ["ed25519"]
}
```
Total weight = 120; threshold = 100.

## Environment Variables
| Variable | Purpose | Values |
|----------|---------|--------|
| `GAUTH_ROTATIONS_V2_CONFIG` | Path to weights config JSON | e.g. `config/multisig_weights.json` |
| `GAUTH_ROTATIONS_V2_SIGN` | Enable signing path | `1` to enable |
| `GAUTH_ROTATIONS_V2_AUTO_GEN` | Auto‑generate ephemeral Ed25519 private keys when no registry/import provided | `1` to enable |
| `GAUTH_ROTATIONS_V2_ED25519_KEYS` | Explicit private key import (comma list `id:base64urlPriv`) | Optional |
| `GAUTH_ROTATIONS_V2_EMBED_PUBS` | Embed public keys (if resolvable) into artifact | `1` to enable |

Notes:
- AUTO_GEN implies signing even if `GAUTH_ROTATIONS_V2_SIGN` is not set.
- Explicit key import takes precedence over auto‑generation when provided.

## Running (Ephemeral Auto‑Gen)
```bash
cd Gauth_go
GAUTH_ROTATIONS_V2_CONFIG=config/multisig_weights.json \
GAUTH_ROTATIONS_V2_SIGN=1 \
GAUTH_ROTATIONS_V2_AUTO_GEN=1 \
./bin/web-server &
SERVER_PID=$!
sleep 2
curl -s http://localhost:8080/api/v1/rotation/summary/v2 | jq .
kill $SERVER_PID
```
Expected output excerpts:
```json
"verified_weight": 120,
"verified_weight_by_alg": {"ED25519": 120},
"threshold_met": true
```

## Running (Explicit Private Keys)
Generate three Ed25519 keys (example Go snippet):
```go
package main
import (
  "crypto/ed25519"; "encoding/base64"; "fmt"
)
func main(){
  ids := []string{"hsm-a","soft-b","notary-c"}
  for _, id := range ids {
    _, priv, _ := ed25519.GenerateKey(nil)
    fmt.Printf("%s:%s\n", id, base64.RawURLEncoding.EncodeToString(priv))
  }
}
```
Capture output like:
```
hsm-a:BASE64URL_PRIV1
soft-b:BASE64URL_PRIV2
notary-c:BASE64URL_PRIV3
```
Set environment:
```bash
GAUTH_ROTATIONS_V2_CONFIG=config/multisig_weights.json \
GAUTH_ROTATIONS_V2_SIGN=1 \
GAUTH_ROTATIONS_V2_ED25519_KEYS="hsm-a:BASE64URL_PRIV1,soft-b:BASE64URL_PRIV2,notary-c:BASE64URL_PRIV3" \
./bin/web-server &
```
Verification should show same weights.

## Optional: Embed Public Keys
If a global Ed25519 registry is active and `GAUTH_ROTATIONS_V2_EMBED_PUBS=1`, each signer will include `public` key (base64url). Embedding does NOT alter the canonical digest.

## Failure Diagnostics
Artifact fields:
- `failures`: array of reason codes (`signature_invalid`, `public_key_not_found`, `signature_decode`, `resolver_nil`, `unknown_alg`, `artifact_nil`).
- `verified_weight` < threshold sets `threshold_met=false` and increments `gauth_rotation_v2_threshold_violations_total`.

Enable stderr debug logging via the signing block messages already present (look for `[rotation-v2]` lines).

## Metrics Exposed
Prometheus metrics (names):
- `gauth_rotation_v2_verified_weight`
- `gauth_rotation_v2_verified_weight_alg{alg="ED25519"}`
- `gauth_rotation_v2_threshold_weight`
- `gauth_rotation_v2_signature_failures_total{reason=...}`
- `gauth_rotation_v2_signature_failures_by_alg_total{alg,reason}`
- `gauth_rotation_v2_continuity_updates_total`, `gauth_rotation_v2_chain_starts_total`

## Web UI Panel
A lightweight dashboard panel is now available (served from `web/static_ui`) displaying:
- Verified weight & threshold badge (green when met, amber when partial)
- Per‑algorithm aggregated verified weights
- Signer table (kid, alg, weight, truncated signature)
- Continuity hashes (previous vs latest canonical digest)
- Failure list (only shown if any verification failures recorded)

### Client-Side Signature Verification
If public keys are embedded in the artifact (`public` field per signer) and the browser supports WebCrypto Ed25519:
- The panel performs a best-effort verification of each signer using the domain separated preimage `GAUTH_ROTATION_V2:<canonical_digest>`.
- A checkmark (✓) appears in the "Verified" column for each successful signature, a cross (✗) for failed verification, and a dash (—) if verification was skipped (missing public key / unsupported alg / browser API limitation).

Limitations:
- Only ED25519 is attempted currently; other algorithms are skipped until implemented.
- Browsers without Ed25519 `crypto.subtle` support (older versions) will skip verification gracefully.
- Client verification is informational and SHOULD NOT replace server-side verification or auditing.

To embed public keys for demo purposes set `GAUTH_ROTATIONS_V2_EMBED_PUBS=1` (requires the server to have or reconstruct the public keys for imported/generated private keys).

### Copy & Download Utilities
The Rotation V2 panel provides quick actions:
- Copy Digest: Copies the current `canonical_digest` to clipboard.
- Copy Prev: Copies `previous_artifact_hash` (useful for chain comparisons).
- Download Artifact: Downloads the full JSON response (including verification and continuity fields) as `rotation_v2_artifact.json` for offline inspection or attestation archival.

If clipboard write fails (browser permission), the button will briefly show "Copy Failed".

## Offline Verification CLI
An offline verifier is available to validate Rotation V2 artifacts independently of the running server.

Build (or use go run):
```bash
go build -o bin/rotation-verify ./cmd/rotation-verify
```

Obtain an artifact (from UI Download button or curl):
```bash
curl -s http://localhost:8080/api/v1/rotation/summary/v2 -o /tmp/rotation_v2.json
```

Run verification (human output):
```bash
bin/rotation-verify --file /tmp/rotation_v2.json
```

JSON output (machine-friendly):
```bash
bin/rotation-verify --file /tmp/rotation_v2.json --json
```

Supply extra public keys (if artifact lacks embedded pubs):
```bash
bin/rotation-verify --file /tmp/rotation_v2.json \
  --pub hsm-a:ED25519:BASE64URLPUB1 \
  --pub soft-b:ED25519:BASE64URLPUB2
```

Integrity assertion (fail if digest differs):
```bash
bin/rotation-verify --file /tmp/rotation_v2.json --expect-digest sha256:abcdef...
```

Exit codes:
- 0: success & threshold met
- 1: general error / parse failure
- 2: usage error (missing flags)
- 3: threshold not met or digest mismatch (when using --expect-digest)

The verifier currently supports ED25519; additional algorithms (ECDSA-P256, etc.) will be added as they become stable in the artifact.

Auto‑refresh interval: 45s (same cadence as legacy rotation panel). The legacy panel remains for backwards compatibility; a small note points to the V2 panel for weighted multi‑sig details.

To view, start the server and open:
```
http://localhost:8080/ui/
```
Scroll to the "Rotation V2" panel.

## Security Considerations
- AUTO_GEN mode is for development only (keys are ephemeral and discarded on process exit).
- Explicit private keys should be supplied via secure secret management in production, not inline environment variables.
- Public key embedding is optional; verification SHOULD rely on a trusted key registry in production.
- Domain separation string `GAUTH_ROTATION_V2:` MUST remain unchanged to avoid cross‑protocol signature reuse.

## Next Steps / Extensions
- Add ECDSA-P256 support using `GAUTH_ROTATIONS_V2_ECDSA_KEYS` for public key embedding.
- Integrate real key rotation manager for persistent key sets.
- Provide `/api/v1/rotation/summary/v2/debug` for richer diagnostics without using stderr logs.

---
_Last updated: 2025-10-28_
