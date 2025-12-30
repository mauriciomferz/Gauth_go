# Next Steps: Type Alignment and Build Fix

**Date**: November 11, 2025
**Priority**: HIGH
**Estimated Time**: 4-6 hours
**Blocking**: Tasks 4-8 (integration tests, monitoring, performance)

---

## Current Situation

### ✅ What's Working
- Core implementation: 5,516 lines, 38/38 tests passing
- Disclosure API structure: 588 lines designed and implemented
- External PVP/PIP clients: 397 lines, production-ready
- Circuit breaker pattern, retry logic, caching - all complete

### ⚠️ What Needs Fixing
- `pkg/agentauth/disclosure_service.go` - Type mismatches preventing compilation
- Build fails due to interface/struct field incompatibilities
- ~4-7 critical type alignment issues

---

## Problem Analysis

### Issue 1: ExtendedTokenStore Interface Methods
**Current Interface** (from `extended_token_store.go`):
```go
type ExtendedTokenStore interface {
    SaveToken(ctx context.Context, token *ExtendedToken) error
    GetToken(ctx context.Context, accessToken string) (*ExtendedToken, error)
    RevokeToken(ctx context.Context, accessToken string) error
    IsRevoked(ctx context.Context, accessToken string) (bool, error)
}
```

**What disclosure_service.go Needs**:
```go
// Missing methods:
ListTokensByResourceOwner(ctx context.Context, ownerID string) ([]*ExtendedToken, error)
GetExtendedToken(ctx context.Context, id string) (*ExtendedToken, error)  // Or use GetToken()?

// Wrong signature:
RevokeToken(ctx, accessToken, reason string) error  // Has 3 params, interface has 2
```

### Issue 2: Struct Field Mismatches
**ResourceOwnerInfo** (from `extended_token.go`):
```go
type ResourceOwnerInfo struct {
    OwnerID   string  // NOT "ID"
    OwnerName string
    OwnerType string
}
```

**disclosure_service.go uses**:
```go
token.ResourceOwner.ID  // WRONG - should be OwnerID
```

### Issue 3: AuditEntry Structure
**Actual AuditEntry** (from `extended_token.go`):
```go
type AuditEntry struct {
    Timestamp time.Time
    Action    string
    Actor     string  // NOT "SubjectID"
    Result    string  // NOT "SubjectType"
    Details   map[string]interface{}
    // NO "ObjectID" or "ObjectType" fields
}
```

**disclosure_service.go creates**:
```go
AuditEntry{
    SubjectID:   "...",  // WRONG
    SubjectType: "...",  // WRONG
    ObjectID:    "...",  // WRONG
    ObjectType:  "...",  // WRONG
}
```

### Issue 4: Missing Struct Fields
**ClientOwnerInfo** - Unknown structure, but disclosure_service.go uses:
```go
token.IssuedBy.ClientOwner.Name      // Unknown if exists
token.IssuedBy.ClientOwner.EntityID  // Unknown if exists
```

**OwnersAuthorizerInfo** - Unknown structure, but uses:
```go
token.IssuedBy.OwnersAuthorizer.Name  // Unknown if exists
```

### Issue 5: ExtendedToken Status Field
```go
token.Status  // DOESN'T EXIST on ExtendedToken type
```

Need to derive status from:
- `IsRevoked()` check
- `ExpiresAt` comparison with current time
- Other token state

---

## Resolution Strategy

### Option 1: Proper Fix (RECOMMENDED) - 4-6 hours

#### Step 1: Analyze Existing Types (30 min)
```bash
# Read complete type definitions
grep -A 20 "type ClientOwnerInfo" pkg/agentauth/*.go
grep -A 20 "type OwnersAuthorizerInfo" pkg/agentauth/*.go
grep -A 30 "type ExtendedToken struct" pkg/agentauth/*.go
```

**Actions**:
1. Document actual field names for `ClientOwnerInfo`
2. Document actual field names for `OwnersAuthorizerInfo`
3. Document all `ExtendedToken` fields
4. Create type mapping document

#### Step 2: Extend ExtendedTokenStore Interface (1 hour)
**File**: `pkg/agentauth/extended_token_store.go`

Add methods:
```go
type ExtendedTokenStore interface {
    // Existing methods
    SaveToken(ctx context.Context, token *ExtendedToken) error
    GetToken(ctx context.Context, accessToken string) (*ExtendedToken, error)
    RevokeToken(ctx context.Context, accessToken string) error
    IsRevoked(ctx context.Context, accessToken string) (bool, error)

    // NEW methods for disclosure
    ListTokensByResourceOwner(ctx context.Context, ownerID string) ([]*ExtendedToken, error)
    RevokeTokenWithReason(ctx context.Context, accessToken string, reason string) error
}
```

