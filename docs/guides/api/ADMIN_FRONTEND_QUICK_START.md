---
title: Admin Frontend Quick Start
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Admin Frontend - Quick Start Guide

## 🚀 Start Development Environment

### 1. Start Backend Server
```bash
cd /Users/mauricio.fernandez_fernandezsiemens.co/Gauth_go

# Option A: Using environment variables
GAUTH_JWT_SIGNING_KEY="test-key" \
DB_HOST="localhost" \
DB_PORT="5432" \
DB_USER="postgres" \
DB_PASSWORD="gauth_dev_password" \
DB_NAME="gauth" \
DB_SSLMODE="disable" \
go run ./cmd/web-server

# Option B: Using VS Code Task
# Press Cmd+Shift+P -> "Tasks: Run Task" -> "Start AgentAuth Web Server"
```

Backend will start on: **http://localhost:8080**

### 2. Start Frontend Dev Server
```bash
cd web/ui-react
npm run dev
```

Frontend will start on: **http://localhost:5173**

---

## 🧪 Quick Test

### Verify Backend is Running
```bash
# Check server status
curl http://localhost:8080/health

# Test PoA endpoint with tenant_id
curl "http://localhost:8080/api/admin/poa?tenant_id=test-tenant-1"
```

### Verify Frontend Integration
1. Open browser: http://localhost:5173
2. Navigate to any admin page
3. Open DevTools (F12) -> Network tab
4. Verify requests include `?tenant_id=test-tenant-1`
5. Check for no console errors

---

## 📂 Key Files

### Utilities (YOU WILL USE THESE)
```typescript
// web/ui-react/src/hooks/useAdminApi.ts
import { usePowerOfAttorneyList, usePoAMutations } from '../../hooks/useAdminApi';

// Example usage
const { data, loading, error, refetch } = usePowerOfAttorneyList();
const { createPoA, updatePoA, deletePoA } = usePoAMutations();
```

### Type Definitions
```typescript
// web/ui-react/src/types/admin.ts
import type { PowerOfAttorney, CircuitBreaker, Event } from '../../types/admin';
```

### API Utilities (LOW-LEVEL)
```typescript
// web/ui-react/src/utils/api.ts
import { apiFetch, apiPost, apiPut, apiDelete } from '../../utils/api';

// Example
const response = await apiFetch('/api/admin/poa'); // auto-adds tenant_id
```

---

## 🎯 Admin Handler Endpoints

### Power of Attorney
- `GET /api/admin/poa` - List all PoAs
- `POST /api/admin/poa` - Create new PoA
- `PUT /api/admin/poa/{id}` - Update PoA
- `DELETE /api/admin/poa/{id}` - Delete PoA

### Resilience Patterns
- `GET /api/admin/resilience/circuit-breakers`
- `GET /api/admin/resilience/rate-limiters`
- `GET /api/admin/resilience/retry-policies`
- `POST /api/admin/resilience/circuit-breakers`

### Event System
- `GET /api/admin/events` - Recent events
- `GET /api/admin/events/types` - Event types
- `POST /api/admin/events/handlers` - Create handler

### Authorization Engine
- `GET /api/admin/authz/policies` - Authorization policies
- `POST /api/admin/authz/policies` - Create policy
- `PUT /api/admin/authz/policies/{id}` - Update policy

### Configuration Manager
- `GET /api/admin/config/variables` - Config variables
- `GET /api/admin/config/feature-flags` - Feature flags
- `POST /api/admin/config/variables` - Create variable

**All endpoints automatically include** `?tenant_id={id}` when using hooks/utilities

---

## 🔑 Environment Variables

### Backend (.env)
```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=gauth_dev_password
DB_NAME=gauth
DB_SSLMODE=disable
PORT=8080
GAUTH_JWT_SIGNING_KEY=test-key
```

### Frontend (.env.local)
```env
VITE_API_BASE_URL=http://localhost:8080
VITE_DEFAULT_TENANT_ID=test-tenant-1
```

---

## 🐛 Common Issues

