# Cyclomatic Complexity Refactoring Guide

## Current Status
- **Total gocyclo warnings**: 13
- **Most complex function**: `routes()` - complexity 392
- **Combined complexity**: ~1000+

## Refactoring Strategy

### Priority 1: Quick Wins (Routes Function - Complexity 392)

**File**: `web/server_clean.go:5479`
**Current**: 2100 lines, 129 route registrations, complexity 392
**Target**: Complexity < 25 per function

**Solution Implemented**:
Created `web/routes_helpers.go` with domain-specific route registration functions:

```go
func (s *BetaServer) routes() {
    s.registerBetaRoutes()        // Health, examples
    s.registerTokenRoutes()        // Token operations
    s.registerDelegationRoutes()   // Delegation management
    s.registerRevocationRoutes()   // Revocation handling
    s.registerPolicyRoutes()       // Policy management
    s.registerMetricsRoutes()      // Metrics & monitoring
    s.registerCapabilityRoutes()   // Capability management
    s.registerAuditRoutes()        // Audit logging
    s.registerMonitoringRoutes()   // Observability
    s.registerUIRoutes()           // UI endpoints
    s.registerDocsRoutes()         // Documentation
}
```

**Impact**: Reduces complexity from 392 to ~11 per function
**Effort**: Low (mostly mechanical refactoring)
**Risk**: Low (no logic changes, only organization)

---

### Priority 2: Validation Pipeline (ValidateDelegationCtx - Complexity 73)

**File**: `pkg/rfc0111/rfc0111.go:2315`
**Current**: 224 lines, complexity 73
**Target**: Complexity < 10 per function

**Refactoring Pattern**: Validation Pipeline

```go
type validationStage interface {
    validate(ctx context.Context, poa *POA) error
}

func (s *Service) ValidateDelegationCtx(ctx context.Context, poaID, grantee, action string) error {
    poa, err := s.loadAndVerifyPOA(poaID)
    if err != nil {
        return err
    }
    
    pipeline := []validationStage{
        &hierarchicalDigestValidator{service: s},
        &signatureValidator{service: s},
        &statusValidator{service: s},
        &revocationValidator{service: s},
        &expirationValidator{service: s},
        &scopeValidator{service: s, grantee: grantee, action: action},
        &chainValidator{service: s},
    }
    
    for _, stage := range pipeline {
        if err := stage.validate(ctx, poa); err != nil {
            return err
        }
    }
    
    return nil
}

// Each validator is 10-20 lines
type hierarchicalDigestValidator struct{ service *Service }

func (v *hierarchicalDigestValidator) validate(ctx context.Context, poa *POA) error {
    if poa.Version < 4 || poa.ParentPOAID == "" {
        return nil // Not applicable
    }
    
    parent, err := v.service.getParent(poa.ParentPOAID)
    if err != nil {
        return err
    }
    
    return v.service.verifyParentDigest(poa, parent)
}

type signatureValidator struct{ service *Service }

func (v *signatureValidator) validate(ctx context.Context, poa *POA) error {
    if poa.Signature == nil {
        return nil // No signature to validate
    }
    
    digest, canonical, err := CanonicalPOADigest(poa)
    if err != nil {
        return err
    }
    
    if err := v.verifyDigestMatch(poa.Signature, digest); err != nil {
        return err
    }
    
    return v.verifySignature(poa.Signature, canonical)
}

// ... 5 more validators, each < 20 lines
```

**Impact**: 73 → ~8 per function (9 functions total)
**Effort**: Medium (requires careful extraction)
**Risk**: Medium (logic reorganization, needs thorough testing)

---

### Priority 3: Token Generation (generateAuthToken - Complexity 57)

**File**: `pkg/rfc0111/rfc0111.go:3773`
**Current**: ~200 lines, complexity 57
**Target**: Complexity < 10

**Refactoring Pattern**: Builder Pattern + Strategy

```go
type tokenBuilder struct {
    service *Service
    req     *DelegationRequest
    token   *Token
}

func generateAuthToken(s *Service, req *DelegationRequest) (*Token, string, error) {
    builder := &tokenBuilder{
        service: s,
        req:     req,
        token:   &Token{},
    }
    
    steps := []func() error{
        builder.generateTokenID,
        builder.setBasicFields,
        builder.applyDelegation,
        builder.calculateExpiration,
        builder.setSubjectInfo,
        builder.addClaims,
        builder.addSignature,
        builder.validateToken,
    }
    
    for _, step := range steps {
        if err := step(); err != nil {
            return nil, "", err
        }
    }
    
    jwt, err := builder.encodeJWT()
    return builder.token, jwt, err
}

func (b *tokenBuilder) generateTokenID() error {
    b.token.ID = generateID()
    return nil
}

func (b *tokenBuilder) setBasicFields() error {
    b.token.Issuer = b.service.issuer
    b.token.IssuedAt = b.service.nowFn()
    return nil
}

// ... 6 more steps, each < 15 lines
```

**Impact**: 57 → ~7 per function (8 functions)
**Effort**: Medium 
**Risk**: Medium (token generation is security-critical)

---

### Priority 4: API Handlers (Complexity 47-54)

**Files**:
- `web/server_clean.go:10064` - `apiTokenValidate` (54)
- `web/server_clean.go:10464` - `apiDelegationStatusUpdate` (52)
- `web/server_clean.go:9302` - `apiAuthorizePOA` (47)
- `web/server_clean.go:10696` - `apiLifecycleTimeline` (46)

**Common Pattern**: Extract Request Processing Pipeline