#### Step 3: Implement New Methods (1-2 hours)
**Files**: All implementations of `ExtendedTokenStore`
- `pkg/agentauth/postgres_extended_token_store.go`
- `pkg/agentauth/memory_extended_token_store.go`
- Any other implementations

**Example**:
```go
func (s *PostgresExtendedTokenStore) ListTokensByResourceOwner(ctx context.Context, ownerID string) ([]*ExtendedToken, error) {
    // Query: SELECT * FROM extended_tokens WHERE resource_owner_id = $1 AND revoked = false
    // Return slice of tokens
    return nil, fmt.Errorf("not yet implemented")
}

func (s *PostgresExtendedTokenStore) RevokeTokenWithReason(ctx context.Context, accessToken string, reason string) error {
    // Update token with reason
    // Could store reason in metadata or separate audit table
    return s.RevokeToken(ctx, accessToken)  // For now, delegate to existing method
}
```

#### Step 4: Fix disclosure_service.go (2 hours)
**File**: `pkg/agentauth/disclosure_service.go`

**Fixes**:
1. Change `GetExtendedToken()` → `GetToken()`
2. Change `RevokeToken(ctx, id, reason)` → `RevokeTokenWithReason(ctx, id, reason)`
3. Change `token.ResourceOwner.ID` → `token.ResourceOwner.OwnerID`
4. Fix `AuditEntry` creation to use `Actor`, `Result`, `Details` fields
5. Remove `token.Status` usage, derive from `IsRevoked()` + expiry check
6. Fix `ClientOwner` and `OwnersAuthorizer` field access based on actual structures
7. Remove duplicate `ExtendedTokenStore` interface declaration (already done)

**Example Fix for tokenToSummary()**:
```go
func (s *DisclosureService) tokenToSummary(token *ExtendedToken) (*AuthorizationSummary, error) {
    // Derive status
    isRevoked, _ := s.tokenStore.IsRevoked(context.Background(), token.AccessToken)
    status := "active"
    if isRevoked {
        status = "revoked"
    } else if time.Now().After(token.ExpiresAt) {
        status = "expired"
    }

    // Use correct field names
    summary := &AuthorizationSummary{
        ID:                token.AccessToken,
        ResourceOwnerID:   token.ResourceOwner.OwnerID,  // FIX: was .ID
        ResourceOwnerName: token.ResourceOwner.OwnerName,
        ClientID:          token.ClientID,
        // Use actual field names from ClientOwnerInfo
        ClientName:        getClientName(token),  // Helper function
        GrantedAt:         token.IssuedAt,
        ExpiresAt:         token.ExpiresAt,
        Status:            status,
        // Use actual PowerOfAttorney fields
        Scope:             token.Scope,  // Assuming this exists
    }
    return summary, nil
}

func getClientName(token *ExtendedToken) string {
    // Adapt based on actual ClientOwnerInfo structure
    if token.IssuedBy != nil && token.IssuedBy.ClientOwner != nil {
        // Use whatever field actually exists
        return token.IssuedBy.ClientOwner.Name  // Or .EntityID, or whatever exists
    }
    return "Unknown"
}
```

#### Step 5: Build and Test (30 min)
```bash
# Build
go build -o bin/web-server ./cmd/web-server

# Run existing tests
go test ./pkg/agentauth

# Manual API test
./bin/web-server &
curl http://localhost:8080/api/v1/disclosure/authorizations?owner_id=test_owner
```

#### Step 6: Add Integration Tests (1 hour)
**File**: `pkg/agentauth/disclosure_service_test.go`

```go
func TestDisclosureService_ListActiveAuthorizations(t *testing.T) {
    // Setup: Create mock store with test tokens
    // Execute: Call ListActiveAuthorizations
    // Assert: Verify correct filtering and response
}

func TestDisclosureService_RevokeAuthorization(t *testing.T) {
    // Setup: Create token
    // Execute: Revoke with reason
    // Assert: Token revoked, audit entry created
}
```

---

### Option 2: Stub Implementation - 1 hour

Keep the stub implementation active and document TODOs:

**File**: `pkg/agentauth/disclosure_service_stub.go` (already exists)

**Update handlers** to use stub:
```go
// In web/handlers/disclosure/disclosure_handlers.go
service := agentauth.NewDisclosureServiceStub()  // Use stub instead
```

