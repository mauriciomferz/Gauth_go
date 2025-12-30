---
title: Performance Baseline
 category: performance-report
 status: baseline
 lastUpdated: 2025-11-12
 owners: performance-team
 refreshCadence: quarterly
 source: benchmark-suite
 ---
# Performance Baseline

> Last Updated: 2025-10-17
> Status: Active

Date: 2025-10-11
Host: macOS (Apple M3 Pro, arm64)
Go Version: (detected via `go test`) - ensure reproducibility by documenting `go version` separately if needed.

## Benchmarks Captured
Command: `make bench B=BenchmarkTokenIssue`

```
BenchmarkTokenIssue-11  14925681  71.43 ns/op  272 B/op  2 allocs/op
```

## Interpretation
- Token issuance path (benchmark stub implementation) performs ~14.9M ops/sec at ~71ns per operation with 2 allocations totaling 272 bytes.
- This serves as a stub baseline; real cryptographic signing will increase both latency and allocation count.

## Next Suggested Benchmarks
Add future captures for:
- BenchmarkTokenValidate
- BenchmarkRateLimiterAllow
- BenchmarkSlidingWindowLimiterAllow (once real implementation added)
- BenchmarkTokenJSONSerialize / Deserialize

## Methodology Notes
- Single run (`-count=1`). For stability tracking, consider `-count=5` and statistical aggregation.
- Use `benchstat` for comparing future changes.

## How to Reproduce
```
make bench B=BenchmarkTokenIssue
```

## Planned Enhancements
- Integrate `bench` target into CI (non-blocking) to persist historical results.
- Add optional `BENCH_OUT_DIR` variable to store timestamped outputs.

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md

## Additional Baseline (Validation Path & Metrics)

Date: 2025-10-17
Host: macOS (Apple M3 Pro, arm64)
Go Version: (same session as above)
Command: `go test -bench . -benchmem ./pkg/aap001 -run ^$ -count=1`

```
BenchmarkCreateDelegation-11             10045 ns/op   10955 B/op    82 allocs/op
BenchmarkValidateDelegation-11           10759 ns/op    3432 B/op    28 allocs/op
BenchmarkValidateDelegationWithMetrics-11  8901 ns/op    3459 B/op    28 allocs/op
```

### Notes
- Single-run values; expect normal variance. Metrics instrumentation adds negligible overhead (atomic increments not clearly visible within noise at this scale).
- Future comparisons should use multi-run aggregation (see new `scripts/run_benchmarks.sh`).

### Next Steps
- Introduce statistical aggregation (`-count=5`, benchstat) for these validation benchmarks.
- Track latency percentiles once histogram metrics land (planned and in progress).

## Cryptographic Authenticity Benchmarks (Signing & Verification)

Date: 2025-10-17
Host: macOS (Apple M3 Pro, arm64)
Go Version: same session

Commands:
```
go test -run=NONE -bench=BenchmarkSignCanonicalPOA -benchmem ./pkg/aap001 -count=1
go test -run=NONE -bench=BenchmarkVerifyCanonicalPOA -benchmem ./pkg/aap001 -count=1
```

Results:
```
BenchmarkSignCanonicalPOA-11     72042     18255 ns/op      952 B/op    10 allocs/op
BenchmarkVerifyCanonicalPOA-11   31942     35084 ns/op      888 B/op     9 allocs/op
```

### Interpretation
- Ed25519 signing adds ~18.3µs latency per canonical POA; verification is ~35.1µs (≈1.9x signing cost), both single-run snapshots.
- Allocation profile for signing (10 allocs, 952 B) and verification (9 allocs, 888 B) is modest; dominant cost is cryptographic work plus JSON canonicalization + hashing.
- Compared to prior delegation validation baseline (~10.8µs), adding signature verification roughly triples end-to-end authenticity validation latency if performed sequentially after structural validation. (Note: a subsequent validation benchmark run including extensive audit logging showed inflated numbers (~44µs); that run is excluded from baseline due to logging overhead — see "Methodology Considerations" below.)

### Methodology Considerations / Next Improvements
- Current benches are single-run and susceptible to variance; adopt `-count=5` + `benchstat` aggregation for stable tracking.
- Audit / event logging inside benchmarks can distort timings (as seen in the later validation run). Provide a build tag or environment flag to suppress high-volume audit emission during micro-benchmarks.
- Investigate reducing allocations in canonicalization (e.g., pre-sized buffers, object pooling) to marginally lower signing latency; crypto itself (Ed25519) is expected to remain dominant.
- Potential optimization: cache canonical JSON + hash for POA objects to avoid recomputation when repeatedly signing/validating unchanged structures.
- Consider parallelizing structural validation and key lookup with pre-hashing where safe to reduce end-to-end verification wall time.

### Tracking & Regression Strategy
- Add these benchmark identifiers to CI optional performance job; fail (or warn) on >20% regression over rolling 7-day median.
- Record median & p95 once multi-run data is collected (future histogram metrics integration).
- Introduce a synthetic load test combining delegation validation + signature verification to approximate realistic issuance/consumption throughput.

### Planned Follow-Ups
- Implement Prometheus exporter to expose counters for signature issue / verify and revocation integrity failures (Milestone 2B pending item).
- Add benchmark variants with: (a) cached canonical digest, (b) disabled audit logging, (c) batch verification scenario.
- Document authenticity overhead expectations in `docs/PERFORMANCE.md` and compliance documentation.

