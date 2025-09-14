// Package token provides secure token management functionality.
//
// # Quick Start
//
//	import "github.com/Gimel-Foundation/gauth/pkg/token"
//
//	store := token.NewMemoryStore(24 * time.Hour)
//	t := &token.Token{
//	    Value:     "jwt-token",
//	    Type:      token.AccessToken,
//	    ExpiresAt: time.Now().Add(time.Hour),
//	    Scopes:    []string{"read", "write"},
//	}
//	err := store.Save(ctx, "user-123", t)
//	// ...
//
// See runnable examples in examples/token/basic/main.go and examples/token/advanced_revocation_flow/main.go.
//
// # See Also
//
//   - package authz: for authorization logic and policies
//   - package audit: for audit trail and event logging
//   - package events: for event types and event bus
//
// For a full protocol flow and advanced usage, see the examples/ directory and the project README.
//
// # RFC111 Mapping
//
// This package implements the token management and verification requirements of GiFo-RfC 0111 (September 2025):
//   - Extended tokens: Strongly typed tokens (access, refresh, etc.) with metadata, scope, and lifecycle management.
//   - Power-of-attorney: Token metadata supports delegation, authority, and attestation fields as required by RFC111.
//   - Revocation and compliance: Built-in revocation, rotation, and audit support for transparent and verifiable authorization.
//   - Exclusions: No Web3/blockchain, DNA-based identity, or AI-controlled GAuth logic is present, as required by RFC111.
//   - Versioning and attestation: Token types and validation logic support version history and attestation fields.
//   - Centralized issuance: All token operations are designed for central, auditable control, not decentralized delegation.
//
// For more details, see the RFC111 summary in the project root and the README.md for usage patterns.
//
// # Overview
//
// The token package implements secure token management with:
//   - Strong typing for different token types (access, refresh, etc.)
//   - Cryptographically secure token generation
//   - Safe storage and retrieval
//   - Automatic expiration handling
//   - Flexible metadata support
//   - Revocation capabilities
//   - Token validation and verification
//
// # Implementations
//
// This package provides the following implementations:
//
// 1. MemoryStore: An in-memory token store with optional automatic cleanup
//   - import "github.com/Gimel-Foundation/gauth/pkg/token"
//   - Use NewMemoryStore() to create a new instance
//
// # Key Components
//
// 1. Token Types
//
//	type Token struct {
//	    ID           string       `json:"id"`
//	    Value        string       `json:"token"`
//	    Type         Type         `json:"type"`
//	    IssuedAt     time.Time    `json:"iat"`
//	    ExpiresAt    time.Time    `json:"exp"`
//	    NotBefore    time.Time    `json:"nbf"`
//	    LastUsedAt   *time.Time   `json:"last_used_at,omitempty"`
//	    Issuer       string       `json:"iss"`
//	    Subject      string       `json:"sub"`
//	    Audience     []string     `json:"aud"`
//	    Scopes       []string     `json:"scope"`
//	    Algorithm    Algorithm    `json:"alg"`
//	    Metadata     *Metadata    `json:"metadata,omitempty"`
//	}
//
// Represents tokens with their metadata and lifecycle information.
//
// 2. Store Interface
//
//	type Store interface {
//	    Save(ctx context.Context, key string, token *Token) error
//	    Get(ctx context.Context, key string) (*Token, error)
//	    Delete(ctx context.Context, key string) error
//	    List(ctx context.Context, filter Filter) ([]*Token, error)
//	    Rotate(ctx context.Context, old, new *Token) error
//	    Revoke(ctx context.Context, token *Token) error
//	    Validate(ctx context.Context, token *Token) error
//	    Refresh(ctx context.Context, refreshToken *Token) (*Token, error)
//	}
//
// Allows implementing different storage backends.
//
// # Usage Examples
//
// Basic token storage:
//
//	store := token.NewMemoryStore(24 * time.Hour)
//
//	token := &token.Token{
//	    Value:     "jwt-token",
//	    Type:      token.AccessToken,
//	    ExpiresAt: time.Now().Add(time.Hour),
//	    Scopes:    []string{"read", "write"},
//	}
//
//	if err := store.Save(ctx, "user-123", token); err != nil {
//	    // Handle error
//	}
//
// Token retrieval with validation:
//
//	token, err := store.Get(ctx, "user-123")
//	if err != nil {
//	    switch err {
//	    case token.ErrTokenNotFound:
//	        // Handle not found
//	    case token.ErrTokenExpired:
//	        // Handle expired
//	    default:
//	        // Handle other errors
//	    }
//	}
//
// # Memory Management
//
// The package provides automatic memory management:
//   - TTL-based token expiration
//   - Background cleanup of expired tokens
//   - Efficient storage using minimal memory
//
// # Thread Safety
//
// All operations are thread-safe and can be used from multiple goroutines.
// The implementation uses appropriate synchronization mechanisms.
//
// 1. [REMOVED] memoryStoreV1: An older in-memory store implementation (memory_store.go)
// # Error Handling
//
// Clear error types for common scenarios:
//   - ErrTokenNotFound: Token doesn't exist
//   - ErrTokenExpired: Token has expired
//   - ErrInvalidToken: Token is malformed
//
// # Best Practices
//
// 1. Token Lifecycle:
//   - Set appropriate expiration times
//   - Implement token rotation
//   - Handle expired tokens gracefully
//
// 2. Storage Selection:
//   - Use memory store for development
//   - Implement persistent store for production
//   - Consider distributed storage for scaling
//
// 3. Security:
//   - Use secure token generation
//   - Implement proper scope checking
//   - Validate tokens thoroughly
//
// # Performance Considerations
//
// The memory store implementation:
//   - Uses minimal memory per token
//   - Performs efficient lookups
//   - Handles cleanup automatically
//
// # Extension Points
//
// 1. Custom Storage:
//   - Implement Store interface
//   - Add persistence
//   - Support clustering
//
// 2. Token Types:
//   - Add custom token types
//   - Implement type-specific validation
//   - Add metadata fields
//
// 3. Validation:
//   - Custom validation logic
//   - Additional security checks
//   - Scope verification
//
// See the examples directory for implementation patterns.
//
// # Licensing
//
// This package is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
//
// See the LICENSE file in the project root for details.
package token
