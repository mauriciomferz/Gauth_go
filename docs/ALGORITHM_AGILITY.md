---
title: Algorithm Agility
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Algorithm Agility in AgentAuth

## Overview

AgentAuth implements **algorithm agility**, supporting multiple cryptographic signature algorithms for authorization tokens. This enables gradual migration between algorithms, interoperability with diverse systems, and future-proofing against cryptographic advances.

### Supported Algorithms

| Algorithm | IANA ID | Key Size | Signature Size | Use Case |
|-----------|---------|----------|----------------|----------|
| **Ed25519** | EdDSA | 32 bytes | 64 bytes | Default - Fast, compact signatures |
| **RSA-PSS** | PS256 | 2048-4096 bits | 256-512 bytes | Legacy compatibility, regulatory compliance |
| **ECDSA P-256** | ES256 | 256 bits | ~72 bytes | Balance of security and performance |

## Architecture

### SignatureAlgorithm Interface

All algorithms implement a common interface:

```go
type SignatureAlgorithm interface {
    Sign(privateKey interface{}, message []byte) ([]byte, error)
    Verify(publicKey interface{}, message, signature []byte) error
    KeyType() string
    AlgorithmID() string
    GenerateKey() (privateKey, publicKey interface{}, err error)
    MarshalPrivateKey(privateKey interface{}) ([]byte, error)
    UnmarshalPrivateKey(pemData []byte) (interface{}, error)
    MarshalPublicKey(publicKey interface{}) ([]byte, error)
    UnmarshalPublicKey(pemData []byte) (interface{}, error)
}
```

### Algorithm Registry

The `AlgorithmRegistry` manages available algorithms:

```go
// Get the default registry with all standard algorithms
registry := crypto.DefaultRegistry

// Retrieve a specific algorithm
provider, err := registry.Get("EdDSA")  // or "PS256", "ES256"

// Register custom algorithm
customRSA := crypto.NewRSAPSSProvider(4096)
registry.Register("PS256-4096", customRSA)

// List available algorithms
algorithms := registry.ListAlgorithms()  // ["EdDSA", "PS256", "ES256"]
```

## Usage Examples

### Basic Signature Operations

#### Ed25519 (Default - Fastest)

```go
import "github.com/agentauth/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/crypto"

// Create provider
provider := &crypto.Ed25519Provider{}

// Generate key pair
privKey, pubKey, err := provider.GenerateKey()
if err != nil {
    log.Fatalf("Key generation failed: %v", err)
}

// Sign message
message := []byte("authorization request payload")
signature, err := provider.Sign(privKey, message)
if err != nil {
    log.Fatalf("Signing failed: %v", err)
}

// Verify signature
err = provider.Verify(pubKey, message, signature)
if err != nil {
    log.Printf("Verification failed: %v", err)
}
```

#### RSA-PSS (Legacy Compatibility)

```go
// Create provider with 3072-bit keys (recommended for new deployments)
provider := crypto.NewRSAPSSProvider(3072)

// Generate key pair
privKey, pubKey, err := provider.GenerateKey()

// Sign and verify (same as Ed25519)
signature, _ := provider.Sign(privKey, message)
err = provider.Verify(pubKey, message, signature)
```

#### ECDSA P-256 (Balanced)

```go
// Create provider
provider := &crypto.ECDSAP256Provider{}

// Generate key pair
privKey, pubKey, err := provider.GenerateKey()

// Sign and verify
signature, _ := provider.Sign(privKey, message)
err = provider.Verify(pubKey, message, signature)
```

### Key Management

#### PEM Encoding/Decoding

```go
// Export private key to PEM
privPEM, err := provider.MarshalPrivateKey(privKey)
// Save to file: ioutil.WriteFile("private.pem", privPEM, 0600)

// Import private key from PEM
privKey, err := provider.UnmarshalPrivateKey(privPEM)

// Export public key to PEM
pubPEM, err := provider.MarshalPublicKey(pubKey)
// Save to file: ioutil.WriteFile("public.pem", pubPEM, 0644)

// Import public key from PEM
pubKey, err := provider.UnmarshalPublicKey(pubPEM)
```

