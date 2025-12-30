---
title: Security Hardening Guide
category: security-guide
status: active
lastUpdated: 2025-11-12
owners: security-team
source: internal
refreshCadence: quarterly
---
# Security Hardening Guide

> Last Updated: 2025-11-12
> Scope: Transition from beta evaluation to production-grade deployment.

This checklist enumerates additional controls and operational practices required before running AgentAuth in a production / regulated environment.

## 1. Runtime Environment
- Enforce Go version >= 1.25.3 across build agents and runtime images (`go version`).
- Enable read-only root filesystem (`--read-only` or Kubernetes `readOnlyRootFilesystem: true`).
- Drop all Linux capabilities beyond `NET_BIND_SERVICE` if required; use non-root UID (already in Dockerfile.unified runtime stage).
- Use seccomp + AppArmor profiles; supply PodSecurityContext for Kubernetes.

## 2. Configuration & Secrets
- Replace `AGENTAUTH_JWT_SIGNING_KEY` with rotated HSM or cloud KMS managed key material.
- Store environment secrets in Vault or sealed secrets; avoid plaintext `.env` in images.
- Restrict `AGENTAUTH_CORS_ALLOW` to explicit domains; never use `*` outside development.
- Set `AGENTAUTH_ENABLE_REPLAY_PROTECTION=true` with persistent store (Redis/PostgreSQL) for multi-instance deployments.

## 3. Cryptography & Key Management
- Turn on automated key rotation (`AGENTAUTH_KEY_ROTATION_AUTO=true`) with external anchoring (`AGENTAUTH_ANCHOR_ROTATIONS=true`).
- Enable multi-tenant key isolation if hosting multiple clients (separate key namespaces / prefix).
- Periodically export key rotation ledger for offline audit.
- Validate BLS or threshold signature aggregations against side-channel safe implementations.

## 4. Network & Transport
- Terminate TLS at ingress; enforce HSTS & modern cipher suites.
- Configure rate limiting / DoS protection (gateway or sidecar).
- Employ mTLS if service-to-service PoA introspection occurs across trust zones.

## 5. Persistence & Storage
- Migrate in-memory token/metrics stores to Postgres + durable metrics path (`AGENTAUTH_METRICS_PERSIST_PATH`).
- Enable backup strategy for persistence volumes (daily snapshot + PIT recovery plan).
- Apply encryption-at-rest for database volumes (cloud provider or LUKS).

## 6. Observability & Audit
- Enable Prometheus metrics (`AGENTAUTH_METRICS_ENABLED=true`), tracing (`AGENTAUTH_TRACING_ENABLED=true`) with sampling guard.
- Ship audit logs to immutable store (S3 object lock or append-only ledger service).
- Configure alerting: revocation failures, key rotation errors, anomaly in authorization denials spike.

## 7. Policy & PDP Hardening
- Implement distributed PDP cache invalidation (planned P0) to avoid stale decisions.
- Enforce maximum PoA chain depth (attestation chain limit) to mitigate recursive delegation abuse.
- Introduce structured error catalog to standardize operational incident triage.

## 8. Supply Chain
- Pin all base image digests (`FROM alpine@sha256:...`).
- Generate SBOM (e.g. `syft . -o json > sbom.json`) and sign (cosign) during CI.
- Run `govulncheck ./...` and `gosec ./...` on each pipeline; fail on HIGH severity.
- Scan Docker images with Trivy (HIGH/CRITICAL must be zero).

## 9. Fuzzing & Negative Testing
- Fuzz critical parsers (CBOR codec, policy evaluator) with `go test -fuzz` targets.
- Add chaos tests for replay protection store and distributed revocation flows.
- Perform load testing to p99 latency & memory pressure thresholds.

## 10. Operational Safeguards
- Configure runbooks for: key rotation failure, audit sink outage, DB failover, revocation backlog.
- DR test at least quarterly: simulate region loss + recovery.
- RPO/RTO targets defined (e.g., RPO < 15m, RTO < 30m).

## 11. Access Control & Least Privilege
- Segregate CI/CD roles for build vs deploy (artifact promotion gate).
- Apply IAM policies limiting secret read scope; no broad wildcard access.
- Enforce MFA on all operational dashboards.

## 12. Compliance Alignment
- Map PoA issuance and revocation events to required regulatory audit trails (e.g., GDPR access logs).
- Jurisdiction enforcement hooks must validate local constraints (data residency, retention windows).
- Integrate incident reporting workflow with compliance ticketing tool.

## 13. Performance Guardrails
- Set authorization latency SLO (e.g., p95 < 50ms under peak load).
- Autoscale based on active token validation rate & error ratio.
- Implement token compaction/persistence TTL for stale authorization artifacts.

## 14. Release Management
- Automate release notes from conventional commits (`scripts/release-notes.sh`).
- Tag releases with signed Git tags (`git tag -s v1.0.x`).
- Maintain changelog diff vs previous minor for audit submission.

## 15. Risk Register (Examples)
| Risk | Mitigation | Status |
|------|------------|--------|
| Stale PDP decisions | Distributed cache invalidation | Pending (P0) |
| Weak CORS policy | Explicit allow-list only | Implemented |
| Key rotation failure silent | Alert on missing rotation interval | Partial |
| Audit sink outage | Buffered retry + external anchoring fallback | Planned |
| Delegation abuse (deep chain) | Chain depth cap + anomaly alert | Planned |

## 16. Hardening Progress Tracking
Maintain `TECH_DEBT.md` for P0/P1 items; cross-link completed controls here with date.

## Quick Verification Commands
```bash
govulncheck ./...            # Vulnerability scan
gosec ./...                  # Static security checks
syft . -o json > sbom.json   # SBOM generation
trivy image agentauth:prod       # Image CVE scan
go test -fuzz=FuzzCBOR -run=^$ -fuzztime=10s ./pkg/poa
```

---
This guide evolves; review quarterly and update with new control families (e.g., data minimization, privacy impact assessment).
