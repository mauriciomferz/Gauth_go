---
title: Composite Authorization Readme
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Composite Authorization Example

This example (`agentauth-plus-authorization.json`) models a rich, multi-layer corporate authorization grant that delegates constrained financial and contractual powers from human executives to an enterprise AI agent under strict governance, dual-control, and audit obligations.

## Purpose
Illustrates how complex authority structures can be encoded as a single machine-consumable artifact:
- Human → Human → AI delegation chain (CEO → CFO → AI system)
- Explicit scope & temporal validity
- Derived and standard powers decomposition
- Decision & escalation matrix
- Transactional boundaries and prohibited actions
- Fine-grained action permissions (systems, human, agent interactions)
- Dual‑control & multi‑signature requirements with delay and witness rules
- Accountability & cascade chain for ultimate responsibility tracing

## File
`examples/agentauth-plus-authorization.json`

## Structural Sections
| Section | Key Fields | Intent |
|---------|------------|--------|
| authorizing_party | id, type, authorized_representative, legal_capacity | Establish legal identity & representative validity |
| authorization_grant | scope, limitations, valid_from/until, revocable | Top-level grant envelope |
| powers_granted.basic_powers | financial_operations, contract_management, vendor_management | Base domains of authority |
| powers_granted.derived_powers | approve_invoices, sign_service_agreements, authorize_payments | Concrete operational abilities implied by base powers |
| powers_granted.power_derivation | mapping of base → derived | Traceability & justification of derived powers |
| powers_granted.standard_powers.financial_powers | signing_authority, approval_limits, banking_operations | Monetary operation limits & channels |
| decision_authority.decision_matrix | routine_payments, large_payments, contract_terminations | Determines autonomy vs approval vs dual control |
| decision_authority.escalation_rules | escalation_path, threshold_triggers | Escalation workflow for higher risk/amount/rules |
| transaction_rights.transaction_limits | per_transaction, cumulative, frequency_limits | Quantitative boundaries against abuse |
| transaction_rights.required_approvals | large_payments, new_vendors | Declarative approval sets |
| action_permissions.resource_actions | system → verbs | CRUD / execute rights per system category |
| action_permissions.human_interactions | authorized_channels, interaction_limits | Guardrails for direct human contact volume/time |
| action_permissions.agent_interactions | authority_validation (blockchain) | Cross-agent trust & recurrent validation |
| dual_control_principle | requires_dual_control, approval_matrix, control_mechanisms | High-risk operations require multi-party confirmation |
| authorization_cascade.cascade_chain | levels with authorizer/authorized | Provenance chain of delegation |
| authorization_cascade.accountability_chain | ordered list of actors | Reverse lookup for responsibility |
| expires_at | timestamp | Temporal validity & revocation boundary |

## Delegation & Cascade
The artifact encodes a two-level cascade:
1. Board-sanctioned CEO authority (ultimate human).
2. CEO delegates scoped financial powers to CFO.
3. CFO delegates limited, routine powers to the AI system (`enterprise_financial_ai_v1`).

`authorization_cascade.cascade_chain` provides immutable provenance of each delegation hop; `accountability_chain` preserves a top-down trace enabling post-incident responsibility resolution.

## Power Derivation Logic
`power_derivation` justifies each derived operational power from higher-level domains (e.g., `financial_operations` → `authorize_payments`). This supports auditable reasoning: revoking a base power automatically invalidates its derived set.

## Decision & Escalation Mechanics
The `decision_matrix` differentiates autonomous vs approval vs dual-approval contexts. `escalation_rules` define triggers (amount, risk_level, vendor_type) and a sequential escalation path with maximum response time SLAs. Override authority ensures urgent remediation (CFO/CEO).

## Transaction Governance
`transaction_rights` split per-transaction vs cumulative vs frequency limits preventing aggregation bypass (e.g., many small payments). Prohibited transactions list enforces categorical bans aligned with policy/compliance.

## Dual Control Enforcement
`dual_control_principle.control_mechanisms` encode multi-signature, time-delayed operations, witness requirements, and mandatory audit trail capture. Large payment attempts must accumulate required signatures before execution and respect time delays (cool-off periods).

## Action Permissions Granularity
- Resource actions enumerate system-level verbs (read/write/approve/execute).
- Human interaction caps volume, duration, and time window (business hours), with required notifications.
- Agent interactions demand periodic blockchain-backed authority re-validation and restrict unauthorized agent delegation.

## Temporal & Revocation Model
Validity windows (`valid_from`, `valid_until`, `expires_at`) enable automatic sunset. `revocable: true` allows proactive governance revocation, while derived power linkage ensures consistent cascading invalidation.

## Integrating Into Your System
1. Load JSON into an authorization engine or policy layer.
2. Validate schema (recommended: define a JSON Schema for structural guarantees).
3. Extract accountability chain for audit context injection.
4. Enforce decision matrix prior to action execution (autonomous vs approval path selection).
5. Apply transaction limit checks (per attempt, cumulative, frequency) before payment commits.
6. Evaluate dual-control operations: hold action in a pending state until signature quorum & time delay satisfied.
7. Log all approvals and escalations with immutable event hashes.

### Example (Pseudo-Go)
```go
artifact := LoadCompositeAuthorization("examples/agentauth-plus-authorization.json")
if time.Now().After(artifact.ExpiresAt) { Deny("grant expired") }
if !WithinScope(artifact, action.Domain) { Deny("outside scope") }
if RequiresDualControl(artifact, action) { QueueForDualControl(action) }
if ExceedsLimits(artifact.TransactionRights, action.Amount) { Escalate(action) }
RecordAuditEvent(action, artifact.AccountabilityChain)
```

