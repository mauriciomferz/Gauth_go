---
title: Test Diagnostics
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Testing Notes (Sunset Enforcement)

`TestCapabilitySunsetEnforcement` confirms that a sunset capability is denied (403) and lifecycle audit metadata marks the phase `sunset`.

Key flags for deterministic isolated runs:

| Flag | Purpose |
|------|---------|
| GAUTH_CAPABILITY_ENFORCE | Enable capability enforcement logic |
| GAUTH_CAP_LIFECYCLE_SUNSET_ENFORCE | Deny usage after sunset timestamp |
| GAUTH_SKIP_SMOKETEST | Skip CSP / smoketest hitting live server |
| GAUTH_DISABLE_BG_POLLS | Suppress autosave/anomaly/metrics loops |
| SKIP_WEB_ASSETS | Skip JS bundling step (faster quiet test build) |
| GAUTH_TEST_TRACE_HEARTBEAT_MS | Override heartbeat phase duration (default 120) when tracing |

Recommended single-test command (no background polls):

```bash
GAUTH_DISABLE_BG_POLLS=1 GAUTH_SKIP_SMOKETEST=1 go test -run ^TestCapabilitySunsetEnforcement$ -count=1 -v ./web
```

Background loops may be re-enabled (omit `GAUTH_DISABLE_BG_POLLS`) when profiling metrics or anomaly sampling.

This file was condensed; historical diagnostic details have been removed now that shutdown and loop suppression are implemented.

## Piped / Truncated Test Output (SIGPIPE) Troubleshooting

When piping `go test` output into commands that terminate early (e.g. `| head -n 10`), the downstream process closes its read end once it has enough lines. The upstream `go test` then encounters a broken pipe (SIGPIPE). Depending on timing, you may see:

* Missing initial `=== RUN` / `--- PASS` lines (appears "silent").
* Partial build banner (asset builder logs dominate early output).
* Heartbeat or diagnostic lines truncated mid-line.

### Replication

```
GAUTH_TEST_TRACE_SIGPIPE=1 go test -run ^TestCapabilitySunsetEnforcement$ -count=1 -v ./web | head -n 15
```

### Why It Happens

`head` exits after reading N lines. The pipe closes; writes from the test process to stdout/stderr after that point raise SIGPIPE at the OS level. Go's runtime treats SIGPIPE by exiting *quietly* (no stack trace) once standard writers error, so final summary lines may not flush.

### Mitigations & Best Practices

1. Avoid destructive consumers: Prefer `tee` over `head` if you need a preview while preserving full output.
2. Use JSON mode for structured capture: `go test -json ./web | jq -r '.Output?'` preserves event ordering and includes run/pass metadata even if truncated.
3. Disable noisy background loops: `GAUTH_DISABLE_BG_POLLS=1 GAUTH_SKIP_SMOKETEST=1` reduces early log flood so essential lines appear before truncation.
4. Harness heartbeat: Set `GAUTH_TEST_TRACE_SIGPIPE=1` to emit a `[trace] test harness starting` and timed heartbeats for early visibility (file: `web/testmain_trace.go`).
5. Early flush strategy: Add minimal `fmt.Fprintln(os.Stderr, "[pre] ready")` in `TestMain` (already implemented via heartbeat delay) if deeper debugging is needed.
6. Heartbeat duration override: Adjust pre-test window with `GAUTH_TEST_TRACE_HEARTBEAT_MS=250` (milliseconds).
6. Limit asset build noise: If front-end build output obscures test lines, run tests with `SKIP_WEB_ASSETS=1` (future enhancement) or run package-specific tests (`./web`) after initial build.
7. Capture full log then slice: `go test -v ./web 2>&1 | tee /tmp/test.log && sed -n '1,50p' /tmp/test.log` avoids SIGPIPE.
8. Fail fast for quicker signal: Add `-failfast` to stop after first failure, reducing window for truncation.
9. Structured artifacts: Consider redirecting stderr separately to ensure heartbeat presence: `... 1>out.log 2>err.log`.
10. CI environments: Avoid piping to `head`; instead rely on test filtering (`-run`) and verbosity controls.

### Recommended Minimal Command (Deterministic & Quiet)

```
SKIP_WEB_ASSETS=1 GAUTH_DISABLE_BG_POLLS=1 GAUTH_SKIP_SMOKETEST=1 go test -run ^TestCapabilitySunsetEnforcement$ -count=1 -v ./web
```

### Future Enhancements (Optional)

* `SKIP_WEB_ASSETS=1` env flag to bypass JS bundling for pure Go test runs.
* Add `go test -json` wrapper script (`scripts/test-json.sh`) for standardized parsing.
* Toggle to extend heartbeat until first `=== RUN` observed (adaptive rather than fixed 120ms delay).

### Quick Decision Matrix

| Goal | Suggested Flags / Approach |
|------|----------------------------|
| See early harness start | `GAUTH_TEST_TRACE_SIGPIPE=1` |
| Reduce noise | `GAUTH_DISABLE_BG_POLLS=1 GAUTH_SKIP_SMOKETEST=1` |
| Speed up build (no JS) | `SKIP_WEB_ASSETS=1` |
| Structured capture | `go test -json` |
| Avoid truncation | Use `tee`, no `head` |
| Faster failure visibility | `-failfast -run ^TestName$` |
| Longer pre-test window | `GAUTH_TEST_TRACE_HEARTBEAT_MS=500 GAUTH_TEST_TRACE_SIGPIPE=1` |

