# Phase 2B Enhancement Plan - PIP/Authorization Integration

**Date**: November 15, 2025  
**Status**: In Progress 🔄  
**Goal**: Complete PIP page integration with real backend authorization endpoint  
**Timeline**: Same day (2-3 hours)

---

## Executive Summary

Phase 2B focuses on enhancing the Policy Information Point (PIP) page to fully integrate with the backend `/api/v1/beta/authz/evaluate` endpoint. The backend endpoint already exists and is functional - this phase is about optimizing the frontend integration and removing any remaining mocks.

---

## Current State Analysis

### ✅ Already Working
1. **Backend Endpoint**: `/api/v1/beta/authz/evaluate` exists and is functional
2. **API Client Integration**: `checkAuthorization()` method already calls the real backend
3. **PIP Page**: Currently using `validateAuthorization()` which wraps `checkAuthorization()`
4. **Response Mapping**: Backend response is properly mapped to frontend types

### 🔄 Needs Enhancement
1. **Cache Stats**: Currently using mock data, need real metrics endpoint
2. **Active Policies Display**: Using hardcoded policy rules
3. **Error Handling**: Can be improved with better user feedback
4. **Evaluation Metrics**: Need real-time metrics from backend

---

## Phase 2B Objectives

### Primary Goals
1. ✅ Verify `/beta/authz/evaluate` endpoint integration (Already done)
2. 🔄 Implement real cache statistics from backend metrics
3. 🔄 Fetch active policies from backend
4. 🔄 Enhance error handling and user feedback
5. 🔄 Add real-time evaluation metrics dashboard

### Success Criteria
- [ ] PIP page shows real cache statistics from backend
- [ ] Active policies fetched from backend policy API
- [ ] All authorization evaluations use real backend
- [ ] Zero mocks or hardcoded data on PIP page
- [ ] Proper error handling with informative messages
- [ ] Performance metrics displayed accurately

---

## Implementation Plan

### Task 1: Verify Current Integration ✅
**Status**: Complete  
**Current Implementation**:
```typescript
// api.ts - Already integrated with backend
async checkAuthorization(data: AuthorizationRequest): Promise<AuthorizationResponse> {
  const response = await this.client.post('/beta/authz/evaluate', {
    subject: data.clientId,
    action: data.action,
    resource: data.geographic || 'default',
    context: data.sector ? { sector: data.sector } : {}
  })
  return {
    authorized: backendData.allowed || backendData.authorized || false,
    allowed: backendData.allowed || backendData.authorized || false,
    policies: backendData.policies || [],
    evaluationTime: backendData.evaluation_time || 0,
    // ... other fields
  }
}
```

### Task 2: Implement Real Cache Statistics 🔄
**Backend Endpoint**: `/api/v1/beta/authz/metrics` or `/api/v1/beta/metrics/prometheus`

**Implementation**:
1. Create `getAuthzCacheMetrics()` method in ApiClient
2. Update PIP page to fetch real cache stats
3. Parse metrics from backend response
4. Update cache stats display with real data

**API Client Method**:
```typescript
async getAuthzCacheMetrics(): Promise<CacheStats> {
  const response = await this.client.get('/beta/authz/metrics')
  const metrics = response.data
  return {
    hits: metrics.cache_hits || 0,
    misses: metrics.cache_misses || 0,
    hitRate: metrics.cache_hit_rate || 0,
    totalRequests: metrics.total_requests || 0,
    evictions: metrics.cache_evictions || 0
  }
}
```

### Task 3: Fetch Active Policies 🔄
**Backend Endpoint**: `/api/v1/beta/policy/head/policies` or `/api/v1/beta/policy/list`

**Implementation**:
1. Create `getActivePolicies()` method in ApiClient
2. Update PIP page to fetch and display real policies
3. Format policy data for display
4. Add policy filtering/search if needed

**API Client Method**:
```typescript
async getActivePolicies(): Promise<PolicyRule[]> {
  const response = await this.client.get('/beta/policy/head/policies')
  const policies = response.data.policies || []
  return policies.map(p => ({
    id: p.id,
    name: p.name,
    description: p.description || '',
    status: p.enabled ? 'active' : 'inactive',
    priority: p.priority || 0
  }))
}
```

### Task 4: Enhanced Error Handling 🔄
**Implementation**:
1. Remove try-catch fallback to mock data
2. Add proper error types and messages
3. Display user-friendly error notifications
4. Log errors for debugging

