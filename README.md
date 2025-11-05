# GiFo RFC 0150 – Go Implementation of GAuth 1.0

[![Lint](https://github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/actions/workflows/lint.yml/badge.svg)](https://github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/actions/workflows/lint.yml)
[![OpenAPI Spec Contract](https://github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/actions/workflows/openapi-coverage.yml/badge.svg)](https://github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/actions/workflows/openapi-coverage.yml)
[![Gap Matrix](docs/badges/gap-matrix.svg)](docs/GAP_MATRIX.auto.md)
[![CI Build](https://github.com/mauriciomferz/Gauth_go/actions/workflows/ci.yml/badge.svg)](https://github.com/mauriciomferz/Gauth_go/actions/workflows/ci.yml)

> **📅 Recent Updates (November 2025)**
> - ✅ **GAP Matrix Enhancement** - Updated to 10 Implemented / 24 Partial / 9 Missing features with comprehensive roadmap
> - ✅ **Interactive Web Features** - Fixed all button handlers and dynamic pattern simulation (25+ unique patterns)
> - ✅ **Model Limit Enforcement** - Multi-dimensional limits with per-user quotas and audit hash chain (sec11.item2)
> - ✅ **Delegation Depth Control** - Dynamic depth enforcement with metrics tracking (sec12.item2)
> - ✅ **Threat Model Documentation** - Comprehensive 12-threat analysis with mitigation mapping (sec14.item1-2)
> - ✅ **Test Coverage** - 100% conformance: 8/8 clauses mapped, 24/24 symbols found
> - ✅ **Audit Ledger** - Hash chain with receipt chain and Merkle roots (sec5.item1)
> - ✅ **OpenAPI Specification** - Complete API docs including provenance endpoints (sec10.item1)

> **Latest Project Status & Compliance Mapping**
> - Progress Report (2025-10-20): see `docs/PROGRESS_STATUS_2025-10-20.md` (executive summary of delivered vs planned + next 10-day focus)
> - Clause-to-Test Coverage Scoping: `docs/CLAUSE_TEST_COVERAGE_SCOPING.md` (data model & generator plan)
> - Coverage Template & Generator: `docs/coverage_template.json` + `cmd/coverage/main.go` (run `go run ./cmd/coverage` to produce `docs/CLAUSE_TEST_COVERAGE.json` after adding markers)



> **⚠️ BETA DEMO NOTICE**
>
> This is a **beta demonstration implementation** of the GAuth authorization framework. It is designed for:
> - 🚀 **Learning** GAuth concepts and patterns
> - 🔬 **Experimenting** with authorization flows
> - 📚 **Understanding** RFC specifications
> - 🧪 **Testing** ideas and concepts
>
> **NOT PRODUCTION READY** – This repository is a beta demonstration only. Do **NOT** use it for real security, production, regulated data, or commercial purposes. See `DISCLAIMER.md` for details on intentionally missing protections.
### ✅ **Implemented Core Features**
  - ✅ **Token Management** - Complete lifecycle with revocation (WORKING)
### ✅ **Technical Stack Implementation**
  - ✅ **High Performance** - All resilience patterns functional and tested
### ✅ **Developer Experience Excellence**
  - ✅ **Interactive Web Demo** - Modern beta web interface with live demos
### 🌐 **Beta Web Interface**
Experience GAuth concepts through an interactive beta web demo:
```bash
# Start the 
# Then visit: http://localhost:8080
```
- **Interactive Token Management** - Create, validate, and revoke beta tokens
 - **New Observability Panels (Beta)** – Live decision metrics (counts + reason codes), capability registry integrity hashes (current & previous), recent audit actions, and lifecycle transition timeline with latency (ns) + reason labeling. These auto-refresh every few seconds and expose in-memory instrumentation for rapid feedback during learning. All panels are additive demo tooling and reset on restart.
[//]: # (----- START UI REVAMP SECTION -----)
#### 🧭 Ledger & Security Metrics Panels (UI Revamp 2025-10)

The beta web interface has been **revamped** to consolidate cryptographic integrity and security observability into a single modular "Ledger & Security Metrics" section. A unified ES module orchestrator (`web/static/js/modules/main.js`) now coordinates refresh cycles, accessibility hooks, and tab state, replacing multiple ad‑hoc script initializers.

Key Panels (auto-refreshing, low overhead):

| Panel | Endpoint | Refresh | Purpose |
|-------|----------|---------|---------|
| Rotation Summary | `GET /api/v1/beta/rotations/summary` | 30s | Aggregated key rotation ledger integrity (head hash, chain length, optional EdDSA signature + anchoring status). |
| Capability Anchor Status | `GET /api/v1/beta/capabilities/anchor/status` | 20s | Canonical capability registry hash, emission aging, stale SLA flag, emission counters, optional notarization receipt summary. |
| Violation Metrics | `GET /api/v1/beta/metrics/violations` | 10s | Token validation / policy violation categorical counters, rolling short/long rate samples, surge detection flag. |
| Semantic Counters | `GET /api/v1/beta/metrics/poa/semantics` | 12s | RFC0111 semantic execution counters (delegation, authorization semantics) + recent rate indicator. |
| Revocation Transparency | `GET /api/v1/token/revocation/head`, `GET /api/v1/token/revocation/root` | 20–30s | Delegation revocation chain integrity (head hash, length, aggregate, verification status) + Merkle root & signed tree head snapshot, with on‑demand inclusion proofs. |

Rotation Summary Determinism:
The JSON artifact is built via a canonical ordering routine (stable field order + deterministic hash concatenation) prior to optional Ed25519 signing. This ensures signature stability across identical ledger states. When `GAUTH_ROTATIONS_SIGN=1` and a valid active EdDSA key is loaded, `kid`, `signature`, and `signed_at` fields are present; otherwise they are omitted without error.

Optional Anchoring:
If `GAUTH_ANCHOR_ROTATIONS=1` is set and an anchor client is configured, the current `head_hash` is externally anchored (idempotent) and the panel displays an "Anchored" badge with receipt hash and timestamp. Anchoring attempts are recorded but skipped when the head hash is unchanged.

Accessibility & UX Improvements:
* Single tab system uses ARIA roles with keyboard focus management (`aria-tabs.js`).
* Panels announce refresh outcomes in hidden `aria-live` regions only on state changes (avoids verbosity).
* High-contrast color tokens selected for status badges (anchored / configured / surge flags).
* All periodic fetches use exponential backoff on transient network errors to prevent log spam and avoid tight retry loops.

Module Architecture:
* `main.js` – Entry orchestrator (theme toggle, nav, tab activation, invoking individual panel initializers).
* `rotation_summary.js` – Fetch + render chain head integrity (hashes, length, signature, anchoring badge).
* `violation_metrics.js` – Fetch + render counters table and rate summary (60s & 300s windows) with surge heuristic.
* `semantic_metrics.js` – Fetch + render semantic execution counters and representative short-term rate.
* `refresh.js` – Shared resilient fetch helper (`backoffFetchJSON`) with capped jitter & max retries.

Adding New Panels:
1. Create module under `web/static/js/modules/<panel>.js` exporting `init<Panel>()`.
2. Register the module call inside `main.js` after DOMContentLoaded.
3. Add DOM container markup (prefer a `<section>` with a deterministic `id` and data attributes) to `templates/index.html` within the metrics tab region.
4. Keep polling intervals ≥8s unless extremely low-cost to minimize impact when running with `-race` tests.

Failure & Edge Cases Handled:
* Empty responses render "No data" placeholder rather than stale values.
* Network errors display a transient warning banner (auto-clears on next successful refresh).
* Signature missing scenarios (unsigned rotation summaries) degrade gracefully—signature fields simply blank.

Future Enhancements:
* Consistency proof visualization & historical tree head diff explorer.
* Client-side percentile estimation (HDR-like sketch) for lifecycle latency without server histogram overhead.
* Dark mode contrast audit & high-visibility focus rings for keyboard nav.

> Implementation note: Legacy `app.js` is retained for backward compatibility but no longer initializes new metrics. New modules live under `web/static/js/modules/`; embedded serving uses a glob so no server code changes were required.
[//]: # (------ END UI REVAMP SECTION ------)
[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![Beta](https://img.shields.io/badge/build-beta-blue.svg)]()
[![Learning](https://img.shields.io/badge/purpose-beta-orange.svg)]()
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)
[![RFC Compliant](https://img.shields.io/badge/RFC-0111%20%7C%200115-orange.svg)](https://gimelfoundation.com)
[![Status](https://img.shields.io/badge/status-beta%20demo-blue.svg)]()
[![Not Production Ready](https://img.shields.io/badge/production-NOT_READY-critical.svg)](DISCLAIMER.md)

### 🔄 GAP Matrix Automation
The implementation vs RFC gap tracking is now partially automated:

| Command | Purpose |
|---------|---------|
| `make gap-matrix` | Generate `docs/GAP_MATRIX.auto.md` (non‑enforcing; still exits 2 on drift but file is emitted). |
| `make gap-matrix-check` | Enforcing drift check (fails CI if CSV vs curated Markdown differ on Status / Priority). |

Artifacts:
* Curated narrative: `docs/GAP_MATRIX.md` (human maintained context & remediation plan).
* Structured source: `artifacts/gap_matrix.csv` (machine generated / flattened rows).
* Generated view: `docs/GAP_MATRIX.auto.md` (capability snapshot + drift report).

ID Stability:
* Each CSV row carries a stable identifier (`secN.itemM`). The generated auto matrix renders these IDs as a first column.
* The curated Markdown currently omits IDs for readability. If you add an `ID` column (as the first column in each table: `| ID | Requirement | ...`) the generator will now detect it and perform drift matching by ID first (falling back to Requirement text when absent).
* Benefit: Editing the human-readable Requirement wording no longer causes false drift, provided the ID stays constant.
* Optional lightweight alternative: add the ID only for sections with frequent wording edits; leaving others unchanged is supported (mixed mode).

Governance Recommendation:
* When deprecating or splitting a requirement, retire the old ID (do not reuse) and introduce new IDs; annotate the retired one in a changelog or an ADR to preserve historical audit continuity.

Badge & Dashboard Integration:
* Generator emits `docs/GAP_MATRIX.auto.json` (machine-readable) + a static SVG badge at `docs/badges/gap-matrix.svg` (Implemented / Total).
* Badge is regenerated via `make gap-matrix` or in CI drift check.
* For richer dashboards (percentages, historical deltas) consume the JSON (`counts.implemented`, `counts.partial`, `counts.missing`, `total`).
   ```
* Shields.io dynamic badge (after publishing raw file, e.g. GitHub Pages or raw link):
   ```
   https://img.shields.io/endpoint?url=<RAW_URL_ENCODED_TO_docs/gap_matrix_badge.json>&label=GAuth%20Coverage&color=blue
   ```
* Suggested derived badges:
   - Implemented coverage (green/yellow/red thresholds e.g. >60 green, >30 yellow else red).
   - Implemented+Partial (overall maturity trajectory).
* Use `implemented_pct` for strict readiness; `with_partial_pct` for roadmap maturity.

Evidence Enrichment (JSON only):
* `docs/GAP_MATRIX.auto.json` now includes per-row `evidence_detail`:
   - `files` – parsed list from the CSV Evidence column (split on `|`).
   - `existing` / `missing` – which referenced files exist on disk (file:line normalized to file path).
   - `test_files` vs `code_files` classification (`*_test.go` vs other `.go`/`.md`).
   - Counts: `existing_count`, `missing_count`, `test_file_count`, `code_file_count`.
* Purpose: enable dashboards / CI lint to detect stale evidence pointers (e.g., missing files) and surface test coverage density per requirement.
* Governance tip: treat a non‑zero `missing_count` as a signal to reconcile or prune outdated evidence references early in PR review.

Historical Trend Logging:
* Each generation appends a snapshot line to `artifacts/history_gap_matrix.jsonl` (JSONL format) with counts & percentages.
* Shape per line:
   ```json
   {"ts":"2025-10-22T04:36:00Z","implemented":7,"partial":19,"missing":17,"total":43,"implemented_pct":16.28,"with_partial_pct":60.47}
   ```
* Use cases:
   - Plot implementation progress over time (e.g., Grafana via log scrape or simple script).
   - Detect regressions (Implemented count drops) in CI by diffing last two lines.
* Suggested guard (future): fail CI if `implemented` decreases unless commit message includes `[allow-coverage-drop]` token.

PR Drift Commenter:
* Workflow: `.github/workflows/gap-matrix-pr-drift.yml` runs on PR changes to CSV / curated Markdown / generator.
* Behavior:
   - Runs `make gap-matrix-check`.
   - Exit code `2` (drift) triggers a PR comment summarizing differences (Status / Priority divergences) extracted from `docs/GAP_MATRIX.auto.md`.
   - Subsequent pushes update the existing bot comment instead of spamming new ones.
   - Exit code `3` (generation error) fails the job immediately.
* Manual resolution: edit either `artifacts/gap_matrix.csv` or `docs/GAP_MATRIX.md` so they align, re-run.
* Extendable: add second step later to open issues for new Missing requirements (hook after drift extraction).

Coverage Regression Guard:
* Workflow: `.github/workflows/gap-matrix-coverage-guard.yml` (PR + selected branch pushes).
* Compares last two snapshots in `artifacts/history_gap_matrix.jsonl` after running the generator.
* Fails if `implemented` count decreases (regression) unless commit history (last 10 messages or PR title) contains override token `[allow-coverage-drop]`.
* Rationale: Prevent silent erosion of actually implemented features while allowing intentional refactors with explicit annotation.
* Future enhancement ideas:
   - Also guard against large negative swings in `with_partial_pct`.
   - Emit a badge delta summary comment on regression.
   - Provide an allowlist file for planned removals.

Missing Requirement Issue Bot:
* Workflow: `.github/workflows/gap-matrix-missing-issues.yml`.
* Detects transitions to `Missing` by comparing last two snapshots in `artifacts/history_gap_matrix_status.jsonl` (per-ID status capture).
* Opens a labeled issue (`gap-matrix`, `gap-missing`) titled `[GAP] New Missing Requirement: <ID>` when a requirement's status changes from Implemented/Partial to Missing.
* Skips if an issue with the same title already exists (idempotent).
* Rationale: surfaces regression or deliberate removal requiring ADR / justification.
* To intentionally remove a feature without triggering an issue, pre-create an issue referencing the de-scope ADR and close it with context, or mark commit with rationale.

Drift Exit Codes (script):
* `0` – In sync.
* `2` – Drift detected (differences listed in generated file).
* `3` – Generation error (missing inputs / parse failure).

Recommended CI pattern:
```bash
make gap-matrix-check
```
If it fails with drift, update either the CSV (regenerate) or the curated Markdown to align Status / Priority for each Requirement.

Future enhancements planned: badge generation (Implemented vs Partial vs Missing counts), JSON output for dashboards, and integration with conformance `report.json` to auto‑link evidence.

**GAuth** is a comprehensive, open-source authorization framework designed specifically for AI systems. This **beta implementation** in Go demonstrates explicit, verifiable, and auditable power-of-attorney flows, showing transparency and accountability concepts in AI decision-making processes.

**Beta Purpose**: This is a **demonstration and learning implementation** - designed for beta use, experimentation, and understanding GAuth concepts.

> 🔐 **Security Note:** This project intentionally omits production security controls. See `docs/SECURITY_ASSESSMENT.md` for a catalog of gaps (hardcoded secrets, unsigned tokens, dev Vault usage). Environment-variable helpers exist only to encourage good habits; they do NOT make this secure.

### 🔐 Capability Anchor & External Notarization (Prototype)
The capability registry is periodically emitted as a canonical JSON anchor artifact for integrity monitoring. Optional external notarization wiring adds freshness and latency metrics.

Key Environment Variables:
| Variable | Purpose |
|----------|---------|
| `GAUTH_CAP_ANCHOR_FILE_PATH` | Path to write anchor artifact (canonical JSON or signed wrapper) |
| `GAUTH_CAP_ANCHOR_WRITE_INTERVAL` | Minimum emission interval (must be >=1m; defaults 5m if unset/invalid) |
| `GAUTH_CAP_ANCHOR_SIGN` | When `1`, signs anchor artifact with active Ed25519 key (if loaded) |
| `GAUTH_CAP_ANCHOR_NOTARIZE` | When `1`, enables prototype external notarization (in-memory provider) & related metrics |
| `GAUTH_CAP_ANCHOR_STALE_THRESHOLD_SECONDS` | Stale threshold for age SLA gauge (default 600) |

Status Endpoint: `GET /api/v1/beta/capabilities/anchor/status` exposes:
```
configured, last_write, registry_hash, age_seconds, stale, stale_threshold_seconds,
emitted_total, skipped_total, hash_changed_total, last_write_unix,
last_notarized_at, notarized_age_seconds, notarization_receipt
```

Metrics (custom exposition):
```
gauth_rfc0111_capability_anchor_last_write_seconds
capability_anchor_age_seconds
capability_anchor_stale
gauth_rfc0111_capability_anchor_emitted_total
gauth_rfc0111_capability_anchor_skipped_total
gauth_rfc0111_capability_anchor_hash_changed_total
capability_anchor_emission_interval_seconds (histogram)
capability_anchor_emission_jitter_seconds
gauth_capability_anchor_notarization_latency_seconds (histogram; when notarize enabled)
gauth_capability_anchor_notarized_age_seconds (when notarize enabled)
gauth_capability_anchor_notarization_failures_total (when notarize enabled)
```

Prototype Receipt (`notarization_receipt`): `hash`, `timestamp`, `provider`, `success` (memory provider only; NOT cryptographically verifiable). See `docs/ALERTING.md` & `docs/COMPLETE_API_REFERENCE.md` for full details.

Roadmap: Replace memory notarizer with real TSA / transparency log (RFC3161, Sigstore Rekor), add inclusion proof verification, and expose signed receipt chain.

### 🌐 External Capability Anchoring (Prototype)
An independent external anchoring provider publishes the canonical capability registry hash for transparency and latency/failure monitoring, separate from the internal notarization path.

Environment Variables:
| Variable | Purpose |
|----------|---------|
| `GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER` | Select provider (`memory` demo in-process, `tsa_stub` simulated latency/failure). |
| `GAUTH_CAP_EXTERNAL_ANCHOR_MIN_MS` | Minimum simulated latency (ms) for `tsa_stub` (default 25). |
| `GAUTH_CAP_EXTERNAL_ANCHOR_MAX_MS` | Maximum simulated latency (ms) for `tsa_stub` (default 120). |
| `GAUTH_CAP_EXTERNAL_ANCHOR_FAIL_PROB` | Failure probability (0.0–1.0) for `tsa_stub` (default 0). |

Endpoints:
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/beta/capabilities/anchor/status` | GET | Includes `external_anchor_receipt` when a successful external anchor exists. |
| `/api/v1/beta/capabilities/anchor/external/receipt` | GET | Latest external anchor receipt. |
| `/api/v1/beta/capabilities/anchor/external/verify` | GET | Basic verification via provider `Verify`. |

Receipt Shape:
```jsonc
{
   "hash": "sha256:...",
   "timestamp": "2025-10-20T19:00:00Z",
   "provider": "memory"|"tsa_stub",
   "version": 1,
   "proof": "..." // optional placeholder in stub
}
```

Prometheus Metrics (adapter):
* `external_anchor_attempts_total{provider}` – attempts initiated
* `external_anchor_failures_total{provider}` – failures
* `external_anchor_latency_seconds{provider}` – latency histogram
* `external_anchor_age_seconds` – seconds since last success
* `external_anchor_last_hash_len` – length of last hash (0 none)

Startup Behavior: Performs an initial anchoring attempt immediately if a registry hash is already computed (static seed scenarios) to populate status.

Failure Simulation: Use `tsa_stub` with `GAUTH_CAP_EXTERNAL_ANCHOR_FAIL_PROB=1` to force failure counters; receipt will be absent from status.

Roadmap: Real RFC3161 / transparency log provider, hash-chain receipt persistence, multi-provider quorum & threshold scoring, proof inclusion endpoints.


### Lifecycle & Multi-Signature (Beta Instrumentation)
Token & delegation lifecycle statuses (`active`, `suspended`, `terminated`) are enforced with terminal guards and updated via:
* `POST /api/v1/token/status/update`
* `POST /api/v1/delegation/status/update`

Metrics counters now wired (memory + Prometheus):
* `token_status_transitions_total`, `token_status_transition_failures_total`
* `delegation_status_transitions_total`, `delegation_status_transition_failures_total`
* `multi_signature_weight_failures_total` (weighted threshold shortfall)

Weighted multi-signature mode (`GAUTH_MULTI_SIG_WEIGHTS`) relaxes raw signer count checks; cumulative weights must meet `GAUTH_MULTI_SIG_THRESHOLD`. See `docs/DEVELOPMENT_ADDENDUM.md` for engineering details.

Lifecycle events emitted:
* `token_status_changed` – token_id, old_status, new_status
* `delegation_status_changed` – delegation_id, old_status, new_status, initialized?

Decision labeling: lifecycle updates also record `decision_total{action="token_status_update"|"delegation_status_update",resource="token:<id>"|"delegation:<id>",outcome="<status>"}` for observability.

#### Labeled Lifecycle Metrics (New)

Lifecycle and delegation transitions now include fine-grained labeled counters and an aggregated JSON breakdown:

Prometheus CounterVecs:
* `token_lifecycle_transition_total{old="<old>",new="<new>",outcome="<success|failure|noop>"}`
* `delegation_lifecycle_transition_total{old="<old>",new="<new>",outcome="<success|failure|noop>"}`

Label semantics:
* `old` – previous status; initialization uses `_` (underscore) when no prior state existed.
* `new` – target status requested/applied.
* `outcome` – `success` (state changed), `failure` (invalid/terminal transition), `noop` (request matched current state, no change).

Memory Metrics Snapshot Endpoint (`/api/v1/beta/metrics/lifecycle`) returns both aggregate counters and breakdown maps:

Key format inside breakdown maps:
* Lifecycle: `entity|old|new|outcome` (e.g. `token|active|suspended|success`, `delegation|_|active|success`)
* Decisions: `action|resource|outcome` (e.g. `token_status_update|token:abc123|success`)

Example response (abridged):
```json
{
   "token_status_transitions_total": 3,
   "token_status_transition_failures_total": 1,
   "delegation_status_transitions_total": 2,
   "delegation_status_transition_failures_total": 1,
   "multi_signature_weight_failures_total": 0,
   "lifecycle_breakdown": {
      "token|active|suspended|success": 1,
      "token|suspended|active|success": 1,
      "token|active|terminated|success": 1,
      "token|terminated|active|failure": 1,
      "delegation|_|active|success": 1
   },
   "decision_breakdown": {
      "token_status_update|token:abc123|success": 3,
      "token_status_update|token:abc123|failure": 1
   }
}
```

Why underscore `_`? It prevents ambiguity between an omitted field and an intentional empty string, making downstream Prometheus / JSON parsing uniform for initialization events.

Use cases:
* Audit: Track failed attempts to resurrect terminated entities (`outcome="failure"`).
* Policy analysis: Measure frequency of suspension cycles.
* Operations: Detect churn or pathological thrashing (repeated `active ↔ suspended`).
* Capacity planning: Correlate decision update volume with other latency metrics (future histogram work).

Future enrichment will add latency histograms and reason codes (`reason="policy_violation"`, etc.).

#### Lifecycle Transition Latency & Reason Codes (Extended)

New latency instrumentation captures the time spent processing each lifecycle status update (token or delegation) in both the in-memory collector and Prometheus adapter:

Prometheus Histogram:
* `gauth_rfc0111_lifecycle_transition_seconds{entity="token"}`
* `gauth_rfc0111_lifecycle_transition_seconds{entity="delegation"}`

Memory snapshot exposes per-entity aggregates:
* `lifecycle_latency_totals_ns` – cumulative nanoseconds per entity.
* `lifecycle_latency_counts` – number of recorded transitions per entity.
* `lifecycle_latency_max_ns` – max single-transition latency (nanoseconds) per entity.

Reason-coded decisions enrich observability:
* `decision_reason_total{action,resource,outcome,reason}` Prometheus CounterVec.
* Memory JSON: `decision_reason_breakdown` map with keys `action|resource|outcome|reason`.

Standard reason values (demo scope):
* `init` – first time a delegation status is set.
* `status_change` – successful state mutation.
* `invalid_transition` – rejected due to terminal or unsupported change.
* `unsupported_status` – payload requested a non-whitelisted status.
* `invalid_payload` – malformed request body.
* `not_found` – token not located.
* `noop` – requested status equals current (idempotent).

Latency Use Cases:
* Regression detection for handler logic (increases in max latency signal accidental blocking).
* Capacity planning (counts vs average duration).
* Policy impact analysis (compare latency distribution for failure vs success once outcome-specific buckets added future).

Planned refinement: add outcome-specific latency labeling, percentile summaries in snapshot, and optional tracing hooks.

### 🔏 Revocation Chain Signatures (Phase 2)
The revocation subsystem now produces a **hash-chained, Ed25519-signed** sequence of revocation events giving cryptographic provenance for delegation invalidation actions.

Key additions:
* `RevocationEvent` now includes `sig_kid` and `signature` fields (base64url Ed25519 signature).
* Each event hash (`hash`) still commits to core fields; the signature commits to those fields **plus** the computed hash and linkage (`prev_hash`).
* Chain verification checks:
   - SHA-256 block hash integrity.
   - Correct `prev_hash` linkage (no breaks, no genesis prev hash).
   - Timestamp sanity (rejects events >2m in the future; tolerates minor skew).
   - Ed25519 signature validity (if present) using in-memory rotating key manager.
* Discovery document exposes:
   - `revocation_support.signatures_enabled` – true if any event carries a signature.
   - `revocation_support.signing_kids` – distinct key IDs observed in the chain.
* New endpoint: `GET /api/v1/token/revocation/verify` returns per-event signature presence and validity plus aggregate chain hash.
* Existing head endpoint `GET /api/v1/token/revocation/head` now benefits from underlying signature validation.

Use cases & rationale:
* Tamper detection: Any mutation of historical revocation events invalidates hash and/or signature upon next verification.
* Forensic audit: Signed events coupled with periodic external anchoring (future phase) allow independent reconstruction and trust minimization.
* Backward compatibility: Unsigned events continue to verify for hash linkage—signature is an additive integrity layer.

Data model (simplified):
```jsonc
{
   "id": "rev-123",            // event id
   "delegation_id": "del-9",   // or delegation_hash
   "reason": "compromise",     // normalized reason code
   "revoked_at": "2025-10-18T12:34:56Z",
   "prev_hash": "<hash-of-prev>",
   "hash": "<sha256-domain-separated>",
   "sig_kid": "AbCdEf12",      // signing key id (Ed25519)
   "signature": "<base64url-signature>"
}
```

Verification endpoint response excerpt:
```json
{
   "success": true,
   "verified": true,
   "aggregate_hash": "<chain-composite-hash>",
   "events": [
      {"id": "rev-1", "hash": "...", "signature_present": true, "signature_valid": true, "sig_kid": "K1"},
      {"id": "rev-2", "hash": "...", "signature_present": true, "signature_valid": true, "sig_kid": "K1"}
   ],
   "length": 2
}
```

Environment interplay:
* Signatures activate automatically when the EdDSA key manager is initialized (`GAUTH_TOKEN_SIG_MODE=eddsa`).
* Rotating keys (Phase 1) remain discoverable via JWKS; historical signed revocations stay valid until their key expires.

Forward roadmap (future phases):
* External anchoring of aggregate chain hash (e.g., ledger or transparency log).
* Merkle accumulator for efficient partial proofs.
* RFC clause mapping manifest (link revocation normative statements to implemented invariants).
* Streaming signature verification metrics & audit emission per revocation.

Security Caveats (demo scope):
* In-memory key manager—no durable storage; restart resets key lineage.
* No replay protection for identical revocation IDs yet (caller responsibility).
* No external time attestation; relies purely on system clock.
* Signature fields omitted for legacy unsigning—upgrade path retains hash chain semantics.

To experiment:
```bash
GAUTH_TOKEN_SIG_MODE=eddsa GAUTH_KEY_ROTATION_HOURS=6 go run ./cmd/gauth-server &
curl -s localhost:8080/api/v1/token/revocation/head | jq
curl -s localhost:8080/api/v1/token/revocation/verify | jq '.events[0]'
```

Tamper demo (local ONLY):
```bash
# Force a revocation, then manually modify server memory with a debugger or by adding a temporary code patch.
# On next /verify call the chain should report 'verified': false.
```

See `pkg/delegation/revocation_chain.go` and tests under `pkg/delegation/revocation_chain_sig_test.go` for implementation details.

### 🔐 EdDSA Key Persistence (Phase 2 Extension)
Ephemeral in-memory keys limited forensic continuity. A persistence layer now optionally stores active and historical Ed25519 keys on disk to preserve signature provenance across restarts.

Enable by setting:
```bash
export GAUTH_TOKEN_SIG_MODE=eddsa
export GAUTH_EDDSA_PERSIST_PATH=./state/eddsa_keys.json    # directory created automatically
```

Behavior:
* On startup, if file exists, keys are loaded (expired keys pruned) and reused; otherwise a fresh active key is generated and persisted.
* Each rotation (manual or auto) appends the previous active key to `history` until expiration.
* File format (simplified):
```jsonc
{
   "ttl_hours": 24,
   "active": {"kid":"AbCd123","created_at":"RFC3339","expires_at":"RFC3339","private_b64":"...","public_b64":"...","alg":"EdDSA","use":"sig"},
   "history": [ { "kid":"...", "expires_at":"..." } ]
}
```
Security Caveats (demo scope):
* Private keys are base64 encoded without encryption—DO NOT use outside demo contexts.
* No passphrase or HSM integration.
* Rotation TTL and auto-rotation interval remain environment-driven (`GAUTH_KEY_ROTATION_HOURS`, `GAUTH_EDDSA_AUTO_ROTATE_MIN`).

Manual rotation example:
```bash
GAUTH_TOKEN_SIG_MODE=eddsa GAUTH_EDDSA_PERSIST_PATH=./state/eddsa_keys.json go run ./cmd/gauth-server &
sleep 1
curl -s localhost:8080/.well-known/jwks.json | jq '.keys | length'
kill %1
gauth-server &   # restart
curl -s localhost:8080/.well-known/jwks.json | jq '.keys[0].kid'   # same active kid if not expired
```

Future Hardening Ideas:
* Encrypt persistence file with age or libsodium sealed boxes.
* Integrate hardware-backed keys (YubiHSM / KMS).
* Add key usage counters for anomaly detection.

### ⛓️ Revocation Aggregate Anchoring (Prototype)
When enabled, the revocation chain aggregate hash is anchored to an in-memory transparency log providing an immutable chronological record prototype.

Enable anchoring:
```bash
export GAUTH_ANCHOR_PROVIDER=memory
export GAUTH_ANCHOR_REVOCATIONS=1
```

Effects:
* Each revocation append attempts `Anchor(aggregate_hash)`; audit entry includes `anchor_hash` and `anchor_at` (RFC3339).
* Duplicate aggregate hashes are idempotent (same record returned).
* Persistence (optional) via `GAUTH_ANCHOR_PERSIST_PATH` stores anchors to disk.

Query latest anchored hashes (audit log snippet):
```bash
curl -s localhost:8080/api/v1/audit/logs | jq '.entries[] | select(.action=="revocation_append") | {revocation_hash: .meta.revocation_hash, anchor_hash: .meta.anchor_hash}'
```

Roadmap:
* External transparency provider integration (Sigstore / blockchain).
* Merkle tree commitments for partial inclusion proofs.
* Signed anchor receipts.

Security Caveats:
* Memory anchor is NOT durable unless persistence enabled.
* No third-party timestamping—clock trust assumed.
* No inclusion proof yet; aggregate is single hash combining sequence & set dimensions.

### 🌳 Revocation Transparency: Merkle Tree, Signed Tree Heads & Verification CLI (Phase 3–4)

The revocation subsystem now exposes a transparency surface enabling external, offline verification of integrity:

Core Components:
1. Merkle Tree over revocation event hashes (append-only)
    * Domain-separated hashing: `GAUTH_MERKLE_LEAF:` and `GAUTH_MERKLE_NODE:` prefixes
    * Odd-leaf promotion (last leaf copied upward) for deterministic roots
    * Endpoints:
       - `GET /api/v1/token/revocation/root` (current root, chain length)
       - `GET /api/v1/token/revocation/proof?id=<event_id>|index=<n>|hash=<event_hash>` (flexible inclusion proof)
2. Signed Tree Heads (STH) – periodic signed snapshots
    * Fields: `version`, `merkle_root`, `chain_length`, `aggregate_hash`, `timestamp`, `signatures[]`
    * Multi-Signature (v2): threshold + weighted signatures if `GAUTH_MULTI_SIG_THRESHOLD > 1`
    * Discovery: `revocation_support.sth_latest` + `sth_history_size`
3. Consistency Proofs (prototype)
    * Endpoint: `GET /api/v1/token/revocation/consistency?start=<tree_head_index>`
    * Demonstrates append-only growth between historical tree head and latest (O(n) naive verification)
4. Persistence of STH history
    * Enable with `GAUTH_STH_PERSIST_PATH=./state/sth_history.json`
    * On startup valid signatures are reloaded; invalid entries skipped
5. Verification CLI (`cmd/verify`)
    * Fetches discovery, events, inclusion proof, STH + JWKS, and optional consistency proof
    * Performs local cryptographic verification (Merkle inclusion + Ed25519 signatures + threshold weight)

Environment Variables:
| Variable | Purpose |
|----------|---------|
| `GAUTH_MULTI_SIG_THRESHOLD` | Required cumulative signature weight (or count) to consider STH valid (>1 activates v2 format) |
| `GAUTH_MULTI_SIG_WEIGHTS` | Comma list `kid=weight` allowing heterogeneous signer weight contributions |
| `GAUTH_STH_PERSIST_PATH` | JSON file path to persist signed tree head history |
| `GAUTH_TOKEN_SIG_MODE=eddsa` | Enables Ed25519 signing of revocation events and tree heads |

Weighted Multi-Sig Semantics:
* If `GAUTH_MULTI_SIG_THRESHOLD=5` and weights: `kA=2,kB=2,kC=3`, any combination meeting ≥5 total weight validates the STH.
* `version` becomes `2` and threshold parameters are bound into the signature payload to prevent replay under altered thresholds.

Inclusion Proof Format:
```jsonc
{
   "success": true,
   "target": "hash:<event_hash>",
   "merkle_root": "<hex_root>",
   "proof": [
      {"sibling": "<hex>", "position": "R"},
      {"sibling": "<hex>", "position": "L"}
   ]
}
```
Verification reconstructs upward applying the domain-separated parent hash until the computed root matches `merkle_root`.

Signed Tree Head Signature Payloads:
* v1 (single-sig): `{"version":1,"merkle_root":"...","chain_length":N,"aggregate_hash":"...","timestamp":"..."}`
* v2 (multi-sig): Adds threshold & weights: `{"version":2,"merkle_root":"...","chain_length":N,"aggregate_hash":"...","timestamp":"...","threshold":T,"weights_total":W}`

Ed25519 signatures are base64url-encoded in `signatures[].sig`; each entry may carry an optional `weight`.

Consistency Proof (prototype) JSON:
```jsonc
{
   "start_length": 10,
   "end_length": 13,
   "start_root": "<root_at_10>",
   "end_root": "<root_at_13>",
   "new_leaves": ["<hash_10>","<hash_11>","<hash_12>"]
}
```
Client replays: build merkle for first 10 leaves, confirm root, then for end length, confirm end root, then ensure appended slice matches `new_leaves`.

#### 🧪 Web UI Client-Side Verification & Consistency (New 2025-10-20)
The beta web UI now includes an **in-browser cryptographic verification panel** nested under *Revocation Transparency*:

Elements Added:
* Inclusion Proof fetch (existing) – enter event id | index | hash → fetch JSON proof.
* Consistency proof fetch – supply `start_index` (historical tree length) and optional `target_length`; if `target_length=0` latest is used.
* "Verify Proof" button – recomputes Merkle root locally from the cached inclusion proof using the same domain separation as the Go backend.
* Status badge – updates to `verified` or `mismatch` based on comparison with currently displayed root.

Implementation Highlights:
* JS module `revocation_transparency.js` now embeds a lightweight **pure JavaScript SHA‑256** (public domain style) to avoid asynchronous WebCrypto complexity for small inputs.
* Leaf digest derivation mirrors backend: `SHA256("GAUTH_MERKLE_LEAF:" + event_hash)`.
* Parent digest: `SHA256("GAUTH_MERKLE_NODE:" + left + right)`; sibling `position` semantics preserved (`R` = sibling right of current hash, `L` = sibling left).
* Verification gracefully disables the button if proof is malformed or empty.
* Consistency proof fetch does not (yet) perform local replay; displayed for inspection only (future: full append-only verification logic client-side).

Usage Flow:
1. Open Revocation Transparency panel.
2. Enter an event identifier (e.g. `rev-1` or leaf index `0`) and click **Proof**.
3. Inspect returned JSON – ensure `proof` array and `merkle_root` fields present.
4. Click **Verify Proof** – UI shows recomputed root + match boolean.
5. (Optional) Enter earlier tree length (e.g. `start_index=5`) and click **Consistency** to view growth slice metadata.

Failure States:
* Network error – proof area shows `Error: <message>`.
* Missing or empty proof – verification disabled (`(no cached proof)`).
* Root mismatch – badge shows `mismatch` (possible tampering or race with chain growth; re-fetch root to confirm).

Limitations (Demo Scope):
* No batched or logarithmic consistency proof (O(n) future improvement).
* No external time attestation – freshness depends on local clock.
* Pure JS SHA-256 not constant-time (acceptable for demo verification workloads only).
* Multi-sig tree head signature verification remains server-side; client trusts reported `verified` flag.

Future Enhancements:
* RFC6962-style logarithmic consistency proofs.
* Signature set display & per-signer weight visualization.
* Local consistency replay (recompute historical and latest roots) & mismatch alerting.
* WebCrypto path (async) with streaming optimization for large proof sets.

Related Files:
* `web/templates/index.html` – Consistency & verification markup (`rev-consistency-*`, `rev-verify-*` IDs).
* `web/static/js/modules/revocation_transparency.js` – Added handlers: `attachConsistencyHandler`, `attachVerifyHandler`, `computeRootFromProof`, `sha256PureJS`.

CLI Alignment: The browser recomputation logic mirrors CLI verification routines ensuring identical domain separation and ordering semantics for educational parity.


### 🔄 Key Rotation Dual-Signature Verification Endpoint (New)
To externally audit key succession events, a rotation verification surface now summarizes continuity and dual-signature validity across the observed rotation descriptors.

Endpoint:
```
GET /api/rotations/verification
```
Response (abridged):
```jsonc
{
   "generated_at": "2025-10-20T12:34:56Z",
   "summary": {
      "total": 2,
      "all_continuity_ok": true,
      "all_signatures_ok": true,
      "failures": 0,
      "results": [
         {
            "index": 0,
            "old_key_id": "ed25519:aa12bb34",
            "new_key_id": "ed25519:cc56dd78",
            "continuity_ok": true,
            "signatures_ok": true
         }
      ]
   }
}
```

Per-descriptor fields:
* `continuity_ok` – `PrevRotationHash` matches previous rotation receipt hash (except first which must be empty).
* `signatures_ok` – both old and new key signatures verified over canonical descriptor payload.
* `reason` – first failure reason (e.g. `continuity_failure`, `missing_old_signature`, `old_sig_invalid`).

Aggregate fields:
* `all_continuity_ok` – all continuity checks passed.
* `all_signatures_ok` – all descriptors had valid dual signatures.
* `failures` – count of descriptors with any failure.

Metrics:
* `gauth_rotation_verification_latency_seconds` – histogram of verification duration.
* `gauth_rotation_verification_total{outcome="success|failure"}` – labeled attempt counter.
* `gauth_rotation_verification_failure_reason_total{reason="<failure_code>"}` – per-reason failure counter (increments once per failing descriptor per request).

Roadmap:
* Persist rotation receipt hashes for replay-resistant continuity proofs.
* Add external timestamping / transparency anchoring of rotation chain heads.
* Expose signed summary artifact.

Tamper Demo (signature removal):
```bash
# (Pseudo) After a rotation event, remove old signature from descriptor in memory.
curl -s localhost:8080/api/rotations/verification | jq '.summary.results[] | select(.reason=="missing_old_signature")'
```

Persistence File Example (`GAUTH_STH_PERSIST_PATH`):
```jsonc
[
   {
      "version":2,
      "merkle_root":"...",
      "chain_length":42,
      "aggregate_hash":"...",
      "timestamp":"2025-10-18T12:34:56Z",
      "threshold":5,
      "weights_total":7,
      "satisfied_weight":5,
      "signatures":[{"kid":"kA","alg":"EdDSA","sig":"...","weight":2},{"kid":"kC","alg":"EdDSA","sig":"...","weight":3}]
   }
]
```

#### 🔍 Using the Verification CLI

Run end-to-end verification against a live server:
```bash
go run ./cmd/gauth-server &           # start server (ensure eddsa + multi-sig env vars if desired)
sleep 1
go run ./cmd/verify --base http://localhost:8080
```

Optional target hash:
```bash
go run ./cmd/verify --base http://localhost:8080 --hash <revocation_event_hash>
```

Output (abridged):
```
[info] latest STH: version=2 root=... length=42 threshold=5 satisfied=5/7 signatures=2
[info] events length=42 aggregate=... verified=true
[info] target hash= ...
[info] merkle inclusion verified=true root=... proof_steps=7
[info] tree head signatures verified
[info] consistency proof verified
[verify] SUCCESS all checks passed
```

Failure Modes:
* Inclusion proof mismatch → potential tampering or wrong hash
* Signature verification failure → unknown `kid`, altered payload, insufficient weight
* Consistency proof failure → non append-only history

Next Roadmap Steps:
* Logarithmic RFC6962-style consistency proofs
* Audit emission of STHs & external transparency anchoring
* BFT-style signature sets & signer attestation metadata
* Client library packaging of verification routines

Implementation References:
* `pkg/delegation/merkle.go` – Merkle construction & proof verification
* `pkg/delegation/revocation_chain.go` – Tree head signing, multi-sig verification, persistence
* `cmd/verify/verify.go` – End-user verification workflow

### 📘 Rotation Ledger & Signed Summary (New)
To provide an append-only, hash-chained provenance of key succession independent from the broader receipt chain, a lightweight `RotationLedger` persists each `KeyRotationDescriptor` along with a chained SHA-256 hash.

Ledger Record Hash Computation:
```
hash = sha256(prev_hash || canonical_rotation_descriptor_bytes)
```
Where `canonical_rotation_descriptor_bytes = 'GAUTH_ROTATION_DESCRIPTOR:' + JSON(without signatures)`.

Persistence File (`GAUTH_ROTATION_LEDGER_PATH`):
```jsonc
{
   "entries": [
      {"index":0,"hash":"<h0>","prev_hash":"","descriptor":{...},"timestamp":"2025-10-20T12:00:00Z"},
      {"index":1,"hash":"<h1>","prev_hash":"<h0>","descriptor":{...},"timestamp":"2025-10-20T13:00:00Z"}
   ],
   "head_hash": "<h1>",
   "updated_at": "2025-10-20T13:00:00Z",
   "version": 1
}
```

Signed Rotation Summary Endpoint:
```
GET /api/v1/beta/rotations/summary
```
Response (with signing/anchoring enabled):
```jsonc
{
   "success": true,
   "configured": true,
   "summary": {
       "chain_length": 7,
       "head_hash": "<ledger_head_hash>",
       "aggregate_hash": "<sha256(all_entry_hashes_concatenated)>",
       "generated_at": "2025-10-20T14:00:00.123Z",
       "kid": "ed25519:aa11bb22",
       "signature": "base64url(ed25519(GAUTH_ROTATION_SUMMARY:...))",
       "mode": "EdDSA"
   },
   "anchored": true,
   "anchor_hash": "<anchor_record_hash>",
   "anchor_at": "2025-10-20T14:00:00Z"
}
```

Aggregate Hash:
```
aggregate_hash = sha256( entry_hash_0 || entry_hash_1 || ... || entry_hash_N )
```

Signature Domain Separation:
```
payload = {"chain_length":N,"head_hash":"...","aggregate_hash":"...","generated_at":"..."}
message = "GAUTH_ROTATION_SUMMARY:" + json(payload)
signature = Ed25519(private_key, message)

Canonical Payload Ordering (Deterministic Signing):
The rotation summary signature now uses a canonical JSON payload helper that marshals a fixed struct with field order: `chain_length`, `head_hash`, `aggregate_hash`, `generated_at`. This eliminates prior non‑determinism caused by map iteration order during JSON encoding. Both the server signer and client verifier use identical helpers to guarantee byte‑for‑byte payload equivalence before domain separation (`GAUTH_ROTATION_SUMMARY:`) and signing. This makes signatures stable across process restarts and prevents intermittent verification failures in tests and external auditors.

Rationale:
* Prevent flaky signature mismatches due to Go map key randomization.
* Provide a stable contract for future multi‑signature or external notarization layers.
* Enable reproducible audit trails (same data → same signature with same key).

Implementation Notes:
* Helper function: internal `canonicalRotationSummaryPayload(sum *RotationSummary)` (mirrored in `pkg/verification`).
* Only the signed subset of fields participates; adding new non‑signed metadata to the summary response will not break existing verifiers.
* Any future expansion of the signed schema will be versioned (e.g. by embedding a `version` field) to preserve backward compatibility.

Testing: A stability test (`internal/notary/rotation_summary_canonical_test.go`) asserts two consecutive sign operations with identical inputs produce equal signatures.
```

Verification Helper (internal): `VerifyRotationSummary(summary, publicKey)` ensures kid derivation matches and signature validates.

Failure Reasons (summary verification): `missing_signature`, `kid_mismatch`, `signature_invalid`, `mode_unsupported`, `serialization_error`.

Anchoring Behavior:
* When `GAUTH_ANCHOR_ROTATIONS=1` the first call after a new head hash is produced will anchor it via the in-memory anchor client.
* Subsequent calls with unchanged `head_hash` return `"anchored": false` (deduplicated) while still exposing the original anchor record through the separate anchor endpoints.

Environment Variables (Rotation Ledger & Summary):
| Variable | Purpose | Default |
|----------|---------|---------|
| `GAUTH_ROTATION_LEDGER_PATH` | Persist rotation ledger (append-rewrite JSON) | empty (disabled) |
| `GAUTH_ROTATIONS_SIGN` | Sign rotation summary using active EdDSA key | `0` (disabled) |
| `GAUTH_ANCHOR_ROTATIONS` | Anchor rotation ledger head hash when summary requested | `0` |

Operational Guidance:
* Enable `GAUTH_ROTATION_LEDGER_PATH` early to avoid losing historical hash continuity.
* Pair `GAUTH_ROTATIONS_SIGN=1` with EdDSA mode (`GAUTH_TOKEN_SIG_MODE=eddsa`) so clients can fetch active public keys via JWKS.
* Anchor selectively (demo) – in production integrate external timestamping / transparency log for stronger non-repudiation.

Audit / Tamper Detection Examples:
```bash
# Detect altered aggregate hash (should invalidate signature)
curl -s $BASE/api/v1/beta/rotations/summary | jq '.summary | {head_hash,aggregate_hash,kid,signature}'

# After offline tamper of ledger file (for demo), signature verification should fail client-side.
```

Future Enhancements:
* Consistency proofs for ledger growth (prove append-only without full replay).
* Multi-signature summary with retiring and successor key joint attestation.
* Optional transparency log inclusion proof for anchored head hashes.

#### 📊 Rotation Summary Metrics (New)

The rotation summary endpoint is now instrumented with dedicated Prometheus metrics to observe generation latency, success/failure rates, and anchoring outcomes. These complement the existing rotation verification metrics and help differentiate failures in building/signing/anchoring the summary from descriptor-level verification issues.

Exported metrics:
* `gauth_rotation_summary_latency_seconds` – Histogram of end-to-end handler latency (summary build → optional signing → optional anchoring → response marshal).
* `gauth_rotation_summary_total{outcome="success"|"error"}` – CounterVec incremented once per request; `error` covers internal build/sign/anchor failures (HTTP still returns JSON with `success:false`).
* `gauth_rotation_summary_anchor_total{result="anchored"|"skipped"|"error"}` – CounterVec tracking anchoring attempts:
   * `anchored` – A new head hash was successfully anchored.
   * `skipped` – Head hash unchanged since last anchor (idempotent dedupe).
   * `error` – Anchor provider returned an error (summary still returned; anchoring not applied).
* `gauth_rotation_summary_chain_length` – Gauge set to current ledger chain length for each request (last observed value).
* `gauth_rotation_summary_head_age_seconds` – Gauge reflecting freshness: wall-time seconds since `generated_at` (client SLO target for rotation frequency).
* `gauth_rotation_summary_last_anchor_age_seconds` – Gauge of seconds elapsed since the most recent successful anchor (monotonic increase until next anchor). Useful for alerting on missed anchoring cadence independent of summary freshness.

Operational Guidance:
* Alert on sustained `outcome="error"` spikes – indicates signing key unavailability or ledger corruption.
* Track ratio of `anchored` to total summary calls; large gaps may signal missed external notarization cadences.
* Use latency histogram to establish internal SLO (p95 <50ms typical under in‑memory demo conditions). Investigate increases for I/O (ledger persistence) regressions.

Example scrape excerpt:
```
# HELP gauth_rotation_summary_total Total rotation summary requests
# TYPE gauth_rotation_summary_total counter
gauth_rotation_summary_total{outcome="success"} 3
gauth_rotation_summary_total{outcome="error"} 0
# HELP gauth_rotation_summary_anchor_total Rotation summary anchoring results
# TYPE gauth_rotation_summary_anchor_total counter
gauth_rotation_summary_anchor_total{result="anchored"} 1
gauth_rotation_summary_anchor_total{result="skipped"} 2
gauth_rotation_summary_anchor_total{result="error"} 0
```

Failure Modes (mapped to `outcome="error"`): `ledger_unavailable`, `build_failed`, `signing_disabled_or_key_missing` (when signing requested), `anchor_failure` (still increments anchor `error` result), `serialization_error`.

Troubleshooting Checklist:
1. Verify environment: `GAUTH_ROTATION_LEDGER_PATH` writable; `GAUTH_ROTATIONS_SIGN=1` + `GAUTH_TOKEN_SIG_MODE=eddsa` for signing.
2. Inspect server logs for `rotation_summary` error entries (include reason string).
3. Confirm active EdDSA key appears in `/.well-known/jwks.json` (kid matches summary `kid`).
4. If anchoring errors: ensure `GAUTH_ANCHOR_PROVIDER` initialized and `GAUTH_ANCHOR_ROTATIONS=1`.
5. Tamper test: modify ledger file head hash – next summary should produce `outcome="error"` (signature mismatch) enabling alert path.

Roadmap (Metrics):
* Add per-outcome latency buckets (success vs error label dimension)
* Emit gauge of current ledger chain length and last anchored head age
* Add signed summary verification counters for external client helper library

##### Client Verification Helper (New)

Programmatic verification of the signed rotation summary is available via the helper in `pkg/verification`:

```go
import (
   "net/http"
   verify "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/verification"
)

func checkRotationSummary(base string) error {
   client := &http.Client{}
   // Ensure JWKS keys are loaded (needed for signature verification)
   if err := verify.LoadJWKS(client, base); err != nil { return err }
   res, err := verify.VerifyRotationSummary(client, base)
   if err != nil { return err }
   if !res.SignatureValid { return fmt.Errorf("rotation summary signature invalid: %s", res.SignatureError) }
   return nil
}
```

Returned struct fields:
* `SignatureValid` / `SignatureError`
* `AgeSeconds` – freshness (client-side computed)
* `Summary` – underlying summary (chain length, head hash, aggregate hash, kid)

Use cases:
* CI guard ensuring published summary remains authentic.
* External auditor polling anchored head hash for unexpected rewinds.
* Alerting if freshness exceeds expected rotation cadence.



Security Notes (Demo Scope):
* STH persistence file unencrypted; treat as demo artifact only
* Threshold & weights loaded via environment (no signed policy) – altering them without re-signing invalidates new STHs but leaves old history intact
* Consistency proof naive (O(n)); not suitable for very large chains yet

> For deeper engineering rationale see `docs/CRYPTOGRAPHY_IMPLEMENTATION.md` and `docs/OBSERVABILITY.md` additions planned in upcoming documentation phase.

### 📸 Snapshot Generation & Verification CLI (Audit Acceleration)

An additive offline audit mechanism supports capturing a point-in-time summary ("snapshot") of the receipt/audit chain and verifying it later without replaying full historical logic.

`cmd/snapshot` provides two modes:
1. Generation (default) – Builds a `Snapshot` JSON from the current receipt chain file.
2. Verification (`-verify`) – Recomputes expected values and validates a previously stored snapshot.

#### Generation Usage
```bash
go run ./cmd/snapshot \
   -receipts ./state/receipts.json \
   -prev "<optional_previous_snapshot_hash>" \
   -out ./snapshots/snap-$(date +%s).json \
   -pretty
```
Flags:
* `-receipts` (required) – Path to persisted receipt chain file.
* `-prev` (optional) – Previous snapshot hash to chain snapshots (linear continuity).
* `-out` (optional) – Write to file atomically (temp + rename). If omitted prints to stdout.
* `-pretty` – Human-readable indentation.
* `-meta` – When printing to stdout, prepend a comment line with generation metadata.

Output (abridged example):
```jsonc
{
   "version": 1,
   "generated_at": "2025-10-20T12:34:56.789Z",
   "receipt_count": 42,
   "chain_head_hash": "sha256:...",
   "merkle_root": "sha256:...",            // present only if merkle feature flag enabled
   "prev_snapshot_hash": "sha256:...",     // optional continuity pointer
   "rotation_head_kid": "K1",              // latest rotation descriptor kid (optional)
   "hash": "sha256:<snapshot_self_hash>"    // domain-separated hash committing to all above fields
}
```

#### Verification Usage
```bash
go run ./cmd/snapshot -verify \
   -receipts ./state/receipts.json \
   -snapshot ./snapshots/snap-1697800000.json \
   -pretty
```
Flags:
* `-verify` – Switch to verification mode.
* `-snapshot` (required in verify mode) – Path to snapshot JSON captured earlier.
* `-receipts` – Same receipts chain file used during generation (should reflect equal or newer state).
* `-pretty` – Pretty-print the combined verification result.

Verification Output Structure:
```jsonc
{
   "snapshot": { /* original snapshot fields */ },
   "result": {
      "valid": true,
      "chain_head_match": true,
      "merkle_root_match": true,
      "snapshot_hash_valid": true,
      "receipt_count_match": true,
      "rotation_head_match": true,
      "reason": ""              // first mismatch reason if invalid
   }
}
```

Exit Codes:
| Code | Meaning |
|------|---------|
| 0 | Verification succeeded (`result.valid=true`) |
| 1 | Verification failed (integrity mismatch) |
| 2 | Missing required flag (`-receipts` or `-snapshot`) |
| 3 | Receipt file load error |
| 4 | Snapshot generation error |
| 5 | Snapshot JSON marshal error (generation) |
| 6 | Temp file write failure (generation with `-out`) |
| 7 | Rename failure finalizing output file |
| 8 | Snapshot file open failure (verify mode) |
| 9 | Snapshot file read failure |
| 10 | Snapshot JSON parse failure |
| 11 | Verification computation error |
| 12 | Result marshal error (verify mode) |

Mismatch Reason Precedence (first failing condition populates `reason`):
1. `merkle_root_mismatch`
2. `chain_head_mismatch`
3. `snapshot_hash_invalid`
4. `receipt_count_mismatch`
5. `rotation_head_mismatch`

Operational Patterns:
* Periodically generate snapshots (cron) committing chain head + merkle root for offline auditors.
* Chain snapshots using `prev_snapshot_hash` for linear provenance.
* Store snapshots in immutable object storage; verification needs the receipt chain file + snapshot artifact.
* Future incremental Merkle optimization will avoid full-tree recompute.

Environment Interaction:
* Merkle inclusion (`merkle_root`) only populated when `GAUTH_NOTARY_MERKLE_ENABLED=1` at generation time.
* Rotation continuity (`rotation_head_kid`) depends on presence of rotation descriptors in receipts.

References:
* `internal/notary/snapshot.go` – `GenerateSnapshot`, `VerifySnapshot`
* `internal/notary/snapshot_test.go` – Generation tests (merkle on/off)
* `internal/notary/snapshot_verify_test.go` – Tamper scenarios & reason precedence
* `internal/notary/snapshot_cli_integration_test.go` – End-to-end CLI exercise
* `cmd/snapshot/main.go` – CLI implementation & exit code mapping

Roadmap:
* Incremental Merkle state to avoid O(n) recompute
* Dual-signature enforcement for rotation descriptors
* Snapshot metrics (generation latency, size, merkle cost)
* Optional external timestamping (TSA / transparency log)

> Snapshots are an additive integrity acceleration mechanism – they do not replace full receipt chain verification but reduce audit overhead for periodic attestations.

#### 📊 Snapshot Metrics (New)
Prometheus metrics emitted during snapshot operations:
* `gauth_snapshot_generation_latency_seconds` – Histogram of generation latency.
* `gauth_snapshot_generation_total{outcome="success|error"}` – Counter of generation attempts.
* `gauth_snapshot_verification_latency_seconds` – Histogram of verification latency.
* `gauth_snapshot_verification_total{outcome="success|failure|error"}` – Counter of verification attempts (failure = integrity mismatch, error = internal serialization issue).

Usage Notes:
* Latency histograms use default buckets; tune later for large chains.
* Errors are differentiated from integrity failures enabling alerting on systemic issues vs tampering.
* Combine with receipt chain integrity gauges for layered monitoring.

#### 🔐 Dual-Signature Key Rotation Enforcement (Implemented)
`KeyRotationDescriptor` now carries optional dual signatures:
```jsonc
"rotation": {
   "old_key_id": "ed25519:abcd...",
   "new_key_id": "ed25519:ef01...",
   "effective_time": "2025-10-20T12:00:00Z",
   "reason": "scheduled",
   "prev_rotation_hash": "<prior_rotation_chain_hash>",
   "old_key_signature": "base64url(ed25519 signature by old key)",
   "new_key_signature": "base64url(ed25519 signature by new key)"
}
```
Signing & Verification:
* Canonical payload excludes signature fields and is domain-separated: `GAUTH_ROTATION_DESCRIPTOR:` prefix + canonical JSON.
* Old key attests succession; new key affirms activation.
* Verification reasons emitted: `kid_mismatch_old`, `kid_mismatch_new`, `missing_old_signature`, `missing_new_signature`, `old_sig_invalid`, `new_sig_invalid`, `serialization_error`.
* Rotation continuity chain formed via `prev_rotation_hash` enables linear audit independent of main receipt chain.

Operational Guidance:
* Require both signatures before marking rotation descriptor accepted.
* Alert on missing or invalid signatures; treat mismatch as potential compromise.
* Future hardening: grace period enforcement, scheduled rotation policy metadata, external timestamping of rotation descriptor.

> Dual-signature descriptors strengthen provenance by ensuring both retiring and successor keys acknowledge the transition, reducing risk of unilateral key hijack.



### Runtime Modes & Secrets (Beta Guardrails)
---
## 🛠️ Operational & Developer Workflow

### Makefile Targets (Web Demo & Full Stack)

| Target                | Description |
|-----------------------|-------------|
| `web-start`           | Start the web demo (scripts/start-web-demo.sh) |
| `web-stop`            | Stop the web demo (scripts/stop-web-demo.sh) |
| `web-restart`         | Restart the web demo (stop+start) |
| `web-logs`            | Tail web demo logs (gauth-web.log) |
| `web-health`          | Health check for web demo (scripts/health.sh) |
| `web-tail-logs`       | Tail logs for web demo (scripts/tail-logs.sh) |
| `web-integration-test`| Run integration tests for web demo (integration tag) |
| `compose-build`       | Build Docker Compose stack (full demo) |
| `compose-up`          | Start Docker Compose stack (full demo) |
| `compose-down`        | Stop Docker Compose stack |
| `compose-logs`        | Tail logs for Docker Compose stack |

#### Example Usage
```bash
# Start/stop web demo
make web-start
make web-stop
# Health check & logs
make web-health
make web-logs
# Run integration tests
make web-integration-test
# Full stack (backend, db, redis, etc.)
make compose-up
make compose-logs
make compose-down
```

### Docker Dev Image (Hot Reload)

For local development with hot reload:
```bash
docker build -f Dockerfile.dev -t gauth-dev .
docker run --rm -it -p 8080:8080 -v $(pwd):/app gauth-dev
# Edits will trigger reload via Air (see .air.toml)
```

---
`GAUTH_MODE` influences startup behavior:
- `development` (default) – Permits ephemeral auto-generated client secret & signing key if `GAUTH_CLIENT_SECRET` / `GAUTH_SIGNING_KEY` unset.
New (beta) cryptographic token signing:
- Tokens are now HMAC-SHA256 signed (JWT-style header.payload.signature) with minimal claims (`sub`, `scope`, `exp`, `iat`).

### Stable vs Ephemeral Runs
By default, running `go run ./cmd/gauth-server` without environment variables generates ephemeral secrets each restart (tokens become invalid after restart).

Use stable secrets for multi-step demo sessions:
```bash
cp .env.example .env   # review & edit values
scripts/start-web-demo.sh
# Stop later
scripts/stop-web-demo.sh
```
Key environment variables:
| Variable | Purpose | Notes |
|----------|---------|-------|
| GAUTH_CLIENT_SECRET | OAuth-style client secret | Beta placeholder — change locally |
| GAUTH_SIGNING_KEY | HMAC signing key for demo tokens | 32+ bytes recommended, rotate frequently |
| GAUTH_PORT | HTTP port | Default 8080 |
| GAUTH_MODE | Mode (`development`) | Only dev mode supported in demo |
| GAUTH_TOKEN_EXPIRY_SECONDS | Token lifetime | Default 3600 |

To run purely ephemeral (stateless) again just kill the process and run:
```bash
go run ./cmd/gauth-server
```
### SBOM (Software Bill of Materials) Generation
Generate an SBOM of dependencies (source or built image) using Syft:
```bash
brew install syft            # or see syft install script
scripts/generate-sbom.sh     # source SBOM -> sbom/sbom-source-<ts>.json
scripts/generate-sbom.sh gauth-beta:latest  # image SBOM
```
Artifacts are placed in `sbom/` (JSON + SPDX JSON). Integrate into CI and sign artifacts for real supply chain security.

### Kubernetes NetworkPolicy (Example)
`deployments/k8s/development/networkpolicy.yaml` restricts ingress to the `gauth` pods in the dev namespace and limits egress to Postgres, Redis and Vault service ports. Extend with stricter namespace or CIDR policies for real clusters.

### Quick Environment Configuration (Demo Only)
---
## 🛡️ Continuous Integration & Compliance

This repository enforces RFC and specification compliance with CI guards:

- **RFC Functional Test**: Every push and pull request runs the RFC functional test (`examples/rfc_functional_test/main.go`) in GitHub Actions. The build fails if any test fails, ensuring all changes remain compliant.

**Run RFC functional test locally:**
```bash
cd examples/rfc_functional_test
go run main.go
```

**In CI:** See `.github/workflows/functional-test.yml` for details.

### 🔒 OpenAPI Route Coverage Enforcement (New)

We enforce a strict 100% mapping between live registered HTTP routes and the OpenAPI specification (`docs/openapi.yaml`). Any divergence (missing spec path for a live route or stale path in the spec) fails CI.

Artifacts & Tooling:
* `internal/specgen/` – Coverage generator canonicalizes Gin `:param` segments to OpenAPI `{param}` placeholders.
* `cmd/specgen/` – CLI wrapper that produces `docs/openapi.coverage.json`.
* `docs/openapi.coverage.json` – JSON snapshot (e.g. `covered_ratio: 1.0`).
* `Makefile` target `openapi_coverage` – Runs coverage, enforces `covered_ratio == 1.0`.
* GitHub Actions workflow `openapi-coverage.yml` – Triggers on pushes / PRs touching spec or generator.

Local verification:
```bash
make openapi_coverage   # Fails if any route is missing from the spec or if extra stale spec paths exist
cat docs/openapi.coverage.json | jq '.covered_ratio'
```

Failure Modes (CI will fail):
* A newly added handler not documented in `openapi.yaml` (reported under `missing_paths`).
* A removed/renamed handler still present in `openapi.yaml` (reported under `extra_spec_paths`).
* Parameter mismatch due to forgetting to convert `:id` → `{id}` in the spec (auto-normalized, but ensure semantic naming).

Why 100%? Prevents silent drift enabling: client generation, security scanning, and contract-first evolution without regressions.

To add a new endpoint:
1. Implement handler.
2. Run `make openapi_coverage` → will fail showing the missing route.
3. Update `docs/openapi.yaml` with the new path (maintain ordering & grouping where possible).
4. Re-run `make openapi_coverage` until ratio returns `1.0`.
5. Commit both code and spec changes together.

Deprecating endpoints:
* Mark with `deprecated: true` in `openapi.yaml` and maintain until removal window expires.
* Only remove after updating code and spec simultaneously (coverage tool will then flag stale entries if you forget to prune).

ETag & Integrity:
* Discovery, JWKS, and OpenAPI endpoints publish ETag headers for cache efficiency.
* Optional HMAC signatures (demo integrity) can be enabled for additional verification (future production scope).

Deprecation Metadata:
* Discovery document now exposes `deprecated_after` and `sunset_after` facilitating automated client warnings.

Badge: The top README badge surfaces current CI status for coverage enforcement.

Spec contract badge now represents both:
* Route coverage (100% required)
* Path parameter description coverage (100% required)

CI runs `make spec-contract` to enforce both simultaneously.

#### 📌 Path Parameter Description Coverage

To improve generated client usability and human readability, all path parameters must include a `description`. A second enforcement target validates this:

```bash
make openapi_param_coverage   # Fails if any path parameter lacks description
```

Combined contract check:
```bash
make spec-contract   # Runs both route & parameter description coverage checks
```

Failure output lists missing param descriptions in the form `/<path>:{param}` pulled from `docs/openapi.coverage.json`.

Rationale:
* Clarifies semantics for identifiers (e.g. `hash` vs `id`).
* Enables richer auto-generated API docs and client SDK comment blocks.
* Prevents silent introduction of ambiguous parameters.

#### 🔎 Query Parameter Description Coverage (New)

All query parameters must include a `description` clarifying semantics, defaults, and bounds.

Local checks:
```bash
make openapi_query_param_coverage   # Enforces 100% query param description coverage
make spec-contract                  # Includes query params now
```

Recommended description elements:
* Purpose (pagination, filtering, time cutoff, etc.)
* Units ("Unix timestamp seconds")
* Defaults ("default 50")
* Bounds ("maximum 250")

Quality benefits: better generated clients, easier SDK decisions, explicit filtering semantics for future caching layers.

#### 🧬 Schema Property Description Coverage (New)

All component schema properties must have a `description` to improve generated documentation, SDK tooltips, and future validation rule clarity.

Check locally:
```bash
make openapi_schema_prop_coverage   # Enforces 100% schema property description coverage
make spec-contract                  # Now includes schema property coverage
```

Why it matters:
* Increases semantic richness for SDK code generation (field comments).
* Clarifies difference between similar fields (`issued_at` vs `expires_at`).
* Facilitates automated rule generation / linting for RFC alignment.

Guidelines:
* Begin with noun phrase ("Internal token identifier").
* Include units and format ("RFC3339 expiration timestamp").
* Avoid restating type unless adding semantic nuance.

#### 🧪 Operation Example Coverage (New)

Every documented operation must include at least one concrete example (in either the request body content or any response content) to improve clarity for SDK code generators and human readers. This prevents ambiguous endpoints with only schemas.

Enforcement targets:
```bash
make openapi_example_coverage   # Fails if any operation lacks an example
make spec-contract              # Includes example coverage now
```

Coverage metrics added to `docs/openapi.coverage.json`:
* `operations_total` – Number of operations across all paths
* `operations_with_example` – Count of operations with ≥1 example
* `operation_example_coverage` – Ratio (must be 1.0)

Failure output lists missing operationIds (or `method:/path` if `operationId` absent) under `missing_operation_examples`.

Benefits:
* Facilitates accurate client generation (example payload scaffolds).
* Reduces onboarding friction; developers copy working payloads.
* Enables automated smoke testing using example fixtures (future enhancement).

Authoring Guidelines:
* Prefer realistic minimal payloads (avoid placeholder lorem ipsum).
* Keep examples small; show only critical fields.
* Use consistent timestamp formats (RFC3339) and representative IDs (`tok_123`).
* Reflect real state transitions or outcomes (e.g., revoked status, inactive token reasons).

Next potential gate: standard error examples for all non-2xx responses and multi-example semantic coverage (success vs failure vs edge cases).

#### 🧯 Error Response Example Coverage (New)

All 4xx/5xx responses must include at least one concrete example payload to document failure contract structure. This enforces that clients can parse standardized error fields reliably.

Local check:
```bash
make openapi_error_example_coverage
```

Included in aggregate contract:
```bash
make spec-contract
```

Coverage metrics added to `docs/openapi.coverage.json`:
* `error_responses_total` – Count of documented 4xx/5xx responses with bodies.
* `error_responses_with_example` – Those providing ≥1 example.

---
## 🔐 Cryptography & Token Signing (Beta Upgrade)

This beta now supports dual signing modes: legacy HMAC-SHA256 and modern Ed25519 (public key) via environment selection.

### Modes
| Mode | Env | Algorithm | Verification | JWKS Exposure |
|------|-----|-----------|--------------|---------------|
| HMAC (legacy) | `GAUTH_TOKEN_SIG_MODE=hmac` (default) | HS256 (manual JWT-like) | Shared secret | Metadata stub (oct) |
| HMAC + JWT lib | `GAUTH_USE_JWT_LIB=1` | HS256 / RS256 | Library path | RSA or HMAC key metadata |
| Ed25519 (public) | `GAUTH_TOKEN_SIG_MODE=eddsa` | EdDSA | Public key verify | OKP keys published |

Switch to Ed25519:
```bash
export GAUTH_TOKEN_SIG_MODE=eddsa
go run ./cmd/gauth-server
# JWKS now includes OKP keys with crv=Ed25519
curl -s localhost:8080/.well-known/jwks.json | jq
```

### JWKS (Unified)
Endpoint: `/.well-known/jwks.json`
Publishes:
* RSA (when `GAUTH_JWT_ALG=RS256` and `GAUTH_USE_JWT_LIB=1`)
* Ed25519 OKP keys (when `GAUTH_TOKEN_SIG_MODE=eddsa`)
* HMAC metadata stub otherwise (`kty=oct`, not a real public key)

Each Ed25519 key entry includes:
```json
{
   "kty": "OKP",
   "crv": "Ed25519",
   "alg": "EdDSA",
   "kid": "<short>",
   "use": "sig",
   "x": "<base64url-public-key>",
   "expires_at": "RFC3339"
}
```

### Discovery Additions
`/.well-known/gauth-configuration` now advertises:
* `eddsa_enabled` (boolean)
* `eddsa_keys` (list of key ids + expiry)
* `eddsa_rotation_hours` (configured TTL via `GAUTH_KEY_ROTATION_HOURS`)

### Key Rotation
Manual rotation utility:
```bash
export GAUTH_TOKEN_SIG_MODE=eddsa
go run ./cmd/rotate-key
# OR via Makefile target
make crypto-rotate
```
Rotating moves the current active key to history until its `expires_at`. Tokens signed before rotation remain valid until expiry (history retention).

### EdDSA Testing
Focused test run:
```bash
make crypto-test
```
Tests cover issuance, validation, tamper detection, unknown `kid`, and rotation validity.

### Backward Compatibility
HMAC tests confirm no regressions:
```bash
go test ./pkg/gauth -run TestHMAC
```
Legacy tokens remain valid when `GAUTH_TOKEN_SIG_MODE=hmac`. Switch modes intentionally—no silent migration.

### Environment Variables (Crypto)
| Variable | Purpose | Default |
|----------|---------|---------|
| `GAUTH_TOKEN_SIG_MODE` | `hmac` or `eddsa` | `hmac` |
| `GAUTH_KEY_ROTATION_HOURS` | Ed25519 key lifetime | `24` |
| `GAUTH_USE_JWT_LIB` | Enable jwt library path | unset |
| `GAUTH_JWT_ALG` | JWT algorithm (HS256/RS256) | HS256 (when lib enabled) |
| `GAUTH_JWKS_SIGNING_KEY` | Optional JWKS integrity HMAC | unset |
| `GAUTH_JWKS_SIGNING_KEY_ENABLED` | Enable JWKS signature headers | `0` |

### Roadmap
Planned enhancements:
* Persistent key store backend (filesystem or KMS adapter)
* Automatic rotation scheduler
* Key revocation list exposure
* Signature audit trail integration
* Detached token integrity proofs (hash-chained logs)

> Note: This implementation is for demonstration; do not rely on OKP key management here for production secrecy or compliance.

---
* `error_response_example_coverage` – Ratio; must be `1.0`.

Failure output lists `missing_error_response_examples` entries as `operationId:statusCode` (or `method:/path:code` if `operationId` absent).

Example minimal error payload:
```json
{
   "error": "VALIDATION_FAILED",
   "message": "missing scope",
   "code": 400,
   "trace_id": "trc_abc123"
}
```

Guidelines:
* Keep examples small and realistic (avoid lorem ipsum).
* Prefer canonical error identifiers (`VALIDATION_FAILED`, `NOT_FOUND`).
* Include a representative `trace_id` placeholder to illustrate correlation.
* One example per distinct status code is sufficient; add more only if semantics differ materially.

Why this gate matters:
* Enables SDKs to scaffold typed error handlers.
* Prevents undocumented shape changes from reaching clients silently.
* Improves human troubleshooting—copy/paste known failure example.

Future enhancements: differentiate semantic categories (validation vs permission), enforce structured error schema consistency, add success/failure parity reporting.

---
Set temporary variables instead of editing code:

```bash
export GAUTH_CLIENT_ID=demo-client
export GAUTH_CLIENT_SECRET=change-me-insecure
export GAUTH_SIGNING_KEY=change-me-also-32-bytes-minimum-please-1234
export GAUTH_TOKEN_EXPIRY_SECONDS=3600
```

Copy `.env.example` to `.env` (if you introduce a loader) and replace placeholders. Never commit real secrets.

---
## 🎯 What is GAuth?

GAuth (Gimel Authorization) is a next-generation authorization framework that goes beyond traditional RBAC/ABAC models. It's specifically designed for AI systems that need to act on behalf of humans or organizations with:

- **Explicit delegation** - Clear, verifiable power-of-attorney flows
---
## 🚀 Key Features

### Core Authorization
- 🔐 **Advanced Authentication** - JWT, PASETO token support
### Technical Excellence
- ⚡ **High Performance** - Circuit breakers, rate limiting, resilience patterns
### Developer Experience
- 📚 **Comprehensive Documentation** - Architecture guides, examples, patterns
---
## 🏗️ Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client/API    │───▶│   GAuth Core    │───▶│   Backend       │
│                 │    │                 │    │   Systems       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │  Audit & Events │
                    │    Storage      │
                    └─────────────────┘
```

### Repository Structure
```
├── cmd/                    # ✅ CLI applications and servers
│   └── gauth-server/      # Main GAuth server (WORKING)
├── pkg/                    # ✅ Core Go packages (13+ packages)
│   ├── gauth/             # ✅ Main GAuth interface & service
│   ├── token/             # ✅ Complete token management
│   ├── events/            # ✅ Event system with typed handlers
│   ├── resilience/        # ✅ Circuit breakers, rate limiting
│   ├── audit/             # ✅ Audit logging & compliance
│   ├── auth/              # ✅ Authentication core
│   ├── authz/             # ✅ Authorization logic
│   └── ...                # Additional specialized packages
├── internal/              # ✅ Internal resilience patterns
│   ├── circuit/           # ✅ Circuit breaker implementation
│   ├── ratelimit/         # ✅ Rate limiting implementation
│   └── resilience/        # ✅ Retry and bulkhead patterns
├── examples/              # ✅ 37+ working examples & tutorials
│   ├── typed_structures_demo/  # ✅ Full event system demo
│   ├── cascade/           # ✅ Resilience patterns demo
│   └── token/             # ✅ Advanced token management
├── test/                  # ✅ Comprehensive test suite
├── docs/                  # ✅ Complete documentation
└── scripts/               # ✅ Build & deployment automation
```

---
## 🚀 Quick Start

### Prerequisites
- **Go 1.24.0+** (beta demo validated with 1.24 toolchain)

### Option A: Run The Beta Web UI (Recommended for exploration)

```bash
# Run directly (defaults to :8080)
go run ./cmd/web-server

# Or choose a port
GAUTH_WEB_PORT=9090 go run ./cmd/web-server

# Visit in browser
open http://localhost:8080  # or chosen port
```

Key endpoints served by the beta UI server:

| Endpoint | Purpose |
|----------|---------|
| `/` | Minimal landing page with link to full UI |
| `/index.html` | Full beta single-page UI (embedded) |
| `/static/css/style.css` | Embedded stylesheet (served from binary) |
| `/static/js/app.js` | Embedded JavaScript (served from binary) |
| `/favicon.ico` | 1x1 transparent GIF to suppress 404s |
| `/api/v1/beta/health` | Health probe |
| `/api/v1/beta/info` | Build/info descriptor |
| `/api/v1/beta/examples/catalog` | Catalog of runnable examples |
| `/api/v1/beta/examples/run` (POST) | Start an example job |
| `/api/v1/beta/examples/run/{id}/status` | Poll job status |
| `/api/v1/beta/examples/run/{id}/logs` | SSE log stream (text/event-stream) |
| `/api/v1/beta/examples/run/jobs/{id}/cancel` (POST) | Cancel queued/running job |
| `/api/v1/poa/authorize` (POST) | Beta PoA authorization stub |
| `/api/v1/poa/metrics` | Simple PoA metrics counter |

### 📊 Beta Governance, Metrics & Export Usage

The beta web server exposes governance & observability endpoints with both JSON and CSV export modes (via `?format=csv`). These reset on restart unless persistence paths are configured.

Core governance endpoints:
| Endpoint | JSON Purpose | CSV Notes |
|----------|--------------|-----------|
| `/api/v1/beta/metrics/decision` | Decision metrics (action/resource/outcome + reason breakdown) | Deterministic header order |
| `/api/v1/beta/metrics/lifecycle` | Lifecycle transition & latency aggregates | Latency fields in nanoseconds |
| `/api/v1/audit/logs` | Recent audit trail entries | Pagination planned |
| `/api/v1/beta/capabilities` | Capability registry + action capability mapping | Includes current & previous registry hash |
| `/api/v1/beta/openapi/governance.yaml` | Governance OpenAPI fragment (YAML) | N/A |
| `/api/v1/beta/openapi/governance.json` | Governance OpenAPI fragment (JSON) | N/A |

CSV examples:
```bash
curl -s 'http://localhost:8080/api/v1/beta/metrics/decision?format=csv' -o decision_metrics.csv
curl -s 'http://localhost:8080/api/v1/beta/metrics/lifecycle?format=csv' -o lifecycle_timeline.csv
curl -s 'http://localhost:8080/api/v1/audit/logs?format=csv' -o audit_logs.csv
```

Health & build info:
```bash
curl -s http://localhost:8080/api/v1/beta/health | jq
curl -s http://localhost:8080/api/v1/beta/info | jq
```

Run with explicit port & healthcheck probe:
```bash
GAUTH_WEB_PORT=8090 go run ./cmd/web-server &
go run ./cmd/web-server -healthcheck   # exits 0 if healthy
```

#### Environment Variables (Web Server & Governance)
| Variable | Default | Purpose |
|----------|---------|---------|
| `GAUTH_WEB_PORT` | `8080` | Bind port for web server |
| `GAUTH_VIOLATION_PERSIST_PATH` | unset | File path for violation counter persistence (hash-chained) |
| `GAUTH_SEMANTIC_PERSIST_PATH` | unset | File path for semantic counters + anomaly EWMA persistence |
| `GAUTH_VIOLATION_AUTOSAVE_SEC` | unset | Autosave interval (seconds) for violation counters |
| `GAUTH_SEMANTIC_AUTOSAVE_SEC` | unset | Autosave interval (seconds) for semantic counters |
| `GAUTH_CAPABILITIES_PATH` | unset | Capability registry JSON file (versioned) |
| `GAUTH_CAPABILITY_ENFORCE` | `0` | Enable capability enforcement when set to `1` |
| `GAUTH_SEMANTIC_HISTORY_DISABLE` | `0` | Disable semantic history (rate calculations) when `1` |
| `GAUTH_VIOLATION_PERSIST_NO_THROTTLE` | `0` | Disable 5s persistence throttle (violations) |
| `GAUTH_SEMANTIC_PERSIST_NO_THROTTLE` | `0` | Disable 5s persistence throttle (semantics) |
| `GAUTH_TOKEN_SIG_MODE` | `hmac` | `hmac` or `eddsa` token signing mode |
| `GAUTH_KEY_ROTATION_HOURS` | `24` | Ed25519 key rotation TTL (hours) |
| `GAUTH_STH_PERSIST_PATH` | unset | Persist signed tree heads (revocation transparency) |
| `GAUTH_SEMANTIC_ANOMALY_BG_SEC` | unset | Background interval for anomaly EWMA updates |

Integrity verification endpoints:
```bash
curl -s http://localhost:8080/api/v1/beta/violations/persistence/verify | jq
curl -s http://localhost:8080/api/v1/beta/semantics/persistence/verify | jq
```
Status codes: `ok`, `mismatch`, `legacy`, `unconfigured`.

Minimal persistent demo:
```bash
export GAUTH_WEB_PORT=8080
export GAUTH_VIOLATION_PERSIST_PATH=./state/violations.json
export GAUTH_SEMANTIC_PERSIST_PATH=./state/semantics.json
mkdir -p state
go run ./cmd/web-server
```

> Planned enhancements: pagination for audit CSV (Accept header CSV negotiation implemented).

#### Lifecycle Timeline Pagination (New)

The lifecycle timeline endpoint now supports cursor-based pagination with filtering and CSV export.

Endpoint: `/api/governance/lifecycle_timeline`

Parameters:
| Name | Type | Description |
|------|------|-------------|
| `entity_type` | string | Optional filter by entity type (`token` or `delegation`) |
| `entity_id` | string | Optional filter by specific entity id |
| `since` | unix seconds | Lower bound timestamp (events before ignored) |
| `outcome` | string | Filter by outcome (`success`,`failure`,`noop`) |
| `reason` | string | Filter by reason code (`init`,`status_change`,`invalid_transition`, etc.) |
| `limit` | int | Page size (default 100, max 250) |
| `cursor` | string | Exclusive event id cursor from previous page's `next_cursor` |
| `format` | string | `csv` for CSV (alternatively use `Accept: text/csv`) |

JSON Response Fields:
```jsonc
{
   "success": true,
   "events": [ { "id": "evt_123", "entity_type": "token", ... } ],
   "count": 50,
   "next_cursor": "evt_073"   // empty when no more pages
}
```

Pagination Semantics:
* Results are ordered newest first.
* The `cursor` is the last `id` from the previous page's `next_cursor` (exclusive); it will not appear again.
* If `next_cursor` is empty you've reached the end.

Example paging loop:
```bash
BASE="http://localhost:8080/api/governance/lifecycle_timeline?limit=50"
cursor=""
while true; do
   url="$BASE"; [ -n "$cursor" ] && url="$url&cursor=$cursor"
   page=$(curl -s "$url")
   count=$(echo "$page" | jq '.count')
   next=$(echo "$page" | jq -r '.next_cursor')
   echo "Fetched $count events; next=$next"
   [ -z "$next" ] && break
   cursor="$next"
done
```

CSV export (same filters):
```bash
curl -H 'Accept: text/csv' "http://localhost:8080/api/governance/lifecycle_timeline?limit=25" -o lifecycle_page1.csv
```

The CSV form does not include `next_cursor`; request JSON when iterating pages.

#### Capability Registry External Anchoring (Prototype)

The capability registry integrity hash (`capability_registry_hash`) can now be externally anchored via a prototype in-memory anchoring provider.

Enable anchoring:
```bash
export GAUTH_ANCHOR_PROVIDER=memory          # initialize memory anchor client
export GAUTH_CAPABILITY_ANCHOR_ENABLE=1      # allow capability anchoring POST
export GAUTH_ANCHOR_PERSIST_PATH=./state/anchors.json   # optional persistence (chronological JSON)
```

Endpoints:
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/beta/capabilities/anchor` | Anchor current capability registry hash (idempotent) |
| GET  | `/api/v1/beta/capabilities/anchor/latest` | Retrieve latest anchored hash + timestamp |

POST Response:
```jsonc
{
   "success": true,
   "hash": "sha256:...",
   "anchored_at": "2025-10-19T12:34:56Z",
   "total": 3,
   "previous_hash": "sha256:<prev>",        // optional (when registry changed)
   "registry_last_changed_at": "2025-10-19T12:30:00Z" // optional
}
```

GET Latest Response (when anchored):
```jsonc
{
   "success": true,
   "anchored": true,
   "latest": {"hash": "sha256:...", "anchored_at": "2025-10-19T12:34:56Z"},
   "total": 3,
   "capability_registry_hash": "sha256:..."
}
```

Behavior & Guarantees:
* Deduplicated: re-anchoring the same hash returns original timestamp.
* Chronological ordering preserved internally.
* Persistence file format:
```jsonc
{"anchors":[{"hash":"sha256:abc","anchored_at":"2025-10-19T12:30:00Z"}, {"hash":"sha256:def","anchored_at":"2025-10-19T12:34:56Z"}]}
```
* Discovery document (`/.well-known/gauth-configuration`) now includes `anchoring.latest_hash` and `anchoring.total` when provider enabled.

Planned Enhancements:
* External timestamp / notarization provider integration (e.g. transparency log).
* Capability audit stream anchoring (record-level hash chaining anchored periodically).
* Inclusion proofs (Merkle commitments) for capability snapshots.
* Fuzz/property tests for canonical hash stability.

Security Caveats:
* Memory-only unless persistence enabled.
* No external time attestation (system clock trusted).
* Anchoring currently covers registry hash only (not audit stream).

Usage Example:
```bash
GAUTH_ANCHOR_PROVIDER=memory GAUTH_CAPABILITY_ANCHOR_ENABLE=1 go run ./cmd/web-server &
sleep 1
curl -s localhost:8080/api/v1/beta/capabilities/anchor | jq
curl -s localhost:8080/api/v1/beta/capabilities/anchor/latest | jq
```

Disable anchoring (server returns 403 on POST):
```bash
unset GAUTH_CAPABILITY_ANCHOR_ENABLE
```

Anchoring adds governance integrity by providing an append-only timeline of published capability sets, enabling external verification of stable capability governance over time.

#### Capability Version Negotiation & Deprecation Metadata (Prototype)

Multi-version capability metadata is now supported via an optional `versions` slice per capability (the single `version` remains for backward compatibility). The server exposes discovery metadata at `/api/v1/beta/info` including each capability's `versions`, `stable`, `deprecated_after`, and `sunset_after` fields.

Negotiation Endpoint:
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/beta/capabilities/negotiate` | Computes agreed versions across client provided capability version lists |

Request Body:
```jsonc
{
   "client_versions": {
      "cap.transfer": ["1.0", "1.1"],
      "cap.issue": ["1.0"],
      "cap.unknown": ["1.0"]
   }
}
```

Response:
```jsonc
{
   "success": true,
   "agreed": {"cap.transfer": "1.0", "cap.issue": "1.0"},
   "unsupported": {"cap.unknown": ["1.0"]}
}
```

Negotiation Rules:
* Server builds the set of supported versions for each capability from `version` + `versions`.
* First matching client version in order is selected (simple intersection-first strategy).
* Unsupported capabilities (not present on server or lacking intersecting versions) are returned with their original proposed list.

Deprecation Lifecycle Fields:
* `deprecated_after` – indicates the earliest timestamp after which capability use is discouraged.
* `sunset_after` – indicates planned removal (clients should stop relying on capability before this time).

These fields are informational in the prototype (no automatic enforcement yet). Planned: discovery will surface deprecation window warnings and negotiation will optionally filter out sunset capabilities when an enforcement flag is enabled.

#### Capability Audit Hash Chain Verification (Prototype)

Capability-related audit events (`delegation_create`, `delegation_revoke`, and denied capability enforcement `capability_enforce`) are now wrapped in a lightweight hash chain when the server is configured with a persistence path. Each chained event stores:
* `prev_hash` – previous chain tip (empty on first event)
* `hash` – SHA256 of canonical event JSON payload
* `timestamp` – UTC RFC3339NANO when persisted

Verification Endpoint:
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/beta/capabilities/audit/verify` | Returns latest chain tip and integrity status |

Example Response:
```jsonc
{
   "success": true,
   "configured": true,
   "latest": {
      "hash": "sha256:...",
      "prev_hash": "sha256:...",
      "timestamp": "2025-10-19T12:33:45.123456789Z"
   },
   "chain_tip": "sha256:...",
   "integrity_ok": true
}
```

To enable persistence of the chain tip (prototype single-file snapshot):
```bash
export GAUTH_CAP_AUDIT_PERSIST_PATH=./state/cap_audit_tip.json
```
(Until full env wiring lands, this may require direct field assignment in code for experimentation.)

Anchoring:
You can externally anchor the current audit chain tip hash (latest capability-related event) using:
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/beta/capabilities/audit/anchor` | Anchor current capability audit chain tip hash |

Response example:
```jsonc
{
   "success": true,
   "hash": "sha256:...",          // anchored chain tip hash (idempotent)
   "anchored_at": "2025-10-19T12:40:00Z",
   "total": 4,                      // total anchors (registry + audit tip anchors share provider)
   "chain_tip": "sha256:...",      // current chain tip
   "type": "capability_audit_chain_tip"
}
```

Planned Enhancements:
* External timestamp transparency log provider
* Merkle subtree commitments for batched events
* Streaming endpoint with inclusion proof verification

Security Caveats:
* Single-file snapshot rewrites latest tip only (not append-only yet)
* No external notarization; system clock trusted
* Chain presently covers a subset of audit actions (capability-related only)


Security hardening (beta scope):
* Per-request Content Security Policy with runtime nonce for inline scripts (`script-src 'nonce-<value>'`).
* All static assets are embedded via `go:embed` (no runtime file dependency).
* Minimal favicon avoids noisy 404s and keeps linter happy.

Example API usage:

```bash
# List examples
curl -s http://localhost:8080/api/v1/beta/examples/catalog | jq

# Start an example
JOB_ID=$(curl -s -X POST -H 'Content-Type: application/json' \
   -d '{"id":"gauth_protocol_basics:minimal_poa"}' \
   http://localhost:8080/api/v1/beta/examples/run | jq -r .job_id)

# Poll status
curl -s http://localhost:8080/api/v1/beta/examples/run/$JOB_ID/status | jq

# Stream logs (Server-Sent Events)
curl -N http://localhost:8080/api/v1/beta/examples/run/$JOB_ID/logs
```

Cancel a job (if still running):
```bash
curl -X POST http://localhost:8080/api/v1/beta/examples/run/jobs/$JOB_ID/cancel
```

Environment variable summary (web UI server):
| Variable | Default | Description |
|----------|---------|-------------|
| `GAUTH_WEB_PORT` | `:8080` | Bind address/port (may be set without leading colon, code normalizes) |

---

### Option B: Build Core Server & Packages
### Installation & Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0.git
   cd GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0
   ```

2. **Build the project**
   ```bash
   go build -o bin/gauth-server ./cmd/gauth-server
   ```

3. **Run working examples**
   ```bash
   # Test advanced token revocation (WORKING ✅)
   cd examples/token/advanced_revocation_flow && go test -v
   
   # Run typed structures demo (WORKING ✅)
   cd examples/typed_structures_demo && go build && ./typed_structures_demo
   
   # Test cascade resilience patterns (WORKING ✅)
   cd examples/cascade && go build
   ```

4. **Verify core functionality**
   ```bash
   # All these work and pass tests ✅
   go test ./pkg/gauth/...
   go test ./pkg/token/...
   go test ./pkg/events/...
   ```

### Security Testing
```bash
# Build security test utility
make build-security-test
# Run comprehensive security tests
./build/bin/security-test
```

---
## 📚 Documentation

| Document | Description |
|----------|-------------|
| [Log Streaming](docs/LOG_STREAMING.md) | How to stream job logs via SSE |

| Document | Description |
|----------|-------------|
| [Getting Started](docs/GETTING_STARTED.md) | Quick setup and basic usage |
| [Architecture](docs/ARCHITECTURE.md) | System design and components |
| [Patterns Guide](docs/PATTERNS_GUIDE.md) | Development patterns & best practices |
| [API Reference](docs/API_REFERENCE.md) | Complete API documentation |
| [Testing Guide](docs/TESTING.md) | Testing strategies and examples |
| [Examples](examples/) | Code samples and tutorials |
| [Disclaimer](DISCLAIMER.md) | Consolidated NOT production ready notice & missing controls |

---
## 🐳 Deployment Options

### 🚀 **Beta Deployment**
```bash
# Build demonstration server 
go build -o bin/gauth-server ./cmd/gauth-server

# Run beta examples
cd examples/typed_structures_demo && go build
cd ../cascade && go build  
cd ../token/advanced_revocation_flow && go test -v

# Docker for beta environment
docker build -t gauth-beta:latest .
```

**⚠️ Important**: Beta implementation only – **NOT production ready**. See `DISCLAIMER.md`.

### Kubernetes
```bash
# Apply manifests (coming soon)
kubectl apply -f k8s/
```

---
## 🧪 Testing & Verification

### 🧪 **Beta Testing**
```bash
# Run beta test suite
./scripts/run_functional_tests.sh

# Beta verification
./scripts/verify_beta_implementation.sh

# Core packages testing
go test ./pkg/...

# Example tests for learning
cd examples/token/advanced_revocation_flow && go test -v  
cd examples/typed_structures_demo && go build
cd examples/cascade && go build
```

**📚 Note**: These tests demonstrate GAuth concepts and are for beta purposes.

**Note**: See [Testing Guide](docs/TESTING_GUIDE.md) for comprehensive testing information.

### 🔍 Revocation Transparency Verification (Programmatic)
You can validate revocation integrity (Merkle inclusion + Signed Tree Head multi-sig + optional consistency) directly from Go code using `pkg/verification` without shelling out to the CLI.

Minimal example:
```go
package main
import (
   "net/http"
   "log"
   verification "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/verification"
)
func main() {
   base := "http://localhost:8080"
   if err := verification.VerifyAll(&http.Client{}, base, ""); err != nil {
      log.Fatalf("verification failed: %v", err)
   }
   log.Println("revocation transparency verified ✅")
}
```

Granular API (all return structured data or errors):
* `FetchDiscovery(client, base)` – discovery + latest STH metadata
* `FetchEvents(client, base)` – chain events list
* `FetchProofByHash(client, base, hash)` – Merkle inclusion proof
* `LoadJWKS(client, base)` – imports public Ed25519 keys for signature verification
* `VerifyInclusion(hash, proof)` – Merkle inclusion check
* `VerifySTHMultiSig(sth)` – single or weighted multi-sig validation (`threshold_not_met`, `signature_invalid_<i>`)
* `FetchConsistency(client, base, start)` + `VerifyConsistency(cons, hashes)` – append-only growth proof (prototype)

Typical failure errors (see docs section 15): `no_events`, `proof endpoint failure`, `inclusion_failed`, `sth_verify: threshold_not_met`, `sth_verify: signature_invalid_1: unknown_kid`, `new_leaves_mismatch`.

CLI alternative:
```bash
go run ./cmd/verify --base http://localhost:8080 --hash <optional_event_hash>
```

For detailed semantics see `docs/REVOCATION_TRANSPARENCY.md`.

#### Structured Verification Errors
`VerifyAll` and granular functions return regular `error` values, but many are instances of `*verification.VerifyError` which carry `Code`, `Detail`, and the original `Cause` (if any). Pattern match safely using `errors.As`:

```go
if err := verification.VerifyAll(client, base, ""); err != nil {
   var vErr *verification.VerifyError
   if errors.As(err, &vErr) {
      switch vErr.Code {
      case "inclusion_failed":
         // integrity breach: log & alert
      case "proof_endpoint_failure":
         // transient or wrong hash provided
      case "sth_verify":
         // multi-sig failure (threshold_not_met or signature_invalid detail)
      case "no_events":
         // empty chain – not an integrity failure
      }
   }
}
```
Codes are documented in `docs/REVOCATION_TRANSPARENCY.md` Section 15; legacy substring checks still work because `Error()` formats `code: detail`.

---
## 📡 Job Log Streaming (Experimental)

You can stream the output of any executed example job in real time using the built-in Server-Sent Events (SSE) endpoint. See [docs/LOG_STREAMING.md](docs/LOG_STREAMING.md) for details and usage examples.

**Quick usage:**
1. Run an example via the web UI or API (returns a job ID)
2. Paste the job ID into the "Live Job Log Streaming" panel on the main page
3. Or use:
   ```bash
   curl -N http://localhost:8080/api/v1/beta/examples/run/JOB_ID/logs
   ```
4. See the docs for event types and frontend integration tips.

---
## 🤝 Contributing

We welcome contributions! Whether you're:
- 🐛 **Fixing bugs** or improving performance
**Get started:**
1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

See [CONTRIBUTORS.md](CONTRIBUTORS.md) for detailed guidelines.

---
## 📄 Legal & Licensing

- **License**: Apache License 2.0 - see [LICENSE](LICENSE)
### Legal Framework
This implementation complies with:
- **GiFo RFC 0080** - Legal Provisions for Gimel Foundation
For complete legal provisions, visit [gimelfoundation.com](https://gimelfoundation.com)

---
## 🔗 Resources

- 🌐 **RFC Specifications**: [gimelfoundation.com](https://gimelfoundation.com)
---
*Built with ❤️ for the future of AI authorization and transparency*