#### Key Rotation Workflow

```go
// 1. Generate new key pair with different algorithm
oldProvider := &crypto.Ed25519Provider{}
newProvider := crypto.NewRSAPSSProvider(3072)

newPrivKey, newPubKey, _ := newProvider.GenerateKey()

// 2. Dual-signing period: sign with both keys
oldSig, _ := oldProvider.Sign(oldPrivKey, message)
newSig, _ := newProvider.Sign(newPrivKey, message)

// 3. Verification tries both algorithms
if err := oldProvider.Verify(oldPubKey, message, oldSig); err == nil {
    log.Println("Verified with old key")
}
if err := newProvider.Verify(newPubKey, message, newSig); err == nil {
    log.Println("Verified with new key")
}

// 4. After migration period, switch to new key only
```

## Integration with AgentAuth

### Token Signing

Add algorithm identifier to token header:

```go
import (
    "encoding/json"
    "github.com/agentauth/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/crypto"
)

type TokenHeader struct {
    Algorithm string `json:"alg"`
    Type      string `json:"typ"`
}

func SignToken(provider crypto.SignatureAlgorithm, privKey interface{}, payload []byte) (string, error) {
    // Create header
    header := TokenHeader{
        Algorithm: provider.AlgorithmID(),  // "EdDSA", "PS256", or "ES256"
        Type:      "AgentAuth",
    }
    
    headerJSON, _ := json.Marshal(header)
    
    // Base64url encode header and payload
    headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
    payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
    
    // Sign header.payload
    signingInput := headerB64 + "." + payloadB64
    signature, err := provider.Sign(privKey, []byte(signingInput))
    if err != nil {
        return "", err
    }
    
    signatureB64 := base64.RawURLEncoding.EncodeToString(signature)
    
    return signingInput + "." + signatureB64, nil
}
```

### Token Verification

Parse algorithm from header and use appropriate provider:

```go
func VerifyToken(registry *crypto.AlgorithmRegistry, token string, pubKeys map[string]interface{}) error {
    parts := strings.Split(token, ".")
    if len(parts) != 3 {
        return errors.New("invalid token format")
    }
    
    // Decode header to determine algorithm
    headerJSON, _ := base64.RawURLEncoding.DecodeString(parts[0])
    var header TokenHeader
    json.Unmarshal(headerJSON, &header)
    
    // Get algorithm provider
    provider, err := registry.Get(header.Algorithm)
    if err != nil {
        return fmt.Errorf("unsupported algorithm: %s", header.Algorithm)
    }
    
    // Get public key for this algorithm
    pubKey, exists := pubKeys[header.Algorithm]
    if !exists {
        return fmt.Errorf("no public key for algorithm: %s", header.Algorithm)
    }
    
    // Verify signature
    signingInput := parts[0] + "." + parts[1]
    signature, _ := base64.RawURLEncoding.DecodeString(parts[2])
    
    return provider.Verify(pubKey, []byte(signingInput), signature)
}
```

## Migration Strategies

### Strategy 1: Immediate Cutover (Simple Systems)

For small deployments or controlled environments:

1. **Day 1**: Deploy new key with RSA-PSS or ECDSA
2. **Day 1**: Switch all signing to new algorithm
3. **Day 1**: Update verification to use new algorithm
4. **Day 2**: Retire Ed25519 keys

**Risk**: Requires synchronized deployment across all services.

### Strategy 2: Gradual Migration (Recommended)

For distributed systems with multiple services:

