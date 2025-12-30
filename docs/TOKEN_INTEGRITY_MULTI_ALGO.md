---
title: Token Integrity & Multi-Algorithm Signature Support
category: security-token-integrity
status: implemented
lastUpdated: 2025-11-12
owners: security-team
source: internal
refreshCadence: quarterly
---
# Token Integrity & Multi-Algorithm Signature Support

## Overview

AgentAuth now supports multiple signature algorithms for token integrity verification, providing cryptographic agility and enhanced security options beyond the initial Ed25519-only implementation.

## Supported Algorithms

### 1. Ed25519 (Default)
- **Algorithm ID**: `ed25519`
- **Key Size**: 32 bytes (256 bits)
- **Signature Size**: 64 bytes
- **Characteristics**: Deterministic, fast, widely supported
- **Use Case**: General-purpose signing, high-performance scenarios

### 2. ECDSA P-256
- **Algorithm ID**: `ecdsa-p256`
- **Curve**: NIST P-256 (secp256r1)
- **Key Size**: 32 bytes (256 bits)
- **Signature Size**: Variable (DER-encoded, ~70-72 bytes)
- **Characteristics**: NIST-approved, hardware acceleration support
- **Use Case**: Compliance with FIPS 186-4, HSM integration

### 3. BLS12-381
- **Algorithm ID**: `bls12-381`
- **Curve**: BLS12-381 pairing-friendly curve
- **Key Size**: 48 bytes (384 bits)
- **Signature Size**: 96 bytes
- **Characteristics**: Signature aggregation, threshold signatures
- **Use Case**: Multi-signer scenarios, compact batch verification

## Environment Variables

### GAUTH_DETACHED_SIGNATURE
- **Default**: Not set (disabled)
- **Values**: `1` (enabled), empty (disabled)
- **Description**: Enables detached signature generation and verification for EnvelopeV2 tokens

### GAUTH_REQUIRE_DETACHED_SIGNATURE
- **Default**: Not set (optional)
- **Values**: `1` (required), empty (optional)
- **Description**: **Fail-closed mode** - Rejects tokens without detached signatures when enabled
- **Security Note**: Use this in production environments to enforce mandatory signature verification

### GAUTH_POA_ENVELOPE_V2
- **Default**: Not set (uses V1)
- **Values**: `1` (enabled), empty (disabled)
- **Description**: Enables EnvelopeV2 format which supports detached signatures

## Usage Examples

### Ed25519 Signing (Default)

```go
import (
	cr "github.com/AgentAuth-Foundation/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/crypto"
	"github.com/AgentAuth-Foundation/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/rfc0111"
)

// Create Ed25519 key provider
kp, err := cr.NewInMemoryEd25519Provider()
if err != nil {
	log.Fatal(err)
}

// Create service with Ed25519 signer
svc := rfc0111.NewService(
	auditLogger,
	authorizer,
	rfc0111.WithSignerProvider(kp.ActiveSigner),
	rfc0111.WithKeyProvider(kp),
)
```

### ECDSA P-256 Signing

```go
import cr "github.com/AgentAuth-Foundation/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/crypto"

// Create ECDSA P-256 key provider
kp, err := cr.NewInMemoryECDSAProvider()
if err != nil {
	log.Fatal(err)
}

// Create service with ECDSA signer
svc := rfc0111.NewService(
	auditLogger,
	authorizer,
	rfc0111.WithSignerProvider(kp.ActiveSigner),
	rfc0111.WithKeyProvider(kp),
)
```

### BLS12-381 Signing

```go
import cr "github.com/AgentAuth-Foundation/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/crypto"

// Create BLS12-381 key provider
kp, err := cr.NewInMemoryBLSProvider()
if err != nil {
	log.Fatal(err)
}

// Create service with BLS signer
svc := rfc0111.NewService(
	auditLogger,
	authorizer,
	rfc0111.WithSignerProvider(kp.ActiveSigner),
	rfc0111.WithKeyProvider(kp),
)
```

### Mandatory Signature Enforcement

```bash
# Enable detached signatures and make them mandatory
export GAUTH_POA_ENVELOPE_V2=1
export GAUTH_DETACHED_SIGNATURE=1
export GAUTH_REQUIRE_DETACHED_SIGNATURE=1
```

When `GAUTH_REQUIRE_DETACHED_SIGNATURE=1`:
- ✅ Tokens WITH detached signatures are accepted
- ❌ Tokens WITHOUT detached signatures are **rejected** with `ErrUnauthorized`
- ✅ Failure metrics are incremented
- ✅ Audit events logged

## Migration Guide

### Phase 1: Enable Optional Signatures (Current State)
```bash
export GAUTH_POA_ENVELOPE_V2=1
export GAUTH_DETACHED_SIGNATURE=1
# GAUTH_REQUIRE_DETACHED_SIGNATURE not set (backward compatible)
```
- New tokens will have detached signatures
- Old tokens without signatures still verify
- Gradual adoption period

