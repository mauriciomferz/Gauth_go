# TokenStore Examples

This directory contains examples for using the `store` package in GAuth.

## Basic Example

The `main.go` file demonstrates:

1. Creating a memory store
2. Storing a token with metadata
3. Retrieving a token
4. Working with token metadata

## Running the Example

```bash
go run main.go
```

## Using Different Storage Backends

The example shows how to use different storage backends:

### Memory Store

```go
store, err := store.NewTokenStore(store.Memory, nil)
```

### Redis Store

```go
store, err := store.NewTokenStore(store.Redis, store.RedisConfig{
    Addr: "localhost:6379",
    KeyPrefix: "myapp:tokens:",
})
```

## Token Operations

The example demonstrates common token operations:

- Storing tokens
- Retrieving tokens
- Listing tokens for a subject
- Revoking tokens
- Checking revocation status
- Cleaning up expired tokens

For more detailed information, refer to the `store` package documentation.
