// Package gauth implements the core GAuth authentication and authorization protocol.
//
// # Overview
//
// GAuth is a comprehensive authentication and authorization framework designed for
// modern distributed systems. It provides secure token management, fine-grained
// access control, and robust resilience patterns for building reliable auth systems.
//
// Core Features
//
//   - Token-based Authentication: Secure JWT-based token management with support
//     for different token types and signing algorithms.
//
//   - Fine-grained Authorization: Granular access control with support for
//     scopes, claims, and resource-specific permissions.
//
//   - Rate Limiting: Configurable rate limiting with multiple strategies including
//     token bucket, sliding window, and distributed rate limiting.
//
//   - Circuit Breaking: Built-in circuit breaker pattern to prevent cascading
//     failures in distributed auth systems.
//
//   - Audit Logging: Comprehensive audit trail with structured logging and
//     transaction tracking.
//
//   - Metrics & Monitoring: Prometheus integration for real-time monitoring of
//     auth operations and system health.
//
// Basic Usage
//
//	import "github.com/Gimel-Foundation/gauth/pkg/gauth"
//
//	// Initialize GAuth with configuration
//	auth := gauth.New(gauth.Config{
//		AuthServerURL: "https://auth.example.com",
//		ClientID:      "client-123",
//		ClientSecret:  "secret-456",
//		Scopes:        []string{"read", "write"},
//		RateLimit: gauth.RateLimitConfig{
//			RequestsPerSecond: 100,
//			BurstSize:        10,
//		},
//	})
//
//	// Create and configure a resource server
//	server := gauth.NewResourceServer(gauth.ResourceConfig{
//		ID:          "resource-123",
//		Auth:        auth,
//		Permissions: []string{"read", "write"},
//	})
//
// # Resilience Patterns
//
// GAuth implements several resilience patterns to ensure system reliability:
//
//   - Circuit Breaker: Prevents system overload by failing fast when error
//     thresholds are exceeded.
//
//   - Retry with Backoff: Automatic retry of failed operations with exponential
//     backoff.
//
//   - Rate Limiting: Protects services from excessive load through configurable
//     rate limiting strategies.
//
//   - Bulkheading: Isolation of critical system components to contain failures.
//
// Security Best Practices
//
//  1. Always use HTTPS in production environments
//  2. Implement appropriate token expiration and rotation
//  3. Use secure token storage mechanisms
//  4. Enable audit logging for security-critical operations
//  5. Configure rate limiting to prevent abuse
//  6. Monitor authentication failures and suspicious patterns
//  7. Regularly rotate client secrets and credentials
//  8. Use strong cryptographic algorithms for token signing
//
// Monitoring & Metrics
//
// GAuth exports Prometheus metrics for key operations:
//
//   - Authentication attempts and success rates
//   - Token validation and generation latencies
//   - Rate limiting statistics
//   - Circuit breaker state changes
//   - Resource server operation counts
//
// For detailed metrics documentation, see the monitoring/metrics.go file.
//
// For complete examples and advanced usage patterns, refer to the examples/
// directory in the repository.
package gauth