**Pros**:
- Build succeeds immediately
- Can continue with Tasks 4-8
- Incremental implementation possible

**Cons**:
- Disclosure API returns "not implemented" errors at runtime
- No actual transparency functionality
- Technical debt accumulates

---

### Option 3: Minimal Adapter Pattern - 2-3 hours

Create adapter layer between disclosure service and existing types:

**New File**: `pkg/agentauth/disclosure_adapter.go`

```go
type DisclosureAdapter struct {
    tokenStore ExtendedTokenStore
}

func (a *DisclosureAdapter) ListTokensByOwner(ctx context.Context, ownerID string) ([]*ExtendedToken, error) {
    // Workaround: Query all tokens and filter
    // TODO: Optimize with proper interface method
    return nil, fmt.Errorf("requires ExtendedTokenStore.ListTokensByResourceOwner")
}

func (a *DisclosureAdapter) DeriveTokenStatus(token *ExtendedToken) string {
    isRevoked, _ := a.tokenStore.IsRevoked(context.Background(), token.AccessToken)
    if isRevoked {
        return "revoked"
    }
    if time.Now().After(token.ExpiresAt) {
        return "expired"
    }
    return "active"
}

func (a *DisclosureAdapter) CreateAuditEntry(action, actor string, details map[string]interface{}) AuditEntry {
    return AuditEntry{
        Timestamp: time.Now(),
        Action:    action,
        Actor:     actor,
        Result:    "success",  // Or derive from details
        Details:   details,
    }
}
```

---

## Recommended Approach

**Choose Option 1: Proper Fix**

**Reasoning**:
1. Adds only 4-6 hours vs. technical debt
2. Results in production-quality code
3. Enables full transparency functionality
4. Completes AAP-001 compliance improvements
5. Provides foundation for Tasks 4-8

**Timeline**:
- **Hour 1**: Analyze types, create mapping document
- **Hour 2**: Extend ExtendedTokenStore interface
- **Hours 3-4**: Implement new methods in all stores
- **Hours 5-6**: Fix disclosure_service.go, build, test
- **Optional Hour 7**: Integration tests

---

## Success Criteria

### Build Success
```bash
$ go build -o bin/web-server ./cmd/web-server
# No errors

$ go test ./pkg/agentauth
ok  github.com/.../pkg/agentauth  X.XXXs
```

### API Functional
```bash
$ ./bin/web-server &

$ curl "http://localhost:8080/api/v1/disclosure/authorizations?owner_id=test"
{
  "authorizations": [...],
  "pagination": {...}
}

$ curl -X POST "http://localhost:8080/api/v1/disclosure/authorizations/token_123/revoke"
{
  "success": true,
  "message": "Authorization revoked"
}
```

### Tests Pass
```bash
$ go test ./pkg/agentauth -run TestDisclosure
ok  github.com/.../pkg/agentauth  X.XXXs
```

---

## After Completion

### Immediate Next (Task 4)
**Integration Tests** for:
- Complete disclosure API flow
- External PVP/PIP clients
- Error scenarios
- Concurrent requests

### Then (Tasks 5-8)
- Authorization chain enhancements
- Formal requirements validation
- Monitoring & alerting
- Performance testing

---

## Resources Needed

### Files to Read
1. `pkg/agentauth/extended_token_store.go` - Interface definition
2. `pkg/agentauth/extended_token.go` - Type definitions
3. `pkg/agentauth/postgres_extended_token_store.go` - Implementation example
4. `pkg/agentauth/memory_extended_token_store.go` - Implementation example

### Files to Modify
1. `pkg/agentauth/extended_token_store.go` - Add methods
2. `pkg/agentauth/*_extended_token_store.go` - Implement methods
3. `pkg/agentauth/disclosure_service.go` - Fix type usage
4. `pkg/agentauth/disclosure_service_test.go` - New file for tests

### Documentation to Update
1. `IMPLEMENTATION_STATUS.md` - Update after completion
2. `IMPLEMENTATION_PROGRESS_REPORT.md` - Document fixes
3. API documentation - Add disclosure endpoints

---

## Decision Point

**Question**: Which option should we proceed with?

1. ✅ **Option 1: Proper Fix** (4-6 hours) - Complete, production-ready
2. ⏭️ **Option 2: Stub Implementation** (1 hour) - Quick, technical debt
3. 🔄 **Option 3: Adapter Pattern** (2-3 hours) - Middle ground

**Recommendation**: **Option 1** - Invest 4-6 hours now for clean, production-ready code

---

**Ready to proceed? Let me know which option you prefer!**
