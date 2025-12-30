---
title: Readme
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# rotation-verify CLI

The `rotation-verify` command validates a Rotation Summary V2 artifact (weighted multi-signature key rotation descriptor).

## Input Artifact Shapes
The tool accepts either:
1. A JSON envelope with a top-level `artifact` field containing a `WeightedRotationArtifact`.
2. A direct artifact object with fields: `version`, `active_key_set_id`, `previous_artifact_hash`, `threshold_weight`, `signers`, `algorithm_suite`, `canonical_digest`, `generated_at`.

## Signer Public Keys
- The verifier attempts to reconstruct a public key resolver from any embedded signer public keys (base64url in each signer entry's `public` field) when present.
- Additional public keys can be supplied via repeated `--pub id:ALG:base64urlpub` flags. Currently only `ED25519` is supported.
- If no public keys are available, verification weight will be 0. The CLI will exit with code 3 if the threshold is not met.

## Exit Codes
- `0`: Verification succeeded and threshold met (or `--json` output produced with threshold met).
- `2`: Missing required `--file` flag.
- `3`: Verification executed but threshold not satisfied.
- `1`: Other errors (e.g., unreadable file, malformed JSON).

## Threshold Handling in Tests
The test `TestRotationVerifyCLI` treats an exit code 3 (threshold unmet) as a skip so that local environments without signing material do not produce false failures.

## Environment Integration
The HTTP endpoint `/api/v1/rotation/summary/v2` may emit an unsigned artifact if signing env vars are not configured. To enable local signing:
```
export AGENTAUTH_ROTATIONS_V2_CONFIG=./path/to/weights_config.json
export AGENTAUTH_ROTATIONS_V2_SIGN=1
# Optional: supply explicit private keys (32-byte seed or full 64-byte private encoded as hex or base64url):
export AGENTAUTH_ROTATIONS_V2_ED25519_KEYS="signer1:BASE64URLPRIV,signer2:HEXPRIV"
# Auto-generate ephemeral keys if none provided:
export AGENTAUTH_ROTATIONS_V2_AUTO_GEN=1
```
When keys are present the endpoint returns a signed artifact and the CLI should pass threshold verification.

## Example Usage
Human-readable output:
```
go run ./cmd/rotation-verify --file artifact.json
```
JSON output:
```
go run ./cmd/rotation-verify --file artifact.json --json
```
With explicit public key (overrides/augments embedded keys):
```
go run ./cmd/rotation-verify --file artifact.json --pub signer1:ED25519:BASE64URLPUB --json
```

## Adding New Algorithms
To support another algorithm:
1. Extend `parsePubSpec` in `main.go` to decode and validate the public key.
2. Add signature attachment & verification logic to the notary package (`internal/notary`).
3. Populate signer entries with `Alg` matching the new algorithm.

## Troubleshooting
- `file did not contain recognizable artifact`: Ensure JSON matches one of the accepted shapes.
- `digest mismatch`: Occurs when `--expect-digest` is set and differs from artifact's `canonical_digest`.
- `public_key_not_found` failures: Provide `--pub` keys or ensure server emits embedded public keys during signing phase.

## Security Notes
This tool performs in-memory verification only. For production-grade use:
- Enforce allowed algorithms.
- Validate timestamps and chain continuity (`previous_artifact_hash`).
- Optionally pin active key set IDs to expected values.

---
Generated documentation to clarify behavior and test assumptions.
