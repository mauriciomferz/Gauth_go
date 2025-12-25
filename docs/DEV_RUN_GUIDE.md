---
title: Dev Run Guide
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Development Run Guide

This guide clarifies how to launch key binaries and examples locally.

## Web Server (`cmd/web-server`)

Run from repository root:
```bash
go run ./cmd/web-server
```
Build binary:
```bash
go build -o bin/web-server ./cmd/web-server
./bin/web-server
```
Custom port:
```bash
GAUTH_WEB_PORT=9090 go run ./cmd/web-server
# or positional arg
go run ./cmd/web-server 9090
```
Health probe:
```bash
go run ./cmd/web-server -healthcheck
```
Common pitfall: Running from inside `web/` while referencing `cmd/web-server/main.go` fails because the relative path does not exist; always invoke from repo root.

## Key Rotation Example (`examples/key_rotation`)
Launch API:
```bash
go run ./examples/key_rotation
```
Then visit:
- http://localhost:8080/api/v1/keys/rotation/status
- http://localhost:8080/api/v1/keys/health

Exit status 127 previously observed was caused by using `timeout` on macOS (GNU `timeout` is not installed by default). Omit it or install coreutils:
```bash
brew install coreutils
# then use gtimeout if needed
gtimeout 10s go run ./examples/key_rotation
```

## Metrics Persistence
Enable snapshot persistence on graceful shutdown:
```bash
GAUTH_METRICS_PERSIST_PATH=~/.gauth-metrics go run ./cmd/web-server
```

## Troubleshooting
| Symptom | Cause | Resolution |
|---------|-------|-----------|
| exit status 127 with `timeout` | Missing GNU `timeout` | Remove or install coreutils (`gtimeout`) |
| exit status 1 from `web/` dir | Wrong relative path | Run from repo root |
| Port already in use | Previous instance alive | `lsof -ti:8080 | xargs kill -9` |

## Smoke Test Script
Run a quick end-to-end health probe (builds, starts, checks, stops):
```bash
bash scripts/smoke_web.sh
```
Optional: pass a port (defaults 8080):
```bash
bash scripts/smoke_web.sh 9090
```
Fails fast if health or discovery endpoints are not 200.

## Next Additions
- (Planned) Root Makefile targets (`make web`, `make key-rotation`, `make smoke`).
- Extend smoke script with capability diff & token issue probes.
