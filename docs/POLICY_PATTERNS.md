# Policy Condition Patterns (Beta Demonstration)

> Last Updated: 2025-10-17
> Status: Active

> Educational / beta scope only. These patterns illustrate how a future full authorization engine (RBAC + ABAC + Delegation) could evaluate decisions. They DO NOT represent a production‑grade policy language. See `DISCLAIMER.md` and `AUTHORIZATION_IMPLEMENTATION.md` for gaps.

## Table of Contents
1. Core Decision Model
2. Subject / Role / Delegation Patterns
3. Scope & Action Patterns
4. Attribute (ABAC‑style) Patterns
5. Temporal & Expiry Patterns
6. Context / Environment Patterns
7. Revocation & Invalidation Patterns
8. Risk / Conditional Elevation Patterns
9. Chaining & Combining Strategies
10. Testing Strategies
11. Roadmap Enhancements

---
## 1. Core Decision Model
A (future) Policy Decision evaluates an `AuthorizationRequest` shaped like:
```go
AuthorizationRequest {
  Subject:  { UserID, Roles[], Attributes },
  Resource: { ID, Type, Owner, Labels },
  Action:   { ID, Verb, Category },
  Environment: map[string]any { "ip": "198.51.100.10", "time": now, "auth_strength": "pwd+mfa" },
  Delegation: *DelegationContext (optional)
}
```
Decision = ALLOW | DENY | NOT_APPLICABLE | INDETERMINATE

Combining (for multiple policy hits): e.g. *deny‑overrides*, *first‑applicable*, *permit‑overrides*.

---
## 2. Subject / Role / Delegation Patterns
### 2.1 Direct Role Allow
Allow if subject holds required role.
```pseudo
policy when roles CONTAINS "finance.approver" allow action IN {"invoice.approve"}
```
### 2.2 Hierarchical Role Inheritance
```pseudo
if any(role IN subject.roles WHERE role.ancestors CONTAINS "org.admin") then allow
```
### 2.3 Delegated Authority (Power‑of‑Attorney)
Grantor delegates restricted scope to grantee.
```pseudo
if delegation.present && delegation.grantor == resource.owner && action IN delegation.scope && now < delegation.valid_until allow
```
### 2.4 Delegation With Restrictions
```pseudo
if delegation.restrictions["max_amount"] >= request.context["amount"] allow else deny
```
### 2.5 Multi‑Principal (Joint) Authorization (future)
Require 2 distinct principals to authorize high‑risk action.
```pseudo
if action == "transfer.execute" require approvals.count >= 2 && distinct(approvals.subjects) >= 2
```

---
## 3. Scope & Action Patterns
### 3.1 Wildcard Scope
```pseudo
if action IN delegation.scope OR "*" IN delegation.scope allow
```
### 3.2 Action Category Mapping
Collapse verbs to a normalized category map.
```pseudo
READ: {"get","list","describe"}
WRITE: {"create","update","patch"}
DELETE: {"delete","revoke"}
```
Then:
```pseudo
if subject.role == "data.reader" && action.category == READ allow
```
### 3.3 Compound Action Gate
```pseudo
if action == "dataset.export" require subject.attributes["dpo_training"] == true && subject.region == resource.region
```

---
## 4. Attribute (ABAC‑style) Patterns
### 4.1 Owner Match
```pseudo
if resource.owner == subject.user_id allow
```
### 4.2 Label / Tag Based
```pseudo
if resource.labels["classification"] == "public" allow
```
### 4.3 Department Alignment
```pseudo
if subject.attributes["department"] == resource.labels["dept"] allow
```
### 4.4 Numeric Threshold
```pseudo
if request.context["amount"] <= subject.attributes["approval_limit"] allow
```
### 4.5 IP Range Control
```pseudo
if cidr_contains(policy.allowed_cidr, env.ip) allow else deny
```
### 4.6 Authentication Strength
```pseudo
if action == "vault.seal" require env.auth_strength >= "mfa"
```
(Requires ordered strength taxonomy: password < password+mfa < hardware_key)

---
## 5. Temporal & Expiry Patterns
### 5.1 Business Hours Window
```pseudo
if time.in_range("09:00-17:00", env.time, subject.timezone) allow else deny
```
### 5.2 Delegation Expiry
```pseudo
if now > delegation.valid_until deny
```
### 5.3 Cooldown / Rate Temporal Guard
```pseudo
if last_action("privileged.reset") < 24h ago deny
```
### 5.4 Scheduled Activation
```pseudo
if now < policy.effective_from deny
```
### 5.5 Weekend Restriction
```pseudo
if weekday(env.time) IN {SAT,SUN} deny
```