#### Phase 1: Dual Verification (Week 1-2)
```go
// Services accept BOTH algorithms
func VerifyTokenDual(token string) error {
    // Try Ed25519 first (existing)
    if err := verifyWithEd25519(token); err == nil {
        return nil
    }
    
    // Fall back to RSA-PSS (new)
    return verifyWithRSAPSS(token)
}
```

#### Phase 2: Dual Signing (Week 3-4)
```go
// Issuer signs with BOTH algorithms
newToken := SignToken(rsaProvider, rsaPrivKey, payload)
legacyToken := SignToken(ed25519Provider, ed25519PrivKey, payload)

// Return new token, keep legacy for backwards compatibility
return newToken
```

#### Phase 3: Monitor and Switch (Week 5-6)
- Monitor metrics: `agentauth_algorithm_usage{algorithm="PS256"}` should approach 100%
- Once all services updated, switch issuer to new algorithm only
- Stop signing with Ed25519

#### Phase 4: Cleanup (Week 7+)
- Remove Ed25519 verification fallback
- Delete old Ed25519 keys
- Update documentation

### Strategy 3: Service-by-Service Migration

Migrate one service at a time:

```go
type ServiceConfig struct {
    ServiceID         string
    SigningAlgorithm  string   // Algorithm this service uses to sign
    AcceptedAlgorithms []string // Algorithms this service accepts
}

var migrationSchedule = []ServiceConfig{
    {ServiceID: "api-gateway",     SigningAlgorithm: "PS256", AcceptedAlgorithms: []string{"EdDSA", "PS256"}},
    {ServiceID: "auth-service",    SigningAlgorithm: "EdDSA", AcceptedAlgorithms: []string{"EdDSA", "PS256"}},
    {ServiceID: "billing-service", SigningAlgorithm: "EdDSA", AcceptedAlgorithms: []string{"EdDSA", "PS256"}},
}
```

## Performance Comparison

Benchmarks on Apple M1 (arm64):

| Algorithm | Key Generation | Sign | Verify | Signature Size |
|-----------|----------------|------|--------|----------------|
| **Ed25519** | 0.05ms | 0.03ms | 0.08ms | 64 bytes |
| **RSA-PSS 2048** | 40ms | 0.8ms | 0.06ms | 256 bytes |
| **RSA-PSS 3072** | 150ms | 2.5ms | 0.12ms | 384 bytes |
| **RSA-PSS 4096** | 450ms | 6ms | 0.20ms | 512 bytes |
| **ECDSA P-256** | 0.2ms | 0.15ms | 0.30ms | ~72 bytes |

### Recommendations by Load

| Scenario | Recommended Algorithm | Rationale |
|----------|----------------------|-----------|
| High-throughput APIs | Ed25519 | Fastest sign/verify, smallest signatures |
| Banking/Financial | RSA-PSS 3072+ | Regulatory compliance, audit requirements |
| IoT/Embedded | Ed25519 | Minimal bandwidth, low compute |
| Enterprise SSO | ECDSA P-256 | Balance of security and compatibility |
| Microservices | Ed25519 | Low latency, high volume token exchange |

## Security Considerations

### Algorithm Selection Guidelines

1. **Ed25519** (Default)
   - ✅ Fast, compact, modern
   - ✅ No parameter configuration needed
   - ⚠️ May not be accepted by legacy systems
   - ⚠️ Less regulatory precedent than RSA

2. **RSA-PSS**
   - ✅ Widely accepted, regulatory compliant
   - ✅ Compatible with legacy systems
   - ⚠️ Slow key generation, large keys/signatures
   - ⚠️ Requires ≥2048 bits (3072+ recommended)

3. **ECDSA P-256**
   - ✅ Good performance, reasonable signature size
   - ✅ NIST-approved, widely supported
   - ⚠️ Signature non-determinism (implementation-dependent)
   - ⚠️ P-256 parameters controversial in some circles

### Key Size Requirements

