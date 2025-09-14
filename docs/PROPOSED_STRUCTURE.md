# GAuth Go Project Structure Proposal

This structure separates core library code, demos, and documentation for maximum clarity and community onboarding.

```
Gauth_go/
├── cmd/
│   └── demo/                # Standalone demo apps (main.go, etc.)
├── pkg/
│   ├── gauth/               # Main library entry point (public API)
│   │   ├── doc.go           # Package-level documentation
│   │   ├── gauth.go         # Main orchestrator
│   │   └── ...
│   ├── token/               # Token management (typed, modular)
│   │   ├── store.go         # TokenStore and related types
│   │   ├── types.go         # Token types, claims, etc.
│   │   ├── validation.go    # Token validation logic
│   │   └── ...
│   ├── events/              # Event system (typed, modular)
│   │   ├── event_types.go   # EventType, EventStatus enums
│   │   ├── dispatcher.go    # Event dispatcher
│   │   ├── handlers/        # Event handlers (metrics, audit, etc.)
│   │   ├── metadata.go      # Metadata types and helpers
│   │   └── ...
│   ├── auth/                # Authentication logic
│   ├── authz/               # Authorization logic
│   ├── audit/               # Auditing logic
│   ├── util/                # Utilities
│   └── ...
├── examples/                # Usage examples (standalone, focused)
│   ├── basic/
│   ├── advanced/
│   └── ...
├── docs/
│   ├── README.md            # Project overview
│   ├── LIBRARY.md           # Library usage, extension, integration
│   ├── MANUAL_TESTING.md    # Manual testing and runtime usage
│   └── ...
├── LICENSE
├── README.md                # Quickstart, install, and project vision
└── ...
```


**Key Points:**
- All library code is under `pkg/`, with each domain in its own package.
- Demos and CLI entry points are under `cmd/`.
- Examples are in `examples/` for discoverability.
- Documentation is in `docs/` and at the root for onboarding.
- No public `map[string]interface{}` in APIs—typed structs everywhere.
- Rate limiting is enforced per user (OwnerID) and per client, using the OwnerID field of the token as the subject for rate limiting.
- Package-level `doc.go` files for every major package.

---

**Next Steps:**
1. Begin moving and splitting code into this structure, starting with the most critical (e.g., token store, event types).
2. Add or update `doc.go` and README files for each package.
3. Refactor APIs for type safety and clarity.
4. Polish documentation and examples for community onboarding.
