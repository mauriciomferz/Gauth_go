---
title: Authorization Implementation
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Real Authorization Implementation

> Last Updated: 2025-10-17
> Status: Active

## 🛡️ Current Implementation Status (Beta Refactor)

Implemented (this branch):
- In-memory `MemoryAuthorizer` with policy matching (subject/resource/action wildcard + conditions).
- Role-based matching via `Policy.Roles` (context key `roles`).
- Required scope enforcement via `Policy.RequiredScopes` against context `scopes` list.
- Advanced ABAC operators: `equals`, `not_equals`, `in`, `contains`, `prefix`, `suffix`.
- Revocation chain (`RevocationChain`) for delegation/POA revocations (tamper-evident hash linkage) integrated into AAP-001 service.
- JWT demo token scopes embedded & extracted (`Claims.HasScope`).

Still Missing (roadmap):
- Persistent backing store & indexing for policies, roles, revocations.
- Combining algorithms (Deny-overrides, Permit-overrides) for multiple matching policies.
- Attribute sourcing (external directories, dynamic context enrichment).
- Delegation issuance migration to hash-linked chain (currently simple map).
- Policy versioning, rollback, real-time reloading.
### **Required Implementation:**

}

type Role struct {
    ID          string
    Name        string
    Description string
    Permissions []Permission
    Inherits    []string // Parent roles
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type Permission struct {
    ID       string
    Resource string
    Action   string
    Effect   PermissionEffect // Allow/Deny
    Conditions []Condition
}

type AccessRequest struct {
    UserID    string
    Resource  string
    Action    string
    Context   map[string]interface{}
    Timestamp time.Time
}

func (rbac *RBACEngine) Authorize(req *AccessRequest) (*AuthorizationDecision, error) {
    // Get user roles
    userRoles, err := rbac.userRoleStore.GetUserRoles(req.UserID)
    if err != nil {
        return nil, fmt.Errorf("failed to get user roles: %w", err)
    }
    
    // Collect all permissions (including inherited)
    allPermissions := make([]Permission, 0)
    for _, roleID := range userRoles {
        permissions, err := rbac.getEffectivePermissions(roleID)
        if err != nil {
            return nil, fmt.Errorf("failed to get permissions for role %s: %w", roleID, err)
        }
        allPermissions = append(allPermissions, permissions...)
    }
    
    // Evaluate permissions
    decision := rbac.evaluatePermissions(allPermissions, req)
    
    // Log the decision
    rbac.auditLogger.LogAuthorizationDecision(req, decision)
    
    return decision, nil
}

func (rbac *RBACEngine) evaluatePermissions(permissions []Permission, req *AccessRequest) *AuthorizationDecision {
    var allowDecisions []Permission
    var denyDecisions []Permission
    
    for _, perm := range permissions {
        if rbac.matchesResourceAction(perm, req.Resource, req.Action) {
            // Evaluate conditions
            if rbac.evaluateConditions(perm.Conditions, req.Context) {
                if perm.Effect == PermissionAllow {
                    allowDecisions = append(allowDecisions, perm)
                } else {
                    denyDecisions = append(denyDecisions, perm)
                }
            }
        }
    }
    
    // Deny takes precedence
    if len(denyDecisions) > 0 {
        return &AuthorizationDecision{
            Decision: DecisionDeny,
            Reason:   "Explicit deny permission found",
            MatchingPermissions: denyDecisions,
        }
    }
    
    if len(allowDecisions) > 0 {
        return &AuthorizationDecision{
            Decision: DecisionAllow,
            Reason:   "Allow permission granted",
            MatchingPermissions: allowDecisions,
        }
    }
    
    return &AuthorizationDecision{
        Decision: DecisionDeny,
        Reason:   "No matching allow permissions found",
    }
}
```

#### **B. Attribute-Based Access Control (ABAC)**
```go
type ABACEngine struct {
    policyStore    PolicyStore
    attributeStore AttributeStore
    evaluator      *PolicyEvaluator
}

type Policy struct {
    ID          string
    Name        string
    Description string
    Rules       []PolicyRule
    Effect      PolicyEffect
    Version     string
    Active      bool
}

type PolicyRule struct {
    Condition   string // XACML-like expression
    Target      Target
    Effect      PolicyEffect
    Obligations []Obligation
}

type Target struct {
    Subjects  []AttributeMatch
    Resources []AttributeMatch
    Actions   []AttributeMatch
    Environment []AttributeMatch
}

type AttributeMatch struct {
    AttributeID string
    Match       MatchType
    Value       interface{}
}

func (abac *ABACEngine) Evaluate(request *AuthorizationRequest) (*PolicyDecision, error) {
    // Get applicable policies
    policies, err := abac.policyStore.GetApplicablePolicies(request)
    if err != nil {
        return nil, fmt.Errorf("failed to get applicable policies: %w", err)
    }
    
    var decisions []*PolicyDecision
    
    for _, policy := range policies {
        decision, err := abac.evaluatePolicy(policy, request)
        if err != nil {
            continue // Log error but continue evaluation
        }
        decisions = append(decisions, decision)
    }
    
    // Combine decisions using policy combining algorithm
    return abac.combineDecisions(decisions), nil
}

func (abac *ABACEngine) evaluatePolicy(policy *Policy, request *AuthorizationRequest) (*PolicyDecision, error) {
    for _, rule := range policy.Rules {
        // Check if rule target matches request
        if !abac.matchesTarget(rule.Target, request) {
            continue
        }
        
        // Evaluate rule condition
        result, err := abac.evaluator.EvaluateCondition(rule.Condition, request.Attributes)
        if err != nil {
            return nil, fmt.Errorf("condition evaluation failed: %w", err)
        }
        
        if result {
            return &PolicyDecision{
                Decision:    rule.Effect,
                PolicyID:    policy.ID,
                RuleID:      rule.Condition,
                Obligations: rule.Obligations,
            }, nil
        }
    }
    
    return &PolicyDecision{
        Decision: PolicyNotApplicable,
        PolicyID: policy.ID,
    }, nil
}
```

#### **C. Policy Decision Point (PDP)**
```go
type PolicyDecisionPoint struct {
    rbacEngine    *RBACEngine
    abacEngine    *ABACEngine
    combiningAlg  CombiningAlgorithm
    obligations   ObligationService
    advices       AdviceService
}

func (pdp *PolicyDecisionPoint) MakeDecision(request *AuthorizationRequest) (*AuthorizationResponse, error) {
    // Validate request
    if err := pdp.validateRequest(request); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }
    
    // Evaluate RBAC
    rbacDecision, err := pdp.rbacEngine.Authorize(&AccessRequest{
        UserID:   request.Subject.UserID,
        Resource: request.Resource.ID,
        Action:   request.Action.ID,
        Context:  request.Environment,
    })
    if err != nil {
        return nil, fmt.Errorf("RBAC evaluation failed: %w", err)
    }
    
    // Evaluate ABAC
    abacDecision, err := pdp.abacEngine.Evaluate(request)
    if err != nil {
        return nil, fmt.Errorf("ABAC evaluation failed: %w", err)
    }
    
    // Combine decisions
    finalDecision := pdp.combineDecisions(rbacDecision, abacDecision)
    
    // Process obligations
    obligations, err := pdp.obligations.ProcessObligations(finalDecision.Obligations, request)
    if err != nil {
        return nil, fmt.Errorf("obligation processing failed: %w", err)
    }
    
    return &AuthorizationResponse{
        Decision:    finalDecision.Decision,
        Reason:      finalDecision.Reason,
        Obligations: obligations,
        Advice:      pdp.advices.GenerateAdvice(finalDecision, request),
        Timestamp:   time.Now(),
    }, nil
}
```

#### **D. Dynamic Policy Evaluation**
```go
type PolicyEvaluator struct {
    expressionEngine ExpressionEngine
    functions        map[string]Function
    cache           *sync.Map
}

