---
title: Readme
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# AgentAuth Cryptography Module

> Last Updated: 2025-10-24
> Status: Active

This module implements all major cryptographic primitives and key management features for the AgentAuth RFC demonstration, including:

## Features
- Ed25519, ECDSA, and BLS signature schemes
- Batch signature verification
- Key rotation (Vault, KMS, file keystore)
- Canonical JSON serialization for hash/signature integrity
- Audit log with hash chain and signature enforcement
- Enforcement flags for compliance and attestation
- Cross-language test vectors for cryptographic validation

## Canonical JSON & Hash Chain Logic
- All hashes and signatures are computed over canonical JSON (stable key ordering)
- Hash chain logic ensures tamper-evident audit logs for key rotation events
- Canonicalization is enforced in both production and test code

## Usage Example
```go
// ECDSA signing
priv, pub := GenerateECDSAKeyPair()
sig, err := SignECDSA(priv, data)
valid := VerifyECDSA(pub, data, sig)

// BLS batch verification
valid := BatchVerifyBLS(pubKeys, messages, signatures)

// Key rotation
err := RotateKey(store, oldKey, newKey)
```

## Compliance & Audit
- All cryptographic operations are covered by integration and unit tests
- Audit log entries are signed and hash-chained
- Enforcement flags and attestation logic validated in compliance tests

## See Also
- [docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md)
- [docs/COMPLETE_API_REFERENCE.md](../../docs/COMPLETE_API_REFERENCE.md)
- [test/](../../test/)
- [examples/token_management/key_rotation.go](../../examples/token_management/key_rotation.go)

---
For RFC details, see [docs/rfc/README.md](../../docs/rfc/README.md)
