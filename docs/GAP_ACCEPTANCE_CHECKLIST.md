# GAP Acceptance Checklist (P0)
Generated: 2025-10-22

| Gap ID | Implemented PR | Tests Added | Metrics Added | Docs Updated | Verification Command | Reviewer Sign-Off |
|--------|----------------|------------|---------------|--------------|----------------------|-------------------|
| sec1.item1 | TBD | TBD | token_signature_verifications_total | TOKEN_SIGNATURES.md, openapi.yaml | go test ./... -run TestECDSASignVerify | TBD |
| sec1.item2 | TBD | TBD | token_claims_validation_fail_total | TOKEN_CLAIMS.md | go test ./... -run TestClaimsSemantics | TBD |
| sec1.item3 | TBD | TBD | token_parse_duration_seconds | TOKEN_CLAIMS.md | go test ./... -run TestParserReplacement | TBD |
| sec1.item5 | TBD | TBD | token_signature_multi_alg_enabled | TOKEN_SIGNATURES.md | go test ./... -run TestMultiAlgEnvelope | TBD |
| sec2.item1 | TBD | TBD | authz_conflicts_total | AUTHORIZATION_IMPLEMENTATION.md | go test ./... -run TestPDPDiagnostics | TBD |
| sec2.item2 | TBD | TBD | abac_function_invocations_total | AUTHORIZATION_IMPLEMENTATION.md | go test ./... -run TestABACRegistry | TBD |
| sec3.item1 | TBD | TBD | poa_validation_fail_total | POA_VALIDATION.md | go test ./... -run TestPoAValidator | TBD |
| sec5.item1 | TBD | TBD | audit_ledger_append_total,audit_anchor_interval_seconds | AUDIT_LEDGER.md | go test ./... -run TestAuditLedgerVerify | TBD |
| sec8.item1 | (this commit) | provider tests | secret_operations_total (future) | SECRETS.md | go test ./internal/secrets -count=1 | crypto-lead |
| sec9.item1 | TBD | clause map script tests | conformance_clause_coverage_percent | clause_map.json | go run scripts/verify_clause_map.go | test-harness |
| sec11.item2 | TBD | stream reason tests | model_limits_notarization_total | MODEL_LIMITS.md | curl /api/v1/model/limits/attestation/stream | governance-lead |
