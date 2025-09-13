# Manual Testing & Runtime Usage

This guide helps you manually test and explore GAuth functionality.

## Running the Demo

1. Build and run the demo:
   ```sh
   cd cmd/demo
   go run main.go
   ```
2. Try API endpoints or CLI commands as described in the README.

## Manual Scenarios
- **Authorize:** Submit an authorization request and observe the grant and audit log.
- **Token Issue:** Request a token and verify its structure and expiry.
- **Audit Trail:** Check audit logs for compliance events.
- **Rate Limiting:** Exceed rate limits and confirm correct error/audit behavior.

## Tips
- Use the `examples/` directory for more usage patterns.
- Logs and errors are printed to stdout or the configured audit backend.

---
For more, see `README.md` and `LIBRARY.md`.