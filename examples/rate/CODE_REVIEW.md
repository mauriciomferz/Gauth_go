# Rate Limiting Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates rate limiting patterns using the AgentAuth framework. It covers token bucket, concurrent clients, window sliding, and reset/retry logic.

## Key Concepts
- **TokenBucket**: Implements rate limiting with burst and window controls.
- **Concurrent Clients**: Shows how multiple clients interact with the limiter.
- **Window Sliding**: Demonstrates window-based rate limiting.
- **Reset & Retry**: Shows how to reset limits and retry requests.

## How to Run
```bash
cd examples/rate
go run main.go
```

## Code Review Notes
- Code is idiomatic Go, modular, and demonstrates rate limiting best practices.
- All patterns are clearly separated and easy to extend.
- Inline comments added for Beta readability.

## Educational Comments
- See `main.go` for all pattern implementations and usage.
- Token bucket and window logic are fully functional and tested.

---

For more, see the [Architecture Guide](../../docs/ARCHITECTURE.md).