---
## 6. Context / Environment Patterns
### 6.1 Geo / Region Match
```pseudo
if env.region == resource.region allow
```
### 6.2 Risk Score Conditional
```pseudo
if env.risk_score > 70 deny
```
### 6.3 Device Posture
```pseudo
if env.device.trusted == true allow
```
### 6.4 Network Zone Tiering
```pseudo
if env.network_zone == "prod-admin" && subject.role == "ops.engineer" allow
```

---
## 7. Revocation & Invalidation Patterns
### 7.1 Explicit Delegation Revocation
```pseudo
if delegation.revoked == true deny
```
### 7.2 Token Invalidation (shadow list)
```pseudo
if token.id IN revocation_list deny
```
### 7.3 Policy Version Supersession
```pseudo
if policy.version < active_version(policy.id) deny (NotApplicable -> Deny upgrade)
```
### 7.4 Emergency Global Freeze
```pseudo
if env.global_freeze == true deny
```
### 7.5 Suspicious Behavior Flag
```pseudo
if subject.flags["suspicious"] == true deny
```

---
## 8. Risk / Conditional Elevation Patterns
### 8.1 Step‑Up Authentication
```pseudo
if action.sensitivity == HIGH && env.auth_strength < "mfa" require_step_up()
```
### 8.2 Progressive Access
```pseudo
if subject.tenure_days < 30 deny high_privilege_actions
```
### 8.3 Behavioral Anomaly
```pseudo
if deviation(user.behavior, baseline) > THRESHOLD deny
```
### 8.4 Dynamic Quota Clamp
```pseudo
if rolling_sum(subject.id, "trade.volume", 24h) > subject.attributes["daily_limit"] deny
```

---
## 9. Chaining & Combining Strategies
| Strategy | Description | When to Use |
|----------|-------------|-------------|
| Deny-Overrides | Any deny wins | Safety critical APIs |
| Permit-Overrides | First allow wins | Open/self-service actions |
| First-Applicable | Ordered evaluation | Layered fallback policies |
| Weighted Score | Score threshold gating | Risk adaptive decisions |

Example (pseudo):
```pseudo
for p in policies ORDERED BY p.priority:
  d = p.evaluate(request)
  if d == DENY: return DENY  // deny-overrides
  if d == ALLOW && combining == PERMIT_OVERRIDES: return ALLOW
```

---
## 10. Testing Strategies
| Test Type | Focus | Example |
|-----------|-------|---------|
| Unit | Single condition | "ipInRange denies 203.0.113.5" |
| Composition | Combining algorithm | Mixed allow/deny ordering |
| Edge Temporal | Boundary times | 08:59 vs 09:00 inclusion |
| Fuzz | Expression parser resilience | Random token sequences |
| Mutation | Tamper detection | Flip operator in condition |

Suggested Go test harness pattern:
```go
cases := []struct{ name string; req AuthorizationRequest; want Decision }{ /* ... */ }
for _, c := range cases {
  t.Run(c.name, func(t *testing.T){
     got := engine.Decide(c.req)
     if got.Decision != c.want { t.Fatalf("want %s got %s", c.want, got.Decision) }
  })
}
```

---
## 11. Roadmap Enhancements
1. DSL Parser (EBNF) with safe sandbox evaluation
2. Attribute fetch plugins (directory, HR, device posture, threat intel)
3. Incremental decision cache with dependency graph invalidation
4. Temporal policies compiled into interval index for O(log n) activation lookup
5. Policy provenance & signed bundles
6. Differential analysis (detect drift between staging vs prod policies)
7. Policy simulation mode (shadow evaluation with metrics)
8. Merkle anchored decision audit trail (pairs with hash-chained events)
9. Sidecar PDP for low latency edge authorization
10. Streaming revocation feed (Zero Trust revocation latency < 5s)

---
## Quick Visual Summary
```
[Request]
  │  (Gather Attributes & Delegation Context)
  ▼
[Match Targets] -> [Evaluate Conditions] -> [Collect Decisions] -> [Combine] -> [Obligations/Advices] -> [Decision]
```

> This document provides conceptual scaffolding for future implementation; current code base includes only minimal placeholder authorization logic.

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
