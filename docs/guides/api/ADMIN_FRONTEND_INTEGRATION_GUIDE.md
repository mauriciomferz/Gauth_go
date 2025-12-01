# Admin Handlers Frontend Integration Guide

**Date:** November 23, 2025  
**Status:** Integration Ready  
**Frontend:** React + TypeScript + Fluent UI  
**Backend:** Go + Gin + PostgreSQL

---

## Overview

This guide provides step-by-step instructions for integrating the React frontend with the newly completed admin handler backend APIs that now use PostgreSQL with full tenant isolation.

---

## API Endpoint Changes

### ✅ All admin endpoints now require `tenant_id` parameter

**Old Format:**
```
GET /api/admin/poa
POST /api/admin/poa
```

**New Format:**
```
GET /api/admin/poa?tenant_id={tenant_id}
POST /api/admin/poa?tenant_id={tenant_id}
```

---

## 1. Power of Attorney Integration

### Backend Endpoint
```
GET  /api/admin/poa?tenant_id={tenant_id}
POST /api/admin/poa?tenant_id={tenant_id}
GET  /api/admin/poa/{id}?tenant_id={tenant_id}
PUT  /api/admin/poa/{id}?tenant_id={tenant_id}
DELETE /api/admin/poa/{id}?tenant_id={tenant_id}
```

### Frontend File
`web/ui-react/src/pages/admin/PowerOfAttorney.tsx`

### Required Changes

**Current Code (Line ~284):**
```typescript
const response = await fetch('/api/admin/poa', {
  headers: {
    Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
  },
});
```

**Updated Code:**
```typescript
const tenantId = localStorage.getItem('tenant_id') || 'default-tenant';
const response = await fetch(`/api/admin/poa?tenant_id=${tenantId}`, {
  headers: {
    Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
  },
});
```

### Response Format
```json
{
  "powerOfAttorneys": [
    {
      "id": "uuid",
      "principalId": "string",
      "principalName": "string",
      "representativeId": "string",
      "representativeName": "string",
      "representativeType": "string",
      "status": "active|revoked|expired",
      "validFrom": "ISO8601",
      "validUntil": "ISO8601",
      "actions": ["string"],
      "resources": ["string"],
      "geoRestrictions": ["string"],
      "approvalStatus": "pending|approved|rejected",
      "createdAt": "ISO8601"
    }
  ],
  "total": 1
}
```

---

## 2. Resilience Patterns Integration

### Backend Endpoints
```
GET  /api/admin/resilience/circuit-breakers?tenant_id={tenant_id}
POST /api/admin/resilience/circuit-breakers?tenant_id={tenant_id}
GET  /api/admin/resilience/rate-limiters?tenant_id={tenant_id}
POST /api/admin/resilience/rate-limiters?tenant_id={tenant_id}
GET  /api/admin/resilience/retry-policies?tenant_id={tenant_id}
POST /api/admin/resilience/retry-policies?tenant_id={tenant_id}
GET  /api/admin/resilience/bulkheads?tenant_id={tenant_id}
POST /api/admin/resilience/bulkheads?tenant_id={tenant_id}
```

### Frontend File
`web/ui-react/src/pages/admin/ResiliencePatterns.tsx`

### Circuit Breaker Response Format
```json
{
  "circuitBreakers": [
    {
      "id": "uuid",
      "name": "string",
      "service": "string",
      "state": "open|closed|half-open",
      "failureThreshold": 5,
      "successThreshold": 2,
      "timeout": 30000,
      "failures": 0,
      "successes": 0,
      "lastStateChange": "ISO8601",
      "totalRequests": 0,
      "failureRate": 0
    }
  ],
  "total": 1
}
```

### Rate Limiter Request Format
```json
{
  "name": "api-rate-limiter",
  "resource": "/api/v1/*",
  "algorithm": "token-bucket",
  "limit": 100,
  "window": 60,
  "burst": 20
}
```

---

## 3. Event System Integration

