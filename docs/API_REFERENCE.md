---
title: AgentAuth RFC API Reference
 category: api-reference
 status: active
 lastUpdated: 2025-11-12
 owners: architecture-team
 refreshCadence: on-change
 source: source-code
 ---
## 🌲 Revocation Transparency Endpoint Examples (Beta)
> For internal model details (Merkle accumulator, Signed Tree Heads v1/v2, multi-sig threshold & weights, persistence, consistency proofs) consult `REVOCATION_TRANSPARENCY.md`.
### Inclusion Proof
`GET /api/v1/token/revocation/proof` (exactly one of `id`, `index`, or `hash` required)
Example:
```
curl -s "http://localhost:8080/api/v1/token/revocation/proof?hash=4f8c..." | jq
```
Success response:
```json
{
    "root": "b6f1c9e3dc...",
    "leaf_hash": "4f8c2e91a5...",
    "leaf_digest": "0f9a7c...",
    "index": 41,
    "chain_length": 42,
    "proof": [
    {"sibling": "ab12...", "position": "R"},
    {"sibling": "77fe...", "position": "L"}
    ],
    "verified": true
}
```
Error (missing query parameter):
```json
{
    "error": "missing_query_param",
    "message": "one of id, index, hash must be provided"
}
```
### Consistency Proof (Prototype)
`GET /api/v1/token/revocation/consistency?start=<start_index>`
Example:
```
curl -s "http://localhost:8080/api/v1/token/revocation/consistency?start=0" | jq
```
Success response:
```json
{
    "start_length": 5,
    "end_length": 42,
    "start_root": "91ab...",
    "end_root": "b6f1c9...",
    "new_leaves": ["4f8c2e91a5...", "8aa7bc11e3..."],
    "verified": true
}
```

### Revocation Consistency Proof V2 (Logarithmic)
Endpoint: `GET /api/v1/token/revocation/consistency_v2?start=<tree_head_index>`

Returns an RFC6962-style (prototype) JSON object:

```
{
    "success": true,
    "proof": {
        "start_length": <int>,
        "end_length": <int>,
        "path": ["<hexNode>", ...],
        "start_root": "<hex>",
        "end_root": "<hex>"
    },
    "latest_tree_head": { ... SignedTreeHead ... }
}
```

Characteristics:
- Path size is logarithmic relative to end_length.
- Demonstrates append-only growth from `start_length` to `end_length`.
- Provides optional prefix subtree decomposition (`prefix_roots`, `prefix_sizes`) representing maximal power-of-two aligned blocks covering the first `start_length` leaves (binary decomposition). Sum(prefix_sizes) == start_length.
- If the revocation chain is currently empty (no events appended), `latest_tree_head` MAY be `null` (or omitted) because `SignTreeHead()` now returns `nil` for empty chains (see "Empty Chain Behavior" in `REVOCATION_TRANSPARENCY.md`). Clients must handle this case gracefully.
- Current verification (`VerifyConsistencyProofV2`) still rebuilds the start tree for canonical root validation, but additionally validates each prefix root matches the corresponding subtree. Experimental fast reconstruction (feature flag `GAUTH_CONSISTENCY_V2_FAST=1`) currently succeeds only when `start_length` is a power of two (single prefix block); multi-block reconstruction is deferred.

Additional Fields (Optimization Phase):
| Field | Description |
|-------|-------------|
| `prefix_bridges` | Sequence of intermediate merged subtree digests produced by right-to-left reduction over the prefix blocks; enables fast deterministic reconstruction of `start_root` without full tree rebuild when fast flag enabled. |

Generation Algorithm (Optimized):
1. Streaming Segment Stack: Leaves up to `start_length` are ingested one by one, forming a stack of `(hash,size)` segments. Adjacent equal-sized segments merge immediately (size doubles) yielding a canonical power-of-two decomposition of the processed prefix.
2. Prefix Blocks: The final stack embodies `prefix_roots`/`prefix_sizes` (left → right). This avoids constructing all tree levels (`O(n)` memory) while retaining `O(n)` leaf hashing cost.
3. Bridges Construction: A right-to-left reduction merges the last two segments repeatedly (with consolidation when equal sizes arise) producing `prefix_bridges`. These bridges mirror the exact merge order of the canonical tree formation for heterogeneous block boundaries.
4. Consistency Path: (Current phase) A temporary Merkle tree is still built to derive the logarithmic path between historical and latest tree heads. Future phase will replace this with pure streaming interval math eliminating remaining `O(n)` generation overhead.

Fast Reconstruction:
`ReconstructStartRootFromPrefixBlocks(prefix_roots, prefix_sizes, start_length, prefix_bridges)` validates invariants and performs a reduction using bridges to obtain `start_root` in `O(k)` where `k = len(prefix_roots) + len(prefix_bridges)` (typically `< 2*log2(start_length)`). If the feature flag `GAUTH_CONSISTENCY_V2_FAST=1` is set, verification attempts this reconstruction and compares the result against the canonical rebuilt root; any mismatch causes verification failure.

Auditor Endpoint Enhancement:
`GET /api/v1/token/revocation/audit_consistency?start=<tree_head_index>` returns both legacy (full rebuild) and fast reconstruction timings:
```jsonc
{
    "success": true,
    "proof": { ... },
    "legacy_duration_ns": 1203487,
    "fast_duration_ns": 19834,
    "fast_root_matches": true
}
```
Use this to externally validate performance characteristics and detect any divergence between approaches.

Benchmark Snapshot (Generation Phase):
| Leaves | New Gen (ns/op) | Legacy Gen (ns/op) | Alloc (new) | Alloc (legacy) |
|--------|-----------------|--------------------|-------------|----------------|
| 64     | ~32.5µs         | ~51.8µs            | 73KB        | 118KB          |
| 256    | ~129.7µs        | ~203.7µs           | 294KB       | 472KB          |
| 1024   | ~530.8µs        | ~837.3µs           | 1.20MB      | 1.93MB         |
| 4096   | ~2.22ms         | ~3.55ms            | 4.77MB      | 7.70MB         |
(*Apple M3 Pro, Go 1.22+, benchtime 0.2s; values approximate; see `consistency_generation_bench_test.go` for details.)

Client Guidance (Updated):
1. Validate prefix decomposition (`prefix_roots`/`prefix_sizes`) and optionally `prefix_bridges` if using fast path.
2. Attempt fast reconstruction when flag enabled; fall back to canonical full rebuild if disabled or reconstruction returns empty string.
3. Verify append-only via path as before; ensure `path` length within logarithmic bound.
4. Record auditor metrics for monitoring (e.g., anomaly detection if fast_root mismatch appears).

Limitations (Updated):
- Consistency path still incurs an `O(n)` temporary tree build (optimization forthcoming).
- Range hashing and interval-based sibling derivation are deferred to next phase to eliminate remaining rebuild cost.
- Prefix bridges rely on deterministic merge ordering; any alteration in merge policy requires bridge sequence regeneration logic update.

Client Guidance:
1. Verify `start_root` and `end_root` against locally reconstructed Merkle trees (or incremental state).
2. Ensure `path` length is within allowed logarithmic bounds.
3. (Future) Reconstruct start root using path-only algorithm to avoid full rebuild.

Use Cases:
- Efficient audit of revocation chain evolution.
- Potential integration with external transparency monitors anchoring only periodic tree heads.

Limitations (Prototype):
- Path-only reconstruction using prefix blocks is not yet active (meta-tree algorithm deferred for correctness); integrity currently relies on full start & end tree recomputation plus prefix subtree matching.
- Merkle tree uses copy-up promotion for odd nodes; path algorithm adapted accordingly.
### 🪪 Power of Attorney Semantic Validation Modes

The service supports selectable semantic validation tiers for delegation issuance controlled by `GAUTH_POA_VALIDATOR`:

| Mode | Env Value | Characteristics |
|------|-----------|-----------------|
| Basic (default) | `basic` or unset | Backwards-compatible minimal invariants: prevent non-wildcard self-delegation, enforce `valid_from < valid_until`, require `jurisdiction` for `regulatory:` scopes, placeholder joint delegation rule (`joint:` requires `signatures` ≥2), numeric format sanity (`max_amount`, `max_daily_amount`, `currency` if present). Wildcard allowed for self-delegation only. |
| Advanced | `advanced` | Superset governance rules: mandatory `currency` restriction and 30‑day duration cap for any `transaction:` scope, wildcard scope disabled unless `GAUTH_ALLOW_WILDCARD=1`, syntax validation for `valid_hours` (`HH-HH`, wrap-around allowed) and `valid_weekdays` (comma-separated 0–6 unique), optional inline multi-signature restriction unless `GAUTH_ALLOW_INLINE_MULTISIG=1`, aggregate scope length limiting via `GAUTH_MAX_SCOPE_AGG_LEN`, runtime enforcement of `valid_hours` + `valid_weekdays` during token verification. |
| None | `none` | Disables semantic validation (not recommended for production). Only basic structural field checks apply. |

Additional related environment variables:

