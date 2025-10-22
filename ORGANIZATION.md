# Project Organization

> Last Updated: 2025-10-17
> Status: Active

> **⚠️ BETA DEMONSTRATION NOTICE**
>
> This is a **beta demonstration implementation** only.
> **NOT production ready. This is for beta learning and demonstration purposes only. Do NOT use for real security, production, or commercial deployment.**

**Repository Layout (Beta Demonstration)**

## 📁 Current Structure (High-Level Overview)

```
GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/
├── pkg/                  # Core packages (auth, authz, audit, delegation, policy, token, rate/ratelimit, events, compliance, validation, etc.)
│   ├── gauth/            # High-level service facade (authorization lifecycle)
│   ├── auth/             # Authentication primitives
│   ├── authz/            # Authorization engine (memory + regex metrics)
│   ├── audit/            # Hash‑chained audit logging
│   ├── delegation/       # RFC0111 / RFC0115 delegation flows
│   ├── policy/           # Experimental provenance chain & evaluation adapter
│   ├── token/            # Token issuance/validation/revocation
│   ├── events/           # Typed event hub
│   ├── ratelimit/        # Unified interface + token bucket wrapper
│   ├── resilience/       # Retry / bulkhead patterns
│   └── ...               # Additional RFC & utility packages (store, testutil, rfc0111, compliance, validation)
│
├── internal/             # Internal helpers (circuit breaker, rate limiting, etc.)
├── cmd/                  # Server entry points (web-server, gauth-server)
├── web/                  # Beta web UI (embedded assets & handlers)
├── examples/             # Extensive runnable examples (delegation, policy, token, resilience, tracing ...)
├── test/                 # Cross‑package tests & benchmarks
├── docs/                 # Comprehensive documentation & RFC summaries
├── scripts/              # Dev, CI, normalization, benchmarking utilities
├── monitoring/           # Prometheus / alert configs (beta)
├── deployments/          # Docker/K8s manifests (demo scope)
├── go.mod, go.sum        # Module definition & deps
└── README.md             # Main project introduction
```

## 🎓 Beta Status: Demonstration Implementation

**⚠️ IMPORTANT**: This is a **beta implementation** designed for learning and demonstration purposes only. **NOT production ready. Do NOT use for real security, production, or commercial deployment.**

**Beta Verification Snapshot (Representative, Not Exhaustive):**
- ✅ Core servers build (`web-server`, `gauth-server`)
- ✅ Delegation + token examples run
- ✅ Policy provenance endpoints exposed
- ✅ Audit chain verification tests pass
- ✅ Authorization metrics & policy metrics panels operational
- ✅ Benchmarks runnable via nightly workflow

**Highlighted Demonstration Features:**
- 🎯 Token lifecycle & revocation
- 🎯 Delegation (RFC0111/RFC0115) flows
- 🎯 Hash‑chained audit logging & tamper detection
- 🎯 Experimental policy chain & provenance
- 🎯 Authorization metrics & regex caching
- 🎯 Resilience patterns & unified rate limiting interface
- 🎯 Examples catalog for guided exploration

---

**Demo Author**: [Mauricio Fernandez](https://github.com/mauriciomferz)
**Copyright (c) 2025 Gimel Foundation gGmbH i.G.**

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
