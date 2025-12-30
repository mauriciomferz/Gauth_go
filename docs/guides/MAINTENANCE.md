---
title: Project Maintenance Guide
category: maintenance
status: active
lastUpdated: 2025-11-12
owners: maintainers
---
# AgentAuth Project Maintenance Guide

This guide documents the lightweight hygiene workflow to keep the codebase healthy, reproducible, and compliant with RFC-0111 implementation quality standards.

## Core Hygiene Loop

1. Fast Tidy (developer inner loop)
   - `make tidy-fast` (runs enhanced formatting, vet, minimal lint)
2. Full Hygiene (pre-commit / pre-push)
   - `make hygiene` (tidy + TODO/FIXME inventory)
3. CI Simulation (before larger PRs)
   - `make ci` (format check + vet + full lint + race tests)

## Formatting

Two levels of formatting are available:

- Basic: `make format` (gofmt + go mod tidy) – quick canonical formatting.
- Enhanced: `make format-enhanced` – executes `scripts/format.sh` which performs:
  - `goimports` (add/remove imports & grouping)
  - `gofmt` (canonical formatting)
  - Optional module tidy across go.work modules
  - Changed file summary (if git present)

Dry-run preview (no writes):
```bash
make format-dry
```
(Uses `DRY=1 scripts/format.sh` internally.)

## Module Management

- `make tidy-all` – runs `go mod tidy` for root and each module listed in `go.work`.
- `make deps` – downloads dependencies and tidies.

## Static Analysis & Linting

| Target            | Purpose                                      |
|-------------------|----------------------------------------------|
| `make vet`        | Go vet static analysis                       |
| `make lint`       | Full golangci-lint (project config)          |
| `make lint-minimal`| Fast lint (format check + minimal linters)  |
| `make lint-strict`| Elevates warnings to errors                  |
| `make lint-fix`   | Auto-fix imports + formatting                |

## Tests & Coverage

| Target                | Purpose                                  |
|-----------------------|------------------------------------------|
| `make test`           | Standard test run                        |
| `make test-race`      | Race detection                           |
| `make test-coverage`  | Coverage artifacts (`coverage.out/html`) |
| `make fuzz-cbor`      | Fuzz PoA CBOR codec                      |

## Benchmarks

Run targeted benchmarks (default token & rate bench set):
```bash
make bench
# Filter specific benchmarks
make bench B=BenchmarkTokenLifecycle
```

## TODO/FIXME Inventory

Generate a report of outstanding inline work items:
```bash
make todo-report
```
Output: `docs/CODE_TODO_REPORT.md` (auto-grouped raw matches).

## Recommended Pre-Push Sequence

```bash
make format-enhanced
make tidy-fast
make test
make lint-minimal
make hygiene
```

If introducing structural / validation changes:
```bash
make test-race
make fuzz-cbor   # For serialization stability
```

## Extended Token & RFC-0111 Integrity

Before merging changes that touch authorization chain, PoA, or token serialization:
```bash
go test ./pkg/gauth -run TestE2E_JWTSerializationRoundTrip -count=1
go test ./pkg/gauth -run TestE2E_ExtendedTokenService_Integration -count=1
```
Both suites must pass to preserve end-to-end token semantics.

## Scripts Overview

| Script                  | Purpose                                  |
|-------------------------|------------------------------------------|
| `scripts/format.sh`     | Enhanced formatting helper               |
| `scripts/tidy.sh`       | Combined format + tidy + vet + tests     |
| `scripts/release-notes.sh` | Auto-generate grouped release notes  |

## Environment Tips

- Use `DRY=1 make tidy` or `DRY=1 scripts/format.sh` to preview changes.
- Set `FAST_LINT=0` when invoking `scripts/tidy.sh` to run full lint.
- For module spanning (go.work), rely on `make tidy-all` after adding/removing modules.

## Troubleshooting

| Symptom                               | Action                                                  |
|---------------------------------------|---------------------------------------------------------|
| Unexpected import diffs               | Run `make format-enhanced` (goimports normalizes)       |
| Failing JWT serialization tests       | Re-run E2E tests; ensure `OwnersAuthorizerInfo` + LegalFramework present and use `EncodeExtendedToken`. |
| Vet warnings about formatting         | Run `make format` or `make format-enhanced`.            |
| Lint complaining about unused code    | Use `make lint-fix` then rerun `make lint-minimal`.     |
| Race failures intermittently          | Investigate with `make test-race`; add synchronization. |

## GNAP Store Cleanup

The GNAP implementation includes automatic cleanup for in-memory grant and token stores to prevent memory leaks in production deployments.

### Quick Start

```go
import "github.com/mauriciomferz/Gauth_go/pkg/gnap"

// Initialize stores
grantStore := gnap.NewMemoryGrantStore()
tokenStore := gnap.NewMemoryTokenStore()

// Create cleanup manager (10 minute interval)
cleanup := gnap.NewCleanupManager(grantStore, tokenStore, 10*time.Minute)

// Start automatic cleanup
ctx := context.Background()
cleanup.Start(ctx)
defer cleanup.Stop()

// Monitor cleanup statistics
stats := cleanup.Stats()
log.Printf("Cleaned: %d grants, %d tokens (last: %v)",
    stats.TotalGrantsCleaned, stats.TotalTokensCleaned, stats.LastCleanup)
```

### Cleanup Policies

**Grant Cleanup:**
- Removes grants after `ExpiresAt` time passes
- Grace period: None (immediate after expiration)

**Token Cleanup:**
- Expired tokens: Removed 1 hour after `ExpiresAt` (clock skew tolerance)
- Revoked tokens: Removed 24 hours after `RevokedAt` (audit retention)

### Manual Cleanup

For testing or manual triggering:
```go
grantsRemoved, tokensRemoved := cleanup.RunOnce()
```

### Recommended Settings

| Environment | Interval | Rationale |
|-------------|----------|-----------|
| Development | 5 minutes | Fast feedback, lower memory |
| Production  | 10-15 minutes | Balance between memory and CPU |
| High-volume | 5 minutes | Prevent memory growth |

### Monitoring

The `CleanupManager` tracks:
- `TotalGrantsCleaned` - Cumulative count
- `TotalTokensCleaned` - Cumulative count  
- `LastCleanup` - Timestamp of last run
- `Running` - Current state

## Policy

- No commit should remove required RFC-0111 fields from token flows.
- Always provide PoA + AuthorizationChain + OwnersAuthorizerInfo + LegalFramework in test fixtures invoking `CreateExtendedToken`.
- New packages must include at least one unit test and be covered by `make test`.

## Future Enhancements (Optional)

- Add pre-commit hook invoking `make tidy-fast`.
- Integrate `govulncheck` into `make ci`.
- Add structured JSON output for `todo-report` to drive dashboards.

---
Maintained: November 12, 2025
