# Token Management Examples

This directory contains comprehensive examples demonstrating the token management features of the GAuth library.

## Core Examples

### 1. Basic Token Management (basic.go)
Shows fundamental token operations:
- Creating and signing JWT tokens
- Token validation
- Basic token rotation
- Using the token blacklist

## Security Examples

### 10. Secure Token Flow (secure_flow.go)
Comprehensive security implementation:
- Key rotation
- Encrypted storage
- Audit logging
- Combined JWT/PASETO tokens
- Security event tracking

### 11. Service Setup (service_setup.go)
Complete service configuration:
- TLS setup
- Rate limiting
- CORS configuration
- Multiple storage backends
- Monitoring and metrics

```bash
go run basic.go
```

### 2. Custom Validation (validation.go)
Demonstrates validation customization:
- Custom validator implementation
- Using validation chains
- Token querying
- Multiple token types

```bash
go run validation.go
```

### 3. Key Rotation (key_rotation.go)
Advanced key management:
- RSA key management
- Key rotation
- Token rotation with key changes
- Secure token invalidation

```bash
go run key_rotation.go
```

## Advanced Examples

### 4. Distributed Token Management (distributed.go)
Demonstrates token handling in distributed systems:
- Multi-node token validation
- Token revocation propagation
- Event-based synchronization
- Cluster-wide token state management

```bash
go run distributed.go
```

### 5. OAuth2/OpenID Connect Flows (oauth2.go)
Implements OAuth2 token flows:
- Authorization code flow
- Refresh token flow
- Token type handling
- Scope management
- Client validation

```bash
go run oauth2.go
```

### 6. Token Monitoring (monitoring.go)
Shows token lifecycle monitoring:
- Token usage statistics
- Expiration tracking
- Revocation monitoring
- Token metrics collection

### 7. Encrypted Store (encrypted_store.go)
Demonstrates secure token storage:
- AES-GCM encryption
- Secure key management
- Transparent encryption/decryption
- Protection of sensitive data

### 8. PASETO Support (paseto.go)
Shows modern token format implementation:
- PASETO v2 support
- Ed25519 signatures
- Enhanced security properties
- Simple, secure API

### 9. Multi-Region Deployment (multi_region.go)
Complex microservices scenario:
- Multi-region token management
- Cross-region validation
- Service-to-service auth
- Token caching strategies

```bash
go run monitoring.go
```

## Features Demonstrated

1. Token Management
   - Creation and signing
   - Validation and verification
   - Rotation and revocation
   - Blacklisting

2. Security Features
   - JWT support (HS256/RS256)
   - Key management and rotation
   - Secure token IDs
   - Token blacklisting

3. Advanced Features
   - Distributed token handling
   - OAuth2 flows
   - Token monitoring
   - Custom validation

4. Store Features
   - In-memory storage
   - TTL support
   - Concurrent access
   - Query capabilities

## Best Practices

These examples demonstrate several security best practices:
1. Always validate tokens before use
2. Implement proper token rotation
3. Use secure key management
4. Handle token revocation properly
5. Monitor token lifecycle
6. Use appropriate token types
7. Manage token scopes
8. Handle distributed scenarios
9. Implement proper OAuth2 flows

## Production Considerations

When using these examples in production:
1. Use secure key storage
2. Implement persistent token storage
3. Add proper logging and monitoring
4. Handle all error cases
5. Add rate limiting
6. Use secure configuration
7. Implement proper access controls
8. Add auditing
9. Use TLS for all communications
10. Implement proper cleanup routines

## Dependencies

- Go 1.21 or later
- GAuth library
- JWT libraries (as needed)