**Example**:
```typescript
async checkAuthorization(data: AuthorizationRequest): Promise<AuthorizationResponse> {
  try {
    const response = await this.client.post('/beta/authz/evaluate', {
      subject: data.clientId,
      action: data.action,
      resource: data.geographic || 'default',
      context: data.sector ? { sector: data.sector } : {}
    })
    return this.mapAuthzResponse(response.data, data)
  } catch (error) {
    if (axios.isAxiosError(error)) {
      throw new Error(error.response?.data?.message || 'Authorization check failed')
    }
    throw error
  }
}
```

### Task 5: Real-Time Metrics Dashboard 🔄
**Implementation**:
1. Add evaluation metrics endpoint call
2. Display decision statistics
3. Show policy evaluation times
4. Add recent evaluations log

---

## API Endpoints Reference

### Already Existing
| Endpoint | Method | Purpose | Status |
|----------|--------|---------|--------|
| `/api/v1/beta/authz/evaluate` | POST | Evaluate authorization | ✅ Working |
| `/api/v1/beta/authz/metrics` | GET | Authorization metrics | 🔄 Need to verify |
| `/api/v1/beta/policy/head/policies` | GET | List active policies | 🔄 Need to verify |
| `/api/v1/beta/metrics/prometheus` | GET | Prometheus metrics | ✅ Exists |

### Request/Response Formats

**Authorization Evaluation Request**:
```json
{
  "subject": "user_123",
  "action": "read",
  "resource": "/api/users",
  "context": {
    "department": "engineering",
    "role": "developer"
  }
}
```

**Authorization Evaluation Response**:
```json
{
  "allowed": true,
  "authorized": true,
  "policies": ["admin-policy", "read-policy"],
  "evaluation_time": 15,
  "cache_hit": true,
  "decision": {
    "allow": true,
    "reason": "Policy admin-policy granted access"
  }
}
```

---

## Timeline

### Phase 2B (2-3 hours)
- **Hour 1**: Verify and document current integration
- **Hour 2**: Implement cache metrics and active policies
- **Hour 3**: Enhanced error handling and testing

### Milestones
1. ✅ Current integration verified
2. 🔄 Cache stats showing real data
3. 🔄 Active policies from backend
4. 🔄 Error handling improved
5. 🔄 PIP page 100% integrated

---

## Testing Plan

### Unit Tests
- [ ] `checkAuthorization()` calls correct endpoint
- [ ] Response mapping handles all fields correctly
- [ ] Error cases handled properly

### Integration Tests
- [ ] Authorization evaluation works end-to-end
- [ ] Cache metrics update correctly
- [ ] Policy list displays properly

### Browser Tests
- [ ] PIP page loads without errors
- [ ] Form submission works correctly
- [ ] Results display properly
- [ ] Error messages show when backend fails
- [ ] Cache stats update after evaluations

---

## Dependencies

### Backend
- ✅ `/api/v1/beta/authz/evaluate` - Already exists
- 🔄 `/api/v1/beta/authz/metrics` - Need to verify
- 🔄 `/api/v1/beta/policy/head/policies` - Need to verify

### Frontend
- ✅ PIP.tsx component
- ✅ ApiClient class
- ✅ Authorization types defined

---

## Risk Assessment

### Low Risk ✅
- Backend endpoint already exists and works
- API client integration already in place
- Types already defined

### Medium Risk 🔄
- Metrics endpoint format might differ from expectations
- Policy API might not exist or have different structure

### Mitigation
- Verify all backend endpoints before implementation
- Use graceful degradation if metrics unavailable
- Keep cache stats display even if real metrics fail

---

## Completion Criteria

### Must Have
- [x] Authorization evaluation using real backend
- [ ] Cache statistics from real metrics
- [ ] Active policies from backend
- [ ] No mock data in PIP page
- [ ] Proper error handling

### Nice to Have
- [ ] Real-time metrics updates
- [ ] Policy filtering/search
- [ ] Recent evaluations log
- [ ] Performance charts

---

## Next Steps

1. Verify backend metrics endpoints exist
2. Implement `getAuthzCacheMetrics()` in ApiClient
3. Implement `getActivePolicies()` in ApiClient
4. Update PIP page to use real metrics
5. Test end-to-end
6. Remove fallback mock code
7. Update documentation
8. Create completion report

---

**Phase 2B Status**: Ready to begin implementation  
**Expected Completion**: Same day (November 15, 2025)  
**Next Phase**: Phase 2C - Metrics with Prometheus parsing