### Issue: "Cannot connect to database"
```bash
# Check PostgreSQL is running
docker ps | grep gauth-postgres

# Start if not running
docker start gauth-postgres
```

### Issue: "Port 8080 already in use"
```bash
# Kill existing process
lsof -ti:8080 | xargs kill -9

# Or use different port
PORT=8081 go run ./cmd/web-server
```

### Issue: "CORS error in browser"
**Solution**: Ensure backend CORS allows `http://localhost:5173`

### Issue: "tenant_id missing in requests"
**Cause**: Using plain `fetch()` instead of hooks/utilities  
**Solution**: Import and use `apiFetch()` or hooks

---

## 📊 Available React Hooks

```typescript
// Power of Attorney
usePowerOfAttorneyList() // → { data, loading, error, refetch }
usePoAMutations() // → { createPoA, updatePoA, deletePoA }

// Resilience
useCircuitBreakers() // → { data, loading, error, refetch }
useRateLimiters()
useRetryPolicies()
useResilienceMutations() // → { createCircuitBreaker, createRateLimiter, ... }

// Events
useEvents() // → { data, loading, error, refetch }
useEventTypes()

// Authorization
useAuthorizationPolicies() // → { data, loading, error, refetch }
useAuthzMutations() // → { createPolicy, updatePolicy, deletePolicy }

// Configuration
useConfigVariables() // → { data, loading, error, refetch }
useFeatureFlags()
useConfigMutations() // → { createVariable, createFeatureFlag, ... }
```

---

## 🎨 Component Pattern

```typescript
import { usePowerOfAttorneyList, usePoAMutations } from '../../hooks/useAdminApi';

function MyComponent() {
  // 1. Use hooks
  const { data, loading, error, refetch } = usePowerOfAttorneyList();
  const { createPoA } = usePoAMutations();

  // 2. Update local state when data changes
  useEffect(() => {
    if (data?.powerOfAttorneys) {
      setPoaList(data.powerOfAttorneys);
    }
  }, [data]);

  // 3. Use mutations
  const handleCreate = async () => {
    await createPoA({ name: 'Test', ... });
    refetch(); // Refresh list
  };

  // 4. Render
  if (loading) return <Spinner />;
  if (error) return <div>Error: {error}</div>;
  return <div>{/* render data */}</div>;
}
```

---

## ✅ Testing Checklist

- [ ] Backend running on port 8080
- [ ] Frontend running on port 5173
- [ ] Database connected (PostgreSQL)
- [ ] Navigate to http://localhost:5173
- [ ] Open admin pages (Power of Attorney, Resilience, etc.)
- [ ] Check browser DevTools Network tab
- [ ] Verify `tenant_id` parameter in requests
- [ ] No console errors
- [ ] CRUD operations work

---

## 📚 Documentation

1. **ADMIN_FRONTEND_INTEGRATION_GUIDE.md** - Comprehensive integration guide
2. **FRONTEND_INTEGRATION_STATUS.md** - Current status and progress
3. **FRONTEND_INTEGRATION_COMPLETION_REPORT.md** - Final completion report
4. **This File** - Quick reference for daily development

---

## 🆘 Need Help?

### Check These Files
- Hook implementation: `web/ui-react/src/hooks/useAdminApi.ts`
- API utilities: `web/ui-react/src/utils/api.ts`
- Type definitions: `web/ui-react/src/types/admin.ts`
- Example component: `web/ui-react/src/pages/admin/PowerOfAttorney.tsx`

### Test Backend Directly
```bash
# List PoAs
curl "http://localhost:8080/api/admin/poa?tenant_id=test-tenant-1" | jq

# Create PoA
curl -X POST "http://localhost:8080/api/admin/poa?tenant_id=test-tenant-1" \
  -H "Content-Type: application/json" \
  -d '{"principal_id":"user123","representative_id":"user456","valid_until":"2025-12-31T23:59:59Z"}' | jq
```

---

**Status**: ✅ Ready for Development  
**Last Updated**: November 23, 2025
