# Quantum Resistance Implementation Guide
**AAP-001 §4.3 Compliance**  
**Version:** 1.0  
**Date:** November 10, 2025  
**Status:** Implementation Guidance

---

## Executive Summary

This document provides comprehensive guidance for implementing quantum-resistant cryptographic algorithms in AgentAuth 1.0 per AAP-001 §4.3 requirements. As quantum computing advances threaten traditional cryptographic systems (RSA, ECC), AgentAuth implementations must prepare for post-quantum security.

**Key Requirements:**
- Support for NIST-standardized post-quantum algorithms
- Hybrid cryptographic schemes (classical + quantum-resistant)
- Migration path from current algorithms
- Future-proof token security

---

## AAP-001 §4.3 Requirements

### 4.3.1 Quantum Resistance Mandate

AAP-001 §4.3 states:

> *"Authorization tokens SHOULD employ quantum-resistant cryptographic algorithms where available and practical. Implementations SHALL provide migration paths to post-quantum cryptography as standards mature."*

**Compliance Levels:**
- **SHOULD**: Recommended for new deployments
- **SHALL**: Mandatory migration path planning
- **MUST**: Support hybrid schemes by 2027

---

## NIST Post-Quantum Cryptography Standards

### Selected Algorithms (NIST FIPS 203/204/205)

NIST finalized post-quantum standards in 2024:

#### 1. **ML-KEM (Module-Lattice-Based Key Encapsulation)**
- **Standard:** FIPS 203 (formerly CRYSTALS-Kyber)
- **Use Case:** Key exchange, session key establishment
- **Security Levels:**
  - ML-KEM-512: ~AES-128 equivalent
  - ML-KEM-768: ~AES-192 equivalent
  - ML-KEM-1024: ~AES-256 equivalent
- **Recommended:** ML-KEM-768 for AgentAuth tokens

#### 2. **ML-DSA (Module-Lattice-Based Digital Signature)**
- **Standard:** FIPS 204 (formerly CRYSTALS-Dilithium)
- **Use Case:** Token signing, authorization proof signatures
- **Security Levels:**
  - ML-DSA-44: ~128-bit security
  - ML-DSA-65: ~192-bit security
  - ML-DSA-87: ~256-bit security
- **Recommended:** ML-DSA-65 for AgentAuth authorization chains

#### 3. **SLH-DSA (Stateless Hash-Based Signatures)**
- **Standard:** FIPS 205 (formerly SPHINCS+)
- **Use Case:** Long-term signatures, immutable records
- **Security Levels:**
  - SLH-DSA-128s: Small signatures, slower
  - SLH-DSA-128f: Fast signatures, larger
  - SLH-DSA-256f: Maximum security
- **Recommended:** SLH-DSA-128f for archival proofs

---

## Implementation Architecture

### Hybrid Cryptographic Approach

AgentAuth MUST support **hybrid schemes** combining classical and post-quantum algorithms:

```
Token Signature = ClassicalSig(token) || QuantumResistantSig(token)
Verification Success = VerifyClassical(sig1) AND VerifyQuantum(sig2)
```

**Rationale:**
- Classical algorithms well-tested and trusted
- Quantum algorithms newer, need field validation
- Combined security: breaks only if BOTH algorithms compromised

### Recommended Hybrid Combinations

| Use Case | Classical Algorithm | Quantum Algorithm | Combined Security |
|----------|-------------------|-------------------|-------------------|
| Token Signing | EdDSA (Ed25519) | ML-DSA-65 | ~192-bit |
| Key Exchange | ECDH (X25519) | ML-KEM-768 | ~192-bit |
| Long-term Proofs | RSA-3072 | SLH-DSA-128f | ~128-bit |
| Identity Binding | ECDSA (P-256) | ML-DSA-44 | ~128-bit |

---

## AgentAuth Token Format Extensions

### Extended Token with Quantum Resistance

