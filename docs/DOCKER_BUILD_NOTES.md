# Docker Build Notes & Troubleshooting

> Last Updated: 2025-10-17
> Status: Active

This document captures recent improvements and guidance for the beta demonstration GAuth Docker build process.

## Recent Changes

1. Added `Dockerfile.minimal` for fast, low-surface, diagnostic builds.
2. Enhanced `scripts/docker-build-robust.sh` to:
   - Automatically create a temporary `Dockerfile.minimal` if missing when the fallback path is triggered.
   - Provide guidance for diagnosing JSON parsing errors caused by misplaced markup or unexpected log formatting.
   - Offer a deterministic minimal build surface when the primary multi-stage build fails.

## When to Use Each Dockerfile

- `Dockerfile` (multi-stage, alpine → scratch-ish runtime): Use for normal builds and publishing.
- `Dockerfile.minimal` (added): Use for rapid debugging when encountering build failures unrelated to application code (e.g., cache path quirks, CI anomalies).

## Common Issue: `invalid character '<' looking for beginning of value`

This error does NOT originate from the Go build itself. It usually appears when a tool expects JSON-formatted output (e.g., GitHub Actions log processors / Docker buildx JSON streaming) but instead receives:

- Raw HTML (perhaps from an error page)
- An `<img ...>` tag accidentally pasted into a terminal / script
- Base64-encoded image data starting with a data URL or unintended blob in the log stream

### How to Diagnose

1. Force plain progress output:
   ```bash
   DOCKER_BUILDKIT=1 docker build --progress=plain -t debug:test . 2>&1 | tee build.log
   ```
2. Search for stray markup or binary prefixes:
   ```bash
   grep -n "<img" build.log || true
   grep -n "iVBORw0KG" build.log || true
   ```
3. If running in CI, check if a wrapper step adds annotation parsing or attempts to decode JSON from raw log lines.

### Likely Root Causes (Observed Context)
- Pasting rich content (HTML/base64) into an interactive terminal where a command awaited input.
- Upstream proxy responding with an HTML error page (check for corporate proxies altering responses during `apk add` or `go mod download`).

## Minimal Build Strategy
The minimal Dockerfile previously omitted:
- Non-root user creation (NOW ADDED: runs as UID 1001 for safer default)
- CA certificates bundle (still omitted; add if outbound TLS is required at runtime)
- Healthcheck instructions (NOW ADDED: uses `ENTRYPOINT -healthcheck` flag hitting `/api/v1/beta/health`)
- Extra utilities (still omitted)

Healthcheck behavior (now probes the long‑running beta web server):
```
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
   CMD ["/web-server", "-healthcheck"]
```
The `web-server` binary performs an HTTP GET to `GAUTH_HEALTH_URL` (default `http://localhost:8080/api/v1/beta/health`) and exits 0 on 2xx.

Why switch from the demo `gauth-server`? The `gauth-server` binary executes a finite protocol walk‑through then exits, which made the container appear unhealthy despite a successful demonstration. Using the persistent `web-server` aligns the healthcheck with an actual served endpoint and avoids false negatives.

Add them back incrementally only after confirming the core binary builds and starts.

### Makefile & Smoke Test Support

Convenience targets / scripts:

| Command | Purpose |
|---------|---------|
| `make docker-build-minimal` | Build the minimal image (`gauth:minimal`) |
| `make docker-smoke-minimal` | Build & run a smoke test (health probe) |
| `scripts/smoke-minimal.sh` | Underlying script (used by target) |

Smoke test steps performed:
1. Build the minimal image
2. Start a disposable container on :8080
3. Poll `/api/v1/beta/health` until 200 OK (timeout ~20s)
4. Exec in-container `web-server -healthcheck` to verify embedded probe
5. Output success summary and clean up container

Example:
```bash
make docker-smoke-minimal
```

### CI Automation

A GitHub Actions workflow (`.github/workflows/minimal-smoke.yml`) runs the same smoke test on every push / PR targeting `main` or `beta-refactor` to ensure the minimal image remains runnable and the health endpoint contract holds.

## Recommended Workflow for Failures

1. Run robust script:
   ```bash
   ./scripts/docker-build-robust.sh
   ```
2. If it falls back, inspect emitted diagnostics.
3. If error persists, capture full plain-progress log and attach it to an issue.
4. Run the built image (if successful):
   ```bash
   docker run --rm -p 8080:8080 gauth-demo:robust-build --help
   ```

## Housekeeping
- `.dockerignore` excludes large / irrelevant directories. Avoid re-adding the `gauth-demo-app` folder that previously caused cache key instability.
- If you add new top-level directories required for build, ensure they are copied explicitly in Dockerfiles.

## Next Steps
- Optionally add multi-arch `buildx bake` config for reproducibility.
- Consider adding a `make docker-build-minimal` target wiring to `Dockerfile.minimal`.
 - After confirming stability, consider adding CA certs stage if HTTPS outbound becomes a necessity for examples.

---
Beta demonstration implementation – NOT production ready. Do NOT use for real security, production, or commercial deployment.

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
