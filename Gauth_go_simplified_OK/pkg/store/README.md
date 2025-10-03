# Token Store Package

The `store` package provides implementations for token storage in different backends. This package is designed to offer a clean interface for managing tokens with different persistence options.

## Overview

The package offers the following functionality:

1. Storage interfaces and common types
2. Multiple storage backend implementations
3. Type-safe error handling
4. Thread-safe token operations
5. Built-in token expiration and cleanup

## Available Stores

### Memory Store

The memory store provides an in-memory implementation of the TokenStore interface. It's suitable for development, testing, and applications that don't require persistence across restarts.

```go
store := store.NewTokenStore(store.Memory, nil)
```

### Redis Store

The Redis store provides a persistent, distributed implementation of the TokenStore interface. It's suitable for development environments where tokens need to be shared across multiple instances.

```go
store := store.NewTokenStore(store.Redis, store.RedisConfig{
    Addr: "localhost:6379",
    KeyPrefix: "myapp:tokens:",
})
```

## Interface

The `TokenStore` interface defines methods for managing tokens:

```go
type TokenStore interface {
    Store(ctx context.Context, token string, metadata TokenMetadata) error
    Get(ctx context.Context, token string) (*TokenMetadata, error)
    GetByID(ctx context.Context, id string) (*TokenMetadata, error)
    Delete(ctx context.Context, token string) error
    List(ctx context.Context, subject string) ([]TokenMetadata, error)
    Revoke(ctx context.Context, token string) error
    IsRevoked(ctx context.Context, token string) (bool, error)
    Cleanup(ctx context.Context) error
}
```

## Usage

```go
// Create a store
store := store.NewTokenStore(store.Memory, nil)

// Store a token
err := store.Store(ctx, "token123", store.TokenMetadata{
    ID: "id123",
    Subject: "user123",
    ExpiresAt: time.Now().Add(1 * time.Hour),
})

// Retrieve a token
metadata, err := store.Get(ctx, "token123")

// Check if a token is revoked
revoked, err := store.IsRevoked(ctx, "token123")

// List tokens for a subject
tokens, err := store.List(ctx, "user123")

// Revoke a token
err := store.Revoke(ctx, "token123")

// Delete a token
err := store.Delete(ctx, "token123")
```

## Error Handling

The package provides structured error handling through `StorageError` type:

```go
if err != nil {
    if storageErr, ok := err.(*store.StorageError); ok {
        fmt.Printf("Operation: %s, Key: %s, Error: %v, Detail: %s\n", 
            storageErr.Op, storageErr.Key, storageErr.Err, storageErr.Detail)
    }
}
```

## Best Practices

1. Always close stores that implement the `Close()` method when done
2. Use proper error handling to distinguish between different error types
3. Consider the appropriate store for your use case based on persistence needs
4. Set appropriate TTL values for tokens based on security requirements
5. Use the cleanup functionality to manage token lifecycle