# Enhanced Token Signature Validation

This document outlines the implementation of a new, extensible token signature validation system in GAuth.

## Summary

The previous signature validation logic was limited and not easily extensible. This work introduces a flexible system that supports multiple signature algorithms concurrently, improves security, and adds detailed monitoring capabilities.

### Key Features

- **Multiple Algorithm Support**: GAuth can now validate tokens signed with various algorithms, including:
  - HMAC (e.g., `HS256`)
  - RSA (e.g., `RS256`)
  - ECDSA (e.g., `ES256`)
- **Extensible Design**: The `TokenSignatureValidator` interface and `ValidatorRegistry` allow new validation algorithms to be added with minimal effort.
- **Comprehensive Unit Tests**: Each validation strategy (HMAC, RSA, ECDSA) is covered by a dedicated set of unit tests to ensure correctness and security.
- **Rich Telemetry**:
  - **Logging**: Detailed logs are generated for both successful and failed validation attempts, including the algorithm used and the time taken.
  - **Metrics**: Prometheus metrics are exported to provide insights into the validation process:
    - `gauthtoken_validation_attempts_total`: A counter for validation attempts, labeled by algorithm and status (`success`, `failed`, `unsupported`, etc.).
    - `gauthtoken_validation_duration_seconds`: A histogram of validation durations, labeled by algorithm.

## Implementation Details

### `TokenSignatureValidator` Interface

The core of the new system is the `TokenSignatureValidator` interface, located in `pkg/token/signature_validator.go`. It defines a single `Validate` method.

```go
// pkg/token/signature_validator.go
type TokenSignatureValidator interface {
    Validate(ctx context.Context, tokenString string) error
}
```

Concrete implementations are provided for HMAC, RSA, and ECDSA.

### `ValidatorRegistry`

The `ValidatorRegistry` in `pkg/token/signature_validator.go` manages all registered validators. When a token is validated, the registry inspects the token's `alg` header and dispatches it to the appropriate validator.

### Integration

The new validation system is integrated into the existing token validation chain in `pkg/token/token.go`. The `signatureValidator` now uses the `ValidatorRegistry` to perform signature checks.

### Testing

Unit tests are located in `pkg/token/signature_validator_test.go`. These tests cover:
- Validation of correctly signed tokens.
- Rejection of tokens with invalid signatures.
- Rejection of tokens signed with the wrong key.
- The `ValidatorRegistry`'s ability to correctly dispatch to the right validator.
- Handling of unsupported algorithms and malformed tokens.

This robust implementation significantly enhances the security and maintainability of GAuth's token handling.
