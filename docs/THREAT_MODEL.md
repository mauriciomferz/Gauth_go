---
title: Threat Model Documentation
category: threat-model
status: active
lastUpdated: 2025-11-12
owners: security-team
source: internal
refreshCadence: quarterly
---

# GAuth Threat Model (RFC 0111 / 0115 Delegation Prototype)

> Status: Updated – October 17, 2025
> Scope: beta-refactor branch including replay protection (in-memory + Redis distributed JTI store), scope/restriction size limits, delegation issuance, revocation chain, token envelope (with JTI), BoltDB repository, metrics (replay hit/miss/store errors/latency), anchoring stub

## 1. Overview
The GAuth prototype implements RFC 0111 delegation (PowerOfAttorney) with optional RFC 0115 definition integration. This threat model documents assets, trust boundaries, key threats, existing mitigations, and gaps to guide hardening work toward production readiness.

## 2. Assets
| Asset | Description | Security Objectives |
|-------|-------------|---------------------|
| Delegation Records (POA) | JSON objects representing grantor→grantee rights | Integrity, Availability, Confidentiality (scope & restrictions) |
| Revocation Chain | Append-only hash chain of revocation events | Integrity (tamper-evidence), Non-repudiation (event trail) |
| Issuance Chain | Hash chain of issuance events (provenance) | Integrity, Traceability |
| Authorization Tokens (PASETO local) | Encrypted envelope referencing delegation | Confidentiality, Integrity, Authenticity |
| Canonical Digests & Signatures | Ed25519 signatures over canonical POA digest | Integrity, Authenticity |
| Symmetric Key Ring | Active + previous encryption keys | Confidentiality, Rotation Robustness |
| Asymmetric Key Provider | Source of Ed25519 public keys (in-memory today) | Availability, Authenticity |
| BoltDB Persistence File | Durable store for delegations | Integrity, Availability |
| Audit Log (memory/file) | Security-relevant events | Integrity, Non-repudiation |
| Metrics / Observability Data | Operational counters and latencies | Integrity (anti-tamper for SLO), Availability |
| Anchor Client Hashes | External anchoring of chain tips | Integrity (external witness) |

## 3. Actors
| Actor | Motivation | Capabilities |
|-------|-----------|--------------|
| Legitimate Grantor | Delegates authority | Valid credentials, authorized actions |
| Legitimate Grantee | Executes delegated actions | Holds token, invokes validation |
| System Admin (future) | Maintenance & key rotation | Elevated access to keys & DB |
| External Auditor | Assesses integrity | Read-only access to chains, logs |
| Passive Network Attacker | Eavesdropping | Packet capture, timing analysis |
| Active Network Attacker | Manipulation, replay | Tamper with messages, attempt forgery |
| Malicious Insider | Abuse legitimate access | Read/Write POA DB, key exposure |
| Token Thief | Uses stolen token | Replay of valid encrypted token |
| Key Compromise Attacker | Extracts encryption/signing key | Decrypts tokens / forges signatures |

## 4. Trust Boundaries
1. Service Process ↔ External Clients (HTTP/API)
2. Service ↔ BoltDB File System (host FS permissions)
3. Service ↔ Anchor External System (stub; future real network boundary)
4. Service ↔ Metrics Backend (Prometheus scrape interface)
5. In-memory Components (key ring, signer, canonical digest) ↔ Untrusted input

## 5. Threat Scenarios
| ID | Threat | Description | Impact | Likelihood (Current) |
|----|--------|-------------|--------|----------------------|
| T1 | Token Replay | Stolen token used within validity window | Unauthorized action | Low (distributed JTI store + metrics) |
| T2 | Token Forgery (Symmetric Key) | Attacker acquires token symmetric key and mint tokens | Privilege escalation | Low-Med (key mgmt immature; replay cache limits reuse of single token) |
| T3 | Signature Forgery | Ed25519 private key compromise enables fake POA authenticity | Integrity loss | Low (key ephemeral/in-memory) |
| T4 | Revocation Bypass | Failure to check revocation chain integrity or event omission | Continued unauthorized use | Low (chain verified each validation) |
| T5 | Chain Tampering | Manipulated hash chain history (revocation/issuance) | Audit trail corruption | Low-Med (no external witness in prod yet) |
| T6 | Persistence Tampering | Direct editing of BoltDB file | Silent data modification | Medium (no file integrity seal) |
| T7 | Principal Enumeration | ListByPrincipal reveals all delegations via insecure endpoint | Privacy leakage | Dependent on API exposure |
| T8 | Key Rotation Gaps | Old keys retained too long enabling extended replay | Increased replay window | Medium |
| T9 | Missing Public Key (Strict Mode) | Authenticity silently skipped allowing downgrade | Integrity loss | Medium (strict mode opt‑in) |
| T10 | DOS via Large Scope/Restrictions | Oversized JSON causing memory pressure | Availability | Low (size limits enforced) |
| T11 | Timing Side Channel | Signature verification timing leaks key info | Confidentiality | Low (Ed25519 constant-time) |
| T12 | Anchor Spoofing | Fake success anchoring hides external tampering | Integrity | Medium (Noop client) |

