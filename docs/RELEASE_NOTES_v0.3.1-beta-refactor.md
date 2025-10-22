# Release v0.3.1-beta-refactor

> Last Updated: 2025-10-17
> Status: Active

Date: 2025-10-17
Status: Beta (NOT production ready)

## Overview
This release focuses on performance and observability improvements to the authorization engine, particularly around regex condition evaluation and decision latency profiling.

## Key Enhancements
### Regex Cache Lifecycle
- Added LRU capacity limit (default 256) with optional TTL expiry per compiled pattern.
- Evictions tracked via `authz_regex_evictions_total`.
- Safe locking and panic-free evaluation (eliminated prior nil dereference path).

### Pre-Validation of Regex Patterns
- During policy reload, patterns are compiled early to surface errors; success/failure increments metrics without populating runtime cache prematurely.

### Match Frequency Metrics
- Total successful regex matches exported (`authz_regex_matches_total`).
- Internal per-pattern counters retained for future top-N exposure.

### Decision Latency Histogram
- Fixed nanosecond buckets (50µs–100ms) exposed via `authz_latency_bucket{le="..."}`.
- Complementary approximate p99 latency retained.

### Metrics Snapshot Extension
- Snapshot now includes: compiles, compile errors, cache size, evictions, matches, latency histogram map.

### Tests Added
- Capacity eviction (LRU) behavior.
- TTL expiry eviction.
- Pre-validation invalid pattern error metric.
- Regex match frequency.
- Latency histogram bucket population.

## Stability & Fixes
- Resolved concurrent map write race in decision cache metadata mutation (cache hit path copies metadata map).
- Fixed regex evaluation panic on invalid pattern (nil pointer dereference) by restructuring compile path.

## Documentation
- `AUTHORIZATION_IMPLEMENTATION.md` expanded: regex caching lifecycle, tuning guidance, latency histogram interpretation, observability quick reference.

## Prometheus Metrics Summary
| Metric | Purpose |
|--------|---------|
| `authz_regex_compiles_total` | Successful pattern compilations (unique + pre-validation) |
| `authz_regex_compile_errors_total` | Invalid pattern attempts |
| `authz_regex_cache_size` | Current cached patterns |
| `authz_regex_evictions_total` | TTL/LRU evictions |
| `authz_regex_matches_total` | Successful regex condition matches |
| `authz_latency_bucket{le="..."}` | Decision latency distribution |
| `authz_latency_average_nanoseconds` | Mean latency (approx) |
| `authz_latency_p99_nanoseconds` | Approximate p99 latency |
| `authz_decisions_total` | Total decisions |
| `authz_cache_hits_total` / `authz_cache_misses_total` | Decision cache performance |
| `authz_policy_reload_total` | Policy reload events |
| `authz_policy_conflicts_total` | Decisions with simultaneous allow+deny matches |

## Upgrade Notes
No breaking API changes to policy or request structures. To leverage new metrics:
```go
ma := authz.NewMemoryAuthorizer()
ma.SetRegexCacheCapacity(128)
ma.SetRegexCacheTTL(5 * time.Minute)
ma.EnableCaching(500 * time.Millisecond)
http.HandleFunc("/metrics", authz.PrometheusHandler(ma))
```

## Future Directions
- Export top-N regex match counts.
- Adaptive regex cache size based on eviction rate.
- Replace static latency buckets with HDR histogram / OTEL distribution.
- Fine-grained decision cache invalidation on policy diff events.

## Disclaimer
Experimental Beta. Do not deploy in production without additional hardening (persistent stores, multi-tenant isolation, cryptographic integrity, comprehensive audit & rollback).

---

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