| Algorithm | Minimum | Recommended | High Security |
|-----------|---------|-------------|---------------|
| Ed25519 | 256 bits | 256 bits | 256 bits |
| RSA-PSS | 2048 bits | 3072 bits | 4096 bits |
| ECDSA | P-256 | P-256 | P-384/P-521 |

**Note**: This implementation enforces minimum key sizes (e.g., RSA ≥2048 bits).

### Cross-Algorithm Verification Prevention

The implementation prevents cross-algorithm attacks:

```go
// This will FAIL - cannot verify Ed25519 signature with RSA key
ed25519Sig, _ := ed25519Provider.Sign(ed25519PrivKey, message)
err := rsaProvider.Verify(rsaPubKey, message, ed25519Sig)
// Error: "RSA-PSS verification failed"
```

Type safety ensures only matching key types are accepted.

## Testing

### Unit Tests

Run comprehensive test suite:

```bash
# All algorithm tests
go test ./pkg/crypto -v

# Specific algorithm
go test ./pkg/crypto -run TestEd25519
go test ./pkg/crypto -run TestRSAPSS
go test ./pkg/crypto -run TestECDSA

# Cross-algorithm verification
go test ./pkg/crypto -run TestCrossAlgorithm

# Registry operations
go test ./pkg/crypto -run TestAlgorithmRegistry

# Coverage
go test ./pkg/crypto -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Coverage

The implementation includes 18 test functions covering:
- ✅ Sign/verify operations for all algorithms
- ✅ Key generation and marshaling (PEM format)
- ✅ Metadata methods (KeyType, AlgorithmID)
- ✅ Registry CRUD operations
- ✅ Cross-algorithm verification prevention
- ✅ Invalid key type handling
- ✅ Malformed PEM data handling
- ✅ Signature consistency
- ✅ Concurrent access to registry
- ✅ RSA key size enforcement

### Integration Testing

Test token issuance and verification:

```bash
# Generate test tokens with all algorithms
./bin/agentauth-token-generator --algorithm EdDSA --output ed25519.token
./bin/agentauth-token-generator --algorithm PS256 --output rsa.token
./bin/agentauth-token-generator --algorithm ES256 --output ecdsa.token

# Verify tokens
./bin/agentauth-token-verifier --token ed25519.token --public-key ed25519.pub
./bin/agentauth-token-verifier --token rsa.token --public-key rsa.pub
./bin/agentauth-token-verifier --token ecdsa.token --public-key ecdsa.pub
```

## Monitoring

### Metrics

Track algorithm usage with Prometheus metrics:

```go
// In metrics exporter
var algorithmUsageCounter = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "agentauth_signature_operations_total",
        Help: "Total signature operations by algorithm and operation type",
    },
    []string{"algorithm", "operation"},  // operation: "sign" or "verify"
)

// Instrument signing
signature, err := provider.Sign(privKey, message)
algorithmUsageCounter.WithLabelValues(provider.AlgorithmID(), "sign").Inc()

// Instrument verification
err = provider.Verify(pubKey, message, signature)
algorithmUsageCounter.WithLabelValues(provider.AlgorithmID(), "verify").Inc()
```

### Dashboard Queries

```promql
# Algorithm distribution
sum by (algorithm) (rate(agentauth_signature_operations_total[5m]))