### Backend Endpoints
```
GET /api/admin/events?tenant_id={tenant_id}
GET /api/admin/events/types?tenant_id={tenant_id}
POST /api/admin/events/types?tenant_id={tenant_id}
GET /api/admin/events/handlers?tenant_id={tenant_id}
POST /api/admin/events/handlers?tenant_id={tenant_id}
GET /api/admin/events/stream?tenant_id={tenant_id}
```

### Frontend File
`web/ui-react/src/pages/admin/EventSystem.tsx`

### Response Format
```json
{
  "events": [
    {
      "id": "uuid",
      "type": "string",
      "category": "string",
      "severity": "info|warning|error",
      "source": "string",
      "timestamp": "ISO8601"
    }
  ],
  "total": 0
}
```

---

## 4. Authorization Engine Integration

### Backend Endpoints
```
GET  /api/admin/authz/policies?tenant_id={tenant_id}
POST /api/admin/authz/policies?tenant_id={tenant_id}
GET  /api/admin/authz/policies/{id}?tenant_id={tenant_id}
PUT  /api/admin/authz/policies/{id}?tenant_id={tenant_id}
DELETE /api/admin/authz/policies/{id}?tenant_id={tenant_id}
GET  /api/admin/authz/roles?tenant_id={tenant_id}
POST /api/admin/authz/simulate?tenant_id={tenant_id}
```

### Frontend File
`web/ui-react/src/pages/admin/AuthorizationEngine.tsx`

### Policy Response Format
```json
{
  "policies": [
    {
      "id": "uuid",
      "name": "string",
      "description": "string",
      "status": "draft|active|inactive",
      "effect": "allow|deny",
      "actions": ["string"],
      "resources": ["string"],
      "createdAt": "ISO8601",
      "updatedAt": "ISO8601"
    }
  ],
  "total": 1
}
```

### Policy Request Format
```json
{
  "name": "admin-access-policy",
  "description": "Full admin access policy",
  "effect": "allow",
  "actions": ["*"],
  "resources": ["*"],
  "priority": 100,
  "enabled": true
}
```

---

## 5. Configuration Management Integration

### Backend Endpoints
```
GET  /api/admin/config/variables?tenant_id={tenant_id}
POST /api/admin/config/variables?tenant_id={tenant_id}
GET  /api/admin/config/variables/{key}?tenant_id={tenant_id}
PUT  /api/admin/config/variables/{key}?tenant_id={tenant_id}
DELETE /api/admin/config/variables/{key}?tenant_id={tenant_id}
GET  /api/admin/config/feature-flags?tenant_id={tenant_id}
POST /api/admin/config/feature-flags?tenant_id={tenant_id}
```

### Frontend File
`web/ui-react/src/pages/admin/ConfigurationManager.tsx`

### Variables Response Format
```json
{
  "variables": [
    {
      "key": "string",
      "value": "string",
      "type": "string",
      "sensitive": false,
      "description": "string"
    }
  ]
}
```

### Variable Request Format
```json
{
  "key": "max_login_attempts",
  "value": "5",
  "type": "integer",
  "sensitive": false,
  "description": "Maximum number of login attempts before lockout"
}
```

---

## Implementation Steps

### Step 1: Create Tenant Management Utility

Create `web/ui-react/src/utils/tenant.ts`:

```typescript
// Tenant management utility
export const getTenantId = (): string => {
  // Try to get from localStorage first
  const storedTenant = localStorage.getItem('tenant_id');
  if (storedTenant) return storedTenant;
  
  // Default tenant for development
  const defaultTenant = 'default-tenant';
  localStorage.setItem('tenant_id', defaultTenant);
  return defaultTenant;
};

export const setTenantId = (tenantId: string): void => {
  localStorage.setItem('tenant_id', tenantId);
};

export const addTenantParam = (url: string): string => {
  const tenantId = getTenantId();
  const separator = url.includes('?') ? '&' : '?';
  return `${url}${separator}tenant_id=${encodeURIComponent(tenantId)}`;
};
```

### Step 2: Create API Utility Helper

Create `web/ui-react/src/utils/api.ts`:

