# Phase 2C Enhancement Plan: Metrics/Prometheus Integration

**Project**: GAuth Enterprise IAM Platform  
**Phase**: Phase 2C - Metrics Page Integration with Prometheus  
**Date**: November 15, 2025  
**Status**: 🚧 In Progress  
**Estimated Duration**: 2-3 hours

---

## Executive Summary

Phase 2C focuses on integrating the Metrics page with real Prometheus metrics from the backend. Currently, the page uses dynamically generated mock data. This enhancement will replace all mock data with real metrics from multiple Prometheus endpoints, providing accurate system health and performance visibility.

### Current State
- ✅ Metrics page exists with complete UI (`web/ui-react/src/pages/Metrics.tsx`)
- ✅ Multiple backend Prometheus endpoints available
- ✅ Beautiful charts and visualizations already implemented
- ❌ All data is currently mock/generated
- ❌ No integration with real backend metrics

### Target State
- ✅ Parse Prometheus metrics format (text exposition format)
- ✅ Fetch real metrics from multiple endpoints
- ✅ Transform Prometheus data into chart-ready format
- ✅ Replace all mock generation with real data
- ✅ Maintain existing UI/UX and visualizations

---

## Objectives

1. **Parse Prometheus Metrics**: Implement parser for Prometheus text exposition format
2. **Integrate Multiple Endpoints**: Fetch and combine metrics from all available endpoints
3. **Real-Time Data**: Display actual system metrics instead of mock data
4. **Maintain UX**: Keep existing charts, auto-refresh, and loading states
5. **Error Handling**: Graceful fallback when metrics unavailable

---

## Available Prometheus Endpoints

Based on `web/server_clean.go` analysis:

| Endpoint | Purpose | Line |
|----------|---------|------|
| `/api/v1/beta/authz/metrics/prometheus` | Authorization cache/evaluation metrics | 5601 |
| `/api/v1/beta/policy/metrics/prometheus` | Policy engine metrics | 5574 |
| `/api/v1/beta/metrics/violations/prometheus` | Compliance violation metrics | 5578 |
| `/api/v1/beta/metrics/revocation/auto-sign/prometheus` | Revocation/auto-sign metrics | 5580 |
| `/api/v1/beta/capabilities/anchor/metrics/prometheus` | Capability anchor metrics | 5605 |
| `/api/v1/beta/metrics/prometheus` | Global system metrics | 5618 |

**Primary Focus**: We'll use `/api/v1/beta/authz/metrics/prometheus` and `/api/v1/beta/metrics/prometheus` for Phase 2C.

---

## Prometheus Text Format

Prometheus metrics are exposed in text exposition format:

```
# HELP metric_name Description of the metric
# TYPE metric_name counter
metric_name{label1="value1",label2="value2"} 123.456 1634567890000

# HELP http_requests_total Total HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",status="200"} 1234567

# HELP http_request_duration_seconds Request latency
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.05"} 1000
http_request_duration_seconds_bucket{le="0.1"} 2500
http_request_duration_seconds_bucket{le="0.5"} 5000
http_request_duration_seconds_sum 125.5
http_request_duration_seconds_count 5000
```

---

## Implementation Plan

### Task 1: Create Prometheus Parser ⏳

**File**: `web/ui-react/src/lib/prometheusParser.ts` (NEW)

**Functions**:
```typescript
interface PrometheusMetric {
  name: string;
  type: 'counter' | 'gauge' | 'histogram' | 'summary';
  help: string;
  values: Array<{
    value: number;
    labels: Record<string, string>;
    timestamp?: number;
  }>;
}

export function parsePrometheusMetrics(text: string): PrometheusMetric[]
export function getMetricValue(metrics: PrometheusMetric[], name: string, labels?: Record<string, string>): number | null
export function getHistogramStats(metrics: PrometheusMetric[], baseName: string): { avg: number; p95: number; p99: number }
```

**Parser Logic**:
1. Split text by lines
2. Identify HELP, TYPE, and value lines
3. Parse metric names, labels, and values
4. Group by metric name
5. Return structured array

### Task 2: Update API Client ⏳

**File**: `web/ui-react/src/lib/api.ts`

**New Methods**:
```typescript
// Fetch Prometheus metrics in text format
async getPrometheusMetrics(endpoint: string = '/beta/metrics/prometheus'): Promise<string>

// Fetch and parse authorization metrics
async getAuthzPrometheusMetrics(): Promise<PrometheusMetric[]>

// Fetch aggregated system metrics
async getSystemMetrics(): Promise<SystemMetrics>
```

**Implementation**:
- Use `axios.get()` with `responseType: 'text'`
- Parse response using `parsePrometheusMetrics()`
- Transform into `SystemMetrics` interface

### Task 3: Update Metrics Page ⏳

**File**: `web/ui-react/src/pages/Metrics.tsx`

**Changes**:
1. Remove `generateMetrics()`, `generateRequestsData()`, `generateLatencyData()` functions
2. Add `useEffect` to load real data on mount
3. Create `loadMetrics()` async function
4. Replace mock data with real Prometheus metrics
5. Keep existing UI/charts unchanged

