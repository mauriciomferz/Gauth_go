---
title: Code Style
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Code Style & Linting Standards

> Last Updated: 2025-10-17
> Status: Active

This repository enforces consistency via `golangci-lint`, `gofumpt`, and Makefile helpers. Adhere to the following guidelines:

## Formatting
- Run `make format` before pushing (`go fmt ./...` + `go mod tidy`).
- Use modern octal notation (`0o644`) for file modes; avoid legacy `0644`.
- Prefer multi-line composite literals (maps, slices, structs) with trailing commas to minimize future diff noise and satisfy `gofumpt`.

## Constants & Duplication
- Hoist repeated JSON fragments or string literals (policy bodies, cache metadata values, test user IDs) into `const` blocks.
- Use shared cache metadata constants: `metadataCacheHitTrue`, `metadataCacheHitFalse`.

## Error Handling
- Do not ignore errors silently—handle or justify with `//nolint:<linter> // reason: ...`.
- Wrap contextual errors (`fmt.Errorf("reload policies: %w", err)`) when propagating.
- In tests prefer `t.Fatalf` for setup failures over `panic`.

## Concurrency & Watching
- Capture event types once in file watch loops; avoid repeated type assertions.
- Use atomic counters for high-frequency metrics in place of coarse mutexes.

## Makefile Targets Quick Reference
| Target | Purpose |
|--------|---------|
| `make lint` | Full linter suite |
| `make lint-minimal` | Fast subset (pre-commit) |
| `make lint-strict` | Zero tolerance (pre-merge) |
| `make lint-fix` | Auto-fix formatting/imports/tidy |
| `make hygiene` | Composite maintenance (format + minimal lint + TODO report) |

## Inline Suppressions
Keep rare and always annotated:
```go
//nolint:errcheck // metrics exposition write failure is non-fatal
```
Never leave bare suppressions lacking a rationale.

## Testing Strategy
- Prefer deterministic tests (inject clocks) over real sleeps; integration tests requiring sleeps are tagged with `//go:build integration`.
- Benchmarks should isolate cache hit vs miss paths (planned: `authorizer_bench_test.go`).

## Layout
- Public APIs under `pkg/`; experimental or internal code under `internal/`.
- Avoid cyclic dependencies—extract interfaces for cross-package coupling.

## Pull Request Hygiene
- Small, focused PRs; CI must be green (tests + lint) before merge.
- Include reasoning in commit messages for any `//nolint` usage.

Adhering to these standards keeps CI green and review cycles efficient.

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
