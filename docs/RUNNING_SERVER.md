---
title: Running Server
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Running the Web Server

The `web-server` command must be executed from the repository root (where `go.mod` lives). If you run it from inside the `web/` directory you will see errors like:

```
stat cmd/web-server/main.go: no such file or directory
```

or

```
stat /.../AgentAuth/web/cmd/web-server: directory not found
```

## Correct Commands (macOS / zsh)

From the repo root (`AgentAuth/`):

```zsh
go run ./cmd/web-server          # preferred (package path)
```

Optional explicit file form (also from root):

```zsh
go run cmd/web-server/main.go
```

## Background Run With Logging

```zsh
go run ./cmd/web-server 2>&1 | tee server.log
```

## Port Selection

Override port:

```zsh
AGENTAUTH_WEB_PORT=9090 go run ./cmd/web-server
```

## Healthcheck

Once started:

```zsh
curl -s http://localhost:8080/api/v1/beta/health
```

Container / script health probe:

```zsh
go run ./cmd/web-server -healthcheck
```

## Common Startup Env Vars

```zsh
export AGENTAUTH_TOKEN_SIG_MODE=eddsa           # enable Ed25519 token signatures
export AGENTAUTH_MULTI_SIG_THRESHOLD=2          # advertise multi-signature support in discovery
export AGENTAUTH_ATTEST_NONCE_TTL=30m           # attestation nonce TTL
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `stat cmd/web-server/main.go` | Wrong working directory | `cd /path/to/AgentAuth` then rerun |
| `address already in use` | Port 8080 occupied | `lsof -ti:8080 | xargs kill -9` or change port |
| Missing discovery fields | Env vars not set | Export required env vars before run |

---
_Generated helper doc (RB2 follow-up) to reduce run path confusion._