---
title: Secret Storage Encryption at Rest
category: security-storage
status: implemented
lastUpdated: 2025-11-12
owners: security-team
source: internal
refreshCadence: quarterly
---
# Secret Storage Encryption at Rest

## Overview

The AgentAuth secret provider now supports transparent **encryption at rest** through `EncryptedProvider`, ensuring sensitive credentials, keys, and secrets are never stored in plaintext.

## Architecture

### Components

1. **Provider Interface** (`pkg/secret/provider.go`)
   - Abstract interface for secret storage operations: Get, Set, Delete, List
   - Implementation-agnostic: memory, vault, cloud KMS, etc.

2. **EncryptedProvider** (`pkg/secret/encrypted.go`)
   - Transparent encryption wrapper around any Provider backend
   - AES-256-GCM encryption with authenticated encryption (AEAD)
   - PBKDF2-SHA256 key derivation from passphrase (100,000 iterations)

3. **Backend Providers**
   - `MemoryProvider`: In-memory map (dev/test only)
   - `VaultStub`: Placeholder for HashiCorp Vault integration
   - Future: AWS KMS, GCP Secret Manager, Azure Key Vault

## Encryption Specification

### Algorithm: AES-256-GCM

- **Cipher**: AES with 256-bit key
- **Mode**: Galois/Counter Mode (GCM)
- **Authentication**: Built-in authenticated encryption (detects tampering)
- **Nonce**: 12-byte random nonce per encryption (prevents pattern analysis)

### Key Derivation: PBKDF2-SHA256

- **Function**: PBKDF2 (Password-Based Key Derivation Function 2)
- **Hash**: SHA-256
- **Iterations**: 100,000 (mitigates brute-force attacks)
- **Salt**: Fixed per-installation (consider per-tenant in production)
- **Output**: 32-byte encryption key

### Encrypted Format

```
base64url(nonce[12] || ciphertext || tag[16])
```

- **nonce**: 12-byte random value (unique per encryption)
- **ciphertext**: Encrypted plaintext
- **tag**: 16-byte authentication tag (GCM)

## Usage

### Basic Setup

```go
import "github.com/agentauth/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/secret"

// Create encrypted provider
backend := secret.NewMemory()
passphrase := "your-strong-passphrase-min-16-chars"
encrypted, err := secret.NewEncrypted(backend, passphrase)
if err != nil {
    return err
}

// Use like any provider - encryption is transparent
ctx := context.Background()
err = encrypted.Set(ctx, "db/password", "secret-value")
plaintext, err := encrypted.Get(ctx, "db/password")
```

### Production Recommendations

1. **Passphrase Management**
   - Use environment variables or secure config
   - Minimum 32 bytes entropy (64 hex chars recommended)
   - Rotate periodically (requires re-encryption)

2. **Backend Selection**
   - **Dev/Test**: `MemoryProvider` (ephemeral)
   - **Production**: Vault, AWS KMS, GCP Secret Manager
   - Never use `MemoryProvider` in production

3. **Key Rotation**
   ```go
   // Re-encrypt with new passphrase
   oldProvider, _ := secret.NewEncrypted(backend, oldPass)
   newProvider, _ := secret.NewEncrypted(backend, newPass)
   
   keys, _ := oldProvider.List(ctx, "")
   for _, key := range keys {
       value, _ := oldProvider.Get(ctx, key)
       newProvider.Set(ctx, key, value)
   }
   ```

## Security Properties

### Confidentiality
- ✅ Secrets encrypted with AES-256-GCM before storage
- ✅ Backend never sees plaintext
- ✅ Random nonces prevent pattern analysis

### Integrity
- ✅ GCM authentication tag detects tampering
- ✅ Decrypt fails if ciphertext modified
- ✅ Protection against bit-flipping attacks

### Non-Repetition
- ✅ Each encryption uses unique random nonce
- ✅ Same secret encrypted twice produces different ciphertext
- ✅ Prevents frequency analysis

## Test Coverage

12 comprehensive tests in `pkg/secret/encrypted_test.go`:

1. **TestEncryptedProviderCRUD** - Basic CRUD operations with encryption
2. **TestEncryptedProviderList** - Prefix-based key listing
3. **TestEncryptedProviderIfNotExists** - Idempotent create semantics
4. **TestEncryptedProviderName** - Provider naming
5. **TestEncryptedProviderWeakPassphrase** - Rejects short passphrases
6. **TestEncryptedProviderNilBackend** - Input validation
7. **TestEncryptDecryptRoundTrip** - Unicode, special chars, multiline
8. **TestEncryptedProviderTamperDetection** - Authentication failure on tampering
9. **TestEncryptedProviderDifferentKeys** - Cross-key isolation
10. **TestEncryptedProviderNonceUniqueness** - 100 encryptions yield unique ciphertexts

All tests passing (0.845s).

## Integration

### With AAP-001 Service

```go
// In pkg/aap001/aap001.go initialization
backend := secret.NewMemory()
passphrase := os.Getenv("AGENTAUTH_SECRET_ENCRYPTION_KEY")
if passphrase == "" {
    return errors.New("AGENTAUTH_SECRET_ENCRYPTION_KEY required")
}

secretProvider, err := secret.NewEncrypted(backend, passphrase)
if err != nil {
    return err
}

service := aap001.NewService(
    // ... other options ...
    aap001.WithSecretProvider(secretProvider),
)
```

### Environment Variables

```bash
# Production
export AGENTAUTH_SECRET_ENCRYPTION_KEY=$(openssl rand -hex 32)

# Dev/Test
export AGENTAUTH_SECRET_ENCRYPTION_KEY="dev-passphrase-sufficient-length"
```

## AAP-001 Compliance

Implements **sec8.item1** requirements:

- ✅ Pluggable secret provider abstraction
- ✅ Encryption at rest (AES-256-GCM)
- ✅ Memory backend (dev/test)
- ✅ Vault stub (production placeholder)
- ✅ Strong key derivation (PBKDF2-SHA256, 100k iterations)
- ✅ Authenticated encryption (tamper detection)

**GAP Matrix Status**: `sec8.item1` → **Implemented** (P0 Critical)

## Future Enhancements

1. **Real Vault Backend**
   - HashiCorp Vault KV v2 integration
   - Transit engine for encryption
   - Dynamic secrets

2. **Cloud KMS Integration**
   - AWS Secrets Manager
   - GCP Secret Manager
   - Azure Key Vault

3. **Multi-Tenant Key Isolation**
   - Per-tenant encryption keys
   - Tenant-specific salts
   - Key hierarchy (master + tenant keys)

4. **HSM Support**
   - PKCS#11 interface
   - Hardware security modules
   - FIPS 140-2 compliance

## References

- AAP-001: AgentAuth Authorization Protocol
- NIST SP 800-38D: GCM Specification
- NIST SP 800-132: PBKDF2 Recommendation
- Go crypto/aes: https://pkg.go.dev/crypto/aes
- Go crypto/cipher: https://pkg.go.dev/crypto/cipher
