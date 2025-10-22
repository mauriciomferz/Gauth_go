# GAP Workstream Ownership Map
Generated: 2025-10-22

| Workstream | Module / Files | Owner (Placeholder) | Reviewers |
|------------|----------------|---------------------|-----------|
| Cryptographic Foundations | internal/crypto/*, pkg/rfc0111/*, docs/TOKEN_SIGNATURES.md | crypto-lead | security-review, perf-review |
| Claims & Parser | pkg/gauth/claims.go, pkg/gauth/parser.go, bench_test/*, docs/TOKEN_CLAIMS.md | token-core | fuzz-review, perf-review |
| Authorization Engine | pkg/authz/*, AUTHORIZATION_IMPLEMENTATION.md | authz-lead | governance-review |
| PoA & Delegation | pkg/rfc0111/poa_validator.go, envelope changes | delegation-lead | crypto-lead |
| Persistence & Ledger | internal/ledger/*, AUDIT_LEDGER.md, MODEL_LIMITS.md (anchors) | ledger-lead | crypto-lead, ops-review |
| Secret & Key Mgmt | internal/secrets/*, internal/crypto/keys.go | secrets-lead | security-review |
| Model Governance | web/server_clean.go (limits paths), internal/notary/* | governance-lead | ledger-lead |
| Testing & Conformance | conformance/*, scripts/verify_clause_map.go | test-harness | authz-lead |
| Interoperability & API | docs/openapi.yaml, web/* endpoints | api-lead | governance-lead |
| Observability & Metrics | internal/observability/*, internal/metrics/*, web/server_clean.go | observability-lead | perf-review |
| Compliance & Jurisdiction | compliance modules (planned) | compliance-lead | legal-advisor |
| Replay & Token Security | pkg/gauth/gauth.go (JTI), durable store future | token-core | security-review |
| Data Hygiene & Risk | validation filters, risk register docs | risk-lead | security-review |

Note: Owners placeholders to be replaced with actual maintainers. Each PR requires owner + one reviewer from distinct domain.
