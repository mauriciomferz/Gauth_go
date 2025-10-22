# Ratelimit Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates rate limiting patterns using the GAuth framework's internal ratelimit package. It covers burst, steady rate, reset, remove, and window sliding logic.

## Key Concepts
- **Limiter**: Implements rate limiting with burst, window, and reset controls.
- **Concurrent Clients**: Shows how multiple clients interact with the limiter.
- **Window Sliding**: Demonstrates window-based rate limiting.
- **Reset & Remove**: Shows how to reset and remove limits for clients.

## How to Run
```bash
cd examples/ratelimit
go run main.go
```

## Code Review Notes
- Code is idiomatic Go, modular, and demonstrates rate limiting best practices.
- All patterns are clearly separated and easy to extend.
- Inline comments added for Beta readability.

## Educational Comments
- See `main.go` for all pattern implementations and usage.
- Limiter logic is fully functional and tested.

---

For more, see the [Architecture Guide](../../docs/ARCHITECTURE.md).
