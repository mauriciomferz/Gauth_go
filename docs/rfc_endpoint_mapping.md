---
title: Rfc Endpoint Mapping
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# RFC Endpoint Mapping

This document correlates HTTP endpoints (from `docs/openapi.yaml` and route registrations) with RFC111 / RFC115 clause references and implementation artifacts.

| Endpoint | Purpose | RFC Reference(s) | Implementation Highlights | Error Codes (Examples) |
|----------|---------|------------------|---------------------------|------------------------|
| `POST /api/v1/model/validate` | Enforce model input/output + rate limits | `rfc111:model_limits` | `apiModelValidate` applies per-user & global limits; audit optional | `model_invalid_payload`, `model_unknown`, `model_user_input_limit_exceeded`, `model_limit_exceeded`, `model_rate_limit_exceeded` |
| `POST /api/v1/multi_sig/bls` (issue/verify) | BLS multi-signature aggregation | `rfc111:multi_sig` | Input validation, base64 decodes, participant ceiling | `invalid_mode`, `missing_message`, `private_key_deserialize_failed`, `aggregated_signature_decode_failed` |
| `POST /api/v1/pop/verify` | Proof-of-possession verification | `rfc111:pop` | Nonce challenge pairs, BLS init, pair validation | `bad_request`, `no_pairs`, `nonce_gen_failed` |
| `POST /api/v1/anchor/external` | External capability+rotation anchoring | `rfc111:external_anchor` | Combined anchor receipt persistence | `anchor_inputs_missing`, `anchor_append_failed`, `anchor_store_missing` |
| `POST /api/v1/anchor/revocation/emit` | Revocation Merkle root anchoring | `rfc111:revocation_anchor` | Chain emptiness guard, idempotent root anchoring | `revocation_chain_empty`, `revocation_root_empty`, `revocation_anchor_failure` |
| `GET /api/v1/diagnostics/semantic` | Semantic counters & anomaly diagnostics | `rfc115:semantic_diagnostics`, `rfc115:anomaly_detection`, `rfc115:integrity_chain` | Rates (60s/300s), EWMA z-scores, integrity hash chain exposure | `semantic_metrics_unavailable` (spec example) |
| `POST /api/v1/token/validate` (implicit via JWT path) | Token/JWT validation + replay protection | `rfc111:token_validation`, `rfc111:replay_protection` | Standardized respondError mapping; strict JTI, algorithm checks | `token_invalid_signature`, `token_expired`, `token_replay_detected`, `token_invalid_algorithm`, `token_malformed` |
| `POST /api/v1/capabilities/anchor` | Capability registry anchoring | `rfc111:capability_anchor` | Registry hash creation & anchoring, client availability checks | `capability_anchor_disabled`, `capability_anchor_client_unavailable`, `capability_anchor_failure` |
| `POST /api/v1/capabilities/audit/anchor` | Capability audit chain tip anchoring | `rfc111:capabilities_audit_anchor` | Chain tip empty guard, anchor emission | `capabilities_audit_chain_tip_empty`, `capabilities_audit_anchor_failure` |
| `GET /api/v1/capabilities/audit/verify` | Capability audit verification retrieval | `rfc111:capabilities_audit_verify` | File read & JSON validation, integrity replay | `capabilities_audit_read_failed`, `capabilities_audit_invalid_json` |
| `POST /api/v1/capabilities/negotiate` | Client version negotiation | `rfc111:capabilities_negotiate` | Compares client vs server versions; strict payload | `capabilities_negotiate_invalid_payload` |
| `POST /api/v1/attestation/verify` | Attestation JSON validation | `rfc111:attestation_verify` | Body read & JSON parsing integrity | `attestation_body_read_failed`, `attestation_invalid_json` |
| `POST /api/v1/rotations/summary` (implied) | Rotation ledger summary & anchoring | `rfc111:rotations` | Ledger type safety, optional anchor integration | `rotation_ledger_type_mismatch` |

## Notes

1. Some endpoints are exposed under both `/api/v1/beta/...` and stable paths; RFC references apply equally.
2. Diagnostics 500 error path (`semantic_metrics_unavailable`) is documented but not yet triggered in code—future enhancement.
3. JWT validation endpoint uses unified respondError variants introduced in recent compliance pass.
4. Revocation anchoring example responses appear multiple times in `openapi.yaml` to cover all error codes.
5. Rotation operations are partially tagged; additional invariants (sequence continuity, signature set completeness) can expand RFC111:rotations coverage.

## Future Mappings (Planned)

- Consistency proofs endpoints (if added) → `rfc111:revocation_consistency`
- Delegation semantic enforcement actions (throttling/revocation triggers) → `rfc115:reactive_controls`
- Extended attestation cryptographic chain validation → `rfc111:attestation_integrity`
