---
title: Code TODO / FIXME Report
category: generated
status: active
lastUpdated: 2025-11-12
owners: tooling
generated: true
source: todo-harvest-script
refreshCadence: daily-ci
tags: [maintenance, technical-debt]
---

# Code TODO / FIXME Report

> Last Updated: 2025-10-17
> Status: Active

Generated technical debt snapshot.

## Categorization

| ID | Location | Summary | Category | Priority | Action |
|----|----------|---------|----------|----------|--------|
| 1 | pkg/auth/auth.go:437 | Incorporate scopes into token serialization beyond demo format | Authorization / Feature completeness | High | Define token scope embedding & validation path |
| 2 | pkg/delegation/delegation.go:33 | Implement RevocationChain with hash linkage + Verify and integrate into authorization path | Security / Integrity / Revocation | High | Design struct + hashing scheme (prev-hash), verify traversal, plug into revocation checks |
| 3 | web/static/js/modules/samples.js:206 | Implement example viewing logic (UI modal/panel/redirect) | Frontend UX | Medium | Add component + route or dynamic modal loader |
| 4 | .git/hooks/sendemail-validate.sample (multiple) | Placeholder TODOs for sample hook checks | Tooling Template | Low (Ignore) | Leave as template or add note to ignore |

## Detail

### 1. Scopes in Token (pkg/auth/auth.go)
Current tokens omit scopes except in ephemeral demo structures. Need to embed scopes in signed representation and surface in validation path. Consider:
- Add `Scopes []string` to canonical token claims struct.
- Normalize/validate (deduplicate, sort) before serialization.
- Extend authorization decision to check token scopes against requested operation.
- Add tests: issue token with scopes, authorize allowed vs forbidden scope.

### 2. RevocationChain (pkg/delegation/delegation.go)
Objective: cryptographically link revocations forming append-only chain for auditability.
Design Sketch:
```go
// RevocationLink links one revocation event to prior via hash chaining.
type RevocationLink struct {
    ID        string    // unique revocation event ID
    Subject   string    // principal / delegation subject
    Reason    string    // textual summary
    Timestamp time.Time // event time (UTC)
    PrevHash  string    // hash of previous link (hex)
    Hash      string    // hash of this link's serialized payload
}

// RevocationChain maintains ordered links.
type RevocationChain struct {
    Links []RevocationLink
}

func (c *RevocationChain) Append(e RevocationLink) RevocationLink { /* compute PrevHash from last.Hash, compute e.Hash, push */ }
func (c *RevocationChain) Verify() error { /* recompute hashes sequentially; detect tampering */ }
```
Hash Function: SHA-256 over canonical JSON (stable field ordering) or binary concatenation: `ID|Subject|Reason|Timestamp|PrevHash`.

Integration:
- Store chain in memory store / persistence layer (future). For now, package-level singleton or injected dependency.
- On revocation, append new link after verifying chain integrity.
- Authorization path: during token/delegation validation, consult revocation index; optionally surface chain integrity status.
- Tests: tamper with a middle link's Reason, expect Verify() failure.

### 3. Frontend Example Viewing (samples.js)
Add UI affordance: clicking example opens detail panel/modal.
Options:
- Lightweight: inject hidden <div id="example-viewer">, populate with example JSON & syntax highlight.
- Route-based: integrate with existing client-side router (if present) using hash fragment `#example/<id>`.

Steps:
1. Add event listener to example list items.
2. Render detail view (title, description, code snippet) using safe escaping.
3. Provide close action.
4. Unit test (if frontend test harness emerges) or manual test plan.

### 4. Sample Git Hook Placeholders
The `.git/hooks/sendemail-validate.sample` contains instructional TODOs. These are not production code. Documented here for completeness; no remediation required.

## Suggested Next Actions
1. Implement RevocationChain (Item 2) - high integrity impact.
2. Integrate scopes into token claims & enforcement (Item 1).
3. Frontend example viewer (Item 3) improves demo usability.
4. Ignore sample hook placeholders (Item 4).

## Automation
Report generated manually (initial). Add `make todo-report` target + `scripts/tidy.sh` to regenerate automatically by scanning for patterns:
```
grep -R "TODO|FIXME" -n --exclude-dir=node_modules --exclude-dir=bin --exclude-dir=build --exclude-dir=.git --exclude='*.map' .
```
Apply categorization filters to exclude sample templates & generated artifacts.

---
Last Updated: $(date)

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