## 6. Existing Mitigations
| Threats | Mitigations |
|---------|------------|
| T1 | Short-lived tokens (delegation duration bounded), status/expiry checks each validation, JTI replay cache (first-seen acceptance then reject), replay metrics for detection |
| T2 | Key ring rotation support (active + previous), planned tighter retention policy |
| T3 | Canonical digest + Ed25519; signature verification path (strict mode flags missing keys) |
| T4 | Revocation chain integrity verification on each validation; status synchronization |
| T5 | Hash chain for issuance & revocation; external anchor attempt metrics |
| T6 | BoltDB append-only semantics with structured JSON; need host FS perms |
| T7 | No public API exposing raw index yet; authorization gating required |
| T8 | Rotation function; tests ensuring previous keys still work then retire plan pending |
| T9 | `strictAuthenticity` option returns integrity failure when public key missing |
| T10 | Explicit size caps on scope slice length, individual scope string length, restrictions count, key/value lengths |
| T11 | Use Ed25519 (designed for constant-time verification) |
| T12 | Metrics fold anchor attempts + failures to detect persistent failure mode |

## 7. Gaps & Weaknesses
- (Addressed) Token replay protection added: JTI issuance & in-memory + distributed (Redis) replay store. Remaining gap: persistent replay window shrink on cold restart (no durable recent JTI snapshot yet).
- Public key distribution and persistence missing (verification relies on key ring only).
- No integrity seal / checksum for BoltDB file (tampering offline undetected on startup).
- Lack of structured authorization policy around List operations (privacy risk).
- Anchor client is a Noop (no real external witness enforceability).
- No rate limiting / circuit breakers on validation endpoint for DOS.
- Soft skip authenticity still allowed when not strict, enabling downgrade attack.
- Absence of encryption-at-rest for BoltDB file.

## 8. Recommended Actions (Prioritized)
| Priority | Action | Rationale |
|----------|--------|-----------|
| P1 | Integrate KeyProvider fully into `VerifyToken` and `ValidateDelegationCtx` | Close authenticity gap / prevent silent downgrade |
| P2 | Implement token replay prevention (nonce/jti + memory/time index) | Completed (in-memory cache). Follow-up: distributed cache / persistence |
| P3 | Add file integrity seal (hash+HMAC) verified at startup | Detect persistence tampering (T6) |
| P4 | Replace Noop Anchor with pluggable external (e.g., append hash to external log) | Strengthen chain tamper detection (T5/T12) |
| P5 | Enforce size limits on scope/restrictions & validate UTF-8 | Completed (size limits enforced; UTF-8 validation pending) |
| P6 | Introduce retention policy + automatic purge of expired delegations | Reduce risk surface & DB bloat |
| P7 | Add optional BoltDB value encryption (AEAD) | Protect confidentiality at rest |
| P8 | Add rate limit on validation endpoint | Availability protection |
| P9 | Authorization check on listing delegations | Privacy boundary control |
| P10 | Periodic external audit log hash anchoring (separate from chains) | Strengthen non-repudiation |

## 9. Abuse Cases & Responses
| Abuse Case | Detection | Response |
|------------|----------|----------|
| Rapid replay of a stolen token | ReplayHits metric increases sharply vs ReplayMisses baseline | Rate limit + revoke delegation; investigate key compromise; consider invalidating key ring if pattern widespread |
| Anchor failure suppression | Anchor failure counter high / attempts constant | Fallback secondary anchor target; alert security team |
| Offline DB modification | Startup hash mismatch (after seal introduction) | Quarantine node; rebuild from verified backup |
| Signature downgrade (missing key) | Public key missing metric increase | Enforce strict mode globally; rotate key distribution mechanism |