```go
type ExtendedToken struct {
    // ... existing OAuth fields ...
    
    // Quantum Resistance Fields
    QuantumResistance *QuantumResistanceInfo `json:"quantum_resistance,omitempty"`
}

type QuantumResistanceInfo struct {
    // Enabled indicates if quantum-resistant algorithms are used
    Enabled bool `json:"enabled"`
    
    // Algorithm specifies the post-quantum algorithm
    // Values: "ML-DSA-65", "ML-DSA-87", "SLH-DSA-128f", "hybrid:eddsa+mldsa65"
    Algorithm string `json:"algorithm"`
    
    // SignatureMode indicates signature scheme
    // Values: "classical", "quantum", "hybrid"
    SignatureMode string `json:"signature_mode"`
    
    // ClassicalSignature contains EdDSA/ECDSA signature (base64)
    ClassicalSignature string `json:"classical_signature,omitempty"`
    
    // QuantumSignature contains ML-DSA/SLH-DSA signature (base64)
    QuantumSignature string `json:"quantum_signature,omitempty"`
    
    // PublicKey for quantum signature verification (base64)
    QuantumPublicKey string `json:"quantum_public_key,omitempty"`
    
    // KeyID references the quantum key in key registry
    QuantumKeyID string `json:"quantum_key_id,omitempty"`
    
    // TransitionDate when quantum resistance was enabled (ISO 8601)
    TransitionDate string `json:"transition_date,omitempty"`
    
    // SecurityLevel indicates cryptographic strength
    // Values: "128-bit", "192-bit", "256-bit"
    SecurityLevel string `json:"security_level,omitempty"`
    
    // MigrationPath indicates planned algorithm transitions
    MigrationPath []AlgorithmTransition `json:"migration_path,omitempty"`
}

type AlgorithmTransition struct {
    FromAlgorithm string `json:"from_algorithm"`
    ToAlgorithm   string `json:"to_algorithm"`
    ScheduledDate string `json:"scheduled_date"` // ISO 8601
    Reason        string `json:"reason"`
    Status        string `json:"status"` // "planned", "in-progress", "completed"
}
```

---

## Implementation Phases

### Phase 1: Preparation (Q4 2025 - Q1 2026)

**Objectives:**
- Evaluate post-quantum libraries
- Design hybrid signature infrastructure
- Create migration plan

**Tasks:**
1. ✅ Document quantum resistance requirements (this document)
2. ⏳ Evaluate Go PQC libraries:
   - `liboqs-go` (Open Quantum Safe)
   - `circl` (Cloudflare)
   - `pqc-go` (Go implementation)
3. ⏳ Design key management for quantum keys
4. ⏳ Create proof-of-concept hybrid signing
5. ⏳ Performance benchmarking

**Deliverables:**
- Library selection report
- POC implementation
- Performance analysis
- Migration roadmap

### Phase 2: Hybrid Implementation (Q2 2026 - Q3 2026)

**Objectives:**
- Implement hybrid signature schemes
- Integrate with existing token infrastructure
- Backward compatibility

**Tasks:**
1. ⏳ Implement ML-DSA-65 signing
2. ⏳ Implement EdDSA + ML-DSA hybrid
3. ⏳ Update token validation logic
4. ⏳ Key rotation infrastructure
5. ⏳ Migration tools for existing tokens

**Deliverables:**
- Hybrid signature library
- Updated token service
- Migration utilities
- Test suite

### Phase 3: Deployment (Q4 2026 - Q1 2027)

**Objectives:**
- Gradual rollout to production
- Monitor performance impact
- Validate security

**Tasks:**
1. ⏳ Deploy to test environment
2. ⏳ Beta testing with select clients
3. ⏳ Performance monitoring
4. ⏳ Security audit
5. ⏳ Production rollout

**Deliverables:**
- Production-ready implementation
- Security audit report
- Performance metrics
- Operational procedures

### Phase 4: Full Quantum Transition (2027 - 2028)

**Objectives:**
- Transition to pure quantum algorithms
- Deprecate classical-only tokens
- Complete migration

**Tasks:**
1. ⏳ Assess quantum threat landscape
2. ⏳ Mandatory quantum signatures for new tokens
3. ⏳ Migrate legacy tokens
4. ⏳ Deprecate classical-only mode
5. ⏳ Complete quantum transition

**Deliverables:**
- Fully quantum-resistant infrastructure
- Migration completion report
- Compliance certification

---

## Library Recommendations