| Env | Purpose |
|-----|---------|
| `GAUTH_ALLOW_WILDCARD` | Set to `1` to permit `*` scope under Advanced mode. |
| `GAUTH_ALLOW_INLINE_MULTISIG` | Set to `1` to allow providing `MultiSignatures` inline at issuance when `threshold > 1`. |
| `GAUTH_MAX_SCOPE_AGG_LEN` | Integer limit for aggregate length of all scope entries (Advanced). |
| `GAUTH_POA_VALIDATOR` | Selects validator mode (`advanced`, `basic`, `none`). |

Restriction Keys (current set): `currency`, `jurisdiction`, `signatures`, `max_amount`, `max_daily_amount`, `valid_hours`, `valid_weekdays`. Future enhancements will add a warning channel and persistent counters for daily limit usage.

Runtime Conditional Enforcement:
`valid_hours` and `valid_weekdays` are syntactically validated at issuance in Advanced mode and enforced during `VerifyToken`. Tokens presented outside permitted windows return an `unauthorized` RFC error (`outside permitted hours` or `outside permitted weekdays`). Wrap-around hour windows (e.g., `22-06`) are supported.

### � Full PoA Embedding (RawPOA / PoAVersion)

EnvelopeV2 supports optional embedding of the full canonical PoA JSON for deterministic replay, external audit, and interoperability.

Environment Variables:
| Env | Purpose | Default |
|-----|---------|---------|
| `GAUTH_POA_ENVELOPE_V2` | Enables EnvelopeV2 issuance (adds canonical digest & multi-sig metadata). | unset (V1) |
| `GAUTH_EMBED_FULL_POA` | When `1`, embeds canonical PoA JSON into `RawPOA` and sets `PoAVersion` (`poa/v1`). | unset (disabled) |
| `GAUTH_MAX_RAW_POA_BYTES` | Max allowed canonical JSON length for embedding; skip & count if exceeded. | 8192 |

Behavior:
1. Canonical JSON produced once via `CanonicalPOADigest` and reused for `RawPOA` (ordering preserved: sorted scope & restriction keys; RFC3339 UTC timestamps).
2. Disabled flags or oversize payload ⇒ `RawPOA` omitted (token still valid) maintaining backward compatibility.
3. Size violations increment omission counter (`envelope_raw_poa_too_large_total`).
4. `PoAVersion` fixed to `poa/v1` (future semantic evolution may introduce `poa/v2`).

Metrics:
| Metric | Type | Description |
|--------|------|-------------|
| `envelope_raw_poa_embedded_total` | Counter | Successful embeddings within size cap. |
| `envelope_raw_poa_too_large_total` | Counter | Embedding skipped due to size limit exceeded. |

Operational Guidance:
* Track adoption ratio with custom recording rule: embedded_total / envelope_v2_issued_total.
* Alert if omission counter spikes (may indicate oversized PoAs or cap tuning need).

Security Notes:
* Always verify `canonical_digest` against recomputed canonical JSON before trusting `RawPOA` content.
* Embedding increases token size; keep cap conservative and benchmark before raising.

Future Roadmap:
* CBOR compact encoding (`GAUTH_EMBED_FULL_POA_FORMAT=cbor`).
* Verification helper exposing RawPOA directly (avoid manual envelope parsing).
* Streaming / chunking strategy for very large PoAs.
* Warning channel & audit persistence of embedded PoA snapshots.


### �🔏 Signed Tree Head (Multi-Sig) & Key Rotation Endpoints

Multi-signature protection provides stronger assurance over append-only revocation transparency by requiring a configured cumulative weight threshold across distinct Ed25519 signing keys.

