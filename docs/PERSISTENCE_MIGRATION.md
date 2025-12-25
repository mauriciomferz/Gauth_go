---
title: Persistence Migration
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Persistence & Migration Guide

Date: 2025-10-30

## Overview
The GAuth Beta introduces optional persistent storage for Power of Attorney (PoA) records and audit ledgers using BoltDB. Persistence is activated via the environment variable `GAUTH_PERSIST_PATH`. When set, the service will:

1. Initialize a BoltDB-backed PoA repository (file created if absent).
2. Store all new PoA issuance, status transitions, and revocation workflow states durably.
3. Maintain an audit ledger (hash-chain) persisted in BoltDB (if configured similarly for ledger path).

This guide provides migration steps for moving from in-memory operation to persistent storage while preserving integrity and minimizing downtime.

## Components
| Component | Persistence Artifact | Activation Variable | Notes |
|----------|----------------------|---------------------|-------|
| PoA Repository | `<GAUTH_PERSIST_PATH>` (BoltDB file) | `GAUTH_PERSIST_PATH` | Enables durable PoA storage & sub-delegation chains |
| Audit Ledger | Separate BoltDB file (e.g. `GAUTH_LEDGER_DB_PATH`) | `GAUTH_LEDGER_DB_PATH` (if present) | Hash-chain entries persisted for provenance |
| Daily Limit Store (Enhanced Validator) | Same or separate BoltDB file | N/A (explicit initialization) | Used for rate/limit policy enforcement |

## Schema & Buckets
PoA repository buckets:
- `poa`: key = PoA ID, value = JSON serialized `PowerOfAttorney` object
- `principal`: key = principal (grantor or grantee), value = JSON array of PoA IDs for quick lookup

Audit ledger buckets (from ledger package):
- `entries`: append-only records with chained hashes
- Optional indexing buckets (TTL pruning, external anchoring metadata)

## Enabling Persistence
1. Choose a filesystem path (directory must exist and be writable):
   ```bash
   export GAUTH_PERSIST_PATH=/var/lib/gauth/poa.db
   ```
2. (Optional) Enable persistent audit ledger:
   ```bash
   export GAUTH_LEDGER_DB_PATH=/var/lib/gauth/ledger.db
   ```
3. Start the service. On startup, BoltDB files will be created if they do not exist.

## Migration Strategy
If you have existing in-memory PoAs you wish to preserve:
1. Quiesce issuance (temporarily block new delegation creation).
2. Export current in-memory PoAs (JSON dump endpoint or internal tool – implement if not available).
3. Enable `GAUTH_PERSIST_PATH` and restart service.
4. Replay/exported PoAs into new persistent repository using admin/import tool. Ensure canonical digests re-computed match prior digests (log mismatches).
5. Resume issuance.

### Integrity Considerations
- Canonical digest versions (V1/V2/V3) must remain unchanged for pre-existing PoAs; do not mutate taxonomy fields post-migration without version bumping.
- Multi-signature aggregation status and revocation workflow state must be serialized intact.
- Sub-delegation depth (`Depth`) derived field should be preserved; if rebuilding hierarchy, recompute depths and validate against `GAUTH_MAX_DELEGATION_DEPTH`.

### Restart Resilience Test
A basic durability validation workflow:
1. Set `GAUTH_PERSIST_PATH` to temp file.
2. Issue root PoA and a child with `parent_poa_id`.
3. Record their IDs.
4. Stop service (or close repository handle).
5. Restart service with same path.
6. Query by ID; assert both records present with identical scope, status, and depth.

Automated test reference: `rfc0111_persistence_durability_test.go`.

## Operational Guidance
| Action | Recommendation |
|--------|---------------|
| Backup | Snapshot BoltDB files daily; use filesystem atomic copy (service can stay online – BoltDB supports read concurrency). |
| Restore | Replace BoltDB file from backup while service stopped; verify file permissions; run ledger verify endpoint before accepting traffic. |
| Compaction | Periodic export + re-import to new file to reclaim space (BoltDB retains freed pages). |
| Monitoring | Track file size growth; alert on rapid expansion (possible issuance flood or misuse). |

## Failure Modes & Mitigations
| Failure | Symptom | Mitigation |
|---------|---------|-----------|
| Corrupted BoltDB | Open error / panic | Keep rotating backups; attempt bolt `db.View()` salvage tooling; fallback to last good backup. |
| Missing persistence path | PoAs not durable post-restart | Ensure env var set in systemd/unit/container spec; add startup check refusing to run without path in production mode. |
| Partial writes (power loss) | Inconsistent record counts | BoltDB transactional safety reduces risk; after restart run integrity scan comparing principal index counts vs. actual PoAs. |

## Future Enhancements
- Postgres repository implementation for HA & horizontal scaling.
- Digest domain bump for hierarchical contexts (exclude dynamic fields until version upgrade strategy defined).
- Export/import CLI for seamless migration and backup verification.
- Background compaction and TTL pruning for principals with large delegation churn.

## Checklist
- [ ] Set `GAUTH_PERSIST_PATH` in deployment manifest
- [ ] Create writable directory; set secure permissions (600 for file)
- [ ] Run restart resilience test in staging
- [ ] Configure backup job
- [ ] Monitor metrics: `delegation_max_observed_depth`, issuance rate, ledger size
- [ ] Document recovery steps in runbook

## Example systemd Unit Snippet
```ini
[Service]
Environment=GAUTH_PERSIST_PATH=/var/lib/gauth/poa.db
Environment=GAUTH_LEDGER_DB_PATH=/var/lib/gauth/ledger.db
ExecStart=/usr/local/bin/gauth-server
Restart=on-failure
RuntimeMaxSec=86400
```

## Security Notes
- Restrict filesystem permissions; PoA records contain governance metadata and revocation workflow state.
- Consider encrypted volume or OS-level disk encryption for sensitive jurisdictions.
- Validate backup integrity by periodically loading copy into staging and running ledger verification + sample PoA queries.

---
Generated: 2025-10-30
