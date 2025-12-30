---
title: AgentAuth Complete API Reference
 category: api-reference
 status: active
 lastUpdated: 2025-11-12
 owners: architecture-team
 refreshCadence: on-change
 source: source-code
 ---
# AgentAuth 1.0 Complete API Reference

 > Last Updated: 2025-10-19
> Status: Active

**Official AgentAuth Community Implementation - Combined Library & Web API Documentation**

> Note: This reference now includes the Decision Metrics governance endpoint (`/api/v1/beta/metrics/decisions`) documenting structured lifecycle decision aggregation.

Complete API reference for AAP-RFC-0111 (AgentAuth 1.0) and AAP-RFC-0115 (PoA Definition) implementation, including both the Go library API and the web demonstration API.

## 📋 **Table of Contents**

1. [Web Demo API](#web-demo-api)
2. [Go Library API](#go-library-api)
3. [Data Types Reference](#data-types-reference)
4. [Error Handling](#error-handling)
5. [Examples](#examples)

## 🛡️ Governance Endpoints Summary

Core observability & integrity surfaces for beta governance:

| Endpoint | Purpose | Key Fields | Filters | Determinism |
|----------|---------|-----------|---------|-------------|
| [`GET /api/v1/beta/metrics/decisions`](#decision-metrics) | Decision & reason counters | `action, resource, outcome, reason, count` | n/a | action→resource→outcome→reason lexicographic |
| [`GET /api/v1/beta/lifecycle/timeline`](#lifecycle-timeline) | Recent lifecycle transition events | `entity_type, entity_id, old_status, new_status, outcome, reason, latency_ns, at` | `entity_type`, `entity_id` | Desc timestamp; tie-break tuple |
| [`GET /api/v1/beta/capabilities`](#capability-registry) | Capability registry + integrity hashes | `capability_registry_hash, capability_registry_prev_hash, capability_registry_last_changed_at, capabilities[]` | n/a | Lexicographic capability id ordering for hash |
| [`GET /api/v1/audit/logs`](#audit-logs) | Recent audit decision entries | `id, action, resource, outcome, reason, at` | `limit`, `since` | Desc timestamp; id tie-break |

Reason code semantics table appears under Decision Metrics; lifecycle & audit sections reuse those classifications.

### 📁 Machine-Readable Schemas

JSON Schema files for governance endpoints live in `docs/schema/`:
`decision_metrics.schema.json`, `lifecycle_timeline.schema.json`, `capabilities_registry.schema.json`, `audit_logs.schema.json`.

Validator Notes: Uses draft 2020-12. If your validator warns about meta-schema features, you can still rely on structural field validation. Enums codify current reason/status vocabularies—treat additions as non-breaking if you allow unknown strings.

---

## 🌐 **Web Demo API**

The web demonstration API provides REST endpoints for testing and demonstrating RFC-0111 and RFC-0115 functionality through a web interface.

### **Base URL**
```
http://localhost:8080
```

### **Authentication**
No authentication required for demo API endpoints.

### **Content Type**
All API requests and responses use `application/json`.

---

### **📋 Demo Scenarios**
### 🔎 Beta Metrics (Governance) – Decision Counters

<a id="decision-metrics"></a>
#### GET /api/v1/beta/metrics/decisions

Deterministically returns labeled decision counters and reason counters captured during lifecycle operations (e.g. `token_status_update`, `delegation_status_update`). Useful for governance audits, anomaly detection, and reproducible diffing.

Minimal curl example:
```bash
curl -s http://localhost:8080/api/v1/beta/metrics/decisions | jq .
```

Response:
```json
{
  "success": true,
  "decisions": {
    "counts": [
      {"action":"token_status_update","resource":"token:abc123","outcome":"suspended","count":2},
      {"action":"delegation_status_update","resource":"delegation:poa_9","outcome":"active","count":1}
    ],
    "reasons": [
      {"action":"token_status_update","resource":"token:abc123","outcome":"suspended","reason":"status_change","count":1},
      {"action":"token_status_update","resource":"token:abc123","outcome":"suspended","reason":"noop","count":1}
    ]
  }
}
```

Fields:
* `action` – high‑level operation category.
* `resource` – namespaced identifier (`token:<id>` / `delegation:<id>`).
* `outcome` – resulting status or outcome label (e.g. `active`, `suspended`, `terminated`, `success`, `noop`).
* `count` – aggregated counter value.
* `reason` – additional classification (`status_change`, `init`, `noop`, `maintenance`, `rate_limited`, `policy_violation`, plus lifecycle-specific: `invalid_transition`, `unsupported_status`, `invalid_payload`, `not_found`).

Reason Code Semantics:
| Code | Category | Description |
|------|----------|-------------|
| `init` | Lifecycle | First time entity status is set (old status sentinel `_`). |
| `status_change` | Lifecycle | Successful mutation from one status to another. |
| `noop` | Lifecycle | Requested status equals current; idempotent. |
| `invalid_transition` | Lifecycle | Disallowed transition (e.g., resurrect terminated). |
| `unsupported_status` | Lifecycle | Status outside allowed whitelist. |
| `invalid_payload` | Validation | Malformed request body / schema mismatch. |
| `not_found` | Lookup | Target token/delegation not found. |
| `maintenance` | System State | Decision influenced by maintenance window gating. |
| `rate_limited` | Throttling | Action blocked or altered due to rate limiting flag. |
| `policy_violation` | Policy | Action denied or altered due to policy engine result. |

Deterministic Ordering Details:
1. Sort keys lexicographically: `action` → `resource` → `outcome` → (`reason` when present).
2. Stable slice materialization prevents hash-order dependence.
3. Tie-break on count (ascending) ensures multi-run diff stability even if equal labels inserted concurrently.
4. Enables reproducible governance diffs and external anomaly detection using simple JSON equality.

Ordering: Stable lexicographic ordering by action, resource, outcome (and reason for reasons array) plus count tie‑breaker for deterministic output and clean diffs.

Unavailable Case: If the in‑memory metrics implementation is not active, returns `{"success":true,"decisions":{"available":false}}`.

Environment Influence:
* `GAUTH_MAINTENANCE_WINDOW=1` may produce `reason=maintenance`.
* `GAUTH_RATE_LIMITED=1` may produce `reason=rate_limited`.
* `GAUTH_POLICY_VIOLATION=1` converts status change reasons to `policy_violation`.

Export Formats:
All governance list endpoints support an optional `?format=csv` parameter for lightweight ingestion:
* Decisions: `curl -s "http://localhost:8080/api/v1/beta/metrics/decisions?format=csv"`
* Lifecycle Timeline: `curl -s "http://localhost:8080/api/v1/beta/lifecycle/timeline?format=csv"`
* Audit Logs: `curl -s "http://localhost:8080/api/v1/audit/logs?format=csv"`

CSV Schemas (headers):
* Decisions: `section,action,resource,outcome,reason,count` (section is `counts` or `reasons`; reason blank for counts rows)
* Lifecycle Timeline: `entity_type,entity_id,old_status,new_status,outcome,reason,latency_ns,at`
* Audit Logs: `id,at,actor,action,resource,outcome,reason` (reason extracted from meta when available)

Governance Use Cases:
* Detect spikes in `noop` decisions (potential redundant client requests).
* Track initialization (`init`) rates for delegation onboarding cadence.
* Feed structured counts into anomaly detectors (EWMA / rate analysis) without additional sorting.

<a id="lifecycle-timeline"></a>
### ⏱️ Lifecycle Transition Timeline

#### GET /api/v1/beta/lifecycle/timeline

Returns a recent window of lifecycle status transition events (token & delegation) including latency measurements and reason codes. Supports lightweight filtering.

Minimal curl examples:
```bash
# All recent events (default limit)
curl -s http://localhost:8080/api/v1/beta/lifecycle/timeline | jq .

# Filter by entity type
curl -s "http://localhost:8080/api/v1/beta/lifecycle/timeline?entity_type=token" | jq .

# Filter by specific id
curl -s "http://localhost:8080/api/v1/beta/lifecycle/timeline?entity_id=abc123" | jq .
```

Query Parameters:
* `entity_type` (optional) – `token` | `delegation`; if omitted returns both.
* `entity_id` (optional) – Specific id to filter events for that entity only.

Response (example):
```json
{
  "success": true,
  "events": [
    {"entity_type":"token","entity_id":"abc123","old_status":"active","new_status":"suspended","outcome":"success","reason":"status_change","latency_ns":210456,"at":"2025-10-19T14:27:11Z"},
    {"entity_type":"delegation","entity_id":"poa_9","old_status":"_","new_status":"active","outcome":"success","reason":"init","latency_ns":90412,"at":"2025-10-19T14:27:05Z"}
  ],
  "count": 2
}
```

Event Fields:
* `entity_type` – `token` or `delegation`.
* `entity_id` – Identifier of the entity.
* `old_status` – Previous lifecycle state; initialization uses `_` sentinel.
* `new_status` – Target state applied or requested.
* `outcome` – `success` | `failure` | `noop` (state unchanged) | future additional.
* `reason` – Reason code (see decision metrics list; adds lifecycle context like `invalid_transition`, `unsupported_status`).
* `latency_ns` – Processing duration for the transition handler (nanoseconds).
* `at` – RFC3339 timestamp when recorded.
* `count` – Convenience total returned.

Reason Code Notes:
* `_` sentinel on `old_status` distinguishes initialization from a missing field.
* `invalid_transition` signals guard rejection (e.g., resurrecting terminated token).
* `unsupported_status` for payload requesting non-whitelisted states.
* `noop` indicates idempotent request (requested = current).

Deterministic Ordering:
* Sorted descending by `at` then by `entity_type|entity_id` tuple.
* Stable formatting for audit diffing.

Use Cases:
* Measure suspension cycles & churn.
* Investigate latency spikes for lifecycle operations.
* Policy impact review (failure vs success distribution).

Edge Cases:
* No events yet: returns empty `events` array with `count=0`.
* Extremely large latency values may indicate blocking I/O or instrumentation issues.

<a id="capability-registry"></a>
### 🧩 Capability Registry Integrity

#### GET /api/v1/beta/capabilities

Returns the currently loaded capability definitions plus governance integrity metadata (stable hash, previous hash, last changed timestamp). Hash values provide deterministic content-addressing for the ordered registry; changes allow external monitoring for drift.

Minimal curl example:
```bash
curl -s http://localhost:8080/api/v1/beta/capabilities | jq .
```

Response (example):
```json
{
  "success": true,
  "capability_registry_hash": "9f5d2c...",
  "capability_registry_prev_hash": "18ae77...",
  "capability_registry_last_changed_at": "2025-10-19T14:22:11Z",
  "capabilities": [
    {"id":"token.lifetime","version":"1.0.0","stable":true},
    {"id":"delegation.multi_sig","version":"1.0.0","stable":false}
  ],
  "count": 2
}
```

Fields:
* `capability_registry_hash` – SHA-256 (domain-separated) hash of the canonical ordered registry serialization.
* `capability_registry_prev_hash` – Prior stable hash (or empty/omitted if first load).
* `capability_registry_last_changed_at` – RFC3339 timestamp of last mutation or reload affecting hash.
* `capabilities[]` – Array of loaded capability descriptors.
  * `id` – Unique capability identifier (namespaced).
  * `version` – SemVer-like version string of the capability spec.
  * `stable` – Boolean signaling stable vs experimental capability.
* `count` – Convenience count of array length.

Determinism & Governance:
* Registry serialized with lexicographic ordering of `id` ensuring cross-instance hash parity.
* Any addition/removal or version change mutates the hash; external monitors can alert on unexpected drift.
* Previous hash retained to enable lightweight integrity chain comparisons and anchoring strategies.

Error / Edge Cases:
* On startup prior to initial load: may return `prev_hash` empty and a reduced capability set.
* If capability loading fails internally: `success` may be true with empty array and placeholder hashes.

Use Cases:
* Compliance attestation snapshots (capture hash + list alongside deployment artifacts).
* Diff tooling: compare current vs previous hash to trigger audit export.
* Capability lifecycle governance (track experimental → stable transitions).

### 🧾 Notarization Receipt Chain Verification (Prototype)

#### GET /api/v1/beta/notarization/receipts/verify

Performs integrity verification of the persisted capability anchor notarization receipt chain. Each receipt is linked via `prev_hash` and a deterministic `chain_hash = sha256(prev_hash || canonical_base_receipt_json)`.

Related Endpoints:
* `GET /api/v1/beta/notarization/receipts/latest` – latest stored receipt object (includes linkage fields).
* `GET /api/v1/beta/notarization/receipts` – array of receipts with `hash`, `timestamp`, `provider`, `prev_hash`, `chain_hash`.

Minimal curl examples:
```bash
curl -s http://localhost:8080/api/v1/beta/notarization/receipts/verify | jq .
curl -s http://localhost:8080/api/v1/beta/notarization/receipts | jq .
```

Success (integrity ok) response:
```json
{
  "success": true,
  "configured": true,
  "integrity": "ok",
  "total": 3,
  "chain_head": "a3e7c8..."
}
```

Mismatch response (first invalid entry diagnostic):
```json
{
  "success": true,
  "configured": true,
  "integrity": "mismatch",
  "total": 3,
  "details": {
    "mismatch_index": 0,
    "expected": "0a9c...",
    "stored": "deadbeef...",
    "prev_expected": "",
    "prev_stored": ""
  }
}
```

Empty chain response:
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
| `integrity` | `ok` (all chain_hash values recompute), `mismatch` (first failure detected), `empty` (no entries), `unconfigured` (persistence disabled) |
| `chain_head` | Last entry chain_hash (only on ok) |
| `mismatch_index` | Zero-based index of failing entry (on mismatch) |
| `expected` / `stored` | Recomputed vs stored chain hash for failing entry |
| `prev_expected` / `prev_stored` | Linkage validation for failing entry predecessor |

Environment Configuration (Notarization Prototype):
| Env | Purpose | Default |
|-----|---------|---------|
| `GAUTH_CAP_ANCHOR_NOTARIZE` | Enable notarization emission path | unset |
| `GAUTH_CAP_ANCHOR_NOTARY_PROVIDER` | Provider selection (`memory`,`external_stub`) | `memory` |
| `GAUTH_NOTARY_RECEIPT_PERSIST_PATH` | Receipt chain persistence file | unset |
| `GAUTH_NOTARY_RECEIPT_VERIFY_INTERVAL` | Background verification interval (seconds, >=30) | 120 |

Metrics:
* `capability_anchor_notarization_receipts_integrity` gauge: ok=1 mismatch=0 unconfigured/legacy/empty=-1. This gauge is now also mirrored in the custom endpoint `/api/v1/beta/capabilities/anchor/metrics/prometheus` (HELP/TYPE + current value) for environments scraping only that minimal surface; full labeled metrics remain available via the standard `/metrics` registry scrape.
* Latency histograms: `capability_anchor_notarization_latency_seconds` and provider-labeled variant.
* Failure counters: `capability_anchor_notarization_failures_total` (+ provider labeled).
* Age gauge: `capability_anchor_notarized_age_seconds` (time since last successful receipt).
 * On-demand verification trigger: append `?verify=1` to `/api/v1/beta/capabilities/anchor/metrics/prometheus` to force a recomputation of receipt chain integrity even if a recent verification occurred. Useful for ad-hoc probes or post-maintenance spot checks.
 * Freshness auto-check: the custom Prometheus endpoint automatically re-verifies if the last verification timestamp is older than `GAUTH_NOTARY_RECEIPT_VERIFY_FRESHNESS_SECONDS` (default 120s). Override by setting that env var to a positive integer seconds value.

Alerting Recommendation:
Alert when integrity gauge == 0 (mismatch) for >=2 consecutive scrapes; suppress alerts for -1 (unconfigured/empty/legacy).

Security Notes:
* Prototype provider does not confer external cryptographic trust; replace with TSA/transparency provider before marking GAP_MATRIX external anchoring items as implemented.
* Persistence file permissions should restrict access (created with 0600).

Testing:
* `web/notarization_receipt_persistence_test.go` verifies append-only linkage.
* `web/notarization_receipt_verification_test.go` exercises integrity endpoint (ok + tamper mismatch).
* Provider-labeled metrics tested in `web/capability_anchor_notarization_provider_metrics_test.go`.



#### **GET /scenarios**
### � Governance JSON Schemas (Reference)
### 📦 Governance JSON Schemas (Reference)
Canonical JSON structure excerpts (simplified) for machine consumers. Field ordering shown is illustrative; treat objects as unordered except where arrays are explicitly ordered.

Decision Metrics Snapshot (`GET /api/v1/beta/metrics/decisions`):
```jsonc
{
  "success": true,
  "decisions": {
    "counts": [
      {"action": "token_status_update", "resource": "token:<id>", "outcome": "active|suspended|terminated|success|noop", "count": 1}
    ],
    "reasons": [
      {"action": "token_status_update", "resource": "token:<id>", "outcome": "active|suspended|terminated|success|noop", "reason": "init|status_change|noop|maintenance|rate_limited|policy_violation|invalid_transition|unsupported_status|invalid_payload|not_found", "count": 1}
    ]
  }
}
```

Lifecycle Timeline Event (`GET /api/v1/beta/lifecycle/timeline`):
```jsonc
{
  "entity_type": "token|delegation",
  "entity_id": "<string>",
  "old_status": "_|active|suspended|terminated",
  "new_status": "active|suspended|terminated",
  "outcome": "success|failure|noop",
  "reason": "init|status_change|noop|invalid_transition|unsupported_status|invalid_payload|not_found|maintenance|rate_limited|policy_violation",
  "latency_ns": 123456,
  "at": "2025-10-19T14:27:11Z"
}
```

Capability Registry Snapshot (`GET /api/v1/beta/capabilities`):
```jsonc
{
  "success": true,
  "capability_registry_hash": "<sha256 hex>",
  "capability_registry_prev_hash": "<sha256 hex>|",
  "capability_registry_last_changed_at": "2025-10-19T14:22:11Z",
  "capabilities": [
    {"id": "token.lifetime", "version": "1.0.0", "stable": true}
  ],
  "count": 1
}
```

Audit Log Entry (`GET /api/v1/audit/logs`):
```jsonc
{
  "id": "evt_1042",
  "action": "token_status_update",
  "resource": "token:<id>",
  "outcome": "active|suspended|terminated|success|noop|failure",
  "reason": "init|status_change|noop|invalid_transition|unsupported_status|invalid_payload|not_found|maintenance|rate_limited|policy_violation",
  "at": "2025-10-19T14:25:09Z"
}
```

Notes:
* Reason code vocabulary shared across lifecycle, decisions, audit; clients can unify classification pipelines.
* `_` sentinel appears only in `old_status` for initialization events.
* Latency currently raw nanoseconds; future versions may add percentile summaries.
* Hash fields allow external attestation; treat empty previous hash as genesis state.
### �📑 Audit Log Stream (Preview)

<a id="audit-logs"></a>
#### GET /api/v1/audit/logs

Returns recent audit decision/event entries in deterministic order (most recent first) with optional pagination limiting. Used by Web UI preview panel.

Minimal curl examples:
```bash
# Default recent audit log entries
curl -s http://localhost:8080/api/v1/audit/logs | jq .

# Limit to 10 entries
curl -s "http://localhost:8080/api/v1/audit/logs?limit=10" | jq .

# Entries since a timestamp
curl -s "http://localhost:8080/api/v1/audit/logs?since=2025-10-19T14:00:00Z" | jq .
```

Query Parameters:
* `limit` (optional, integer) – Max entries to return (default 50).
* `since` (optional, RFC3339 timestamp) – Include only entries with `at` > provided timestamp.

Response (example):
```json
{
  "success": true,
  "entries": [
    {"id":"evt_1042","action":"token_status_update","resource":"token:abc123","outcome":"suspended","reason":"status_change","at":"2025-10-19T14:25:09Z"},
    {"id":"evt_1041","action":"delegation_status_update","resource":"delegation:poa_9","outcome":"active","reason":"init","at":"2025-10-19T14:25:07Z"}
  ],
  "count": 2
}
```

Fields:
* `id` – Unique audit entry id.
* `action` – Operation recorded (e.g. `token_status_update`).
* `resource` – Namespaced resource (`token:<id>`, `delegation:<id>`).
* `outcome` – Resulting status or decision outcome label.
* `reason` – Classified reason code (see Decision Metrics list).
* `at` – RFC3339 timestamp of event.
* `count` – Returned entry count (convenience).

Ordering & Determinism:
* Sorted descending by timestamp then id for tie-break.
* Stable formatting for diff-friendly exports.

Edge / Error Cases:
* Empty: `{ "success": true, "entries": [], "count": 0 }` when no audit yet.
* Invalid `since`: may yield `400` with error message.

Use Cases:
* UI operational preview.
* Incident reconstruction (filter with `since`).
* Batch export / external ledger anchoring.
Lists all available demo scenarios for testing different RFC configurations.

**Request:**
```http
GET /scenarios HTTP/1.1
Host: localhost:8080
```

**Response:**
```json
[
  {
    "id": "rfc0111-basic",
    "name": "RFC-0111 Basic AgentAuth 1.0",
    "description": "Basic RFC-0111 AgentAuth 1.0 scenario with P*P Architecture",
    "config": {
      "p2p_enabled": true,
      "exclusions": ["resource1", "resource2"],
      "extended_tokens": true,
      "ai_client": false
    },
    "rfc_type": "RFC-0111"
  },
  {
    "id": "rfc0111-ai",
    "name": "RFC-0111 AI Client",
    "description": "RFC-0111 with AI client capabilities enabled",
    "config": {
      "p2p_enabled": true,
      "exclusions": [],
      "extended_tokens": true,
      "ai_client": true
    },
    "rfc_type": "RFC-0111"
  },
  {
    "id": "rfc0115-basic",
    "name": "RFC-0115 Basic PoA Definition",
    "description": "Basic RFC-0115 Power of Attorney definition scenario",
    "config": {
      "parties": {
        "grantor": "User A",
        "grantee": "User B", 
        "witness": "System"
      },
      "authorization_type": "limited",
      "legal_framework": "standard"
    },
    "rfc_type": "RFC-0115"
  },
  {
    "id": "rfc0115-advanced",
    "name": "RFC-0115 Advanced PoA",
    "description": "Advanced RFC-0115 with complex authorization requirements",
    "config": {
      "parties": {
        "grantor": "Corporation A",
        "grantee": "Agent B",
        "witness": "Legal System",
        "notary": "Certified Notary"
      },
      "authorization_type": "full",
      "legal_framework": "enterprise"
    },
    "rfc_type": "RFC-0115"
  },
  {
    "id": "combined-demo",
    "name": "Combined RFC Demo",
    "description": "Demonstration of combined RFC-0111 and RFC-0115 functionality",
    "config": {
      "rfc0111": {
        "p2p_enabled": true,
        "exclusions": ["restricted"],
        "extended_tokens": true,
        "ai_client": true
      },
      "rfc0115": {
        "parties": {
          "grantor": "System",
          "grantee": "Client"
        },
        "authorization_type": "limited"
      }
    },
    "rfc_type": "Combined"
  }
]
```

---

### **🔐 Authentication**

#### **POST /authenticate**
Authenticates using a selected demo scenario and returns a mock authentication token.

**Request:**
```json
{
  "scenario_id": "rfc0111-basic"
}
```

**Response (Success):**
```json
{
  "success": true,
  "token": "gauth_abc123def456...",
  "message": "Authentication successful for RFC-0111 Basic AgentAuth 1.0",
  "metadata": {
    "scenario": "RFC-0111 Basic AgentAuth 1.0",
    "rfc_type": "RFC-0111",
    "timestamp": 1696260000,
    "config": {
      "p2p_enabled": true,
      "exclusions": ["resource1", "resource2"],
      "extended_tokens": true,
      "ai_client": false
    }
  },
  "rfc_type": "RFC-0111"
}
```

**Response (Error):**
```json
{
  "success": false,
  "message": "Scenario not found"
}
```

**Error Codes:**
- `400 Bad Request`: Invalid JSON or missing scenario_id
- `404 Not Found`: Scenario not found

---

#### **POST /validate**
Validates an authentication token received from the authenticate endpoint.

**Request:**
```json
{
  "token": "gauth_abc123def456..."
}
```

**Response:**
```json
{
  "valid": true,
  "message": "Token is valid",
  "timestamp": 1696260000
}
```

**Error Codes:**
- `400 Bad Request`: Invalid JSON or missing token

---

### **🎯 RFC-0111 Configuration**

#### **POST /rfc0111/config**
Creates and validates an RFC-0111 configuration using the combined RFC implementation.

**Request:**
```json
{
  "p2p_enabled": true,
  "extended_tokens": true,
  "ai_client": false,
  "exclusions": ["web3_blockchain", "dna_based_identities"]
}
```

**Response (Success):**
```json
{
  "success": true,
  "message": "RFC-0111 configuration created successfully",
  "config": {
    "pp_architecture": {
      "pep": {
        "supply_side": {
          "entity": "client",
          "enforcement": ["oauth_flow", "token_validation"],
          "status": "active"
        },
        "demand_side": {
          "entity": "resource_server",
          "enforcement": ["scope_validation", "exclusions_check"],
          "status": "active"
        }
      },
      "pdp": {
        "primary_pdp": "authorization_server",
        "secondary_pdp": "resource_owner",
        "decision_rules": ["scope_validation", "exclusions_enforcement"]
      },
      "pip": {
        "authorization_server": "gauth_server",
        "data_sources": ["user_profile", "resource_metadata"],
        "info_types": ["identity", "permissions", "exclusions"]
      },
      "pap": {
        "client_owner_authorizer": "system_admin",
        "resource_owner_authorizer": "resource_admin",
        "policy_management": ["exclusions_policy", "ai_governance_policy"]
      },
      "pvp": {
        "trust_service_provider": "certification_authority",
        "verification_methods": ["digital_certificate", "oauth_token"],
        "identity_types": ["human", "ai_agent", "organization"]
      }
    },
    "exclusions": {
      "web3_blockchain": {
        "prohibited": true,
        "description": "Web3 and blockchain technologies are prohibited",
        "license_required": false
      },
      "ai_operators": {
        "prohibited": false,
        "description": "AI operators allowed with proper licensing",
        "license_required": true
      },
      "dna_based_identities": {
        "prohibited": true,
        "description": "DNA-based identities are prohibited",
        "license_required": false
      },
      "decentralized_auth": {
        "prohibited": true,
        "description": "Decentralized authentication is prohibited",
        "license_required": false
      },
      "enforcement_level": "strict"
    },
    "extended_tokens": {
      "token_type": "gauth_extended",
      "scope": ["authorization", "compliance"],
      "duration": "1h0m0s",
      "authorization": {
        "transactions": ["approve", "delegate"],
        "decisions": ["authorize", "revoke"],
        "actions": ["create", "modify", "delete"],
        "resource_rights": ["read", "write", "execute"]
      },
      "compliance": {
        "compliance_tracking": true,
        "audit_trail": ["creation", "usage", "revocation"],
        "revocation_status": "active"
      }
    },
    "gauth_roles": {
      "resource_owner": {
        "identity": "user_principal",
        "legal_capacity": true,
        "transaction_authority": ["financial", "legal"],
        "decision_acceptance": ["authorization", "delegation"],
        "action_impact": ["medium", "high"]
      },
      "resource_server": {
        "identity": "protected_service",
        "asset_types": ["data", "services", "financial"],
        "protected_resources": ["user_data", "financial_info"],
        "token_validation": "jwt_validation",
        "ai_capable": false
      },
      "client": {
        "type": "digital_agent",
        "identity": "ai_client_v1",
        "ai_capabilities": ["nlp", "decision_making"],
        "autonomy_level": "supervised",
        "request_types": ["data_access", "transaction"],
        "compliance_mode": "strict"
      },
      "authorization_server": {
        "identity": "gauth_server",
        "extended_token_issuing": true,
        "compliance_tracking": true,
        "pp_architecture_support": true,
        "exclusions_enforced": true
      },
      "client_owner": {
        "identity": "system_owner",
        "authorization_level": "admin",
        "ai_system_ownership": ["full"],
        "delegated_powers": ["configuration", "monitoring"]
      },
      "owner_authorizer": {
        "identity": "legal_authority",
        "statutory_authority": true,
        "authorization_scope": ["legal", "compliance"],
        "verification_method": "legal_certification"
      }
    },
    "version": "1.0.0",
    "status": "active",
    "created_at": "2025-10-02T20:00:00Z",
    "updated_at": "2025-10-02T20:00:00Z"
  },
  "rfc_version": "RFC-0111",
  "timestamp": 1696260000
}
```

**Error Codes:**
- `400 Bad Request`: Invalid configuration parameters

---

### **📋 RFC-0115 PoA Definition**

#### **POST /rfc0115/poa**
Creates and validates an RFC-0115 Power of Attorney definition.

**Request:**
```json
{
  "parties": {
    "grantor": "System Owner",
    "grantee": "AI Agent", 
    "witness": "Authorization Server"
  },
  "authorization_type": "limited",
  "legal_framework": "standard"
}
```

**Response (Success):**
```json
{
  "success": true,
  "message": "RFC-0115 Power of Attorney definition validated successfully",
  "poa_definition": {
    "parties": {
      "principal": {
        "type": "individual",
        "identity": "system_owner_001",
        "individual": {
          "name": "System Owner",
          "citizenship": "US"
        }
      },
      "representative": null,
      "authorized_client": {
        "type": "digital_agent",
        "identity": "ai_agent_v1",
        "version": "1.0.0",
        "operational_status": "active"
      }
    },
    "authorization": {
      "authorization_type": {
        "representation_type": "sole",
        "restrictions": ["financial_limits"],
        "sub_proxy_authority": false,
        "signature_type": "single"
      },
      "applicable_sectors": ["information_communication"],
      "applicable_regions": [
        {
          "type": "country",
          "identifier": "US",
          "name": "United States"
        }
      ],
      "authorized_actions": {
        "digital_actions": ["data_processing", "api_calls"],
        "transaction_actions": [],
        "physical_actions": []
      }
    },
    "requirements": {
      "validity_period": {
        "start_time": "2025-10-02T20:00:00Z",
        "end_time": "2026-10-02T20:00:00Z",
        "time_windows": [
          {
            "start": "09:00",
            "end": "17:00",
            "timezone": "UTC",
            "days": ["Mon", "Tue", "Wed", "Thu", "Fri"]
          }
        ],
        "geo_constraints": ["US"],
        "suspension_rules": ["security_breach", "compliance_violation"]
      },
      "formal_requirements": {
        "notarization_required": false,
        "witness_required": true,
        "legal_review_required": true,
        "registration_required": false
      },
      "power_limits": {
        "power_levels": [
          {
            "type": "transaction_amount",
            "limit": 1000.0,
            "currency": "USD",
            "description": "Maximum transaction amount"
          }
        ],
        "interaction_boundaries": ["read_only_data"],
        "tool_limitations": ["approved_apis_only"],
        "outcome_limitations": ["non_binding_recommendations"],
        "model_limits": [
          {
            "parameter_count": 1000000000,
            "reasoning_methods": ["logical", "statistical"],
            "training_methods": ["supervised"],
            "description": "AI model constraints"
          }
        ],
        "behavioral_limits": ["no_autonomous_transactions"],
        "quantum_resistance": true,
        "explicit_exclusions": ["financial_trading", "legal_contracts"]
      },
      "specific_rights": {
        "data_access_rights": ["user_profile", "preferences"],
        "modification_rights": [],
        "delegation_rights": [],
        "revocation_rights": ["immediate_revocation"]
      },
      "special_conditions": {
        "emergency_protocols": ["system_shutdown"],
        "escalation_procedures": ["human_oversight"],
        "monitoring_requirements": ["activity_logging", "compliance_tracking"],
        "reporting_obligations": ["weekly_reports"]
      },
      "death_incapacity": {
        "death_termination": true,
        "incapacity_suspension": true,
        "successor_designation": "backup_admin",
        "notification_procedures": ["immediate_alert"]
      },
      "security_compliance": {
        "encryption_requirements": ["AES-256", "RSA-4096"],
        "audit_requirements": ["quarterly_review"],
        "compliance_frameworks": ["SOC2", "ISO27001"],
        "security_monitoring": ["real_time", "anomaly_detection"]
      },
      "jurisdiction_law": {
        "language": "English",
        "governing_law": "US_Federal_Law",
        "place_of_jurisdiction": "US",
        "attached_documents": ["terms_of_service", "privacy_policy"]
      },
      "conflict_resolution": {
        "dispute_resolution_method": "arbitration",
        "arbitration_rules": "AAA_Commercial",
        "governing_law": "Delaware_Law",
        "jurisdiction": "Delaware_Courts"
      }
    },
    "gauth_context": {
      "pp_architecture_role": "client_authorization",
      "exclusions_compliant": true,
      "extended_token_scope": ["poa_validation", "compliance_tracking"],
      "ai_governance_level": "supervised"
    }
  },
  "rfc_version": "RFC-0115",
  "timestamp": 1696260000
}
```

**Error Codes:**
- `400 Bad Request`: Invalid PoA definition or missing parties information

---

### **🔄 Combined RFC Demo**

#### **POST /combined/demo**
Demonstrates the combined functionality of RFC-0111 and RFC-0115 in a unified configuration.

**Request:**
```json
{
  "rfc0111": {
    "p2p_enabled": true,
    "extended_tokens": true,
    "ai_client": true,
    "exclusions": ["web3_blockchain"]
  },
  "rfc0115": {
    "parties": {
      "grantor": "System",
      "grantee": "AI Agent"
    },
    "authorization_type": "limited"
  }
}
```

**Response (Success):**
```json
{
  "success": true,
  "message": "Combined RFC configuration validated successfully",
  "combined_config": {
    "rfc_0111": {
      "pp_architecture": { /* RFC-0111 configuration */ },
      "exclusions": { /* exclusions configuration */ },
      "extended_tokens": { /* token configuration */ },
      "gauth_roles": { /* role definitions */ },
      "version": "1.0.0",
      "status": "active",
      "created_at": "2025-10-02T20:00:00Z",
      "updated_at": "2025-10-02T20:00:00Z"
    },
    "rfc_0115": {
      "parties": { /* party definitions */ },
      "authorization": { /* authorization scope */ },
      "requirements": { /* requirements structure */ },
      "gauth_context": { /* integration context */ }
    },
    "integration_level": "combined_rfc",
    "combined_version": "1.0.0",
    "compatibility": {
      "rfc0111_version": "1.0.0",
      "rfc0115_version": "1.0.0",
      "integration_status": "fully_compatible"
    }
  },
  "rfc_versions": ["RFC-0111", "RFC-0115"],
  "timestamp": 1696260000
}
```

**Error Codes:**
- `400 Bad Request`: Invalid combined configuration or missing RFC specifications

---

### **📁 Static Files**

#### **GET /**
Serves the frontend application files from the `/frontend/` directory.

**Main Files:**
- `/` - Main demo application (index.html)
- Static assets (CSS, JS, images) served from frontend directory

---

## 📚 **Go Library API**

*[Previous Go library API documentation remains the same...]*

The existing library API documentation in the original API_REFERENCE.md file covers:
- RFCCompliantService
- RFC-0111 Authorization API (AuthorizeAgentAuth)
- RFC-0115 PoA Definition structures
- Professional Foundation API (ProperJWTService)
- Legal Framework Validation
- Complete data type definitions

---

## 🔧 **Error Handling**

### **HTTP Status Codes**
- `200 OK`: Successful request
- `400 Bad Request`: Invalid request format or parameters
- `404 Not Found`: Resource not found (e.g., scenario not found)
- `500 Internal Server Error`: Server-side processing error

### **Error Response Format**
```json
{
  "success": false,
  "message": "Error description",
  "error_code": "ERROR_TYPE",
  "timestamp": 1696260000
}
```

---

## 💡 **Examples**

### **Complete Demo Flow**

1. **Get Available Scenarios**
```bash
curl -X GET http://localhost:8080/scenarios
```

2. **Authenticate with a Scenario**
```bash
curl -X POST http://localhost:8080/authenticate \
  -H "Content-Type: application/json" \
  -d '{"scenario_id": "rfc0111-basic"}'
```

3. **Validate the Token**
```bash
curl -X POST http://localhost:8080/validate \
  -H "Content-Type: application/json" \
  -d '{"token": "gauth_abc123def456..."}'
```

4. **Configure RFC-0111**
```bash
curl -X POST http://localhost:8080/rfc0111/config \
  -H "Content-Type: application/json" \
  -d '{
    "p2p_enabled": true,
    "extended_tokens": true,
    "ai_client": true,
    "exclusions": ["web3_blockchain", "dna_based_identities"]
  }'
```

5. **Create RFC-0115 PoA Definition**
```bash
curl -X POST http://localhost:8080/rfc0115/poa \
  -H "Content-Type: application/json" \
  -d '{
    "parties": {
      "grantor": "System Owner",
      "grantee": "AI Agent",
      "witness": "Authorization Server"
    },
    "authorization_type": "limited",
    "legal_framework": "standard"
  }'
```

6. **Run Combined Demo**
```bash
curl -X POST http://localhost:8080/combined/demo \
  -H "Content-Type: application/json" \
  -d '{
    "rfc0111": {
      "p2p_enabled": true,
      "extended_tokens": true,
      "ai_client": true,
      "exclusions": ["web3_blockchain"]
    },
    "rfc0115": {
      "parties": {
        "grantor": "System",
        "grantee": "AI Agent"
      },
      "authorization_type": "limited"
    }
  }'
```

---

## � Capability Anchor & Notarization (Prototype)

### GET /api/v1/beta/capabilities/anchor/status
Returns freshness & integrity state of the capability anchor artifact plus optional external notarization receipt when `GAUTH_CAP_ANCHOR_NOTARIZE=1`.

Fields:
```
success, configured, last_write, registry_hash, age_seconds, stale_threshold_seconds,
stale, emitted_total, skipped_total, hash_changed_total, last_write_unix,
last_notarized_at, notarized_age_seconds, notarization_receipt
```

Receipt (prototype):
```json
{
  "hash": "sha256:...",
  "timestamp": "2025-10-19T23:40:00.123456789Z",
  "provider": "memory",
  "version": 1,
  "success": true,
  "latency_seconds": 0.00012
}
```

### Custom Prometheus Exposition
`GET /api/v1/beta/capabilities/anchor/metrics/prometheus` emits anchor & notarization metrics:
- gauth_rfc0111_capability_anchor_last_write_seconds (gauge)
- capability_anchor_age_seconds (gauge)
- capability_anchor_stale (gauge)
- gauth_rfc0111_capability_anchor_emitted_total (counter)
- gauth_rfc0111_capability_anchor_skipped_total (counter)
- gauth_rfc0111_capability_anchor_hash_changed_total (counter)
- capability_anchor_emission_interval_seconds (histogram)
- capability_anchor_emission_jitter_seconds (gauge)
- gauth_capability_anchor_notarization_latency_seconds (histogram, when notarize enabled)
- gauth_capability_anchor_notarized_age_seconds (gauge, when notarize enabled)
- gauth_capability_anchor_notarization_failures_total (counter, when notarize enabled)

### Environment Flags
| Flag | Purpose |
|------|---------|
| GAUTH_CAP_ANCHOR_FILE_PATH | Anchor artifact file path |
| GAUTH_CAP_ANCHOR_WRITE_INTERVAL | Minimum interval (>=1m) between emissions (default 5m) |
| GAUTH_CAP_ANCHOR_SIGN | Enable Ed25519 signing if key present |
| GAUTH_CAP_ANCHOR_NOTARIZE | Enable prototype external notarization & metrics |
| GAUTH_CAP_ANCHOR_STALE_THRESHOLD_SECONDS | SLA stale threshold (default 600) |

See `docs/ALERTING.md` for alert examples (latency, stale notarized age, failures surge).

Prototype Notice: External notarization currently uses an in-memory stub provider (`provider=memory`). For production, integrate a real TSA/transparency log (e.g., RFC3161, Sigstore Rekor) and extend receipt with inclusion proof & signature.

## �🚀 **Getting Started**

1. **Start the Demo Server**
```bash
cd gauth-demo-app/web/backend
go build -o gauth-backend main.go
./gauth-backend
```

2. **Access the Web Interface**
Open http://localhost:8080 in your browser

3. **Test the API**
Use the provided curl examples or test through the web interface

---

*This comprehensive API reference covers both the web demonstration API and the complete Go library implementation. For additional guides and documentation, see the [Getting Started Guide](../docs/GETTING_STARTED.md) and [Webapp Rebuild Summary](../WEBAPP_REBUILD_SUMMARY.md).*

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