## 10. Future Enhancements
- Integrate OpenTelemetry tracing with security span attributes (delegation_id, result_code).
- Explore multi-region replication integrity (Merkle tree of chain tips).
- Introduce formal policy engine for restriction evaluation (avoid ad-hoc parsing).
- Add structured security events for anomaly detection (e.g., consecutive integrity failures).

## 11. Residual Risks
Even after recommended actions, residual risks include key compromise via host intrusion, sophisticated supply chain attacks on dependency libraries, and advanced cryptographic attacks beyond Ed25519 assumptions (currently considered low likelihood).

## 12. Glossary
- POA: Power Of Attorney (delegation object)
- Chain Tip: Latest hash of issuance or revocation chain
- Anchor: External committing of hash to external system (e.g., transparency log)
- Strict Authenticity: Mode enforcing failure when signature public key is missing.

---
*This document is living; update as architecture evolves (e.g., when distributed replay cache, real anchoring, and file integrity seals are implemented).*

---

## 13. Capability Anchor & Receipt Chain Extension (beta-refactor additions)

### New Assets
| Asset | Description | Security Objectives |
|-------|-------------|---------------------|
| Capability Anchor Artifact | Periodic hash of registry state (sha256 + optional signature) | Integrity, Freshness |
| Notarization Receipts | Append-only chain of receipt objects (hash, timestamp, provider) | Integrity, Ordering |
| Receipt Chain Hash | Head chain hash (sha256(prev_hash || entry_bytes)) | Tamper-evidence |
| Merkle Root (optional) | Tree root summarizing all entries up to point | Efficient verification |
| Rotation Descriptor | Metadata linking old/new signing keys | Continuity, Non-repudiation |
| Snapshot Object (planned) | Authenticated summary enabling pruning | Integrity, Scalability |

### Additional Threats
| ID | Threat | Description | Impact | Mitigations (Current/Planned) |
|----|--------|-------------|--------|------------------------------|
| CA1 | Receipt Chain Tampering | Modify/insert/delete entries | Integrity loss | Hash chain verification, integrity gauge, mismatch tests, future external anchoring |
| CA2 | Merkle Root Forgery | Recompute with altered entry set | Auditor deception | Recompute from raw entries, chain hash mismatch still reveals tampering, planned signed snapshots |
| CA3 | Suppressed Verification | Adaptive interval abused to delay checks | Delayed detection | Freshness threshold & verify=1 trigger, last verify age gauge |
| CA4 | Rotation Descriptor Injection | Fake rotation event inserted | Key continuity break | Planned dual-signature, continuity hash (PrevRotationHash) |
| CA5 | Snapshot Rollback (planned) | Replace latest snapshot with older valid one | Hide tampering | Snapshot hash chain + timestamp monotonicity + external anchoring |
| CA6 | External Provider Replay | Replay stub TSA/tlog metadata | False sense of inclusion | Real integration will validate cryptographic signatures and inclusion proofs |
| CA7 | Merkle Performance DoS | Large tree recomputation each append | CPU exhaustion | Incremental optimization pending, adaptive verification frequency, feature flag disable |

### Metrics-Based Detection Enhancements
- `capability_anchor_notarization_receipts_integrity` gauge transitions to 0 on mismatch; alert immediate.
- `capability_anchor_notarization_receipts_last_verify_age_seconds` rising beyond SLA triggers forced re-verify.
- Potential future metric: merkle_build_latency_seconds (histogram) to detect performance abnormal spikes.

### Roadmap Security Controls (Anchor Layer)
Short Term:
1. Add tests validating Merkle root field reproducibility & absence when disabled.
2. Document rotation dual-signature requirement & implement.

Medium Term:
1. Implement snapshot signature & previous snapshot hash chain.
2. Integrate real TSA (RFC3161) and transparency log (Rekor) with proof validation.
3. Add pruning policy engine (retention + external anchor preconditions).

Long Term:
1. Provide offline auditor CLI to verify snapshot -> receipts -> Merkle root -> chain head -> external proofs.
2. Formal verification of chain hash & merkle computation correctness.

### Residual Risks Post-Roadmap
- Until real external services & signatures are in place, local tampering may persist if detection disabled and host compromised.
- Rotation descriptor still unsigned allowing theoretical injection.
- Merkle recomputation cost at scale could lead to verification delays if not optimized.

### Open Questions (Anchor Layer)
- Should each receipt include per-entry signature for stronger authenticity?
- Do we anchor Merkle root or chain head (or both) externally for redundancy?
- How frequently should snapshots be externally anchored vs internal-only?
