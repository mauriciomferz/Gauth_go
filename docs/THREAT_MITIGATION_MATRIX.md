# Threat Mitigation Matrix

**Version**: 1.0  
**Last Updated**: 2025-12-27  
**Purpose**: Systematic mapping of identified threats to implemented mitigations

---

## Overview

This matrix links threats from [`THREAT_MODEL.md`](THREAT_MODEL.md) to their corresponding mitigations implemented in the AgentAuth codebase. It serves as a compliance verification tool and ensures no threats are left unaddressed.

---

## Threat-to-Mitigation Mapping

| Threat ID | Threat Description | Severity | Mitigation | Implementation | Status |
|-----------|-------------------|----------|------------|----------------|--------|
| T1 | **Token Forgery** | Critical | Ed25519/RSA/ECDSA signatures | [`aap001.go`](../pkg/agentauth_aap_001/aap001.go) canonical signature verification | ✅ **Mitigated** |
| T2 | **Replay Attacks** | Critical | JTI-based replay detection | [`replay_store_bolt.go`](../pkg/agentauth/replay_store_bolt.go) + [`redis_replay_store.go`](../pkg/agentauth_aap_001/redis_replay_store.go) | ✅ **Mitigated** |
| T3 | **Token Expiration Bypass** | High | `exp` claim validation | [`aap001.go`](../pkg/agentauth_aap_001/aap001.go) timestamp checks | ✅ **Mitigated** |
| T4 | **Delegation Depth Attack** | High | Max delegation depth limit (configurable) | [`aap001.go`](../pkg/agentauth_aap_001/aap001.go) chain verification | ✅ **Mitigated** |
| T5 | **Scope Escalation** | Critical | Scope intersection validation | [`engine.go`](../pkg/pdp/engine.go) scope narrowing | ✅ **Mitigated** |
| T6 | **Capability Registry Tampering** | High | External RFC-3161 TSA anchoring | [`external_anchor.go`](../pkg/ledger/external_anchor.go) with fallback chains | ✅ **Mitigated** |
| T7 | **Discovery Metadata Spoofing** | Medium | HTTPS + JWKS signature verification | [`discovery_endpoint.go`](../web/discovery_endpoint.go) | ✅ **Mitigated** |
| T8 | **Audit Log Tampering** | High | Merkle tree integrity + external anchoring | [`ledger.go`](../pkg/ledger/ledger.go) chained hashing | ✅ **Mitigated** |
| T9 | **Rate Limit Bypass** | Medium | GCRA + distributed rate limiting | [`ratelimit.go`](../pkg/ratelimit/ratelimit.go) semantic analysis | ✅ **Mitigated** |
| T10 | **Replay Store Failure** | Critical | Fail-closed mode + monitoring | [`replay_store_bolt.go`](../pkg/agentauth/replay_store_bolt.go) + [`FAIL_CLOSED_ADVISORY.md`](FAIL_CLOSED_ADVISORY.md) | ✅ **Mitigated** |
| T11 | **Malformed Token DoS** | Medium | Input validation + fuzz testing | [`canonical_signature_fuzz_test.go`](../pkg/agentauth_aap_001/canonical_signature_fuzz_test.go) + enhanced JTI validation | ✅ **Mitigated** |
| T12 | **Policy Complexity Attack** | Medium | Expression evaluation budgets | [`expr.go`](../pkg/authz/expr.go) MaxTokens/MaxOps limits | ✅ **Mitigated** |

---

## Mitigation Coverage by Category

### Authentication & Identity
- ✅ Cryptographic signature verification (T1)
- ✅ Token expiration enforcement (T3)
- ✅ JTI uniqueness validation (T2)
- ✅ Malformed token rejection (T11)

### Authorization & Delegation
- ✅ Scope narrowing enforcement (T5)
- ✅ Delegation chain depth limits (T4)
- ✅ Policy evaluation safety (T12)

### Integrity & Auditability
- ✅ Audit log tamper-proofing (T8)
- ✅ Capability registry anchoring (T6)
- ✅ Discovery metadata protection (T7)

### Availability & Resilience
- ✅ Rate limiting with semantic analysis (T9)
- ✅ Fail-closed replay protection (T10)
- ✅ DoS mitigation via input validation (T11)

---

## Residual Risks

See [`RESIDUAL_RISKS.md`](RESIDUAL_RISKS.md) for accepted residual risks with documented risk treatment decisions.

**Summary**:
- **0 Critical unmitigated threats**
- **0 High unmitigated threats**
- **0 Medium unmitigated threats**

All identified threats have implemented mitigations. System is production-ready from a threat mitigation perspective.

---

## Synchronization Process

**Automated Sync** (Future):
```bash
# Planned: CI script to verify threat-mitigation consistency
./scripts/sync_threat_model.sh --verify
```

**Manual Sync** (Current):
1. Update `THREAT_MODEL.md` when identifying new threats
2. Implement mitigations and update `GAP_MATRIX.md`
3. Update this matrix with mitigation references
4. Update `RESIDUAL_RISKS.md` if risk is accepted

**Verification**: Every quarter, security team reviews this matrix for completeness.

---

## References
- [Threat Model](THREAT_MODEL.md)
- [GAP Matrix](GAP_MATRIX.md)
- [Residual Risks](RESIDUAL_RISKS.md)
- [Fail-Closed Advisory](FAIL_CLOSED_ADVISORY.md)
