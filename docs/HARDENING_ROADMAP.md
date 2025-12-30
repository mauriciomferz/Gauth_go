---
title: Hardening Roadmap
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# AgentAuth Hardening Roadmap (From Beta Demonstration to Production Candidate)

> Last Updated: 2025-10-17
> Status: Active

> Status: Draft
> Scope: Defines phased implementation steps to transform the current **beta AgentAuth-RFC-001 (formerly RFC 111) / 115 demonstration** into a security, compliance, validation, and operations capable system.

## Phase 0 – Current State (Beta Baseline)
- Mock / simplified cryptography (basic signing)
- Hardcoded legal & compliance responses
- In‑memory stores only; no persistence
- Minimal audit & monitoring
- No policy engine, no multi‑tenant isolation, no supply chain controls

## Phase 1 – Foundations (Security & Observability)
### Goals
Establish trustworthy cryptographic, logging, and monitoring primitives.
### Key Deliverables
- Replace beta demo token logic with PASETO or hardened JWT library (alg agility, explicit claims validation)
- Key management abstraction + ephemeral dev keys; design for KMS/HSM plug‑in
- Structured logging (JSON) with correlation IDs & redaction helpers
- Basic metrics (Prometheus) + OpenTelemetry tracing spans around auth & delegation flows
- Graceful shutdown, health (`/healthz`), readiness (`/ready`), liveness probes
- Add input validation scaffolding (central validator package)
### Exit Criteria
- All tokens verified with audience / issuer / expiry enforcement
- Unit tests for cryptographic service boundaries
- Logs machine‑parseable, PII scrub pipeline documented

## Phase 2 – Policy & Authorization Maturity
### Goals
Introduce dynamic, versioned authorization and delegation controls.
### Key Deliverables
- Integrate policy engine (OPA sidecar or embedded CEL evaluation) for authorization & delegation decisions
- Versioned policy registry (GitOps or signed bundle)
- Attestation requirement evaluation (multi‑attester rules)
- Rate limiting & anomaly detection stub (sliding window + counters)
### Exit Criteria
- All authorization decisions traceable to a policy evaluation record
- Failing policies produce structured denials with reason codes

## Phase 3 – Persistence & Multi‑Tenancy
### Goals
Enable durable state and safe tenant isolation.
### Key Deliverables
- Replace in‑memory token & delegation stores with Postgres + migrations
- Introduce Redis for cache / rate limiting counters
- Tenant scoping (org / namespace) enforced at repository layer
- Data retention policies (configurable TTL per data class)
### Exit Criteria
- Cold restart does not lose tokens / delegations
- Migration pipeline tested (up/down)
- Isolation tests confirm no cross‑tenant leakage

## Phase 4 – Compliance & Audit Integrity
### Goals
Meet baseline SOC2/GDPR style auditability & governance.
### Key Deliverables
- Tamper‑evident audit ledger (hash‑chain or Merkle root per block)
 - External capability & audit ledger anchoring: periodic submission of chain tip (hash + timestamp) to transparency / timestamp service (e.g., RFC3161 TSA or open-source Rekor). Export notarization receipt & expose `capability_anchor_notarized_age_seconds` gauge.
- Data flow map + classification (personal, operational, cryptographic)
- Subject rights API stubs (export, delete requests)
- Configurable retention & purge jobs
- SBOM generation (Syft or CycloneDX) + artifact signing (Cosign)
### Exit Criteria
- Audit entries verifiable end‑to‑end
- SBOM & signed image available for each CI build
- Data classification documented & enforced in code paths

## Phase 5 – Security Hardening & Supply Chain
### Goals
Advance trust posture & resilience.
### Key Deliverables
- Key rotation scheduler + compromised key revocation procedure
 - External notarization verification pipeline: scheduled task validates latest notarization receipts for capability registry and revocation/audit chains; sets freshness gauges and emits alert on mismatch or stale (> configured threshold).
- Replay protection (nonce / jti store)
- Dependency vulnerability scanning & license policy gate
- Build provenance attestation (SLSA level draft)
- Fuzz & property‑based testing integrated into CI
### Exit Criteria
- Rotating keys without downtime
- Fuzz suite covers token parsing, policy evaluation, and delegation flows

## Phase 6 – Operational Excellence
### Goals
Production readiness in reliability & response.
### Key Deliverables
- SLOs & alerting rules (latency, error %, policy denial spike, ledger integrity failure)
- Blue/Green or Canary deploy workflow
- Disaster recovery drill runbook + RPO/RTO targets
- Threat model & periodic security review cadence
### Exit Criteria
- On‑call runbook validated in simulated incident
- DR test passes within documented RTO

## Phase 7 – Advanced Compliance & Privacy
### Goals
Extend to regulated contexts.
### Key Deliverables
- Fine‑grained consent + purpose limitation enforcement
- Differential privacy or minimization strategies for analytics
- Encryption at rest (KMS envelope) for sensitive tables
- Automated DPIA & risk register template
### Exit Criteria
- Every data field tagged with purpose & retention metadata
- Periodic compliance report can be generated automatically

---
## Cross‑Cutting Non‑Functional Requirements
| Area | Requirement |
|------|-------------|
| Performance | P99 auth < 150ms (baseline after Phase 3) |
| Scalability | Horizontal scale with stateless app tier |
| Security | No critical / high vulnerabilities in dependency scan |
| Test Coverage | >= 80% critical path (auth, delegation, policy) |
| Documentation | Updated per phase with CHANGELOG entries |

## Next Immediate Actions (Proposed)
1. Add scaffolding packages: `policy`, `validation`, `audit` (ledger), `compliance`
2. Implement interfaces & stubs (no external deps yet)
3. Wire minimal usage examples in README section (future)
4. Add SBOM generation script placeholder

## Tracking
Each phase should map to GitHub Milestones & issues referencing this roadmap.

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
