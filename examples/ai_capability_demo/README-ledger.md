---
title: Readme-Ledger
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# AI Capability Demo Ledger Extension

This document augments the main README with audit ledger anchoring details.

## Overview
The demo includes an append-only Merkle ledger (in-memory by default; optional BoltDB persistence when `GAUTH_AI_DEMO_LEDGER_DB_PATH` is set) to anchor events (e.g., PoA lifecycle, decisions). Each append produces a new Merkle root that can be fetched and inclusion proofs can be generated per entry.

## Data Model
Entry fields:
- id (sequential uint64)
- type (string) semantic category, e.g. `decision`, `poa`, `audit`
- payload (string) opaque content hashed as a leaf
- parent_root (string) optional previous external root for cross-ledger stitching
- timestamp (RFC3339)
- hash (hex SHA-256 of leaf canonical form)

## Hashing
Leaf canonical form:
```
<id>|<type>|<payload>|<parent_root>|<timestamp>
```
Merkle layer construction duplicates the last node when odd count (power-of-two padding) to stabilize tree height.

## Endpoints
- POST `/demo/ledger/append` body: `{"type":"decision","payload":"...","parent_root":"optional"}`
  - Response: `{"id":0,"root":"<hexroot>","entry_hash":"<hash>"}`
- GET `/demo/ledger/root/latest` response: `{"root":"<hexroot>","entries":N}`
- GET `/demo/ledger/entry/:id/proof` response: `{"entry_id":2,"entry_hash":"<hash>","siblings":["hex","..."],"root":"<hexroot>"}`
- GET `/demo/ledger/roots` response: `{"roots":["<root1>","<root2>",...],"count":N}` (historical root sequence for external verifiers / auditors)

## Metrics
Core ledger metrics:
- `ai_demo_ledger_appends_total{type="<type>"}` incremented per append
- `ai_demo_ledger_root_emissions_total` incremented per root retrieval

Unified events metric:
- `ai_demo_ledger_events_total{event="<event>"}` increments for all anchored lifecycle events, providing a single series for dashboarding. Current event labels:
  - `decision_enforce`
  - `poa_issue`
  - `poa_prepare`
  - `poa_sign`
  - `poa_finalize`
  - `poa_revoke`
  - `poa_token_issue`
  - `ledger_append_manual`
  - `ledger_root_emit`

You can correlate specific PoA or decision operations with their corresponding ledger append by filtering on the event label.

Proof request metric:
- `ai_demo_ledger_proof_requests_total` increments on each inclusion proof retrieval.

## Proof Verification
Client recomputation algorithm used in tests (simplified):
1. Start from `entry_hash` (leaf digest).
2. Iterate through `siblings` in order; for each level pair do deterministic left-right pairing (current, sibling) and SHA-256 the concatenation bytes.
3. If odd node duplication occurred server-side the test logic mirrors by pairing the node with itself.
4. Result must equal supplied `root`.

Orientations: Proof responses now include an `orientations` array parallel to `siblings` where each element is `L` or `R` indicating whether the original node at that level was a left or right child. Rebuild algorithm:

```
acc = leaf(entry_digest)
for i in range(len(siblings)):
  if orientations[i] == 'L':
    acc = SHA256(acc || siblings[i])
  else: # node was right child
    acc = SHA256(siblings[i] || acc)
assert acc == root
```

This removes ordering ambiguity and prepares for future compressed proof formats.

## Token Issuance Anchoring
Issuance of a PoA-bound token now appends a `poa_token_issue` entry, ensuring traceability of credential creation events alongside lifecycle transitions.

## Persistence
Enable BoltDB-backed persistence (experimental):
```bash
export GAUTH_AI_DEMO_LEDGER_DB_PATH=/tmp/ledger.db
```
On startup you'll see a log: `[ledger] bolt persistence enabled at /tmp/ledger.db`. Entries are stored sequentially in a single bucket; roots are recomputed incrementally.

## Future Extensions
- Persistent storage enhancements (batch write, compaction)
- Batch append for throughput
- Cross-ledger anchoring using `parent_root`
- SNARK / STARK proof integration
- Expiration and pruning strategy (archival store)
- Proof orientation bits & size optimization

_Last updated: historical roots endpoint, unified events metric, token issuance anchoring, and proof recomputation test._
