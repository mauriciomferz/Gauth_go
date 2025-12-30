---
title: Key Rotation Example
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Key Rotation Example (Multi-Tenant)

The example at `examples/key_rotation/main.go` demonstrates:
* File-backed key store (`/tmp/gauth-keys`)
* Multi-tenant key manager with per-tenant policies
* HTTP API for rotation operations

## Run (macOS / zsh)

From repository root:
```zsh
go run ./examples/key_rotation/main.go
```

macOS does not include GNU `timeout` by default; avoid `timeout 10s go run ...` (Exit 127). Use Ctrl+C to stop.

## Initial Output (abridged)
```
Generated and activated initial key <keyId> for tenant tenant-a
Generated and activated initial key <keyId> for tenant tenant-b
Generated and activated initial key <keyId> for tenant tenant-c
Starting AgentAuth Key Rotation API server on :8080
```

## Key Endpoints

| Operation | Method & Path | Notes |
|-----------|---------------|-------|
| Global rotation status | GET /api/v1/keys/rotation/status | Lists tenants & active schedules |
| Tenant rotation status | GET /api/v1/keys/rotation/status/:tenant | Per-tenant view |
| List tenant keys | GET /api/v1/keys/list/:tenant | Active, archived, grace-period keys |
| Trigger rotation | POST /api/v1/keys/rotation/trigger/:tenant | Forces immediate rotation |
| Update policy | PUT /api/v1/keys/rotation/policy/:tenant | Adjust interval/jitter/grace |
| Activate key | POST /api/v1/keys/activate/:tenant/:keyId | Promote a generated key |
| Archive key | POST /api/v1/keys/archive/:tenant/:keyId | Mark no longer active, but usable if in grace |
| Delete key | DELETE /api/v1/keys/:tenant/:keyId | Irreversible removal |
| Health | GET /api/v1/keys/health | Basic liveness check |

## Demo: Trigger All Rotations
```zsh
curl -X POST http://localhost:8080/demo/trigger-all-rotations | jq .
```

## Example Policy Update
Change tenant-a rotation interval to 12h:
```zsh
curl -X PUT http://localhost:8080/api/v1/keys/rotation/policy/tenant-a \
  -H 'Content-Type: application/json' \
  -d '{"enabled":true,"interval_hours":12,"jitter_hours":1,"max_key_age_hours":168,"grace_period_hours":24}'
```

## File Layout
Keys stored under `/tmp/gauth-keys/<tenant>/<kid>.json`. Inspect:
```zsh
find /tmp/gauth-keys -maxdepth 2 -type f -print
```

## Grace Period Concept
When a new key is activated:
1. Previous key enters grace period (still accepted for verification).
2. After `grace_period_hours` expires it becomes archival-only (verification disabled).

## Common Issues
| Symptom | Cause | Resolution |
|---------|-------|-----------|
| Exit 127 running with `timeout` | GNU timeout not installed | Omit `timeout`, or brew install coreutils (`gtimeout`) |
| 403/404 on rotation trigger | Tenant not registered | Confirm tenant name in request path |
| No new key after trigger | Rotation policy disabled | PUT updated policy with `enabled=true` |

## Stopping the Server
Ctrl+C or send SIGINT. For background run with log capture:
```zsh
go run ./examples/key_rotation/main.go 2>&1 | tee key_rotation.log
```

---
_Adjunct to RB2: operational clarity for crypto governance demonstration._