```typescript
import { getTenantId } from './tenant';

interface FetchOptions extends RequestInit {
  skipTenant?: boolean;
}

export const apiFetch = async (
  url: string,
  options: FetchOptions = {}
): Promise<Response> => {
  const { skipTenant, ...fetchOptions } = options;
  
  // Add tenant_id parameter unless explicitly skipped
  let finalUrl = url;
  if (!skipTenant) {
    const tenantId = getTenantId();
    const separator = url.includes('?') ? '&' : '?';
    finalUrl = `${url}${separator}tenant_id=${encodeURIComponent(tenantId)}`;
  }
  
  // Add authorization header if token exists
  const token = localStorage.getItem('admin_token');
  const headers = {
    'Content-Type': 'application/json',
    ...(token && { Authorization: `Bearer ${token}` }),
    ...fetchOptions.headers,
  };
  
  return fetch(finalUrl, {
    ...fetchOptions,
    headers,
  });
};
```

### Step 3: Update PowerOfAttorney Component

Update `web/ui-react/src/pages/admin/PowerOfAttorney.tsx`:

```typescript
import { apiFetch } from '../../utils/api';

// Replace all fetch calls with apiFetch
const fetchPoAList = async () => {
  try {
    const response = await apiFetch('/api/admin/poa');
    if (response.ok) {
      const data = await response.json();
      setPoaList(data.powerOfAttorneys || []);
    }
  } catch (error) {
    console.error('Failed to fetch PoA list:', error);
  }
};

const handleCreatePoA = async () => {
  setLoading(true);
  try {
    const response = await apiFetch('/api/admin/poa', {
      method: 'POST',
      body: JSON.stringify(formData),
    });

    if (response.ok) {
      fetchPoAList();
      setBuilderDialogOpen(false);
      resetForm();
    }
  } catch (error) {
    console.error('Failed to create PoA:', error);
  } finally {
    setLoading(false);
  }
};
```

### Step 4: Update ResiliencePatterns Component

Update `web/ui-react/src/pages/admin/ResiliencePatterns.tsx`:

```typescript
import { apiFetch } from '../../utils/api';

const fetchCircuitBreakers = async () => {
  try {
    const response = await apiFetch('/api/admin/resilience/circuit-breakers');
    if (response.ok) {
      const data = await response.json();
      setCircuitBreakers(data.circuitBreakers || []);
    }
  } catch (error) {
    console.error('Failed to fetch circuit breakers:', error);
  }
};

const fetchRateLimiters = async () => {
  try {
    const response = await apiFetch('/api/admin/resilience/rate-limiters');
    if (response.ok) {
      const data = await response.json();
      setRateLimiters(data.rateLimiters || []);
    }
  } catch (error) {
    console.error('Failed to fetch rate limiters:', error);
  }
};
```

### Step 5: Update EventSystem Component

Update `web/ui-react/src/pages/admin/EventSystem.tsx`:

```typescript
import { apiFetch } from '../../utils/api';

const fetchEvents = async () => {
  try {
    const response = await apiFetch('/api/admin/events');
    if (response.ok) {
      const data = await response.json();
      setEvents(data.events || []);
    }
  } catch (error) {
    console.error('Failed to fetch events:', error);
  }
};

const fetchEventTypes = async () => {
  try {
    const response = await apiFetch('/api/admin/events/types');
    if (response.ok) {
      const data = await response.json();
      setEventTypes(data.eventTypes || []);
    }
  } catch (error) {
    console.error('Failed to fetch event types:', error);
  }
};
```

### Step 6: Update AuthorizationEngine Component

Update `web/ui-react/src/pages/admin/AuthorizationEngine.tsx`:

```typescript
import { apiFetch } from '../../utils/api';

const fetchPolicies = async () => {
  try {
    const response = await apiFetch('/api/admin/authz/policies');
    if (response.ok) {
      const data = await response.json();
      setPolicies(data.policies || []);
    }
  } catch (error) {
    console.error('Failed to fetch policies:', error);
  }
};

const handleCreatePolicy = async (policyData: any) => {
  try {
    const response = await apiFetch('/api/admin/authz/policies', {
      method: 'POST',
      body: JSON.stringify(policyData),
    });

    if (response.ok) {
      fetchPolicies();
    }
  } catch (error) {
    console.error('Failed to create policy:', error);
  }
};
```