// Evaluate complex policy expressions
func (pe *PolicyEvaluator) EvaluateCondition(condition string, attributes map[string]interface{}) (bool, error) {
    // Check cache first
    cacheKey := fmt.Sprintf("%s:%x", condition, hash(attributes))
    if cached, ok := pe.cache.Load(cacheKey); ok {
        return cached.(bool), nil
    }
    
    // Parse expression
    expr, err := pe.expressionEngine.Parse(condition)
    if err != nil {
        return false, fmt.Errorf("expression parsing failed: %w", err)
    }
    
    // Create evaluation context
    ctx := &EvaluationContext{
        Attributes: attributes,
        Functions:  pe.functions,
        Timestamp:  time.Now(),
    }
    
    // Evaluate
    result, err := expr.Evaluate(ctx)
    if err != nil {
        return false, fmt.Errorf("expression evaluation failed: %w", err)
    }
    
    boolResult, ok := result.(bool)
    if !ok {
        return false, fmt.Errorf("expression must evaluate to boolean, got %T", result)
    }
    
    // Cache result
    pe.cache.Store(cacheKey, boolResult)
    
    return boolResult, nil
}

// Built-in policy functions
func (pe *PolicyEvaluator) initializeFunctions() {
    pe.functions = map[string]Function{
        "hasRole": func(args []interface{}) (interface{}, error) {
            if len(args) != 2 {
                return false, fmt.Errorf("hasRole requires 2 arguments")
            }
            userRoles, ok := args[0].([]string)
            if !ok {
                return false, fmt.Errorf("first argument must be []string")
            }
            requiredRole, ok := args[1].(string)
            if !ok {
                return false, fmt.Errorf("second argument must be string")
            }
            
            for _, role := range userRoles {
                if role == requiredRole {
                    return true, nil
                }
            }
            return false, nil
        },
        
        "inTimeRange": func(args []interface{}) (interface{}, error) {
            // Implementation for time-based access control
            now := time.Now()
            startTime, _ := args[0].(time.Time)
            endTime, _ := args[1].(time.Time)
            return now.After(startTime) && now.Before(endTime), nil
        },
        
        "ipInRange": func(args []interface{}) (interface{}, error) {
            // Implementation for IP-based access control
            clientIP := args[0].(string)
            allowedRange := args[1].(string)
            _, ipNet, err := net.ParseCIDR(allowedRange)
            if err != nil {
                return false, err
            }
            ip := net.ParseIP(clientIP)
            return ipNet.Contains(ip), nil
        },
    }
}
```

### **Implementation Complexity: EXTREMELY HIGH**
- **Time Estimate**: 10-16 weeks
- **Required Skills**: Authorization systems, policy languages, distributed systems
- **Performance Requirements**: Sub-millisecond decision times
- **Scalability**: Handle millions of authorization requests/second
- **Testing**: Extensive policy testing framework required

### **Critical Features Required:**
1. **Policy Versioning and Rollback**
2. **Real-time Policy Updates**
3. **Decision Caching and Invalidation**
4. **Performance Monitoring**
5. **Policy Conflict Detection**
6. **Distributed Decision Points**
7. **Integration with External Attribute Stores**

---

## Quick Usage (Implemented Subset)

### Define a Policy with Roles & Required Scopes
```go
authzMem := authz.NewMemoryAuthorizer()
authzMem.AddPolicy(authz.Policy{
    ID: "read-doc",
    Roles: []string{"doc_reader"},
    Resource: "documents/123",
    Actions: []string{"read"},
    Effect: authz.Allow,
    RequiredScopes: []string{"docs:read"},
    Conditions: []authz.Condition{{Key: "classification", Operator: "in", Values: []string{"public","internal"}}},
})