#### Core Endpoints
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/crypto/eddsa/keys` | GET | List active & historical Ed25519 public keys (base64url raw 32-byte) with `kid`, optional expiry metadata. |
| `/api/v1/crypto/rotate` | POST | Force a key rotation (development / testing). Automatically signs latest Merkle tree head post-rotation. |
| `/api/v1/crypto/ensure-threshold` | POST | Iteratively rotates until satisfied weight ≥ configured threshold (bounded attempts). Useful in demos to auto-populate multi-sig state. |
| `/api/v1/token/revocation/head` | GET | Latest signed tree head (STH) with multi-sig signature array, threshold data and aggregate hash. |

#### Signed Tree Head JSON Structure
Example (abridged):
```json
{
    "version": 2,
    "merkle_root": "b6f1c9e3dc...",
    "chain_length": 42,
    "aggregate_hash": "e1f2...",             // hash chaining revocation events
    "threshold": 5,                           // required cumulative weight
    "weights_total": 7,                       // sum of all active key weights
    "satisfied_weight": 5,                    // sum of weights of included signatures
    "signatures": [
        {"kid":"ed25519:ab12","alg":"EdDSA","weight":3,"sig":"base64urlSignature1"},
        {"kid":"ed25519:cd34","alg":"EdDSA","weight":2,"sig":"base64urlSignature2"}
    ]
}
```
Fields:
| Field | Description |
|-------|-------------|
| `version` | Tree head schema version. |
| `merkle_root` | Current Merkle root over all revocation event hashes. |
| `chain_length` | Number of revocation events appended. |
| `aggregate_hash` | Linear linkage accumulator hash (integrity chain). |
| `threshold` | Required minimum cumulative weight for multi-sig validity. |
| `weights_total` | Sum of all active signer weights. |
| `satisfied_weight` | Weight achieved by present signatures. |
| `signatures[].kid` | Key identifier of signer. |
| `signatures[].weight` | Assigned integer weight for signer. |
| `signatures[].sig` | Base64url Ed25519 detached signature over canonical STH payload. |
| `signatures[].alg` | Algorithm (currently `EdDSA`). |

#### Canonical Signing Payload
```
GAUTH_TREE_HEAD:{"version":<int>,"merkle_root":"<hex>","chain_length":<int>,"aggregate_hash":"<hex>","threshold":<int>,"weights_total":<int>,"satisfied_weight":<int>}
```
(Order fixed; no whitespace; numeric values unquoted.)

#### Client Verification Flow (Multi-Sig Panel)
1. Fetch latest STH (`/api/v1/token/revocation/head`).
2. Fetch public keys (`/api/v1/crypto/eddsa/keys`).
3. Reconstruct canonical payload (above) and obtain bytes via UTF-8.
4. Attempt WebCrypto Ed25519 verification (`subtle.verify`).
5. If unsupported, fallback to tweetnacl-based verification via `verifyEd25519` (constant-time-ish detached signature verify). The fallback is injected at runtime and can be overridden via `setEd25519Impl` for testing.
6. Aggregate per-signature results and compute cumulative satisfied weight; display threshold status badge.

Verification Badges:
| Badge | Meaning |
|-------|---------|
| `Threshold Met` (green) | `satisfied_weight >= threshold` |
| `Partial` (amber) | Not enough cumulative weight yet |
| `Verified X/Y` (green) | All Y signatures verified (X = Y) |
| `Verified X/Y` (red) | At least one signature failed verification |

Per-Signature Status Cell:
| Value | Meaning |
|-------|---------|
| `ok` | Signature verified (WebCrypto or fallback) |
| `fail` | Verification failed (error tooltip with reason) |
| `n/a` | Missing verification attempt (no signature present) |

Error Codes (tooltips):
| Code | Cause |
|------|-------|
| `missing_public_key` | Key map lacked entry for signer `kid` |
| `webcrypto_unsupported` | Browser lacks Ed25519 WebCrypto implementation |
| `verify_failed` | WebCrypto verification returned false |
| `fallback_unverified` | Fallback verify returned false |
| `exception` | Exception thrown during processing |

#### Rotation Semantics
Each successful rotation (development endpoint) triggers automatic signing of the new tree head. This accelerates demo environments by quickly accumulating multi-sig signatures after multiple rotations. Production rotation mechanism would persist prior keys and follow stricter authorization & dual-signature protocols.

#### Environment Variables (Multi-Sig)
| Env | Purpose |
|-----|---------|
| `GAUTH_MSIG_THRESHOLD` | Numeric threshold required for multi-sig validity. |
| `GAUTH_MSIG_WEIGHTS` | Comma-separated weights for keys in rotation order (e.g. `3,2,1`). |
| `GAUTH_MSIG_AUTO_SIGN=1` | Auto-sign tree head after any rotation. |
| `GAUTH_EDDSA_ROTATION_INTERVAL_SEC` | Background rotation interval (if scheduler enabled). |
| `GAUTH_EDDSA_MAX_HISTORY` | Max historical keys retained/exposed via keys endpoint. |

#### Security & UX Notes
* Fallback verification is only used when native Ed25519 unavailable; it relies on audited tweetnacl implementation (public domain). Avoid downgrading silently—panel exposes `webcrypto_unsupported` reason.
* Threshold satisfaction shown independently from signature cryptographic verification; a satisfied threshold with failed signature(s) should be treated as untrusted.
* Canonical payload versioning ensures forward-compatible extensions (additional fields must be added only after version increment).
* For accessibility, badges include `aria-label` and deterministic color contrast; future enhancement will add live region updates.

Roadmap (Multi-Sig):
* Dual-signature rotation descriptors anchoring both retiring and successor keys.
* External auditor endpoint returning historical STH sequence + batch verification summary.
* Path-only consistency proof derivation eliminating full tree rebuild in verification.
* Optional signature aggregation (MLS-style) to compress multiple Ed25519 signatures.

### Discovery STH Excerpt
`GET /.well-known/gauth-configuration`
```json
{
    "revocation_support": {
    "enabled": true,
    "sth_latest": {
    "version": 2,
    "merkle_root": "b6f1c9e3dc...",
    "chain_length": 42,
    "aggregate_hash": "e1f2...",
    "threshold": 5,
    "weights_total": 7,
    "satisfied_weight": 5,
    "signatures": [
    {"kid": "signerA", "alg": "EdDSA"},
    {"kid": "signerC", "alg": "EdDSA"}
    ]
    }
    }
}
```
Clients must fetch JWKS (`/.well-known/jwks.json`) to obtain Ed25519 public keys for each `kid` and verify multi-signature threshold satisfaction.

### 🔐 JWKS Endpoint

`GET /.well-known/jwks.json`

Returns a JSON Web Key Set containing:
- RSA key (RS256) when `GAUTH_USE_JWT_LIB=1` and `GAUTH_JWT_ALG=RS256`
- Symmetric placeholder (oct) when only HMAC mode active
- Ed25519 public keys as OKP JWK objects when `GAUTH_TOKEN_SIG_MODE=eddsa`

Example response (EdDSA + RS256 hybrid):
```json
{
    "keys": [
        {"kty":"RSA","alg":"RS256","kid":"demo-rsa","use":"sig","n":"...","e":"AQAB"},
        {"kty":"OKP","crv":"Ed25519","alg":"EdDSA","kid":"AbCdEf12","use":"sig","x":"base64urlPublicKey","expires_at":"2025-10-19T00:00:00Z"}
    ]
}
```
Headers:
- `ETag`: Weak hash of canonical JWKS JSON for client caching (supports `If-None-Match` → `304`).
- `X-JWKS-Signature` + `X-JWKS-Signature-Alg`: Present only when `GAUTH_JWKS_SIGNING_KEY_ENABLED=1` (HMAC-SHA256 integrity hint).
- `X-Key-Rotation-Interval-Days`: Optional rotation interval metadata if provided via env.

Rotation Metadata surfaced in discovery (`/.well-known/gauth-configuration`):
- `jwks_etag`: Mirrors current JWKS ETag once fetched at least once.
- `jwks_last_rotated`: Timestamp updated whenever ETag changes (new key material).

### 🧾 Notarization Receipt Chain Verification (Prototype)

`GET /api/v1/beta/notarization/receipts/verify`

Verifies integrity of the persisted capability anchor notarization receipt chain.

Response (configured, non-empty, integrity ok):
```json
{
    "success": true,
    "configured": true,
    "integrity": "ok",
    "total": 3,
    "chain_head": "b2f4..."
}
```

Response (mismatch detected):
```json
{
    "success": true,
    "configured": true,
    "integrity": "mismatch",
    "total": 3,
    "details": {
        "mismatch_index": 0,
        "expected": "0a9c...",      // recomputed chain hash for entry
        "stored": "deadbeef...",     // stored chain_hash field
        "prev_expected": "",         // previous linkage hash expected
        "prev_stored": ""            // stored prev_hash for entry
    }
}
```

Response (empty chain):
```json
{
    "success": true,
    "configured": true,
    "integrity": "empty",
    "total": 0
}
```

Fields:
| Field | Description |
|-------|-------------|
| `integrity` | `ok` | `mismatch` | `empty` |
| `chain_head` | Last entry `chain_hash` (present only when integrity ok) |
| `mismatch_index` | Zero-based index of first invalid entry (only on mismatch) |
| `expected` / `stored` | Recomputed vs stored chain hash for failing entry |
| `prev_expected` / `prev_stored` | Linkage validation for the failing entry's predecessor |

Related Endpoints:
* `GET /api/v1/beta/notarization/receipts/latest` – latest stored receipt object.
* `GET /api/v1/beta/notarization/receipts` – list of chain entries with linkage fields.

Metrics Integration:
* Gauge: `capability_anchor_notarization_receipts_integrity` (ok=1 mismatch=0 unconfigured/legacy/empty=-1)
* Histogram: `capability_anchor_notarization_latency_seconds` (+ provider labeled variant)
* Counter: `capability_anchor_notarization_failures_total` (+ provider labeled variant)
 * Custom Prometheus Endpoint: `/api/v1/beta/capabilities/anchor/metrics/prometheus` mirrors the integrity gauge (HELP/TYPE + value) for light scrapes. Force a fresh recomputation with `?verify=1`.

Environment Variables:
| Env | Purpose |
|-----|---------|
| `GAUTH_CAP_ANCHOR_NOTARIZE=1` | Enables notarization+receipt path |
| `GAUTH_NOTARY_RECEIPT_PERSIST_PATH` | Receipt chain persistence file |
| `GAUTH_NOTARY_RECEIPT_VERIFY_INTERVAL` | Background verify loop interval seconds (>=30) |
| `GAUTH_CAP_ANCHOR_NOTARY_PROVIDER` | Provider selector (`memory`,`external_stub`) |
 | `GAUTH_NOTARY_RECEIPT_VERIFY_FRESHNESS_SECONDS` | Max age (seconds) before auto re-verification in custom Prometheus endpoint (default 120) |

Additional Environment Variables (Recent Additions):
| Env | Purpose |
|-----|---------|
| `GAUTH_NOTARY_VERIFY_MIN_INTERVAL_SECONDS` | Adaptive verification lower bound (seconds) controlling minimum recomputation cadence |
| `GAUTH_NOTARY_VERIFY_MAX_INTERVAL_SECONDS` | Adaptive verification upper bound; interval expands toward this under low append/mismatch rates |
| `GAUTH_NOTARY_MERKLE_ENABLED=1` | Enable Merkle root computation per receipt (field `merkle_root` appears) |
| `GAUTH_TSA_STUB_MIN_LATENCY_MS` / `GAUTH_TSA_STUB_MAX_LATENCY_MS` | Configure stub TSA latency simulation window |
| `GAUTH_TSA_STUB_PROVIDER_NAME` | Override provider name for stub TSA receipts |
| `GAUTH_TSA_STUB_POLICY_OID` | Policy OID placeholder for RFC3161 scaffold |
| `GAUTH_TLOG_STUB_MIN_LATENCY_MS` / `GAUTH_TLOG_STUB_MAX_LATENCY_MS` | Configure stub transparency log latency simulation window |
| `GAUTH_TLOG_STUB_PROVIDER_NAME` | Override provider name for transparency log stub |

### 🌐 External Capability Anchoring (Prototype)

External anchoring publishes the canonical capability registry hash to an external timestamping / transparency provider distinct from the internal memory anchor client and notarization path.

Environment Variables:
| Env | Purpose |
|-----|---------|
| `GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER` | Select external provider (`memory` in-process demo or `tsa_stub` latency/failure simulation). |
| `GAUTH_CAP_EXTERNAL_ANCHOR_MIN_MS` | Minimum simulated latency (ms) for `tsa_stub` provider (default 25). |
| `GAUTH_CAP_EXTERNAL_ANCHOR_MAX_MS` | Maximum simulated latency (ms) for `tsa_stub` provider (default 120). |
| `GAUTH_CAP_EXTERNAL_ANCHOR_FAIL_PROB` | Failure probability (0.0–1.0) for `tsa_stub` to exercise retry/error paths (default 0). |
| `GAUTH_CAP_EXTERNAL_ANCHOR_RETRIES` | Number of retry attempts after the initial attempt (exponential backoff). Default 0 (no retries). |
| `GAUTH_CAP_EXTERNAL_ANCHOR_RETRY_BASE_MS` | Base backoff (ms) before first retry (default 50). Each retry doubles this base (2^n) and applies ±20% jitter. |

Receipt Object (provider-specific, memory & stub share shape):
```json
{
    "hash": "sha256:...",
    "timestamp": "2025-10-20T18:59:12.345678901Z",
    "provider": "tsa_stub",
    "version": 1,
    "proof": "..."
}
```

Provider Label Normalization:
The metrics label for the TSA stub provider uses a dash (`tsa-stub`) while the environment selector uses an underscore (`GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER=tsa_stub`). Internally the provider implementation emits `provider:"tsa-stub"` in receipts; the metrics helper normalizes the env value `tsa_stub` → `tsa-stub` to ensure consistent labeled counters/histograms. Memory provider passes through unchanged (`memory`). Custom providers should prefer dash-separated labels for Prometheus cardinality hygiene.

Endpoints:
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/beta/capabilities/anchor/status` | GET | Includes `external_anchor_receipt` when at least one external anchor succeeded. |
| `/api/v1/beta/capabilities/anchor/verify` | POST | Verifies a posted receipt object (hash/timestamp/provider) against provider logic. |
| `/api/v1/beta/capabilities/anchor/external/receipts/latest` | GET | Latest persisted external anchor receipt (with `prev_hash` and `chain_hash`). |
| `/api/v1/beta/capabilities/anchor/external/receipts` | GET | Full external anchor receipt chain (append-only order). |
| `/api/v1/beta/capabilities/anchor/external/receipts/verify` | GET | External receipt chain integrity verification (reports ok | mismatch | empty plus diagnostics). |