### Example curl (distributing artifact)
```bash
curl -X POST https://agentauth.example/api/v1/authorization/artifacts \
  -H 'Content-Type: application/json' \
  --data-binary @examples/agentauth-plus-authorization.json
```
(Endpoint name illustrative; adapt to actual API path.)

## Security Considerations
- Treat artifact as a signed object (recommend envelope + signatures for integrity).
- Enforce immutability: modifications require re-sign + version bump.
- Maintain chain-of-custody logs for each delegation hop.
- Apply least privilege drift detection: periodically recompute derived powers vs base.
- Continuous validation of blockchain authority if required.
- Time-delayed operations: monitor for bypass attempts (e.g., splitting large payment into many smaller ones near limits).

## Extension Ideas
- Add cryptographic signatures per cascade link.
- Attach revocation list URL & OCSP-style checks.
- Integrate with rotation artifacts for signer key authority alignment.
- Provide machine-readable JSON Schema & optional SBOM of authority dependencies.
- Include risk scoring block updating escalation paths dynamically.

## Testing Suggestions
| Test | Purpose |
|------|---------|
| Expiry enforcement | Deny after `expires_at` |
| Derived power revocation | Remove base power → verify derived invalidation |
| Dual control path | Attempt high-value payment with only one signature → hold |
| Escalation timing | Simulate urgent scenario → ensure SLA path followed |
| Frequency limit | Exceed per_hour count → deny subsequent attempts |
| Prohibited transaction | Submit banned type (e.g., cryptocurrency) → immediate denial |

## Alignment With AI Safety & Governance
This structure supports:
- Transparent delegation lineage
- Quantitative guardrails (limits, frequency controls)
- Context-sensitive human oversight (escalation & dual control)
- Continuous authorization verification (blockchain validators)
- Strong auditability & witness capture for critical operations

## Next Steps
- Wire artifact ingestion into enforcement middleware.
- Implement signature & hash continuity (reuse rotation digest semantics).
- Build UI panel summarizing cascade & limits (parallel to Rotation V2 panel).
- Add OpenAPI schema & endpoint for composite authorization retrieval.

## API Endpoints
The server exposes beta and stable endpoints:

- GET `/api/v1/beta/authorization/composite` (alias: `/api/v1/authorization/composite`) – Fetch active artifact.
- POST `/api/v1/beta/authorization/composite` (alias: `/api/v1/authorization/composite`) – Activate a new artifact.

### Activation Workflow
1. Client constructs JSON artifact (see earlier sections).
2. POST activation; server validates required blocks and non-overlapping validity window.
3. Response returns:
   - `canonical_hash`: stable SHA-256 over canonicalized subset.
   - `previous_artifact_hash`: continuity pointer (empty on first activation).
   - `version`: timestamp-based version identifier.
4. Subsequent GET returns the active artifact envelope.

### Example Activation
```bash
curl -X POST http://localhost:8080/api/v1/authorization/composite \
  -H 'Content-Type: application/json' \
  -d '{
    "ai_system_id": "enterprise_financial_ai_v1",
    "authorization_grant": {"type": "general", "scope": ["financial_operations"], "valid_from": "2025-01-01T00:00:00Z", "valid_until": "2025-12-31T23:59:59Z", "revocable": true},
    "powers_granted": {"basic_powers": ["financial_operations"]},
    "decision_authority": {"autonomous_decisions": ["routine_invoice_approval"]},
    "transaction_rights": {"allowed_transaction_types": ["vendor_payments"]},
    "action_permissions": {"system_actions": ["generate_reports"]},
    "dual_control_principle": {"enabled": true},
    "authorization_cascade": {"accountability_chain": ["ceo_001","cfo_001","enterprise_financial_ai_v1"]},
    "expires_at": "2025-12-31T23:59:59Z"
  }'
```

### Example Retrieval
```bash
curl http://localhost:8080/api/v1/authorization/composite | jq .
```

### Conflict Conditions
Returns 409 when new artifact `valid_from` precedes current artifact `expires_at` (overlapping validity windows).

### Error Codes
| HTTP | Code | Meaning |
|------|------|---------|
| 400 | composite_authorization_invalid | Structural validation failed or expired artifact |
| 400 | composite_authorization_invalid_json | Malformed JSON payload |
| 404 | composite_authorization_missing | No active artifact |
| 409 | composite_authorization_conflict | Overlapping activation window |
| 500 | composite_authorization_activation_failed | Internal activation error |

### Canonical Hash
The `canonical_hash` is computed from selected deterministic fields (scope, basic powers, decision matrix entries, expiration, accountability chain). Changing any of those requires reactivation and produces a new hash.

## Prometheus Metrics
Composite authorization lifecycle instrumentation (prefix `agentauth_composite_`):

| Metric | Type | Description |
|--------|------|-------------|
| `agentauth_composite_activation_total` | counter | Successful activations (201). |
| `agentauth_composite_conflicts_total` | counter | Activation conflicts (409). |
| `agentauth_composite_invalid_total` | counter | Invalid/expired activation attempts (400). |
| `agentauth_composite_current_hash_present` | gauge | 1 when artifact active; 0 if none configured. |
| `agentauth_composite_chain_updates_total` | counter | Continuity advances (non-empty previous hash). |

Example scrape (after 2 activations, 1 conflict):
```
agentauth_composite_activation_total 2
agentauth_composite_conflicts_total 1
agentauth_composite_invalid_total 0
agentauth_composite_current_hash_present 1
agentauth_composite_chain_updates_total 1
```

Use these for dashboards: activation velocity, conflict ratio, continuity health.

---

---
Generated README to assist developers in adopting composite authorization patterns. Adapt field names or structure as implementation evolves.
