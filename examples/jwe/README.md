---
title: "JWE Encryption Examples"
category: example
status: active
lastUpdated: 2025-11-12
owners: architecture-team
refreshCadence: on-change
---
# JWE Examples

This directory contains example applications demonstrating JWE (JSON Web Encryption) usage with AgentAuth Extended Tokens.

## Examples

### 1. Simple Authorization Server (`auth_server.go`)
Demonstrates creating and encrypting Extended Tokens with JWE.

**Run:**
```bash
go run auth_server.go
```

**Features:**
- Creates Extended Tokens with comprehensive AAP-001 fields
- Encrypts tokens using RSA-OAEP-256 + A256GCM
- Outputs JWE token for transmission to resource server

### 2. Simple Resource Server (`resource_server.go`)
Demonstrates validating and decrypting JWE tokens.

**Run:**
```bash
go run resource_server.go <jwe-token>
```

**Features:**
- Accepts JWE token as command-line argument
- Decrypts JWE to JWT
- Parses and validates Extended Token
- Checks authorization chain, PoA, restrictions

### 3. Key Rotation Demo (`key_rotation.go`)
Demonstrates key rotation and backward compatibility.

**Run:**
```bash
go run key_rotation.go
```

**Features:**
- Generates multiple key versions
- Encrypts tokens with different keys
- Shows graceful key rotation (old tokens remain valid during transition)
- Demonstrates key registry pattern

## Quick Start

### Generate Keys

```bash
# Generate RSA key pair (2048-bit)
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem
```

### Run Complete Flow

```bash
# Terminal 1: Start resource server
go run resource_server.go

# Terminal 2: Get token from auth server and send to resource server
TOKEN=$(go run auth_server.go | tail -1)
go run resource_server.go "$TOKEN"
```

## Configuration

All examples use environment variables for configuration:

- `AGENTAUTH_JWE_ENABLED` - Enable/disable JWE (default: true)
- `AGENTAUTH_JWE_PUBLIC_KEY` - Path to RSA public key PEM file
- `AGENTAUTH_JWE_PRIVATE_KEY` - Path to RSA private key PEM file
- `AGENTAUTH_JWE_KEY_ID` - Key identifier for rotation (default: "agentauth-prod-2025-11")
- `AGENTAUTH_JWE_ALGORITHM` - Key encryption algorithm (default: "RSA-OAEP-256")
- `AGENTAUTH_JWE_ENCRYPTION` - Content encryption (default: "A256GCM")

## Performance Notes

Based on benchmarks:

- **Encryption**: ~126 μs per token (125,735 ns/op)
- **Decryption**: ~833 μs per token (832,552 ns/op)
- **Full cycle**: ~1.02 ms per token (1,015,617 ns/op)
- **Throughput**: ~980 encrypt+decrypt cycles/second (single core)

JWE adds approximately 97% size overhead to JWT tokens (typical: 487 bytes → 958 bytes).

## Security Notes

1. **Key Management**: Store private keys securely (never commit to version control)
2. **Key Rotation**: Rotate keys annually (365 days recommended)
3. **Key Distribution**: Only distribute public keys; keep private keys on auth server
4. **Encryption Algorithm**: Use RSA-OAEP-256 for production (not RSA-PKCS1-v1_5)
5. **Content Encryption**: Use A256GCM (AES-256-GCM with authentication)
6. **Compression**: DEFLATE compression reduces token size by 30-40%

## Troubleshooting

### "Public key file not found"
Ensure `AGENTAUTH_JWE_PUBLIC_KEY` points to a valid PEM file or generate keys:
```bash
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem
```

### "Failed to decrypt JWE token"
- Verify using the correct private key (must match public key used for encryption)
- Check token hasn't been tampered with
- Ensure Key ID (kid) matches between encryption and decryption

### "Token validation failed"
- Check token expiration (default: 1 hour)
- Verify authorization chain is valid
- Ensure PoA is within validity period

## Further Reading

- [RFC 7516 - JSON Web Encryption (JWE)](https://tools.ietf.org/html/rfc7516)
- [RFC 7518 - JSON Web Algorithms (JWA)](https://tools.ietf.org/html/rfc7518)
- [AAP AAP-001 - AgentAuth 1.0 Specification](../../AAP-001.md)
- [JWE Phase 1 Completion Report](../../JWE_PHASE1_COMPLETION_REPORT.md)