```go
func (s *BetaServer) apiTokenValidate(c *gin.Context) {
    // Parse request
    req, err := s.parseTokenValidateRequest(c)
    if err != nil {
        s.respondError(c, 400, err)
        return
    }
    
    // Validate token
    result, err := s.validateToken(c.Request.Context(), req)
    if err != nil {
        s.handleValidationError(c, err)
        return
    }
    
    // Audit
    s.auditValidation(c.Request.Context(), req, result)
    
    // Respond
    c.JSON(200, result)
}

func (s *BetaServer) parseTokenValidateRequest(c *gin.Context) (*ValidateRequest, error) {
    var req ValidateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }
    
    if err := s.validateRequest(&req); err != nil {
        return nil, err
    }
    
    return &req, nil
}

func (s *BetaServer) validateToken(ctx context.Context, req *ValidateRequest) (*ValidationResult, error) {
    // Break into sub-steps
    token, err := s.decodeToken(req.Token)
    if err != nil {
        return nil, err
    }
    
    if err := s.checkExpiration(token); err != nil {
        return nil, err
    }
    
    if err := s.checkRevocation(token); err != nil {
        return nil, err
    }
    
    if err := s.verifySignature(token); err != nil {
        return nil, err
    }
    
    return &ValidationResult{Valid: true, Token: token}, nil
}
```

**Impact**: 47-54 → ~8-10 per function
**Effort**: Low-Medium (straightforward extraction)
**Risk**: Low (request/response handling, well-tested)

---

### Priority 5: Remaining Functions (Complexity 27-39)

**Files**:
- `pkg/rfc0111/rfc0111.go:2740` - `ValidateDelegationRich` (58)
- `pkg/rfc0111/rfc0111.go:3562` - `validateDelegationRequest` (39)
- `web/server_clean.go:11050` - `initUIRevamp` (29)
- `web/server_clean.go:2020` - `maybeAugmentAndSignAttestation` (28)
- `pkg/rfc0111/rfc0111.go:2607` - `ApproveRevocation` (27)
- `pkg/replay/durable_replay_store.go:341` - `applyEviction` (27)

**Strategy**: Apply same patterns
- Validation functions → Pipeline pattern
- Init functions → Builder pattern
- Processing functions → Strategy pattern
- Eviction functions → Table-driven approach

---

## Implementation Plan

### Phase 1: Low-Hanging Fruit (1-2 days)
1. ✅ Extract route registration helpers (`routes_helpers.go`) - **DONE**
2. Refactor API handlers (4 functions, ~300 lines each)
3. Extract init/setup functions

**Expected reduction**: 5 warnings

### Phase 2: Core Validation Logic (3-5 days)
1. Refactor `ValidateDelegationCtx` (complexity 73)
2. Refactor `ValidateDelegationRich` (complexity 58)
3. Refactor `validateDelegationRequest` (complexity 39)

**Expected reduction**: 3 warnings

### Phase 3: Token Generation (2-3 days)
1. Refactor `generateAuthToken` (complexity 57)
2. Add comprehensive tests

**Expected reduction**: 1 warning

### Phase 4: Remaining Functions (2-3 days)
1. Refactor remaining 4 functions
2. Comprehensive testing
3. Performance validation

**Expected reduction**: 4 warnings

---

## Testing Strategy

For each refactoring:

1. **Before**: Run existing tests, capture coverage
2. **During**: Write unit tests for extracted functions
3. **After**: 
   - Verify all existing tests pass
   - Verify coverage maintained or improved
   - Run integration tests
   - Performance benchmarks

---

## Rollout Strategy

1. **Create feature branch**: `refactor/reduce-complexity`
2. **One PR per priority group** (4 PRs total)
3. **Each PR includes**:
   - Refactored code
   - New unit tests
   - Updated documentation
   - Complexity metrics (before/after)
4. **Review criteria**:
   - All tests pass
   - Coverage >= baseline
   - Complexity reduction verified
   - No performance regression

---

## Success Metrics

**Current state**: 84 warnings (13 gocyclo)
**Target state**: 71 warnings (0 gocyclo)

**By function**:
| Function | Current | Target | Reduction |
|----------|---------|--------|-----------|
| routes() | 392 | 11 | 381 (97%) |
| ValidateDelegationCtx | 73 | 8 | 65 (89%) |
| ValidateDelegationRich | 58 | 8 | 50 (86%) |
| generateAuthToken | 57 | 7 | 50 (88%) |
| apiTokenValidate | 54 | 8 | 46 (85%) |
| Others (8 functions) | 27-52 | <10 | ~200 total |

**Total complexity reduction**: ~800 points (80%+ improvement)

---

## Benefits

1. **Maintainability**: Smaller functions are easier to understand
2. **Testability**: Extracted functions can be unit tested independently  
3. **Debugging**: Easier to isolate and fix issues
4. **Performance**: No impact (pure refactoring, no algorithmic changes)
5. **Code Quality**: Passes all linters with flying colors
6. **Team Velocity**: Faster onboarding, faster feature development

---

## Risks & Mitigation

| Risk | Mitigation |
|------|------------|
| Introduce bugs during refactoring | Comprehensive test suite, careful review |
| Performance regression | Benchmarks before/after |
| Breaking changes | Keep public APIs unchanged |
| Incomplete refactoring | Incremental PRs, each fully functional |

---

## Next Steps

1. **Review this guide** with the team
2. **Prioritize** which functions to tackle first
3. **Create tracking issues** for each refactoring
4. **Set timeline** based on team capacity
5. **Begin Phase 1** with route helpers (already started!)

---

## Additional Resources

- [Refactoring: Improving the Design of Existing Code](https://martinfowler.com/books/refactoring.html)
- [Working Effectively with Legacy Code](https://www.goodreads.com/book/show/44919.Working_Effectively_with_Legacy_Code)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go)

---

**Status**: Phase 1 started - routes_helpers.go created ✅
**Next**: API handler refactoring (Priority 4)