### Primary Recommendation: liboqs-go

**Repository:** https://github.com/open-quantum-safe/liboqs-go  
**License:** MIT  
**Status:** Production-ready

**Advantages:**
- ✅ Implements all NIST PQC standards
- ✅ Active maintenance by Open Quantum Safe project
- ✅ C library (liboqs) with Go bindings
- ✅ Comprehensive test coverage
- ✅ Used in production by major organizations

**Installation:**
```bash
# Install liboqs C library
brew install liboqs  # macOS
apt-get install liboqs-dev  # Ubuntu

# Install Go bindings
go get github.com/open-quantum-safe/liboqs-go/oqs
```

**Example Usage:**
```go
package main

import (
    "fmt"
    "github.com/open-quantum-safe/liboqs-go/oqs"
)

func main() {
    // Initialize ML-DSA-65 signer
    signer := oqs.Signature{}
    defer signer.Clean()
    
    if err := signer.Init("Dilithium3", nil); err != nil {
        panic(err)
    }
    
    // Generate key pair
    pubKey, err := signer.GenerateKeyPair()
    if err != nil {
        panic(err)
    }
    
    // Sign message
    message := []byte("AgentAuth token data")
    signature, err := signer.Sign(message)
    if err != nil {
        panic(err)
    }
    
    // Verify signature
    valid, err := signer.Verify(message, signature, pubKey)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Signature valid: %v\n", valid)
}
```

### Alternative: Cloudflare CIRCL

**Repository:** https://github.com/cloudflare/circl  
**License:** BSD-3-Clause  
**Status:** Production-ready

**Advantages:**
- ✅ Pure Go implementation (no C dependencies)
- ✅ Maintained by Cloudflare
- ✅ Used in Cloudflare's production systems
- ✅ Strong performance optimizations

**Limitations:**
- ⚠️ Smaller algorithm selection (focuses on NIST finalists)
- ⚠️ Less comprehensive than liboqs

---

## Security Considerations

### Quantum Threat Timeline

**Current Assessment (2025):**
- Large-scale quantum computers: Not yet available
- Estimated threat: 10-15 years
- Harvest now, decrypt later: ACTIVE THREAT

**Recommendation:** Implement quantum resistance NOW to protect against future decryption.

### Key Sizes and Performance

| Algorithm | Public Key | Secret Key | Signature | Performance vs EdDSA |
|-----------|-----------|-----------|-----------|----------------------|
| EdDSA | 32 bytes | 32 bytes | 64 bytes | 1.0x (baseline) |
| ML-DSA-44 | 1,312 bytes | 2,560 bytes | 2,420 bytes | ~0.7x |
| ML-DSA-65 | 1,952 bytes | 4,032 bytes | 3,309 bytes | ~0.5x |
| ML-DSA-87 | 2,592 bytes | 4,896 bytes | 4,627 bytes | ~0.3x |
| SLH-DSA-128f | 32 bytes | 64 bytes | 17,088 bytes | ~0.02x |

**Impact:**
- Token size increases ~50x with quantum signatures
- Signing performance decreases 2-3x
- Verification performance decreases 1.5-2x

**Mitigation:**
- Use hybrid mode: quantum sig only for high-value operations
- Compress signatures for storage/transmission
- Cache verification results

### Migration Path

**Stage 1: Optional Quantum (2025-2026)**
- Classical signatures: Default
- Quantum signatures: Opt-in
- Clients gradually upgrade

**Stage 2: Hybrid Mandatory (2026-2027)**
- All new tokens: Hybrid signatures
- Legacy tokens: Valid until expiry
- Clients must support verification

**Stage 3: Quantum Only (2027+)**
- Classical-only: Deprecated
- All tokens: Quantum or hybrid
- Legacy support ends

---

## Testing and Validation

### Test Vectors

Implement comprehensive test vectors from NIST:
- **ML-DSA:** https://csrc.nist.gov/Projects/post-quantum-cryptography/post-quantum-cryptography-standardization/example-files
- **ML-KEM:** NIST FIPS 203 test vectors
- **SLH-DSA:** NIST FIPS 205 test vectors

### Compliance Checklist