### Step 7: Update ConfigurationManager Component

Update `web/ui-react/src/pages/admin/ConfigurationManager.tsx`:

```typescript
import { apiFetch } from '../../utils/api';

const fetchVariables = async () => {
  try {
    const response = await apiFetch('/api/admin/config/variables');
    if (response.ok) {
      const data = await response.json();
      setVariables(data.variables || []);
    }
  } catch (error) {
    console.error('Failed to fetch variables:', error);
  }
};

const fetchFeatureFlags = async () => {
  try {
    const response = await apiFetch('/api/admin/config/feature-flags');
    if (response.ok) {
      const data = await response.json();
      setFlags(data.flags || []);
    }
  } catch (error) {
    console.error('Failed to fetch feature flags:', error);
  }
};
```

---

## Testing the Integration

### 1. Start the Backend Server

```bash
GAUTH_JWT_SIGNING_KEY="test-key" \
DB_HOST="localhost" \
DB_PORT="5432" \
DB_USER="postgres" \
DB_PASSWORD="gauth_dev_password" \
DB_NAME="gauth" \
DB_SSLMODE="disable" \
go run ./cmd/web-server
```

### 2. Start the Frontend Development Server

```bash
cd web/ui-react
npm install
npm run dev
```

### 3. Test Each Admin Page

Navigate to:
- `http://localhost:5173/admin/power-of-attorney`
- `http://localhost:5173/admin/resilience-patterns`
- `http://localhost:5173/admin/event-system`
- `http://localhost:5173/admin/authorization-engine`
- `http://localhost:5173/admin/configuration-manager`

### 4. Verify API Calls in Browser DevTools

1. Open Browser DevTools (F12)
2. Go to Network tab
3. Filter by "admin"
4. Check that all requests include `?tenant_id=` parameter
5. Verify responses match expected format

---

## Environment Variables

Update `.env.development`:

```env
VITE_API_BASE_URL=http://localhost:8080
VITE_DEFAULT_TENANT_ID=default-tenant
```

Update `.env.production`:

```env
VITE_API_BASE_URL=https://api.your-domain.com
VITE_DEFAULT_TENANT_ID=production-tenant
```

---

## Error Handling

Add global error handling for tenant-related errors:

```typescript
// In api.ts
export class TenantError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'TenantError';
  }
}

export const apiFetch = async (
  url: string,
  options: FetchOptions = {}
): Promise<Response> => {
  // ... existing code ...
  
  const response = await fetch(finalUrl, {
    ...fetchOptions,
    headers,
  });
  
  // Handle tenant-specific errors
  if (response.status === 403) {
    throw new TenantError('Access denied for this tenant');
  }
  
  if (response.status === 400) {
    const error = await response.json();
    if (error.message?.includes('tenant')) {
      throw new TenantError(error.message);
    }
  }
  
  return response;
};
```

---

## CORS Configuration

Ensure the backend allows CORS from frontend:

In `web/server_clean.go`, verify CORS middleware:

```go
r.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Tenant-ID"},
    ExposeHeaders:    []string{"Content-Length"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour,
}))
```

---

## Next Steps

1. ✅ Create utility files (`tenant.ts`, `api.ts`)
2. ✅ Update all 5 admin page components
3. ✅ Test each endpoint with Browser DevTools
4. ✅ Add error handling and loading states
5. ✅ Update environment variables
6. ✅ Test with real PostgreSQL data
7. ✅ Add pagination support
8. ✅ Add filtering/search functionality
9. ✅ Deploy to staging environment

---

## Summary

All backend admin handlers are operational with PostgreSQL and tenant isolation. The frontend components exist and need minor updates to add `tenant_id` parameters to all API calls. The integration can be completed by:

1. Creating helper utilities for tenant management
2. Updating fetch calls to use new utilities
3. Testing each page with real data
4. Adding proper error handling

**Estimated Time:** 2-3 hours for full integration

---

*Last Updated: November 23, 2025*  
*Backend Status: ✅ Complete*  
*Frontend Status: 🔄 Integration Required*