Metrics Adapter Injection:
To ensure the initial external anchoring attempt (performed during server startup when a capability registry hash is already available) records into a custom metrics backend, use the new constructor:

```go
reg := prometheus.NewRegistry()
pm  := metrics.NewPrometheusMetrics(metrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem: "rfc0111", Registry: reg})
srv := web.NewBetaServerWithMetrics(":0", pm)
```

This avoids the earlier pattern of constructing with `NewBetaServer` then replacing `srv.metrics` (which missed startup side-effects such as the initial external anchor attempt). Tests targeting strict Prometheus assertions should prefer `NewBetaServerWithMetrics`.
| `/api/v1/beta/capabilities/anchor/external/receipt` | GET | Latest external anchor receipt only (configured provider). |
| `/api/v1/beta/capabilities/anchor/external/verify` | GET | Performs provider-level verification of latest receipt (`verified:true|false`). |

Metrics (Prometheus Adapter):
| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `external_anchor_attempts_total` | Counter | `provider` | Total external anchoring attempts initiated. |
| `external_anchor_failures_total` | Counter | `provider` | Failed anchoring attempts. |
| `external_anchor_latency_seconds` | Histogram | `provider` | Latency distribution of successful external anchoring operations. |
| `external_anchor_age_seconds` | Gauge | none | Seconds since last successful external anchor (0 when never). |
| `external_anchor_last_hash_len` | Gauge | none | Length of last externally anchored hash (0 when none). |
| `capability_external_anchor_receipts_integrity` | Gauge | none | External receipt chain integrity (1=ok,0=mismatch,-1=unconfigured/empty). |
| `capability_external_anchor_receipts_last_verify_age_seconds` | Gauge | none | Seconds since last external receipt chain integrity verification. |
| `capability_external_anchor_receipts_total` | Counter | none | Total persisted external anchor receipt entries. |

Startup Behavior:
An initial anchoring attempt is performed immediately during server initialization (if a capability registry hash already exists) to ensure observability for static demo seeds.

Failure Simulation:
Retry Behavior:
If `GAUTH_CAP_EXTERNAL_ANCHOR_RETRIES` > 0, the server performs additional attempts upon failure using exponential backoff: `delay = base_ms * 2^attempt` with a ±20% jitter adjustment. Each attempt (initial + retries) records its own metrics (attempt counter increment; failures or latency histogram sample). Successful attempts short-circuit remaining retries. The hash length gauge is updated only on success. Provider-labeled counters reflect total attempts and failures across all retries.
Set `GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER=tsa_stub` with `GAUTH_CAP_EXTERNAL_ANCHOR_FAIL_PROB=1` to force failures; status payload will omit `external_anchor_receipt` and failure counter increments.

Verification Semantics:
`/external/verify` invokes provider `Verify`; memory provider always succeeds when hash matches, while stub may return errors if simulated state diverges.

Observability Guidance:
1. Scrape `/metrics` for labeled counters/histogram buckets.
2. Monitor `external_anchor_age_seconds` for freshness.
3. Alert on high failure ratio; consider provider fallback.

Roadmap:
* Add real TSA / transparency log provider implementation (RFC3161, RFC6962).
* (Completed) Persist external receipts with hash-chain linkage similar to notarization chain (see new `/external/receipts` endpoints & integrity metrics).
* Integrate multi-provider quorum anchoring.
* Export proof verification / inclusion path endpoints when transparency log present.

Receipt Object Extensions:
Unified Recording Helper:
When instrumenting a single external anchoring operation manually (e.g. custom provider integration code outside the server), prefer the consolidated helper to ensure unlabeled and provider-labeled metrics remain synchronized:

```go
// success path
pm.RecordExternalAnchorResult(providerTag, true, latency, len(hashHex))
// failure path
pm.RecordExternalAnchorResult(providerTag, false, 0, 0)
```
Behavior:
* Always increments attempts counters (unlabeled + provider).
* On success: observes latency (both histograms) and sets last hash length gauge.
* On failure: increments failure counters (unlabeled + provider).
* Does NOT set age gauge directly (background server loop handles freshness).

Backward compatibility: existing `IncExternalAnchorAttempts`, `IncExternalAnchorFailures`, `ObserveExternalAnchorLatency`, and `SetExternalAnchorLastHashLen` remain available; the server now prefers the unified helper when present.

The persisted receipt entries (via `/api/v1/beta/notarization/receipts` and latest endpoint) may now include optional fields:
| Field | Description |
|-------|-------------|
| `merkle_root` | Present when Merkle feature enabled; binary Merkle tree root over all leaf entry digests up to this entry. Leaves = sha256 of base JSON (excluding `chain_hash`, `merkle_root`). |
| `rotation` | Key rotation descriptor if the receipt represents a rotation event (continuity metadata). |

Key Rotation Descriptor (`rotation`):
```jsonc
"rotation": {
    "old_key_id": "ed25519:abcd...",
    "new_key_id": "ed25519:ef01...",
    "effective_time": "2025-10-20T12:00:00Z",
    "reason": "scheduled",
    "prev_rotation_hash": "<chain_hash_of_previous_rotation_receipt>"
}
```
Purpose: preserve continuity and audit trail of signing keys; future enhancement will require dual signatures (old + new key) over descriptor.
### 🔄 Key Rotation Verification (Prototype)

`GET /api/v1/beta/rotations/verification`

Audits continuity (`prev_rotation_hash` linkage) and dual-signature validity of recorded key rotation descriptors embedded in notarization receipts.

Example Response (abridged):
```jsonc
{
    "success": true,
    "configured": true,
    "generated_at": "2025-10-20T12:34:56Z",
    "summary": {
        "total": 2,
        "all_continuity_ok": true,
        "all_signatures_ok": false,
        "failures": 1,
        "results": [
            {"index":0,"old_key_id":"ed25519:aa12bb34","new_key_id":"ed25519:cc56dd78","continuity_ok":true,"signatures_ok":true},
            {"index":1,"old_key_id":"ed25519:cc56dd78","new_key_id":"ed25519:ee90ff11","continuity_ok":true,"signatures_ok":false,"reason":"missing_old_signature"}
        ]
    }
}
```

Per-Descriptor Fields:
| Field | Description |
|-------|-------------|
| `index` | Sequence position (0-based) |
| `old_key_id` / `new_key_id` | Key identifiers bound to descriptor |
| `continuity_ok` | Whether `prev_rotation_hash` matches previous rotation receipt hash (empty allowed for first) |
| `signatures_ok` | Both old & new key signatures present and verified |
| `reason` | First failure reason (present only on failure) |
| `prev_hash_expected` / `prev_hash_actual` | Continuity diagnostics when prior hash known |

Aggregate Fields:
| Field | Description |
|-------|-------------|
| `total` | Number of descriptors evaluated |
| `all_continuity_ok` | No continuity failures encountered |
| `all_signatures_ok` | All dual signatures validated |
| `failures` | Count of descriptors with any failure |

Failure Reasons:
| Reason | Meaning |
|--------|---------|
| `descriptor_nil` | Nil descriptor entry |
| `continuity_failure` | Previous linkage mismatch or unexpected non-empty genesis prev hash |
| `kid_not_found_old` | Old key public key material unavailable |
| `kid_not_found_new` | New key public key material unavailable |
| `kid_mismatch_old` | Old key ID mismatch vs derived ID |
| `kid_mismatch_new` | New key ID mismatch vs derived ID |
| `missing_old_signature` | Old key signature field empty |
| `missing_new_signature` | New key signature field empty |
| `old_sig_invalid` | Old signature fails Ed25519 verify / decode |
| `new_sig_invalid` | New signature verification failure |
| `serialization_error` | Canonical payload marshal failed |

Metrics:
| Metric | Description |
|--------|-------------|
| `gauth_rotation_verification_latency_seconds` | Histogram of verification latency |
| `gauth_rotation_verification_total{outcome="success|failure"}` | Counter labeled by outcome (any failure => failure) |
| `gauth_rotation_verification_failure_reason_total{reason="<code>"}` | Counter per failing descriptor categorized by first failure reason |

Current Limitations:
* Public key resolution not yet wired—missing keys produce `kid_not_found_old/new`.
* Rotation descriptors must be embedded in receipts; external persistence forthcoming.
* No signed summary artifact yet (planned for hardening phase).

Roadmap:
* Persistent rotation receipt log with independent integrity chain.
* RFC3161/Sigstore timestamping of rotation chain heads.
* Inclusion proofs & Merkle subtree optimization for descriptor sets.


Stub Provider Interfaces:
- TSA Scaffold (`internal/notary/tsa_stub.go`): `Timestamp` method returns synthetic `TSAResponse` with `emulated_serial` and `emulated_policy_oid`.
- Transparency Log Scaffold (`internal/notary/transparency_log_stub.go`): `Log` method returns synthetic `TransparencyLogEntryResponse` with `emulated_leaf_id`, `emulated_root_ref`.