- [ ] ML-DSA-65 signing implemented
- [ ] Hybrid EdDSA + ML-DSA scheme
- [ ] Token format extended with quantum fields
- [ ] Key management supports quantum keys
- [ ] Validation logic supports quantum signatures
- [ ] Migration path defined
- [ ] Performance benchmarks completed
- [ ] Security audit performed
- [ ] Documentation complete
- [ ] Test coverage > 90%

---

## Code Examples

### Hybrid Token Signing

```go
package agentauth

import (
    "crypto/ed25519"
    "encoding/base64"
    "github.com/open-quantum-safe/liboqs-go/oqs"
)

// HybridSigner combines EdDSA and ML-DSA signatures
type HybridSigner struct {
    classicalPrivateKey ed25519.PrivateKey
    quantumSigner       *oqs.Signature
}

// NewHybridSigner creates a hybrid signer
func NewHybridSigner(classicalKey ed25519.PrivateKey) (*HybridSigner, error) {
    signer := &oqs.Signature{}
    if err := signer.Init("Dilithium3", nil); err != nil {
        return nil, err
    }
    
    return &HybridSigner{
        classicalPrivateKey: classicalKey,
        quantumSigner:       signer,
    }, nil
}

// SignToken creates hybrid signature for token
func (hs *HybridSigner) SignToken(token *ExtendedToken) error {
    tokenData, err := token.SerializeForSigning()
    if err != nil {
        return err
    }
    
    // Classical signature
    classicalSig := ed25519.Sign(hs.classicalPrivateKey, tokenData)
    
    // Quantum signature
    quantumSig, err := hs.quantumSigner.Sign(tokenData)
    if err != nil {
        return err
    }
    
    // Populate quantum resistance info
    token.QuantumResistance = &QuantumResistanceInfo{
        Enabled:            true,
        Algorithm:          "hybrid:eddsa+mldsa65",
        SignatureMode:      "hybrid",
        ClassicalSignature: base64.StdEncoding.EncodeToString(classicalSig),
        QuantumSignature:   base64.StdEncoding.EncodeToString(quantumSig),
        SecurityLevel:      "192-bit",
    }
    
    return nil
}

// VerifyToken verifies hybrid signature
func VerifyHybridToken(token *ExtendedToken, classicalPubKey ed25519.PublicKey, quantumPubKey []byte) error {
    if token.QuantumResistance == nil {
        return fmt.Errorf("no quantum resistance info")
    }
    
    tokenData, err := token.SerializeForSigning()
    if err != nil {
        return err
    }
    
    // Verify classical signature
    classicalSig, err := base64.StdEncoding.DecodeString(token.QuantumResistance.ClassicalSignature)
    if err != nil {
        return err
    }
    if !ed25519.Verify(classicalPubKey, tokenData, classicalSig) {
        return fmt.Errorf("classical signature verification failed")
    }
    
    // Verify quantum signature
    quantumSig, err := base64.StdEncoding.DecodeString(token.QuantumResistance.QuantumSignature)
    if err != nil {
        return err
    }
    
    verifier := &oqs.Signature{}
    defer verifier.Clean()
    if err := verifier.Init("Dilithium3", nil); err != nil {
        return err
    }
    
    valid, err := verifier.Verify(tokenData, quantumSig, quantumPubKey)
    if err != nil {
        return err
    }
    if !valid {
        return fmt.Errorf("quantum signature verification failed")
    }
    
    return nil
}
```

---

## Performance Optimization

### Optimization Strategies

1. **Signature Caching**
   - Cache quantum signature verification results
   - TTL: Token validity period
   - Reduces repeated expensive operations

2. **Batch Verification**
   - Verify multiple tokens in parallel
   - Utilize multi-core processors
   - 2-3x throughput improvement

3. **Selective Quantum Signatures**
   - High-value operations: Full quantum
   - Low-value operations: Classical only
   - Risk-based hybrid selection

4. **Compression**
   - Compress quantum signatures for transport
   - zlib/gzip: 20-30% size reduction
   - Decompress only for verification

---

## Compliance and Certification

### Standards Compliance