req := authz.Request{
    Subject: "alice",
    Resource: "documents/123",
    Action: "read",
    Context: map[string]string{
        "roles": "doc_reader", // comma-separated
        "scopes": "docs:read other:scope",
        "classification": "public",
    },
}

decision, _ := authzMem.Authorize(context.Background(), req)
fmt.Println(decision.Allow, decision.Reason)
```

### Persistent Policy Store & Hot Reload (New)
Load policies from a JSON file and auto-reload them when the file changes via polling.

#### JSON Format
Policies file is a JSON array of objects matching the `Policy` struct:
```json
[
    {
        "id": "read-doc",
        "roles": ["doc_reader"],
        "resource": "documents/123",
        "actions": ["read"],
        "effect": "allow",
        "required_scopes": ["docs:read"],
        "conditions": [
            {"key": "classification", "operator": "in", "values": ["public", "internal"]}
        ]
    },
    {
        "id": "deny-secret",
        "subject": "*",
        "resource": "documents/secret",
        "actions": ["read"],
        "effect": "deny"
    }
]
```

#### Usage
```go
store, err := authz.NewFilePolicyStore("./policies.json")
if err != nil { panic(err) }
pa, err := authz.NewPersistentAuthorizer(store, 2*time.Second)
if err != nil { panic(err) }
pa.Start(); defer pa.Stop()
req := authz.Request{Subject: "alice", Resource: "documents/123", Action: "read", Context: map[string]string{
    "roles": "doc_reader",
    "scopes": "docs:read other:scope",
    "classification": "public",
}}
dec, _ := pa.Authorize(context.Background(), req)
fmt.Println(dec.Allow, dec.Reason)
```

Editing `policies.json` (e.g. flip an effect to `deny`) and saving will update decisions after the next poll interval.

#### Error Handling
- Missing file => empty policy set (default deny).
- Malformed JSON => reload error; previous policies retained.
- Atomicity: slice replaced wholesale; no partial updates.

#### Limitations & Future Work
- Polling only (fsnotify planned).
- No policy validation or combining algorithms.
- No versioning / audit trail.
- No decision cache invalidation hooks yet.

#### Planned Enhancements
1. fsnotify event-driven reload.
2. Policy lint & schema validation.
3. Change diff audit events.
4. Hierarchical directory store (base + env overrides).
5. Index structures for large policy counts.

### Revocation Chain Integration (AAP-001 Service)
After revocation, validation checks chain integrity and revoked state:
```go
svc := rfc0111.NewService(audit.NewMemoryLogger(nil), authzMem)
// create delegation (ensure create_delegation policy permits it)
_ = svc.RevokeDelegationCtx(ctx, poaID, grantor) // appends RevocationEvent
err := svc.ValidateDelegationCtx(ctx, poaID, grantee, "read") // returns error: revoked
```

### ABAC Condition Operators
Use `Condition.Operator` with: `equals`, `not_equals`, `in`, `contains`, `prefix`, `suffix`, `regex`, `numeric_gt`, `numeric_lt`, `time_before`, `time_after`.

```go
authzMem.AddPolicy(authz.Policy{ID: "env-prod", Subject: "alice", Resource: "svc", Actions: []string{"deploy"}, Effect: authz.Allow,
    Conditions: []authz.Condition{{Key: "env", Operator: "equals", Values: []string{"prod"}}}})

// Regex match on email
authzMem.AddPolicy(authz.Policy{ID: "corp-email", Subject: "alice", Resource: "mail", Actions: []string{"send"}, Effect: authz.Allow,
    Conditions: []authz.Condition{{Key: "email", Operator: "regex", Values: []string{"^alice@corp\\.example\\.com$"}}}})

// Numeric greater than (age > 21)
authzMem.AddPolicy(authz.Policy{ID: "age-check", Subject: "bob", Resource: "bar", Actions: []string{"enter"}, Effect: authz.Allow,
    Conditions: []authz.Condition{{Key: "age", Operator: "numeric_gt", Values: []string{"21"}}}})

// Time window (request timestamp before deadline)
deadline := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)
authzMem.AddPolicy(authz.Policy{ID: "submit-before-deadline", Subject: "carol", Resource: "reports", Actions: []string{"submit"}, Effect: authz.Allow,
    Conditions: []authz.Condition{{Key: "now", Operator: "time_before", Values: []string{deadline}}}})