# Migration progress (RSA-PSS adoption rate)
sum(rate(agentauth_signature_operations_total{algorithm="PS256"}[5m]) /
sum(rate(agentauth_signature_operations_total[5m]) * 100

# Verification failures by algorithm
sum by (algorithm) (rate(agentauth_signature_verification_failures_total[5m]))
```

## Troubleshooting

### Issue: "Unknown algorithm: XYZ"

**Cause**: Algorithm not registered in registry.

**Solution**:
```go
// Check available algorithms
algorithms := crypto.DefaultRegistry.ListAlgorithms()
log.Printf("Available: %v", algorithms)

// Register if custom algorithm needed
provider := crypto.NewRSAPSSProvider(4096)
crypto.DefaultRegistry.Register("PS256-4096", provider)
```

### Issue: "Invalid key type: expected *rsa.PrivateKey, got ed25519.PrivateKey"

**Cause**: Using wrong key type with algorithm provider.

**Solution**: Ensure key type matches provider:
```go
// Wrong
rsaProvider := crypto.NewRSAPSSProvider(2048)
rsaProvider.Sign(ed25519PrivKey, message)  // ERROR

// Correct
rsaProvider := crypto.NewRSAPSSProvider(2048)
rsaPrivKey, _, _ := rsaProvider.GenerateKey()
rsaProvider.Sign(rsaPrivKey, message)  // OK
```

### Issue: "RSA-PSS verification failed" with valid signature

**Cause**: Key size mismatch or signature corruption.

**Solution**:
```go
// Verify key size
rsaPriv := privKey.(*rsa.PrivateKey)
log.Printf("Key size: %d bits", rsaPriv.N.BitLen())

// Regenerate with correct size
provider := crypto.NewRSAPSSProvider(3072)
newPrivKey, newPubKey, _ := provider.GenerateKey()
```

### Issue: PEM decoding fails

**Cause**: Invalid PEM format or encoding.

**Solution**:
```go
// Verify PEM headers
if !strings.Contains(string(pemData), "BEGIN PRIVATE KEY") {
    return errors.New("missing PEM header")
}

// Check for whitespace issues
pemData = bytes.TrimSpace(pemData)

// Validate with OpenSSL
// openssl pkey -in private.pem -text -noout
```

## Compliance and Standards

### IANA JOSE Algorithm Registry

This implementation follows [RFC 7518](https://tools.ietf.org/html/rfc7518) (JSON Web Algorithms):

- **EdDSA**: RFC 8032 (Ed25519 and Ed448)
- **PS256**: RFC 8017 (RSASSA-PSS with SHA-256)
- **ES256**: RFC 7518 (ECDSA using P-256 and SHA-256)

### Regulatory Compliance

| Framework | Ed25519 | RSA-PSS | ECDSA P-256 |
|-----------|---------|---------|-------------|
| FIPS 140-2 | ❌ | ✅ | ✅ |
| NIST SP 800-57 | ⚠️ | ✅ | ✅ |
| PCI DSS | ✅ | ✅ | ✅ |
| GDPR | ✅ | ✅ | ✅ |
| Common Criteria | ⚠️ | ✅ | ✅ |

**Legend**: ✅ Approved, ⚠️ Conditionally accepted, ❌ Not approved

**Note**: Ed25519 is not FIPS-approved but is widely used in modern systems (OpenSSH, Signal, Tor, etc.).

## Further Reading

- [RFC 8032: Edwards-Curve Digital Signature Algorithm (EdDSA)](https://tools.ietf.org/html/rfc8032)
- [RFC 8017: PKCS #1: RSA Cryptography Specifications Version 2.2](https://tools.ietf.org/html/rfc8017)
- [RFC 7518: JSON Web Algorithms (JWA)](https://tools.ietf.org/html/rfc7518)
- [NIST SP 800-186: Recommendations for Discrete Logarithm-Based Cryptography](https://doi.org/10.6028/NIST.SP.800-186)
- [Go crypto package documentation](https://pkg.go.dev/crypto)

## Summary

Algorithm agility in AgentAuth provides:

✅ **Flexibility**: Support for Ed25519, RSA-PSS, and ECDSA P-256  
✅ **Interoperability**: Compatible with diverse systems and requirements  
✅ **Future-proofing**: Easy addition of new algorithms via registry  
✅ **Security**: Type-safe operations prevent cross-algorithm attacks  
✅ **Performance**: Optimal algorithm selection for each use case  
✅ **Migration**: Gradual rollout strategies minimize disruption  

Choose the algorithm that best fits your security, performance, and compliance requirements.