Adaptive Verification Notes:
- Background verification interval dynamically recalculates between min/max bounds based on append rate and mismatch occurrences.
- Force immediate recomputation via custom Prometheus endpoint `?verify=1` or when freshness threshold exceeded.
- Gauge `capability_anchor_notarization_receipts_last_verify_age_seconds` exposes age since last integrity verification; alert if near max interval * 2.

Merkle Root Verification (Auditor Guidance):
1. Fetch `/api/v1/beta/notarization/receipts`.
2. For each entry reconstruct leaf digest (sha256 over base JSON minus chain fields) maintaining ordered list.
3. Recompute Merkle root using pairwise hashing duplicating last node when level has odd count; compare to final entry's `merkle_root`.
4. Cross-check final `chain_hash` for chain linkage integrity.

Rotation Audit Flow (Future):
1. Identify all receipts with `rotation` present.
2. Verify linear continuity via `prev_rotation_hash` forming rotation chain.
3. Validate dual signatures over descriptor referencing both key IDs.

Dual-Signature Fields (Implemented):
| Field | Description |
|-------|-------------|
| `old_key_signature` | Ed25519 signature produced by retiring key over canonical, prefixed descriptor payload |
| `new_key_signature` | Ed25519 signature produced by successor key over same canonical payload |

Verification Failure Codes:
| Code | Meaning |
|------|---------|
| `kid_mismatch_old` | Provided old public key does not derive same key id |
| `kid_mismatch_new` | Provided new public key does not derive same key id |
| `missing_old_signature` | Old key signature absent |
| `missing_new_signature` | New key signature absent |
| `old_sig_invalid` | Old signature fails verification |
| `new_sig_invalid` | New signature fails verification |
| `serialization_error` | Canonical descriptor could not be serialized |

Canonical Signing Payload:
```
GAUTH_ROTATION_DESCRIPTOR:{"old_key_id":"...","new_key_id":"...","effective_time":"...","reason":"...","prev_rotation_hash":"..."}
```

### 📊 Snapshot Metrics & Integrity
Metrics exposed during snapshot operations (Prometheus):
| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `gauth_snapshot_generation_latency_seconds` | Histogram | - | Generation latency |
| `gauth_snapshot_generation_total` | Counter | outcome | Attempts (success|error) |
| `gauth_snapshot_verification_latency_seconds` | Histogram | - | Verification latency |
| `gauth_snapshot_verification_total` | Counter | outcome | Attempts (success|failure|error) |

Snapshot Verification Result Fields:
| Field | Meaning |
|-------|---------|
| `valid` | All comparisons succeeded (hash, merkle, chain head, count) |
| `hash_match` | Snapshot self-hash matches recomputed digest |
| `merkle_match` | Merkle root matches recomputed tree (or absent when disabled) |
| `chain_head_match` | Chain head hash equals current store head |
| `receipt_count_ok` | Receipt count equals current store length |
| `reason` | First failing condition (precedence: merkle>hash>chain_head>receipt_count) |

Exit Codes (CLI): reference README section for mapping (0 success; non-zero failure/error categories).


Alerting Recommendation:
Trigger alert when integrity gauge == 0 (mismatch) for 2 consecutive scrapes. Suppress alerts for value -1 (unconfigured/empty/legacy).

### 🕒 JTI TTL & Replay Protection

The token service enforces fail-closed replay detection on the `jti` claim. In-memory JTIs expire after a TTL window:
- Configure with `GAUTH_JTI_TTL_SEC` (default 600 seconds).
- On validation, a duplicate JTI within the TTL window ⇒ token rejected.
- Opportunistic eviction runs when the in-memory map exceeds 500 entries.
- Pluggable persistence layer can be injected via `WithReplayStore` to enforce durable uniqueness.

Edge Cases:
- Expired JTI entries (older than TTL) are removed lazily, allowing reuse (treated as a fresh token).
- Absent or empty `jti` ⇒ immediate rejection.

Security Considerations:
- For production deployments use a durable store (e.g., Redis) implementing `ReplayStore` to survive restarts and coordinate across replicas.
- TTL tuning balances memory footprint vs. replay window risk; choose a value aligned with average token lifetime.
---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
# (Metadata moved to top) AgentAuth RFC API Reference

> Last Updated: 2025-10-21
> Status: Active

> Beta reference – NOT production ready. Interfaces and examples may be incomplete or simplified. See `../DISCLAIMER.md` for intentionally missing security, policy, and lifecycle controls.

**Beta Demonstration API Documentation (NOT Production Ready)**

Complete Go library API reference for AAP-RFC-0111 (AgentAuth 1.0) and AAP-RFC-0115 (PoA Definition) implementation.

> **📝 Note**: This document covers the Go library API. For the web demonstration API documentation, see [COMPLETE_API_REFERENCE.md](./COMPLETE_API_REFERENCE.md) which includes both library and web API documentation.

## 📋 **Table of Contents**

