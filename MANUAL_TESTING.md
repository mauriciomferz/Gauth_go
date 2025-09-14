# Manual Testing & Familiarization Guide

This guide helps you manually test and explore GAuth’s functionality using type-safe, modular APIs.

---

## 1. Run the Demo
- Go to `cmd/demo/` or `examples/` and run the provided `main.go`.
   ```sh
   cd cmd/demo
   go run main.go
   ```
- Observe the output for grant, token, and event flows.

## 2. Try Custom Grants/Tokens
- Modify the example to create custom grants or tokens.
- Test edge cases: expired tokens, revoked grants, invalid scopes.

## 3. Audit Logging
- Check that all actions (grant, token issue, resource access) are logged.
- Try extending the audit logger for your own needs.

## 4. Error Handling
- Intentionally trigger errors (e.g., invalid client, revoked token) and observe responses.

## 5. Extend the Library
- Implement a custom token store or event type.
- Add new roles or attributes as needed.

---
For more, see the package docs in `pkg/gauth/doc.go` and the README.