# Token Management Examples

This directory contains examples for using the token package in GAuth.

## Recommended Usage

When working with tokens in GAuth, we recommend using the main token package implementations:

```go
import (
    "github.com/Gimel-Foundation/gauth/pkg/token"
)

func main() {
    // Create a memory store with 24-hour token cleanup
    store := token.NewMemoryStore(24 * time.Hour)
    
    // Use the store to manage tokens
    // ...
}
```

## Store Implementations

GAuth provides multiple store implementations for different use cases:

1. `MemoryStore`: In-memory token storage with optional automatic cleanup
   - Best for single-instance applications or testing
   - Tokens are lost on service restart

2. `RedisStore`: Redis-backed token storage (coming soon)
   - Best for distributed applications
   - Persistent across service restarts

## Deprecated Implementations

All deprecated in-memory store implementations (`memoryStoreV1`, `token/store`) have been removed as of the latest API migration. Please use the current token store APIs described below.

## Examples

- `basic/main.go`: Shows basic token creation and validation
- `type_safe_usage/main.go`: Demonstrates type-safe token operations
- Other examples showcase specific token-related functionality

## Migration Note

GAuth now uses the following APIs for token management:
- `RequestToken` for token creation
- `ValidateToken` for token validation
- `InvalidateToken` for token revocation

If you are migrating from older code that directly manipulates token structs or uses legacy methods, see the Migration Guide in `docs/CODE_IMPROVEMENTS.md` for details on updating to the new API.