**Data Mapping**:
```typescript
// Map Prometheus metrics to SystemMetrics
const systemMetrics = {
  requests: {
    total: getMetricValue(metrics, 'http_requests_total'),
    perSecond: getMetricValue(metrics, 'http_requests_rate')
  },
  latency: getHistogramStats(metrics, 'http_request_duration_seconds'),
  cache: {
    hitRate: calculateHitRate(metrics),
    size: getMetricValue(metrics, 'cache_entries')
  },
  // ... etc
};
```

### Task 4: Chart Data Transformation ⏳

**Challenge**: Prometheus metrics are instantaneous values, but charts need time-series data.

**Solutions**:
- **Option A**: Use histogram buckets if available (best for latency)
- **Option B**: Fetch metrics multiple times and store history (for trends)
- **Option C**: Use Prometheus `_over_time` queries if backend supports
- **Option D**: Keep time-series mock data, use real instant values for stats (pragmatic)

**Recommended**: Option D for Phase 2C - Use real metrics for stat cards, keep mock time-series for charts (document for future enhancement).

### Task 5: Error Handling ⏳

Add graceful fallback:
```typescript
try {
  const metrics = await apiClient.getSystemMetrics();
  setMetrics(metrics);
} catch (error) {
  console.error('Failed to fetch metrics:', error);
  toast.error('Failed to load metrics, showing cached data');
  // Keep existing data or show error state
}
```

---

## Testing Plan

### Unit Tests
- ✅ Test Prometheus parser with sample text
- ✅ Test metric value extraction
- ✅ Test histogram statistics calculation

### Integration Tests
- ✅ Fetch real metrics from backend
- ✅ Verify parsing works with real data
- ✅ Test error handling (backend down)

### Browser Testing
- ✅ Open Metrics page
- ✅ Verify stat cards show real values
- ✅ Verify auto-refresh updates data
- ✅ Test error states

---

## API Documentation

### GET /api/v1/beta/metrics/prometheus

**Description**: Global system metrics in Prometheus format

**Response** (text/plain):
```
# HELP http_requests_total Total HTTP requests processed
# TYPE http_requests_total counter
http_requests_total{method="GET",endpoint="/api/v1/beta/authz/evaluate"} 1234567

# HELP cache_hit_rate Cache hit rate (0-1)
# TYPE cache_hit_rate gauge
cache_hit_rate 0.938

# HELP system_uptime_seconds System uptime in seconds
# TYPE system_uptime_seconds counter
system_uptime_seconds 86400
```

### GET /api/v1/beta/authz/metrics/prometheus

**Description**: Authorization service metrics

**Response** (text/plain):
```
# HELP authz_cache_hits Total cache hits
# TYPE authz_cache_hits counter
authz_cache_hits 125000

# HELP authz_cache_misses Total cache misses
# TYPE authz_cache_misses counter
authz_cache_misses 8500

# HELP authz_evaluations_total Total authorization evaluations
# TYPE authz_evaluations_total counter
authz_evaluations_total 133500
```

---

## Success Criteria

- ✅ Prometheus parser implemented and tested
- ✅ API client methods fetch real metrics
- ✅ Metrics page displays real stat card values
- ✅ Auto-refresh fetches updated metrics
- ✅ Error handling works gracefully
- ✅ No TypeScript compilation errors
- ✅ Browser testing passes
- ✅ Documentation updated

---

## Timeline

| Task | Duration | Status |
|------|----------|--------|
| Create enhancement plan | 20 min | ⏳ In Progress |
| Implement Prometheus parser | 30 min | ⏳ Pending |
| Update API client | 20 min | ⏳ Pending |
| Update Metrics page | 40 min | ⏳ Pending |
| Testing & debugging | 30 min | ⏳ Pending |
| Documentation | 20 min | ⏳ Pending |
| **Total** | **2.5 hours** | |

---

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Prometheus format parsing errors | Medium | Medium | Extensive parser testing with real data |
| Backend metrics not matching expected format | Low | High | Verify endpoints first, add format validation |
| Time-series data not available | High | Low | Use instant values for stats, keep mock charts |
| Performance impact from multiple endpoints | Low | Medium | Fetch only necessary endpoints, cache results |

---

## Dependencies

- ✅ Backend server running (`go run ./cmd/web-server`)
- ✅ Frontend server running (`vite`)
- ✅ Prometheus endpoints available at `/api/v1/beta/*`
- ✅ Existing Metrics page UI
- ✅ recharts library for charts (already installed)

---

## Future Enhancements

**Post-Phase 2C**:
1. Add time-series data collection (store metrics history)
2. Integrate additional Prometheus endpoints (policy, violations, etc.)
3. Add custom metric queries
4. Implement Prometheus alerting integration
5. Add metric export functionality

---

## Related Documentation

- `PHASE_2A_COMPLETION_REPORT.md` - Reference for Phase 2 workflow
- `PHASE_2B_COMPLETION_REPORT.md` - Reference for PIP integration patterns
- `web/ui-react/API_INTEGRATION_GUIDE.md` - API integration guidelines
- `web/server_clean.go` (lines 5574-5618) - Prometheus endpoint definitions

---

**Next Phase**: Phase 2D - E2E Testing Page Integration (if needed)

---

*Plan created: November 15, 2025*  
*Estimated completion: Same day*  
*Status: Ready to implement*
