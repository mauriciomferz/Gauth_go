# Project Structure & Organization Guide

This document summarizes the current directory layout and proposes incremental cleanup / consolidation actions for maintainability.

## High-Level Layout

```
cmd/                # Entry points (web-server, examples, tooling)
pkg/                # Public packages (gauth core, policy, poa, crypto, etc.)
internal/           # Internal implementations (replay store, jurisdiction, rotation)
web/                # Server wiring + handlers + tests (gin)
web/ui-react/       # Frontend React application (Vite)
docs/               # Curated + generated docs (gap matrix, compliance reports)
deployments/        # Docker Compose, k8s manifests, infra configs
scripts/            # Helper scripts (generation, start/stop, verification)
examples/           # Stand-alone runnable examples (audit, errors, cmd flows)
test/               # Benchmark suites, integration tests
artifacts/          # Audit and status reports (may consolidate into docs/ later)
```

## Observed Duplication / Candidates for Consolidation

- Multiple Dockerfiles (`Dockerfile.*`) → propose `Dockerfile.unified` with build args for modes (dev, prod, minimal).
- Numerous root-level reports (`*_REPORT.md`) → consider grouping under `docs/reports/`.
- Scripts using similar patterns (start/stop web demo) → unify via `scripts/dev-up.sh` (added) and future `dev-down.sh`.
- Environment examples currently frontend-only → added backend example `.env.backend.example`.

## Recommended Near-Term Cleanup Tasks

1. Introduce CODEOWNERS for clear ownership of critical packages.
2. Group executive / QA / gap closure reports under `docs/reports/` (non-breaking symlink or move + update index).
3. Create `Dockerfile.unified` multi-stage build (base -> builder -> final) with optional JS asset build stage.
4. Add CI step for `make hygiene` (format + mod tidy + TODO report) artifact upload.
5. Add `docs/SECURITY_HARDENING.md` capturing production-only steps (separate from README Beta notes).
6. Convert large README performance numbers into generated `docs/PERFORMANCE.auto.md` (scriptable).

## Package Ownership (Draft)

| Area            | Path                | Proposed Owner Alias |
|-----------------|---------------------|----------------------|
| Core Auth       | `pkg/gauth`         | @core-auth          |
| Delegation      | `pkg/delegation`    | @delegation-team    |
| PDP / Policy    | `pkg/pdp`, `pkg/policy` | @policy-engine  |
| Cryptography    | `pkg/crypto`, `internal/crypto` | @crypto-sec |
| PoA / RFC0115   | `pkg/poa`           | @poa-rfc            |
| Compliance      | `pkg/compliance`    | @compliance         |
| Audit / Ledger  | `pkg/audit`, `pkg/ledger` | @audit-ledger  |
| Web / Handlers  | `web/`              | @web-platform       |
| Frontend        | `web/ui-react`      | @frontend           |
| Infra / Deploy  | `deployments/`      | @infra              |

## Future Refactors (Non-Blocking)

- Extract replay protection to distinct module for potential externalization.
- Introduce structured error catalog package (currently examples/errors placeholder + TODO markers).
- Replace ad-hoc metrics interface expansions with a consolidated `metrics` subpackage contract.
- Evaluate moving compliance + jurisdiction logic into a `governance` umbrella module.

## Automation Opportunities

| Automation | Benefit | Outline |
|------------|---------|---------|
| `make release-notes` | Fast changelog synthesis | Parse commits -> group by conventional type |
| `make security-scan` (extended) | One-step security gate | gosec + govulncheck + SBOM generation |
| `make doc-bundle` | Portable doc artifact | Package curated docs into tar/zip for distribution |
| `make poa-cbor-fuzz` | Robust codec | Add fuzz target for newly added CBOR codec |

## Style & Formatting Enforcement

- `.editorconfig` (present) – extend with frontend indent specificity if needed.
- Consider adding `pre-commit` hooks: go fmt, golangci-lint fast subset, license header check.
- Run `go vet` + `golangci-lint` on changed paths only for speed (Git diff integration).

## Immediate Wins Implemented in This Cleanup Wave

- Backend env example (`.env.backend.example`)
- Unified dev startup script (`scripts/dev-up.sh`)
- Structure overview (`docs/STRUCTURE.md` – this document)

---
Generated: $(date +%Y-%m-%d) – Update as modules evolve.