1. [Core Service API](#core-service-api)
2. [AgentAuth-RFC-001 (formerly RFC 111) Authorization API](#rfc-111-authorization-api)
3. [AgentAuth-RFC-002 (formerly RFC 115) PoA Definition API](#rfc-115-poa-definition-api)
4. [Professional Foundation API](#professional-foundation-api)
5. [Data Types Reference](#data-types-reference)
6. [Error Handling](#error-handling)
7. [Examples](#examples)
 8. [Infrastructure & Resilience Utilities](#infrastructure--resilience-utilities)
    - [Circuit Breaker](#circuit-breaker)
    - [Rate Limiter (Simplified)](#rate-limiter-simplified)
    - [Metrics Collector](#metrics-collector)
    - [Authorization Policy Model Update](#authorization-policy-model-update)
    - [Audit Event API Change](#audit-event-api-change)
    - [Token Store Changes](#token-store-changes)

> **🌐 Web API**: For REST API endpoints used by the webapp demo, see [COMPLETE_API_REFERENCE.md](./COMPLETE_API_REFERENCE.md)

## 🏗️ **Core Service API**

### **RFCCompliantService**

The main service implementing both AgentAuth-RFC-001 (formerly RFC 111) and AgentAuth-RFC-002 (formerly RFC 115) specifications.

```go
type RFCCompliantService struct {
    jwtService         *ProperJWTService
    legalValidator     *LegalFrameworkValidator
    delegationManager  *DelegationManager
    attestationService *AttestationService
}
```

#### **Constructor**

```go
func NewRFCCompliantService(issuer, audience string) (*RFCCompliantService, error)
```

**Parameters:**
- `issuer` (string): The issuer identifier for JWT tokens
- `audience` (string): The intended audience for JWT tokens

**Returns:**
- `*RFCCompliantService`: Configured service instance
- `error`: Configuration or initialization error

**Example:**
```go
service, err := auth.NewRFCCompliantService("my-company", "ai-authorization")
### Signature & Semantic Validation Options (AAP-001 Service)

The lower-level `rfc0111.Service` (delegation issuance / token verification) now supports additional functional options:

```go
// Enforce signature presence (abort issuance if signing fails)
func WithMandatorySignatures() rfc0111.Option
// Provide a signing key (returned Signer must implement ed25519 or supported algorithm)
func WithSignerProvider(fn func() (cr.Signer, error)) rfc0111.Option
// Install semantic validator executing cross-field invariants before signing/persistence
func WithSemanticValidator(v rfc0111.PoAValidator) rfc0111.Option
// Enforce strict authenticity (missing public key at verification becomes integrity failure)
func WithStrictAuthenticity() rfc0111.Option
```

Canonical Digest & Signature:
```go
digestHex, canonicalJSON, err := rfc0111.CanonicalPOADigest(&poa)
// Signature covers domain-separated canonical JSON; exclusion of mutable fields preserves validity through lifecycle.
```

Validator Interface:
```go
type PoAValidator interface {
    Validate(*rfc0111.PowerOfAttorney) error // return rfc.ErrInvalidRequest on semantic violation
}

// Provided prototype implementation:
type BasicPoAValidator struct{}
```

Prototype Semantic Rules (BasicPoAValidator):
1. Disallow self-delegation unless wildcard-only scope.
2. Require `currency` restriction for any `transaction:` scope.
3. Enforce temporal ordering (`valid_from` < `valid_until`).
4. Cap `transaction:` delegation duration to 30 days.
5. Normalize nil restriction map.

Activation Example:
```go
svc := rfc0111.NewService(audit.NewMemoryLogger(nil), authz.NewMemoryAuthorizer(),
    rfc0111.WithSignerProvider(func() (cr.Signer, error) { kp, _ := cr.NewInMemoryEd25519Provider(); return kp.ActiveSigner() }),
    rfc0111.WithMandatorySignatures(),
    rfc0111.WithSemanticValidator(rfc0111.BasicPoAValidator{}),
)
```

Token Verification now sets `SignatureValid` and `PublicKeyFound` flags in `TokenVerificationResult` if signature present.

RawPOA Exposure:
`TokenVerificationResult` now includes `raw_poa` and `poa_version` fields when a token was issued as EnvelopeV2 with full PoA embedding enabled and within size limits. These fields are omitted when embedding disabled or size cap exceeded. Always verify `canonical_digest` independently before trusting embedded JSON. Future versions may add `raw_poa_cbor` for compact representation.

if err != nil {
    return fmt.Errorf("service creation failed: %w", err)
}
```

## 🎯 **AgentAuth-RFC-001 (formerly RFC 111) Authorization API**

### **AuthorizeAgentAuth**

Main authorization method implementing complete AgentAuth-RFC-001 (formerly RFC 111) flow with AgentAuth-RFC-002 (formerly RFC 115) PoA Definition validation.

```go
func (s *RFCCompliantService) AuthorizeAgentAuth(ctx context.Context, req AgentAuthRequest) (*AgentAuthResponse, error)
```

**Parameters:**
- `ctx` (context.Context): Request context for cancellation and timeout
- `req` (AgentAuthRequest): Complete RFC-compliant authorization request

**Returns:**
- `*AgentAuthResponse`: Authorization response with compliance validation
- `error`: Authorization or validation error

**Process Flow:**
1. Validates PoA Definition (AgentAuth-RFC-002 (formerly RFC 115))
2. Validates principal capacity (AgentAuth-RFC-001 (formerly RFC 111))
3. Validates AI client capabilities
4. Validates legal compliance
5. Generates authorization code
6. Creates comprehensive audit record

**Example:**
```go
response, err := service.AuthorizeAgentAuth(ctx, auth.AgentAuthRequest{
    ClientID:     "ai_agent_v1",
    ResponseType: "code",
    Scope:        []string{"financial_advisory"},
    PowerType:    "financial_advisory_powers",
    PrincipalID:  "corp_ceo_123",
    AIAgentID:    "ai_financial_advisor",
    Jurisdiction: "US",
    PoADefinition: poaDefinition, // Complete AgentAuth-RFC-002 (formerly RFC 115) structure
})
```

### **CreateAgentAuthToken** (Future Enhancement)

Exchange authorization code for extended token with comprehensive metadata.

```go
func (s *RFCCompliantService) CreateAgentAuthToken(ctx context.Context, authCode string) (*AgentAuthToken, error)
```

## 📋 **AgentAuth-RFC-002 (formerly RFC 115) PoA Definition API**

### **PoA Definition Structure**

Complete implementation of AgentAuth-RFC-002 (formerly RFC 115) Power-of-Attorney Credential Definition.

```go
type PoADefinition struct {
    Principal         Principal         `json:"principal"`          // Section 3.A
    Authorizer        Authorizer        `json:"authorizer"`         // Section 3.A
    Client           ClientAI          `json:"client"`             // Section 3.A
    AuthorizationType AuthorizationType `json:"authorization_type"` // Section 3.B
    ScopeDefinition   ScopeDefinition   `json:"scope_definition"`   // Section 3.B
    Requirements     Requirements      `json:"requirements"`       // Section 3.C
}
```

### **Section A: Parties**

#### **Principal**

```go
type Principal struct {
    Type         PrincipalType `json:"type"`                    // "individual" or "organization"
    Identity     string        `json:"identity"`                // Unique identifier
    Organization *Organization `json:"organization,omitempty"`  // Required if type is "organization"
}

type PrincipalType string
const (
    PrincipalTypeIndividual   PrincipalType = "individual"
    PrincipalTypeOrganization PrincipalType = "organization"
)
```

#### **Organization**

```go
type Organization struct {
    Type                OrganizationType `json:"type"`                 // Commercial, public, non-profit, etc.
    Name                string           `json:"name"`                 // Organization name
    RegisterEntry       string           `json:"register_entry"`       // Commercial register entry
    ManagingDirector    string           `json:"managing_director"`    // Current managing director
    RegisteredAuthority bool             `json:"registered_authority"` // Has registered authority
}

type OrganizationType string
const (
    OrgTypeCommercial   OrganizationType = "commercial_enterprise"    // AG, Ltd., partnership
    OrgTypePublic       OrganizationType = "public_authority"         // Federal, state, municipal
    OrgTypeNonProfit    OrganizationType = "non_profit_organization"  // Foundation, gGmbH
    OrgTypeAssociation  OrganizationType = "association"              // Non-profit or non-charitable
    OrgTypeOther        OrganizationType = "other"                    // Church, cooperative, etc.
)
```

#### **ClientAI**

```go
type ClientAI struct {
    Type              ClientType `json:"type"`               // LLM, agent, agentic AI, robot
    Identity          string     `json:"identity"`           // Unique AI identifier
    Version           string     `json:"version"`            // AI version
    OperationalStatus string     `json:"operational_status"` // "active", "revoked", etc.
}

type ClientType string
const (
    ClientTypeLLM       ClientType = "llm"            // Large Language Model
    ClientTypeAgent     ClientType = "digital_agent"  // Single digital agent
    ClientTypeAgenticAI ClientType = "agentic_ai"     // Team of agents
    ClientTypeRobot     ClientType = "humanoid_robot" // Physical humanoid robot
    ClientTypeOther     ClientType = "other"          // Other AI types
)
```

### **Section B: Authorization Type & Scope**

#### **AuthorizationType**

```go
type AuthorizationType struct {
    RepresentationType    RepresentationType `json:"representation_type"`     // Sole or joint
    RestrictionsExclusions []string          `json:"restrictions_exclusions"` // Specific exclusions
    SubProxyAuthority     bool              `json:"sub_proxy_authority"`     // Can delegate further
    SignatureType         SignatureType     `json:"signature_type"`          // Single, joint, collective
}

type RepresentationType string
const (
    RepresentationSole  RepresentationType = "sole"  // Sole representation
    RepresentationJoint RepresentationType = "joint" // Joint representation
)

type SignatureType string
const (
    SignatureSingle     SignatureType = "single"     // Single signature authority
    SignatureJoint      SignatureType = "joint"      // Joint signature required  
    SignatureCollective SignatureType = "collective" // Collective signature required
)
```

#### **ScopeDefinition**

```go
type ScopeDefinition struct {
    ApplicableSectors  []IndustrySector   `json:"applicable_sectors"`  // ISIC/NACE industry codes
    ApplicableRegions  []GeographicScope  `json:"applicable_regions"`  // Geographic constraints
    AuthorizedActions  AuthorizedActions  `json:"authorized_actions"`  // Permitted actions
}
```

#### **Industry Sectors (ISIC/NACE)**

```go
type IndustrySector string
const (
    SectorAgriculture     IndustrySector = "agriculture_forestry_fishing"
    SectorMining          IndustrySector = "mining_quarrying"
    SectorManufacturing   IndustrySector = "manufacturing"
    SectorEnergy          IndustrySector = "energy_supply"
    SectorWater           IndustrySector = "water_supply"
    SectorWaste           IndustrySector = "waste_management"
    SectorConstruction    IndustrySector = "construction"
    SectorTrade           IndustrySector = "trade"
    SectorTransport       IndustrySector = "transport_storage"
    SectorHospitality     IndustrySector = "hospitality"
    SectorICT             IndustrySector = "information_communication"
    SectorFinancial       IndustrySector = "financial_insurance"
    SectorRealEstate      IndustrySector = "real_estate"
    SectorProfessional    IndustrySector = "professional_scientific"
    SectorBusiness        IndustrySector = "business_services"
    SectorPublicAdmin     IndustrySector = "public_administration"
    SectorEducation       IndustrySector = "education"
    SectorHealth          IndustrySector = "health_social_work"
    SectorArts            IndustrySector = "arts_entertainment"
    SectorOtherServices   IndustrySector = "other_services"
)
```

#### **Geographic Scope**

```go
type GeographicScope struct {
    Type        GeographicType `json:"type"`                   // Global, national, regional, etc.
    Identifier  string         `json:"identifier"`             // Country code, region name, etc.
    Description string         `json:"description,omitempty"`  // Human-readable description
}

type GeographicType string
const (
    GeoTypeGlobal      GeographicType = "global"           // Global operations
    GeoTypeNational    GeographicType = "national"         // Specific country (specify identifier)
    GeoTypeRegional    GeographicType = "regional"         // DACH, Benelux, NAFTA, etc.
    GeoTypeSubnational GeographicType = "subnational"      // States, provinces, municipalities
    GeoTypeSpecific    GeographicType = "specific_location" // Specific branches or locations
)
```

#### **Authorized Actions**

```go
type AuthorizedActions struct {
    Transactions      []TransactionType   `json:"transactions"`        // Financial transactions
    Decisions         []DecisionType      `json:"decisions"`           // Business decisions
    NonPhysicalActions []NonPhysicalAction `json:"non_physical_actions"` // Information actions
    PhysicalActions   []PhysicalAction    `json:"physical_actions"`    // Physical world actions
}

// Transaction Types
type TransactionType string
const (
    TransactionLoan     TransactionType = "loan_transactions"
    TransactionPurchase TransactionType = "purchase_transactions"  
    TransactionSale     TransactionType = "sale_transactions"
    TransactionLease    TransactionType = "leasing_rental"
)

// Decision Types
type DecisionType string
const (
    DecisionPersonnel    DecisionType = "personnel_decisions"    // Hiring, dismissal, development
    DecisionFinancial    DecisionType = "financial_commitments"  // Contracts, expenses, investments
    DecisionBuySell      DecisionType = "buy_sell_transactions"  // Asset acquisition/disposal
    DecisionConceptual   DecisionType = "conceptual_determinations" // Business models, concepts
    DecisionDesign       DecisionType = "design_decisions"       // Branding, architecture
    DecisionInformation  DecisionType = "information_sharing"    // Data disclosure, PR
    DecisionStrategic    DecisionType = "strategic_decisions"    // M&A, partnerships, strategy
    DecisionLegal        DecisionType = "legal_proceedings"      // Legal actions
    DecisionAsset        DecisionType = "asset_management"       // Asset management decisions
)

// Non-Physical Actions
type NonPhysicalAction string
const (
    ActionSharing      NonPhysicalAction = "sharing_presenting"
    ActionBrainstorm   NonPhysicalAction = "brainstorming"
    ActionResearch     NonPhysicalAction = "researching_rag"    // Including RAG operations
)

// Physical Actions (primarily for humanoid robots)
type PhysicalAction string
const (
    ActionShipment     PhysicalAction = "shipments"      // Ocean, air, truck shipments
    ActionProduction   PhysicalAction = "production"     // Manufacturing processes (for demonstration only; NOT production ready)
    ActionRecycling    PhysicalAction = "recycling"      // Recycling operations
    ActionStorage      PhysicalAction = "storage"        // Physical storage operations
    ActionCustomization PhysicalAction = "customization" // Product customization
    ActionPackage      PhysicalAction = "package"        // Packaging operations
    ActionClean        PhysicalAction = "clean"          // Cleaning operations
)
```

### **Section C: Requirements**

#### **Requirements Structure**

```go
type Requirements struct {
    ValidityPeriod    ValidityPeriod    `json:"validity_period"`     // Time constraints
    FormalRequirements FormalRequirements `json:"formal_requirements"` // Legal formalities
    PowerLimits       PowerLimits       `json:"power_limits"`        // Authority limitations
    SpecificRights    SpecificRights    `json:"specific_rights"`     // Rights and obligations
    SpecialConditions SpecialConditions `json:"special_conditions"`  // Special conditions
    DeathIncapacity   DeathIncapacity   `json:"death_incapacity"`    // Death/incapacity rules
    SecurityCompliance SecurityCompliance `json:"security_compliance"` // Security requirements
    JurisdictionLaw   JurisdictionLaw   `json:"jurisdiction_law"`    // Legal framework
    ConflictResolution ConflictResolution `json:"conflict_resolution"` // Dispute resolution
}
```

#### **ValidityPeriod**

```go
type ValidityPeriod struct {
    StartTime       time.Time      `json:"start_time"`        // Authorization start time
    EndTime         time.Time      `json:"end_time"`          // Authorization end time (max 1 year)
    TimeWindows     []TimeWindow   `json:"time_windows"`      // Operational time windows
    GeoConstraints  []string       `json:"geo_constraints"`   // Geographic restrictions
    SuspensionRules []string       `json:"suspension_rules"`  // Automatic suspension conditions
}

type TimeWindow struct {
    Start    string   `json:"start"`    // Start time (HH:MM format)
    End      string   `json:"end"`      // End time (HH:MM format)
    Timezone string   `json:"timezone"` // Timezone identifier
    Days     []string `json:"days"`     // Days of week (Mon, Tue, etc.)
}
```

#### **PowerLimits**

```go
type PowerLimits struct {
    PowerLevels        []PowerLevel     `json:"power_levels"`         // Amount and transaction limits
    InteractionBoundaries []string      `json:"interaction_boundaries"` // Data access, collaboration limits
    ToolLimitations    []string         `json:"tool_limitations"`     // Permitted tools and APIs
    OutcomeLimitations []string         `json:"outcome_limitations"`  // Intended outcome constraints
    ModelLimits        []ModelLimit     `json:"model_limits"`         // AI model restrictions
    BehavioralLimits   []string         `json:"behavioral_limits"`    // Action restrictions
    QuantumResistance  bool             `json:"quantum_resistance"`   // Require quantum-resistant crypto
    ExplicitExclusions []string         `json:"explicit_exclusions"`  // Explicitly forbidden actions
}

type PowerLevel struct {
    Type        string  `json:"type"`        // "amount", "transaction_type", etc.
    Limit       float64 `json:"limit"`       // Numerical limit value
    Currency    string  `json:"currency"`    // Currency code (if applicable)
    Description string  `json:"description"` // Human-readable description
}

type ModelLimit struct {
    ParameterCount   int64    `json:"parameter_count"`   // Maximum model parameters
    ReasoningMethods []string `json:"reasoning_methods"` // Permitted reasoning methods
    TrainingMethods  []string `json:"training_methods"`  // Permitted training approaches
    Description      string   `json:"description"`       // Limit description
}
```

#### **JurisdictionLaw**

```go
type JurisdictionLaw struct {
    Language           string   `json:"language"`             // Contract language
    GoverningLaw       string   `json:"governing_law"`        // Applicable legal framework
    PlaceOfJurisdiction string   `json:"place_of_jurisdiction"` // Legal jurisdiction
    AttachedDocuments  []string `json:"attached_documents"`   // Referenced legal documents
}
```

## 🔐 **Professional Foundation API**

### **ProperJWTService**

Development JWT implementation with RSA-256 signatures.

```go
type ProperJWTService struct {
    privateKey *rsa.PrivateKey
    publicKey  *rsa.PublicKey
    issuer     string
    audience   string
}

func NewProperJWTService(issuer, audience string) (*ProperJWTService, error)
func (s *ProperJWTService) CreateToken(subject string, scope []string, expiry time.Duration) (string, error)
func (s *ProperJWTService) ValidateToken(tokenString string) (*CustomClaims, error)
```

### **LegalFrameworkValidator**

Multi-jurisdiction legal validation.

```go
type LegalFrameworkValidator struct {
    supportedJurisdictions map[string]bool
    supportedEntityTypes   map[string]bool
    complianceRules       map[string][]string
}

func (v *LegalFrameworkValidator) ValidateFramework(ctx context.Context, framework LegalFramework) error
```

## 📊 **Response Types**

### **AgentAuthResponse**

```go
type AgentAuthResponse struct {
    AuthorizationCode string              `json:"code"`              // OAuth authorization code
    State            string              `json:"state"`             // CSRF protection state
    ExtendedToken    string              `json:"extended_token"`    // Optional immediate token
    LegalCompliance  bool                `json:"legal_compliance"`  // Legal validation result
    AuditRecordID    string              `json:"audit_record_id"`   // Audit trail identifier
    ExpiresIn        int                 `json:"expires_in"`        // Code expiration (seconds)
    Scope           []string            `json:"scope"`             // Granted scopes
    PoAValidation   PoAValidationResult `json:"poa_validation"`    // PoA validation details
}
```

### **PoAValidationResult**

```go
type PoAValidationResult struct {
    Valid             bool     `json:"valid"`              // Overall validation result
    ValidationErrors  []string `json:"validation_errors"`  // Specific validation errors
    ComplianceLevel   string   `json:"compliance_level"`   // "rfc115_compliant", etc.
    AttestationStatus string   `json:"attestation_status"` // "validated", "pending", etc.
}
```

### **AgentAuthToken**

```go
type AgentAuthToken struct {
    AccessToken      string           `json:"access_token"`      // JWT access token
    TokenType        string           `json:"token_type"`        // "bearer"
    ExpiresIn        int              `json:"expires_in"`        // Token expiration (seconds)
    Scope           []string         `json:"scope"`             // Token scopes
    ExtendedMetadata ExtendedMetadata `json:"extended_metadata"` // PoA metadata
}

type ExtendedMetadata struct {
    PowerType        string        `json:"power_type"`         // Type of power granted
    PrincipalID      string        `json:"principal_id"`       // Principal identifier
    AIAgentID        string        `json:"ai_agent_id"`        // AI agent identifier
    PoADefinition    PoADefinition `json:"poa_definition"`     // Complete PoA definition
    AttestationLevel string        `json:"attestation_level"`  // Attestation level achieved
    ComplianceProof  []string      `json:"compliance_proof"`   // Compliance validation proof
}
```

## 🚨 **Error Handling**

### **Error Types**

```go
// RFC validation errors
type RFCValidationError struct {
    Code    string `json:"code"`    // Error code (e.g., "invalid_principal")
    Message string `json:"message"` // Human-readable message
    Field   string `json:"field"`   // Field that caused the error
}

// Legal compliance errors
type LegalComplianceError struct {
    Jurisdiction string `json:"jurisdiction"` // Jurisdiction where error occurred
    Regulation   string `json:"regulation"`   // Specific regulation violated
    Description  string `json:"description"`  // Error description
}

// AI capability errors
type AICapabilityError struct {
    ClientID   string `json:"client_id"`   // AI client identifier
    Capability string `json:"capability"`  // Missing capability
    PowerType  string `json:"power_type"`  // Power type requiring capability
}
```

### **Common Error Codes**

| Error Code | Description | HTTP Status |
|------------|-------------|-------------|
| `invalid_request` | Malformed request structure | 400 |
| `invalid_principal` | Principal validation failed | 400 |
| `invalid_client` | AI client validation failed | 400 |
| `unsupported_jurisdiction` | Jurisdiction not supported | 400 |
| `insufficient_capabilities` | AI lacks required capabilities | 403 |
| `excessive_power_limits` | Requested powers exceed limits | 403 |
| `legal_compliance_failure` | Legal framework validation failed | 403 |
| `invalid_poa_definition` | PoA Definition validation failed | 400 |

## 📖 **Usage Examples**

### **Complete RFC Implementation Example**

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/AgentAuth-Foundation/gauth/pkg/auth"
)

func main() {
    // Create service
    service, err := auth.NewRFCCompliantService("GlobalTech", "ai-authorization")
    if err != nil {
        panic(err)
    }
    
    // Create comprehensive PoA Definition
    poa := auth.PoADefinition{
        // A. Parties
        Principal: auth.Principal{
            Type:     auth.PrincipalTypeOrganization,
            Identity: "GlobalTech-Corp-2025",
            Organization: &auth.Organization{
                Type:                auth.OrgTypeCommercial,
                Name:                "GlobalTech Corporation",
                RegisterEntry:       "HRB-123456",
                ManagingDirector:    "Dr. Sarah Johnson",
                RegisteredAuthority: true,
            },
        },
        Client: auth.ClientAI{
            Type:              auth.ClientTypeAgenticAI,
            Identity:          "ai_financial_advisor_v3",
            Version:           "3.2.1-prod", # (for demonstration only; NOT production ready)
            OperationalStatus: "active",
        },
        
        // B. Authorization Type & Scope
        AuthorizationType: auth.AuthorizationType{
            RepresentationType: auth.RepresentationSole,
            SignatureType:      auth.SignatureSingle,
            SubProxyAuthority:  false,
        },
        ScopeDefinition: auth.ScopeDefinition{
            ApplicableSectors: []auth.IndustrySector{
                auth.SectorFinancial, auth.SectorICT,
            },
            ApplicableRegions: []auth.GeographicScope{
                {Type: auth.GeoTypeNational, Identifier: "US"},
                {Type: auth.GeoTypeRegional, Identifier: "EU"},
            },
            AuthorizedActions: auth.AuthorizedActions{
                Transactions: []auth.TransactionType{
                    auth.TransactionPurchase, auth.TransactionSale,
                },
                Decisions: []auth.DecisionType{
                    auth.DecisionFinancial, auth.DecisionInformation,
                },
                NonPhysicalActions: []auth.NonPhysicalAction{
                    auth.ActionResearch, auth.ActionSharing,
                },
            },
        },
        
        // C. Requirements
        Requirements: auth.Requirements{
            ValidityPeriod: auth.ValidityPeriod{
                StartTime: time.Now(),
                EndTime:   time.Now().Add(90 * 24 * time.Hour),
                TimeWindows: []auth.TimeWindow{
                    {Start: "09:00", End: "17:00", Timezone: "EST"},
                },
            },
            PowerLimits: auth.PowerLimits{
                PowerLevels: []auth.PowerLevel{
                    {Type: "transaction_value", Limit: 500000.0, Currency: "USD"},
                    {Type: "daily_limit", Limit: 1000000.0, Currency: "USD"},
                },
                QuantumResistance: true,
                ExplicitExclusions: []string{"cryptocurrency_trading"},
            },
            JurisdictionLaw: auth.JurisdictionLaw{
                Language:           "English",
                GoverningLaw:       "Delaware_Corporate_Law",
                PlaceOfJurisdiction: "US",
            },
        },
    }
    
    // Create AgentAuth request
    request := auth.AgentAuthRequest{
        ClientID:     "ai_financial_advisor_v3",
        ResponseType: "code",
        Scope:        []string{"financial_advisory", "asset_management"},
    State:        "secure_state_token_2025", # (for demonstration only; NOT production ready)
        PowerType:    "financial_advisory_powers",
        PrincipalID:  "GlobalTech-Corp-2025",
        AIAgentID:    "ai_financial_advisor_v3",
        Jurisdiction: "US",
        PoADefinition: poa,
    }
    
    // Authorize with full RFC validation
    response, err := service.AuthorizeAgentAuth(context.Background(), request)
    if err != nil {
        fmt.Printf("❌ Authorization failed: %v\n", err)
        return
    }
    
    fmt.Printf("✅ Authorization successful!\n")
    fmt.Printf("Authorization Code: %s\n", response.AuthorizationCode[:20]+"...")
    fmt.Printf("Legal Compliance: %v\n", response.LegalCompliance)
    fmt.Printf("Compliance Level: %s\n", response.PoAValidation.ComplianceLevel)
    fmt.Printf("Attestation Status: %s\n", response.PoAValidation.AttestationStatus)
    fmt.Printf("Audit Record: %s\n", response.AuditRecordID)
}
```

---

*This API reference provides complete documentation for the official AgentAuth Community AgentAuth RFC implementation. For additional examples and guides, see the [Getting Started Guide](../docs/GETTING_STARTED.md) and [RFC Architecture Documentation](../docs/RFC_ARCHITECTURE.md).*

---

## Infrastructure & Resilience Utilities

### Circuit Breaker

The circuit breaker now uses an options-driven constructor:

```go
type Options struct {
    Name             string
    FailureThreshold int
    ResetTimeout     time.Duration
    HalfOpenLimit    int // reserved
}

cb := circuit.NewBreaker(circuit.Options{ Name: "auth-service", FailureThreshold: 5, ResetTimeout: 10 * time.Second })
err := cb.Execute(ctx, func() error { return maybeFailingOperation() })
state := cb.GetState() // circuit.StateClosed | StateOpen | StateHalfOpen
```

Deprecated shim retained (will be removed in a future release):

```go
// Deprecated: use NewBreaker
legacy := circuit.New("auth-service", 5, 10*time.Second)
```

### Rate Limiter (Simplified)

Unified token bucket implementation in `pkg/rate`:

```go
type Config struct {
    RequestsPerSecond int
    BurstSize         int
    WindowSize        time.Duration // used for aliasing / demos
}

lim := rate.NewLimiter(rate.Config{RequestsPerSecond: 50, BurstSize: 100, WindowSize: time.Second})
if lim.Allow() { /* proceed */ }
_ = lim.Wait(ctx) // blocks until allowed
remaining := lim.GetRemainingRequests("client-id") // approximate
```

Aliases for demo compatibility:

```go
enh := rate.NewEnhancedConfig(25, time.Second) // returns EnhancedConfig with underlying Config
tb := rate.NewTokenBucket(enh)                 // wrapper with Allow() error
```

Removed / Deprecated:
- Sliding window & distributed public constructors (examples and benchmarks now skipped / removed).

### Metrics Collector

Enhanced collector returns aggregated metric map:

```go
mc := monitoring.NewMetricsCollector()
mc.IncrementWithLabels("transactions_total", map[string]string{"status":"success"})
mc.GaugeWithLabels("response_time_seconds", 0.123, nil)
metrics := mc.GetAllMetrics() // map[string]monitoring.MetricValue{ Name: {Value, Labels} }
```

Structural change: `GetAllMetrics()` no longer returns per-label series grouped by name — tests should assert presence and basic value thresholds rather than specific label combinations.

### Authorization Policy Model Update

New simplified policy struct (plural resource/action containers removed):

```go
type Policy struct {
    Subject  string
    Resource string
    Actions  []string
    Effect   string // typically "allow" or "deny"
}

authorizer.AddPolicy(Policy{Subject: "user-1", Resource: "payments", Actions: []string{"read","create"}, Effect: "allow"})
decision := authorizer.Authorize(Request{Subject: "user-1", Resource: "payments", Action: "read"})
if decision.Allow { /* authorized */ }
```

### Audit Event API Change

Deprecated fluent builder (`audit.NewEntry()...WithActor()...`) replaced by event factory:

```go
evt := audit.NewEvent(audit.EventTypeAuthentication, audit.ActionLogin, audit.ResultSuccess)
evt.Subject = "user123"
logger := audit.NewAuditLogger()
_ = logger.Log(ctx, evt)
```

Benchmarks updated accordingly; the old chain methods are removed.

### Token Store Changes

Distributed token store helpers (`NewDistributedStore`, `DistributedConfig`) removed from integration tests (no implementation in current module scope). Memory store remains primary:

```go
store := token.NewMemoryStore(time.Hour)
// save / get / revoke tokens
```

### HTTP Deprecation Headers

Beta HTTP routes now emit the following headers to signal API evolution:

```
X-API-Deprecated: true
X-API-Replacement: /api/v1/beta
```

Clients should plan migration paths accordingly.

---

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
