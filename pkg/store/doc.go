/*
Package store provides comprehensive token and session storage implementations.

This package implements various storage backends and interfaces for managing
tokens, sessions, and related security data with high performance and reliability.

Key Features:

1. Storage Backends:
  - In-memory storage with TTL support
  - Redis backend with cluster support
  - SQL database storage with optimized queries
  - Distributed caching with sharding
  - Pluggable custom storage providers

2. Token Management:
  - Secure token storage with encryption
  - Fast token retrieval with indexing
  - Atomic token revocation
  - Scheduled token cleanup
  - Token metadata management

3. Session Handling:
  - Secure session creation
  - Fast session validation
  - Configurable session expiration
  - Scheduled session cleanup
  - Session attributes and claims

4. Cache Management:
  - Layered caching strategies
  - Cache invalidation patterns
  - Cache warming capabilities
  - Memory usage controls

Basic Usage:

	// Create an in-memory store
	memStore := store.NewMemoryStore(store.MemoryConfig{
		CleanupInterval: time.Minute * 5,
		DefaultTTL:      time.Hour * 24,
	})

	// Create a Redis store
	redisStore := store.NewRedisStore(store.RedisConfig{
		Addresses: []string{"localhost:6379"},
		Password:  "secret",
		DB:        0,
	})

	// Store a token
	err := store.Store(ctx, token, TokenMetadata{
		ID:        "token123",
		Subject:   "user456",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	// Retrieve token metadata
	metadata, err := store.Get(ctx, token)

Storage Types:

 1. Memory Store:
    Fast, non-persistent storage for development and testing.

    store := store.NewMemoryStore()

 2. Redis Store:
    Distributed storage for production deployments.

    store := store.NewRedisStore(store.RedisConfig{
    Addresses: []string{"localhost:6379"},
    })

 3. Database Store:
    Persistent storage with query capabilities.

    store := store.NewDatabaseStore(store.DatabaseConfig{
    Driver: "postgres",
    DSN:    "postgres://user:pass@localhost/db",
    })

Error Handling:

The package provides specific error types:

	type StorageError struct {
		Op  string // Operation that failed
		Key string // Key that caused the error
		Err error  // Underlying error
	}

Thread Safety:

All types in this package are designed to be thread-safe
and can be used concurrently.

Monitoring:

Storage operations can be monitored:

1. Statistics:
  - Total tokens
  - Active tokens
  - Revoked tokens
  - Storage size

2. Performance:
  - Operation latency
  - Cache hit rates
  - Storage errors
  - Cleanup metrics

Best Practices:

1. Storage Selection:
  - Consider persistence needs
  - Evaluate scalability requirements
  - Plan for failures
  - Monitor performance

2. Token Management:
  - Implement regular cleanup
  - Monitor storage usage
  - Handle storage errors
  - Consider encryption

3. Session Handling:
  - Set appropriate timeouts
  - Implement session invalidation
  - Handle concurrent access
  - Monitor session counts

4. Error Handling:
  - Handle transient failures
  - Implement retry logic
  - Log storage errors
  - Monitor error rates

See Also:
- Package token for token management
- Package resilience for storage resilience
- Package monitoring for metrics integration
*/
package store