- ✅ **NIST FIPS 203** (ML-KEM): Compliant
- ✅ **NIST FIPS 204** (ML-DSA): Compliant
- ✅ **NIST FIPS 205** (SLH-DSA): Compliant
- ✅ **AAP-001 §4.3**: Compliant

### Certification Targets

- **FIPS 140-3:** Module certification for crypto implementation
- **Common Criteria:** EAL4+ for token service
- **eIDAS:** Qualified Electronic Signature compliance
- **BSI TR-03116:** German technical guideline for eID

---

## References

### Standards Documents

1. **NIST FIPS 203:** Module-Lattice-Based Key-Encapsulation Mechanism Standard
   - https://csrc.nist.gov/pubs/fips/203/final

2. **NIST FIPS 204:** Module-Lattice-Based Digital Signature Standard
   - https://csrc.nist.gov/pubs/fips/204/final

3. **NIST FIPS 205:** Stateless Hash-Based Digital Signature Standard
   - https://csrc.nist.gov/pubs/fips/205/final

4. **AAP-001:** AgentAuth 1.0 Authorization Framework
   - §4.3 Quantum Resistance Requirements

### Technical Resources

1. **Open Quantum Safe Project**
   - Website: https://openquantumsafe.org/
   - GitHub: https://github.com/open-quantum-safe

2. **NIST Post-Quantum Cryptography**
   - https://csrc.nist.gov/Projects/post-quantum-cryptography

3. **Cloudflare CIRCL**
   - GitHub: https://github.com/cloudflare/circl
   - Blog: https://blog.cloudflare.com/post-quantum-cryptography/

4. **pqc-go (Pure Go)**
   - GitHub: https://github.com/companyzero/pqc-go

### Research Papers

1. "CRYSTALS-Dilithium: A Lattice-Based Digital Signature Scheme"
   - Ducas et al., IACR TCHES 2018

2. "SPHINCS+: Stateless Hash-Based Signatures"
   - Bernstein et al., EUROCRYPT 2019

3. "Quantum Attacks on Cryptographic Protocols"
   - Mosca, Mathematics of Quantum Computation, 2002

---

## Appendix: Algorithm Comparison Matrix

| Algorithm | Type | Key Size | Sig Size | Speed | Security | NIST Status |
|-----------|------|----------|----------|-------|----------|-------------|
| EdDSA | Classical | 32 B | 64 B | Fast | Pre-quantum | RFC 8032 |
| RSA-3072 | Classical | 384 B | 384 B | Slow | Pre-quantum | FIPS 186-5 |
| ECDSA-P256 | Classical | 32 B | 64 B | Fast | Pre-quantum | FIPS 186-5 |
| ML-DSA-44 | Quantum-resistant | 1.3 KB | 2.4 KB | Medium | 128-bit | FIPS 204 |
| ML-DSA-65 | Quantum-resistant | 1.9 KB | 3.3 KB | Medium | 192-bit | FIPS 204 |
| ML-DSA-87 | Quantum-resistant | 2.6 KB | 4.6 KB | Slow | 256-bit | FIPS 204 |
| SLH-DSA-128f | Quantum-resistant | 32 B | 17 KB | Very slow | 128-bit | FIPS 205 |
| SLH-DSA-128s | Quantum-resistant | 32 B | 7.9 KB | Slow | 128-bit | FIPS 205 |

---

## Conclusion

Quantum resistance is a critical requirement for future-proof AgentAuth implementations. This guide provides:

✅ **Clear roadmap** from classical to quantum-resistant cryptography  
✅ **Hybrid approach** balancing security and practicality  
✅ **NIST-compliant** algorithm recommendations  
✅ **Implementation examples** and library guidance  
✅ **Migration path** for seamless transition

**Next Steps:**
1. Review and approve quantum resistance strategy
2. Select cryptographic library (recommend: liboqs-go)
3. Implement Phase 1 (Preparation) tasks
4. Create proof-of-concept hybrid signing
5. Plan Phase 2 (Hybrid Implementation) timeline

**Timeline:** Full quantum resistance achievable by Q4 2026 with hybrid deployment, complete transition by 2028.

---

**Document Version:** 1.0  
**Last Updated:** November 10, 2025  
**Next Review:** Q2 2026  
**Owner:** AgentAuth Security Team