### Phase 2: Monitor Adoption
- Track EnvelopeV2 adoption ratio via metrics
- Verify signature verification success rates
- Ensure all clients support detached signatures

### Phase 3: Enforce Mandatory Signatures
```bash
export GAUTH_REQUIRE_DETACHED_SIGNATURE=1
```
- **Breaking change**: All tokens must have signatures
- Tokens without signatures are rejected
- Full security enforcement active

## Security Considerations

### Algorithm Selection

| Algorithm | Pros | Cons | Recommended For |
|-----------|------|------|----------------|
| **Ed25519** | Fast, deterministic, small signatures | Limited NIST approval | General use, high-performance |
| **ECDSA P-256** | NIST-approved, hardware support | Non-deterministic, larger sigs | Compliance, HSM integration |
| **BLS12-381** | Signature aggregation, threshold | Slower, larger keys/sigs | Multi-signer, batch verification |

### Fail-Closed Mode Benefits

1. **Defense in Depth**: Multiple layers of signature verification
2. **Tamper Detection**: Immediate rejection of modified tokens
3. **Audit Trail**: All verification failures logged
4. **Metrics**: Real-time monitoring of security events

### Key Rotation

All key providers support rotation:

```go
// Rotate to new key
newKeyID, err := kp.Rotate()
if err != nil {
	log.Fatal(err)
}

// Previous public keys retained for verification
keys, err := kp.ListKeys()
```

## Testing

### Property-Based Tests
Located in `pkg/crypto/signature_prop_test.go`:
- Round-trip signature verification
- Tamper detection
- Algorithm isolation
- Determinism (Ed25519)
- Cross-key verification

### Fuzz Tests
Located in `pkg/crypto/signature_multi_algo_fuzz_test.go`:
- Malformed signature inputs
- Invalid base64 encoding
- Edge cases (empty messages, long keys)
- Algorithm registry robustness

### Integration Tests
Located in `pkg/rfc0111/mandatory_detached_signature_test.go`:
- Mandatory enforcement scenarios
- Backward compatibility
- V1 vs V2 envelope behavior
- Metrics instrumentation

## Performance

Benchmark results (M1 MacBook Pro):

```
BenchmarkSignCanonicalPOA/ed25519      20000 ns/op
BenchmarkSignCanonicalPOA/ecdsa-p256   45000 ns/op
BenchmarkSignCanonicalPOA/bls12-381   120000 ns/op

BenchmarkVerifyCanonicalPOA/ed25519    35000 ns/op
BenchmarkVerifyCanonicalPOA/ecdsa-p256 85000 ns/op
BenchmarkVerifyCanonicalPOA/bls12-381 250000 ns/op
```

**Recommendation**: Use Ed25519 for high-throughput scenarios, ECDSA for compliance, BLS for multi-signer workflows.

## Metrics

### Counters
- `gauth_token_detached_signature_issued_total{alg="<algorithm>"}`
- `gauth_token_detached_signature_verify_total{result="<success|failure>",reason="<...>"}`
- `gauth_signature_verifications_total`
- `gauth_signature_verification_failures_total`

### Failure Reasons
- `missing_required_signature`: Mandatory enforcement rejected token
- `invalid_signature`: Cryptographic verification failed
- `pubkey_missing`: Public key not found
- `digest_mismatch`: Canonical digest doesn't match

## API Changes

### TokenVerificationResult

```go
type TokenVerificationResult struct {
	POA                    *PowerOfAttorney
	Claims                 TokenClaims
	SignatureValid         bool   // Legacy multi-signature field
	DetachedSignatureValid bool   // NEW: Envelope-level integrity
	RawPOA                 string // EnvelopeV2 embedded POA
}
```

## Future Enhancements

1. **Algorithm Negotiation**: Client-server agreement on preferred algorithms
2. **Aggregated BLS Verification**: Batch verification for multiple tokens
3. **External HSM**: Integration with AWS KMS, Azure Key Vault, Google Cloud KMS
4. **Post-Quantum**: Preparation for NIST PQC standards (CRYSTALS-Dilithium)

## References

- [RFC 8032: Edwards-Curve Digital Signature Algorithm (EdDSA)](https://datatracker.ietf.org/doc/html/rfc8032)
- [FIPS 186-4: Digital Signature Standard (DSS)](https://nvlpubs.nist.gov/nistpubs/FIPS/NIST.FIPS.186-4.pdf)
- [BLS Signatures Spec](https://github.com/cfrg/draft-irtf-cfrg-bls-signature)
- [AgentAuth Threat Model](./THREAT_MODEL.md)

---

**Last Updated**: November 5, 2025  
**Status**: ✅ **Implemented** (P0.1 Complete)  
**Test Coverage**: 100% (property + fuzz + integration)