```

Operator Semantics:
- regex: Go RE2 pattern; invalid patterns ignored (non-match).
- numeric_gt / numeric_lt: float parsing (`strconv.ParseFloat`); invalid numbers ignored.
- time_before / time_after: RFC3339 timestamps; invalid timestamps ignored.
- Multiple threshold values: match if ANY value condition holds (e.g., numeric_gt: value > any threshold).

Edge Cases & Notes:
- All operators are ANDed across the `Conditions` array; each condition individually uses OR semantics across its `Values` slice.
- Failure to parse (regex/number/time) is treated as non-match (safe default).
- Time comparisons assume UTC input; convert prior to policy evaluation if necessary.

### Regex Compilation Caching (New)
The `regex` operator now uses an internal compilation cache to avoid recompiling identical patterns on every authorization decision. This reduces CPU overhead and GC pressure for frequently used regex-based policies.

Mechanics:
- On first encounter of a pattern string, it is compiled and stored in `regexCache`.
- Subsequent evaluations reuse the compiled `*regexp.Regexp` instance.
- Invalid patterns increment a `regex_compile_errors` metric and are ignored (treated as non-match), preserving safe deny defaults.

Exposed Metrics (Prometheus + Snapshot):
- `authz_regex_compiles_total`: Count of successful pattern compilations (each unique pattern once).
- `authz_regex_compile_errors_total`: Count of failed compilation attempts (malformed patterns).
- `authz_regex_cache_size`: Current number of cached compiled patterns.

Access via snapshot:
```go
snap := ma.GetMetricsSnapshot()
fmt.Printf("regex_compiles=%d regex_errors=%d cache_size=%d\n", snap.RegexCompiles, snap.RegexCompileErrors, snap.RegexCacheSize)
```

Best Practices:
- Prefer anchoring (`^...$`) to avoid unintended substring matches.
- Keep patterns simple; complex backtracking not supported (RE2 engine semantics).
- Log or monitor compile error counter for policy hygiene; a rising error count indicates malformed policy entries.

Future Enhancements:
- TTL or LRU for regex cache when pattern cardinality grows large.
- Pre-validation/linting step during policy load to catch invalid regex early.
- Metrics for per-pattern match frequency.


### Decision Caching (New)
`MemoryAuthorizer` now supports an in-memory decision cache to reduce policy evaluation overhead for repeated identical requests.

#### Enabling
```go
ma := authz.NewMemoryAuthorizer()
ma.EnableCaching(500 * time.Millisecond) // TTL for each cached decision
```

#### Behavior
- Key fields: `subject|resource|action|roles|scopes` (context keys `roles`, `scopes`).
- First evaluation stores decision with `Metadata["cache_hit"] = "false"`.
- Subsequent evaluations before TTL expiry return cached decision with `Metadata["cache_hit"] = "true"`.
- Expired entries are lazily removed on access; next evaluation recomputes and stores anew.
- `InvalidateAll()` clears the entire cache (recommended after policy reload or revocation-impacting changes).

#### Example
```go
req := authz.Request{Subject:"alice", Resource:"vault", Action:"read", Context: map[string]string{"roles":"admin","scopes":"vault:read"}}
dec1, _ := ma.Authorize(ctx, req) // cache_hit=false
dec2, _ := ma.Authorize(ctx, req) // cache_hit=true (served from cache)
ma.InvalidateAll()
dec3, _ := ma.Authorize(ctx, req) // cache_hit=false (cache cleared)
```

#### Limitations & Future Work
- No size limit / eviction policy (risk of unbounded growth in high-cardinality usage).
- No negative caching differentiation (deny vs default deny treated uniformly).
- No proactive invalidation hooks (manual `InvalidateAll()` or future fine-grained invalidation on policy change).
- Currently single-node only (not distributed / no coherence protocol).

#### Planned Enhancements (Caching)
1. LRU / LFU eviction with configurable max entries.
2. Fine-grained invalidation (per subject/policy) when persistence triggers change events.
3. Metrics: hit ratio, entry count, eviction count, latency impact.
4. Optional cryptographic integrity tag for cached decisions (defense-in-depth).
5. Adaptive TTL (shorter for deny decisions, longer for stable allow decisions).

### Policy Combining Algorithms (New)
### Event-Driven Policy Reload (fsnotify) (New)
`PersistentAuthorizer` can watch a JSON policy file and trigger immediate reload on write/create/rename/remove events using `fsnotify`.

#### Enable Watch
```go
store, _ := authz.NewFilePolicyStore("policies.json")
pa, _ := authz.NewPersistentAuthorizer(store, 5*time.Second) // long poll interval (fallback)
_ = pa.StartWatch() // start fsnotify watcher
// optional: also call pa.Start() to keep polling as fallback if watcher fails
```

#### Behavior
- On Write/Create/Rename: reload file, update in-memory slice, invalidate decision cache.
- On Remove: treat as empty policy set (default deny).
- On watcher error: `watchErr` set and loop exits; polling (if started) continues.

#### Testing
Automated test `TestFsnotifyWatchReload` modifies the file and asserts decision change from allow to deny without manual reload call.

#### Limitations
- Single file path only (no directory recursive watch).
- No batching/debounce; rapid consecutive writes may cause multiple reloads.
- Does not emit structured events for external observers yet.

#### Planned Enhancements (Watch)
1. Debounce / coalesce rapid changes.
2. Directory-level watch with pattern filtering.
3. Event bus integration (emit PolicyChanged events with diff summary).
4. Health metrics: watch error count, reload latency.
5. Fallback auto-disable when persistent errors detected.

### Metrics & Observability (In Progress)
Lightweight in-process counters and latency tracking added to `MemoryAuthorizer` for rapid introspection without external dependencies.

#### Collected Metrics
- decisions (total authorization evaluations)
- cache_hits / cache_misses
- reloads (incremented by persistence wrapper on successful policy reload)
- avg_latency_ns (Welford mean)
- p99_latency_ns (approximate assuming normal distribution; non-critical best-effort)
- conflicts (decisions where allow and deny policies matched simultaneously)

#### Access Snapshot
```go
ma := authz.NewMemoryAuthorizer()
// ... configure policies, caching, combining ...
snap := ma.GetMetricsSnapshot()
fmt.Printf("decisions=%d hits=%d misses=%d p99=%.0fns\n", snap.Decisions, snap.CacheHits, snap.CacheMisses, snap.P99LatencyNs)
```

#### Notes
- Latency measured wall-clock per Authorize call; cache hits included; all decision paths (including early combining strategy exits) now recorded.
- Welford algorithm used (constant space) for mean / variance; race-tolerant.
- P99 is heuristic; for production replace with HDR histogram or Prometheus summary.
- `Reloads` counter increment deferred to `PersistentAuthorizer` integration (TODO).

#### Future Enhancements (Metrics)
1. Export Prometheus metrics or OpenTelemetry.
2. Add histogram buckets for latency (p50, p90, p99 accurate).
3. Separate counters per combining strategy & effect outcome.
 4. Error counters (policy parse failures, watcher errors).
 5. Structured decision trace logging toggle.
 6. Conflict counter (number of decisions with simultaneous allow+deny matches).

Multiple policies can match a single request. `MemoryAuthorizer` supports three combining strategies:

| Strategy | Description | Security Bias | Example Outcome (Allow + Deny match) |
|----------|-------------|---------------|--------------------------------------|
| `deny_overrides` | Any deny wins immediately | Conservative | Deny |
| `permit_overrides` | Any allow wins immediately | Permissive | Allow |
| `first_applicable` | First matching policy decides (order dependent) | Neutral / Legacy | Depends on order |

#### Usage
```go
ma := authz.NewMemoryAuthorizer()
ma.SetCombiningStrategy(authz.DenyOverrides) // default
// Add policies...
dec, _ := ma.Authorize(ctx, req)
```

#### Guidance
- Prefer `deny_overrides` for high assurance environments (default).
- Use `permit_overrides` only when explicit allow policies must trump broad catch-all denies; audit carefully for gaps.
- Avoid `first_applicable` unless migrating legacy ordered rule sets; ordering errors can create unintended privilege.

Conflict Diagnostics: When both at least one allow and one deny policy match the request, metadata key `policy_conflict` is populated with a comma-separated list of involved policy IDs. This occurs regardless of final outcome, enabling downstream audit or alerting.

### Policy Reload Diff (New)
Each successful reload computes added and removed policy IDs relative to the previous snapshot.

Access via `PersistentAuthorizer.LastDiff()`:
```go
added, removed := pa.LastDiff()
fmt.Println("added:", added, "removed:", removed)
```
Notes:
- Initial load treats all policies as added.
- Diff lists are ephemeral (only last reload); persist externally if historical tracking required.
- Future enhancement: emit structured events with full prior/next metadata union for audit.

#### Future Enhancements
1. Add `ordered` strategy with explicit priority weights.
2. Conflict diagnostics: emit event when both allow and deny matched (with listing of policy IDs).
3. Policy grouping by namespace with per-group combining semantics.
4. Metrics: counts of matches per strategy, conflict occurrences.


## Test Coverage Summary (New)
| Area | File | Purpose |
|------|------|---------|
| Scopes Round Trip | `test/auth/jwt_scope_test.go` | Ensures token scopes encode/decode correctly |
| Revocation Chain | `pkg/delegation/revocation_chain_test.go` | Hash linkage integrity & tamper detection |
| AAP-001 Revocation Integration | `pkg/rfc0111/rfc0111_revocation_integration_test.go` | Service-level revocation enforcement |
| Roles & Required Scopes | `pkg/authz/authz_enhanced_test.go` | Policy match via roles and scope set |
| ABAC Operators | `pkg/authz/authz_enhanced_test.go` | Advanced condition operator semantics |

---

## Roadmap Next Steps
1. Composite example combining: JWT scopes + roles + revocation enforcement + ABAC conditions.
2. Policy persistence (file/DB) + hot reload.
3. Decision caching layer with configurable TTL and invalidation on policy/revocation change.
4. Extend condition operators (regex, numeric comparison, time windows).
5. Introduce policy combining strategies & conflict resolution diagnostics.
6. Migrate POA issuance to hash-linked Delegation Chain (for creation events) and unify with RevocationChain in integrity verification.
7. Structured metrics: latency histogram, match counts, cache hit rate.

---

## Regex Caching (Enhanced)

The `regex` operator uses a compiled pattern cache to eliminate redundant compilations. Recent enhancements add lifecycle management and observability:

### Features
- On-demand compilation: first evaluation of a pattern compiles and caches it.
- LRU capacity limit (default 256). Configure via `SetRegexCacheCapacity(cap int)` (<=0 disables eviction).
- Optional TTL expiry for cached patterns via `SetRegexCacheTTL(ttl time.Duration)` (<=0 disables TTL).
- Eviction policy: first prune expired (TTL) entries, then evict least recently accessed patterns until under capacity.
- Pre-validation: `PersistentAuthorizer.reload()` compiles regex patterns during policy load to surface invalid patterns early (counts towards metrics) without storing them in the runtime cache until first match evaluation.
- Match frequency: total successful regex matches tracked globally (`RegexMatches`) plus per-pattern internal counters (internal map, not yet exported individually).

### Metrics (Prometheus / Snapshot)
| Metric | Description |
|--------|-------------|
| `authz_regex_compiles_total` | Successful unique pattern compilations (includes pre-validation). |
| `authz_regex_compile_errors_total` | Failed compilations (invalid patterns). |
| `authz_regex_cache_size` | Current number of compiled patterns held. |
| `authz_regex_evictions_total` | Count of evictions (TTL expiry or capacity LRU). |
| `authz_regex_matches_total` | Total successful regex condition matches across decisions. |

Latency and decision-level metrics coexist; see below.

### Behavior Summary
1. Pre-validation phase (reload): compile patterns; increment success/error counters; do not populate cache.
2. First runtime evaluation: if not cached, compile again (double-count avoided only if pattern compiled during pre-validation—both phases legitimately count since they represent distinct compile events). Cached insertion sets timestamps for LRU & TTL.
3. Subsequent matches: pattern reused; last-access timestamp updated; per-pattern match count incremented.
4. Eviction trigger: occurs after a new compile (post-insert call to `pruneRegexCache`) or explicit prune calls; TTL-expired entries removed first, then LRU if over capacity.

### Safety
- Invalid patterns never cached; treated as non-match (fail closed) while incrementing error counter.
- All map updates guarded by `regexMu`; reads use RLock where safe.

### Tuning Guidance
| Scenario | Recommendation |
|----------|---------------|
| Few stable patterns | Keep default capacity 256; TTL off. |
| Burst of many ephemeral patterns | Lower capacity (e.g. 64) + short TTL (1–5m) to limit memory. |
| Memory pressure concerns | Monitor `regex_cache_size` alongside RSS; adjust capacity downward. |
| High invalid pattern rate | Investigate policy authoring pipeline; integrate lint to prevent bad deployments. |

### Future Extensions
- Export top-N pattern match counts.
- Adaptive capacity based on eviction rate (auto-scale up/down).
- Pattern risk classification (length/complexity thresholds) for observability.

## Decision Latency Histogram (New)

A lightweight histogram tracks decision latency distribution using fixed nanosecond bucket upper bounds:

Buckets (ns): `50µs, 100µs, 250µs, 500µs, 1ms, 2.5ms, 5ms, 10ms, 25ms, 50ms, 100ms`.

Prometheus output format:
```
authz_latency_bucket{le="500000"} 42
```
Each counter represents decisions with latency ≤ bucket upper bound. (Implicit +Inf bucket can be derived by total decisions minus highest bucket count.)

Snapshot field `LatencyHistogram` maps bucket upper bounds → counts (only populated buckets to reduce JSON size).

Use Cases:
- Detect latency regressions (e.g., spike in >10ms decisions).
- Guide performance tuning and cache strategy adjustments.
- Establish SLOs (e.g. 99% ≤ 5ms) using cumulative buckets.

### Interpreting p99 vs Histogram
- `P99LatencyNs` is an approximate statistic assuming normal distribution; may diverge under skewed workloads.
- Histogram provides an empirical distribution; prefer it for production dashboards.

### Future Enhancements
- Replace static buckets with dynamic HDR histogram or Prometheus Summary.
- Separate buckets for cache hits vs misses.
- Outcome-specific latency (allow vs deny vs default deny).

## Configuration Snippets
```go
ma := authz.NewMemoryAuthorizer()
ma.SetRegexCacheCapacity(128)
ma.SetRegexCacheTTL(10 * time.Minute)
ma.EnableCaching(500 * time.Millisecond) // decision cache TTL
```

## Observability Quick Reference
| Concern | Metric / Signal | Action |
|---------|-----------------|--------|
| High compile error count | `authz_regex_compile_errors_total` | Validate patterns; block bad policy deploy. |
| Frequent evictions | `authz_regex_evictions_total` rising with stable workload | Increase capacity or TTL; inspect pattern churn. |
| Latency spikes | Histogram buckets >10ms increasing | Profile evaluation path; check I/O or large policy set. |
| Low cache hit ratio | Compare `cache_hits` vs `cache_misses` | Increase decision TTL or refine request key normalization. |

---

## Detached Signature (Public Verifiable Token Integrity) (New)

### Rationale
Previously token integrity outside the issuing system depended on either (a) embedded POA content plus internal validation or (b) local symmetric constructs. To enable third‑party / offline verification and cryptographic non-repudiation, EnvelopeV2 now supports a detached Ed25519 signature over the canonical POA JSON bytes. This allows lightweight envelopes while preserving a stable canonical digest binding, and enables future algorithm agility (ECDSA, BLS aggregate, Ed25519 batch) without altering the embedded POA layout.

### EnvelopeV2 Additions
| Field | JSON Key | Type | Description |
|-------|----------|------|-------------|
| DetachedSignature | `detached_sig` | base64url string | Ed25519 signature over canonical POA bytes (exact bytes used for `CanonicalPOADigest`). |
| DetachedSignatureAlgorithm | `detached_sig_alg` | string | Algorithm identifier (currently `ed25519`). Reserved for future multi‑alg negotiation. |
| DetachedSignatureKeyID | `detached_sig_kid` | string | Key identifier (active signing key ID) facilitating key lookup / rotation verification. |

### Feature Flags & Preconditions
| Flag | Purpose | Default |
|------|---------|---------|
| `GAUTH_POA_ENVELOPE_V2=1` | Enables V2 issuance/verification path | Off (0) |
| `GAUTH_DETACHED_SIGNATURE=1` | Adds detached signature during issuance & enforces verification if present | Off (0) |
| `GAUTH_EMBED_FULL_POA=1` | (Optional) Embeds full canonical PoA JSON (`RawPOA`) | Off (0) |

Detached signature issuance only occurs when BOTH V2 and `GAUTH_DETACHED_SIGNATURE` are enabled. Verification is adaptive: if the fields are present they are validated; absence when the flag is off does not fail verification (forward compatibility window).

### Issuance Flow (Simplified)
1. Construct canonical POA JSON and marshal to bytes (pre-sorted / normalized).
2. Compute digest (existing `CanonicalPOADigest`).
3. If detached signature enabled:
    - Sign canonical bytes with active Ed25519 private key (domain separation constants already applied at higher layer where applicable).
    - Populate `detached_sig`, `detached_sig_alg`, `detached_sig_kid`.
4. Populate multi-signature satisfaction metadata (if any) & envelope timing fields.
5. Return EnvelopeV2 (optionally embed `RawPOA`).

### Verification Flow (Adaptive)
1. Parse envelope; detect V1 vs V2 (existing logic).
2. Recompute canonical digest from provided (or externally retrieved) PoA definition.
3. Compare recomputed digest with `CanonicalDigest`; mismatch => failure (`digest_mismatch` counter increment).
4. If detached signature fields present:
    - Resolve public key by `detached_sig_kid` (active + previous + external provider).
    - Verify Ed25519 signature over the canonical bytes.
    - On success, mark `DetachedSignatureValid=true`; on failure, increment invalid signature counter (reuses existing signature failure buckets) and return error.
5. Continue multi-signature threshold & weight verification (if applicable) and temporal / revocation checks.

### Failure Modes
| Condition | Result | Metrics Impact |
|-----------|--------|----------------|
| Digest recomputation mismatch | Hard fail | `digest_mismatch` counter |
| Missing public key for `kid` | Hard fail | `pubkey_missing` (existing bucket) |
| Signature invalid | Hard fail | `invalid_signature` bucket |
| Absent detached fields (flag disabled) | Accepted | N/A (legacy / rollout window) |

### Observability
Current implementation reuses existing signature verification latency histogram & failure counters. A follow-up may split counters into embedded vs detached forms (proposed: `detached_signature_verifications_total` with `outcome` label). Adoption can be inferred temporarily via envelope issuance counters plus presence frequency sampling.

### Backward Compatibility & Migration
| Phase | Behavior |
|-------|----------|
| 0 – Disabled | No detached fields emitted; verifiers ignore absence. |
| 1 – Opt‑In (current) | Selected environments enable flag; verifiers accept both forms. |
| 2 – Dual Metrics | Track adoption %; add explicit detached verification counters. |
| 3 – Enforce | New flag (`GAUTH_REQUIRE_DETACHED_SIGNATURE=1` planned) rejects envelopes lacking detached signature once adoption ≥ threshold. |
| 4 – Algorithm Agility | Introduce alternate algorithms (ecdsa-p256, ed25519-batch, bls12-381) with negotiation. |

### Test Coverage
File: `pkg/rfc0111/rfc0111_detached_signature_test.go`
| Test | Purpose |
|------|---------|
| `TestDetachedSignatureIssuanceAndVerification` | Happy path: issuance + verify sets `DetachedSignatureValid` |
| `TestDetachedSignatureTamper` | Digest tamper triggers failure (integrity invariant) |
| `TestDetachedSignatureDisabled` | Ensures absence when flag off (no accidental emission) |

### Security Notes
- Detached signature binds only canonical PoA JSON; envelope fields outside canonical digest scope (timestamps, satisfaction metadata) are protected indirectly by digest + multi-signature semantics (future consideration: extend domain separation to include selected envelope headers).
- Key rotation safety: `kid` ensures verifier can locate previous key; rotation tests should include detached signature backward validation.
- Replay protection unchanged; detached signature does not alter JTI / store semantics.

### Future Enhancements
1. Algorithm negotiation & multi-alg support (JWT / COSE style `crit` or alg set negotiation).
2. Aggregated / batch verification for multi-signature detached schemes.
3. Property / fuzz tests targeting canonicalization stability under randomized PoA permutations.
4. Cross-language test vector package (Go → JSON fixtures) for ecosystem verifiers.
5. Prometheus counters labeled by `signature_mode` (embedded|detached) & `alg`.
6. Optional CBOR canonical form for RawPOA (size reduction) with dual-hash bridging period.

---

## KMS Metrics (Prometheus) (New)

The KMS abstraction now optionally emits Prometheus metrics for core operations. Enable via environment variable:

```bash
export GAUTH_KMS_METRICS=1
```

Must be set before KMS construction (e.g. service start) so registration occurs once. The mock implementation invokes `maybeEnableKMSMetrics()` inside `NewMockKMS()`.

### Exposed Metrics
| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `kms_active_signer_requests_total` | counter | provider | Count of ActiveSigner retrievals |
| `kms_rotate_total` | counter | provider | Successful key rotations |
| `kms_list_keys_total` | counter | provider | ListKeys operations invoked |
| `kms_operation_latency_seconds` | histogram | provider, operation | Latency per operation (active_signer, public_key, rotate, list_keys) |

Histogram buckets use Prometheus defaults (no custom buckets specified yet). Each operation wraps its core logic with `recordKMSMetric(provider, op, fn)` to record latency and increment counters. Provider label for mock is `mock`; real adapters should use backend identifier (e.g. `vault`, `aws`, `gcp`).

### Sample Scrape Output (Abbreviated)
```
# HELP kms_rotate_total Successful key rotations
# TYPE kms_rotate_total counter
kms_rotate_total{provider="mock"} 3
# HELP kms_operation_latency_seconds Latency of KMS operations
# TYPE kms_operation_latency_seconds histogram
kms_operation_latency_seconds_bucket{provider="mock",operation="rotate",le="0.005"} 2
kms_operation_latency_seconds_bucket{provider="mock",operation="rotate",le="0.01"} 3
...
```

### Integration Notes
1. Metrics registration is idempotent; repeated enable calls are ignored after first success.
2. Disabled by default to avoid overhead in minimal / test runs.
3. Public key fetches (`PublicKey`) use operation label `public_key` even on cache hits (future optimization: separate cache/miss labels).
4. Rotate latency includes ED25519 key generation time.

### Planned Enhancements
| Area | Plan |
|------|------|
| Latency buckets | Define explicit buckets (e.g. 1ms..500ms) tuned to real KMS backends |
| Error metrics | Counters for failures (rotate errors, fetch misses) |
| Cache labeling | Distinguish cold vs warm `public_key` retrievals |
| Key age gauge | Track age of active key for rotation SLO alerts |
| Operation summary | Add Prometheus summary for high-resolution quantiles |

### Usage Example
After enabling, hit service `/metrics` endpoint:
```
curl -s http://localhost:8080/metrics | grep kms_rotate_total
```
Returns current rotation count for the provider.

---

## Secret Storage Provider Abstraction (New)

Status: Partial (memory + Vault stub) — sec8.item1

Introduces a pluggable secret storage interface decoupling higher‑level components (issuance, external key adapters, future capability enforcement) from the underlying secret persistence. A minimal memory implementation and a `vaultstub` (placeholder) are provided to enable early integration and tests while deferring real Vault / HSM wiring.

### Interface
`pkg/secret/provider.go`
```
type Provider interface {
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key, value string, opts ...Option) error
    Delete(ctx context.Context, key string) error
    List(ctx context.Context, prefix string) ([]string, error)
    Name() string
}
```
Options (extensible):
- `WithTTL(seconds)` (hint – ignored by memory)  
- `IfNotExists()` (create semantics)

### Implementations (Current)
| Name | Persistence | Intended Use | Notes |
|------|-------------|--------------|-------|
| memory | in‑process map | tests, dev, default fallback | NOT durable; no encryption |
| vaultstub | in‑process map (namespaced) | smoke path for future Vault adapter | Same semantics as memory; distinct Name() for metrics separation |

### Usage Example
```
sp := secret.NewMemory()
_ = sp.Set(ctx, "oauth/client_secret", "s3cr3t", secret.IfNotExists())
val, _ := sp.Get(ctx, "oauth/client_secret")
keys, _ := sp.List(ctx, "oauth/")
_ = sp.Delete(ctx, "oauth/client_secret")
```

### Roadmap to Implemented
- Real provider: Vault (v1 KV) adapter with token renewal & namespace support.
- Optional envelope encryption (KMS‑backed DEK) for at‑rest confidentiality even if provider returns plaintext.
- Metrics: `secret_ops_total{provider,op}` + latency histogram, error counters, cache hit ratio (if local cache added).
- Audit trail: append-only JSONL or ledger chain of secret material changes (metatdata only, not raw values).
- Policy gating: capability matrix enforcement of which components may access which secret prefixes.

### Security Considerations
- Memory provider never suitable for production sensitive material (documented).  
- Stub is a drop‑in: upgrading to real Vault requires no call‑site changes.

---

## Policy Versioning & Rollback (New)

Status: Partial — sec2.item4

Adds lightweight in‑memory version snapshots to `MemoryAuthorizer` enabling rollback during development and providing a scaffold for future persistent, auditable policy registries.

### Additions
| Concept | API | Description |
|---------|-----|-------------|
| Working Version | `ma.version` | Monotonically increasing policy set version counter. |
| Snapshot | `Snapshot() int64` | Captures a copy of current policies, stores in `ma.versions`, returns version number. |
| Version Listing | `ListVersions() []int64` | Returns stored snapshot version numbers. |
| Rollback | `Rollback(v int64)` | Replaces current working policy slice with snapshot `v`; sets next working version to `v+1`. |
| Policy Tagging | `Policy.Version` | Each policy stamped with version when added. |

### Example
```
ma := authz.NewMemoryAuthorizer()
ma.AddPolicy(authz.Policy{ID:"p1", Subject:"alice", Resource:"doc", Actions:[]string{"read"}, Effect:authz.Allow})
v1 := ma.Snapshot()
ma.AddPolicy(authz.Policy{ID:"p2", Subject:"alice", Resource:"doc", Actions:[]string{"write"}, Effect:authz.Allow})
_ = ma.Snapshot() // v2
_ = ma.Rollback(v1) // policies now only contain p1 again
```

### Current Limitations
- In‑memory only (lost on restart); no persistence or hash chain integrity.
- No diff inspection API (callers must compute manual set difference pre/post).
- No audit log of rollbacks (future: append event with actor + rationale).

### Path to Implemented
1. Persistent store (e.g., BoltDB / Postgres) maintaining bundle chain (hash(prev_hash||policies_json)).
2. Rollback event recorded & signed (integration with upcoming immutable audit ledger).  
3. CLI / API endpoints: `GET /policies/versions`, `POST /policies/rollback/{v}`.  
4. Metrics: `policy_snapshot_total`, `policy_rollback_total`, `policy_active_version`.  
5. Conformance tests: rollback determinism, chain continuity, tamper detection (altered historical snapshot hash failure).  

### Risks & Mitigations
| Risk | Mitigation |
|------|------------|
| Silent rollback causing unexpected permission loss | Emit structured audit + increment rollback metric; optionally require confirmation token. |
| Snapshot storms (too frequent) | Rate limit or debounce snapshots; expose advisory metric. |

---

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
