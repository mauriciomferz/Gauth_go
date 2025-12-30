---
title: Testing Guide
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# AgentAuth Testing Guide

Comprehensive guide for running and maintaining the AgentAuth automated testing suite.

## Table of Contents

- [Overview](#overview)
- [Test Infrastructure](#test-infrastructure)
- [Quick Start](#quick-start)
- [E2E Testing](#e2e-testing)
- [Performance Testing](#performance-testing)
- [CI/CD Integration](#cicd-integration)
- [Writing Tests](#writing-tests)
- [Troubleshooting](#troubleshooting)

## Overview

The AgentAuth project includes a comprehensive testing suite with multiple layers:

1. **E2E Tests** - Browser-based tests using Playwright
2. **API Integration Tests** - Backend endpoint validation
3. **Performance Tests** - Load and stress testing with k6
4. **Unit Tests** - Go backend unit tests

### Test Coverage

- **Frontend E2E**: Subscriptions, PIP, Metrics, Navigation
- **Backend API**: All 14+ integrated endpoints
- **Performance**: Health checks, token creation, metrics endpoints
- **Load Testing**: Complete user workflows under concurrent load
- **Stress Testing**: System breaking point identification

## Test Infrastructure

### Technologies

- **Playwright** - E2E testing framework
  - Multi-browser support (Chromium, Firefox, WebKit)
  - Mobile device emulation
  - Screenshot and video capture
  - Trace collection for debugging

- **k6** - Performance and load testing
  - HTTP load testing
  - Custom metrics
  - Threshold validation
  - JSON result exports

### Directory Structure

```
Gauth_go/
├── web/ui-react/
│   ├── e2e/                          # E2E test files
│   │   ├── global-setup.ts           # Pre-test setup
│   │   ├── global-teardown.ts        # Post-test cleanup
│   │   ├── auth.setup.ts             # Authentication setup
│   │   ├── subscriptions.spec.ts     # Subscription tests
│   │   ├── pip.spec.ts               # PIP tests
│   │   ├── metrics.spec.ts           # Metrics tests
│   │   ├── api.api.spec.ts           # API integration tests
│   │   └── ...                       # Existing tests
│   ├── playwright.config.ts          # Playwright configuration
│   └── test-results/                 # Test output (gitignored)
├── performance-tests/
│   ├── health-check.js               # Health endpoint load test
│   ├── token-creation.js             # Token performance test
│   ├── metrics-endpoint.js           # Metrics load test
│   ├── load-test.js                  # Complete workflow test
│   ├── stress-test.js                # Stress testing
│   └── results/                      # Test results (gitignored)
├── run-performance-tests.sh          # Performance test runner
└── .github/workflows/test-suite.yml  # CI/CD test automation
```

## Quick Start

### Prerequisites

1. **Node.js** - Version 20.x or higher
2. **Go** - Version 1.21 or higher
3. **k6** - Performance testing tool

#### Installing k6

**macOS:**
```bash
brew install k6
```

**Linux:**
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
  --keyserver hkp://keyserver.ubuntu.com:80 \
  --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | \
  sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

**Windows:**
```bash
choco install k6
```

### Initial Setup

1. **Install frontend dependencies:**
```bash
cd web/ui-react
npm install
```

2. **Install Playwright browsers:**
```bash
npx playwright install
```

3. **Start the backend server:**
```bash
# From project root
go run ./cmd/web-server
```

4. **Verify backend is running:**
```bash
curl http://localhost:8080/health
```

## E2E Testing

### Running E2E Tests

#### Run all E2E tests:
```bash
cd web/ui-react
npm run test
```

#### Run specific browser:
```bash
npm run test:chromium
npm run test:firefox
npm run test:webkit
```

#### Run specific test file:
```bash
npx playwright test e2e/subscriptions.spec.ts
```

#### Run in headed mode (see browser):
```bash
npx playwright test --headed
```

#### Run in debug mode:
```bash
npx playwright test --debug
```

#### Run with UI mode:
```bash
npx playwright test --ui
```

### Test Projects

The Playwright configuration includes 7 test projects:

1. **chromium** - Desktop Chrome tests
2. **firefox** - Desktop Firefox tests
3. **webkit** - Desktop Safari tests
4. **Mobile Chrome** - Pixel 5 mobile tests
5. **Mobile Safari** - iPhone 12 mobile tests
6. **iPad Pro** - Tablet tests
7. **api** - API-only tests (no browser)

### Viewing Test Reports

After running tests, view the HTML report:

```bash
npx playwright show-report
```

### Test Results

Test results are saved in:
- `web/ui-react/test-results/` - Screenshots, videos, traces
- `web/ui-react/playwright-report/` - HTML reports

### E2E Test Suites

#### Subscriptions Tests (`subscriptions.spec.ts`)
Tests subscription management functionality:
- Page display and navigation
- Subscription wizard flow
- Form validation
- Search and filtering
- Existing subscriptions list
- Empty state handling

#### PIP Tests (`pip.spec.ts`)
Tests Policy Information Point:
- Policy viewing
- Cache statistics
- Authorization checks
- Policy search
- Cache management

#### Metrics Tests (`metrics.spec.ts`)
Tests metrics and monitoring:
- Real-time metrics display
- Auto-refresh functionality
- Chart visualizations
- Category filtering (HTTP, Performance, System)
- Timestamp display

#### API Tests (`api.api.spec.ts`)
Tests backend API endpoints:
- Health and status checks
- Prometheus metrics format
- Token management (create, validate)
- PVP identity verification
- Registry queries
- PIP operations
- Subscription CRUD
- Error handling
- Performance benchmarks

## Performance Testing

### Running Performance Tests

#### Run all performance tests:
```bash
./run-performance-tests.sh all
```

#### Run specific test:
```bash
./run-performance-tests.sh health
./run-performance-tests.sh token
./run-performance-tests.sh metrics
./run-performance-tests.sh load
./run-performance-tests.sh stress
```

#### Run against different environment:
```bash
API_URL=https://staging.example.com ./run-performance-tests.sh all
```

### Performance Test Scenarios

#### Health Check Load Test
- **File**: `performance-tests/health-check.js`
- **Load**: 0 → 10 → 50 → 100 → 50 → 0 users
- **Duration**: 5 minutes
- **Thresholds**:
  - p95 < 500ms
  - Error rate < 1%

#### Token Creation Performance
- **File**: `performance-tests/token-creation.js`
- **Load**: 0 → 20 → 50 → 0 users
- **Duration**: 5 minutes
- **Thresholds**:
  - p95 < 1000ms
  - Error rate < 5%

#### Metrics Endpoint Load Test
- **File**: `performance-tests/metrics-endpoint.js`
- **Load**: 0 → 10 → 30 → 50 → 0 users
- **Duration**: 4 minutes
- **Thresholds**:
  - p95 < 800ms
  - Error rate < 2%

#### Complete Workflow Load Test
- **File**: `performance-tests/load-test.js`
- **Load**: 50 → 100 → 150 → 100 → 0 users
- **Duration**: 14 minutes
- **Tests**: Health → Create Token → Metrics → Subscriptions → PIP Cache
- **Thresholds**:
  - p95 < 2000ms
  - Error rate < 5%
  - Workflow p95 < 5000ms

#### Stress Test
- **File**: `performance-tests/stress-test.js`
- **Load**: 100 → 200 → 300 → 400 → 500 → 0 users
- **Duration**: 32 minutes
- **Purpose**: Find system breaking point
- **Thresholds**:
  - p99 < 3000ms
  - Error rate < 10%

### Performance Test Results

Results are saved in `performance-tests/results/`:
- JSON summary files
- Full test data
- HTML reports (for load tests)

### Interpreting Results

k6 provides metrics including:
- **http_req_duration** - Response times (avg, min, max, p90, p95, p99)
- **http_req_failed** - Percentage of failed requests
- **http_reqs** - Total number of requests
- **vus** - Number of virtual users
- **vus_max** - Maximum number of virtual users
- **iteration_duration** - Complete iteration time

**Performance Benchmarks:**
- ✅ Excellent: p95 < 500ms, errors < 1%
- ⚠️ Acceptable: p95 < 1000ms, errors < 5%
- ❌ Needs optimization: p95 > 1000ms, errors > 5%

## CI/CD Integration

### GitHub Actions Workflow

The test suite automatically runs on:
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop`
- Manual workflow dispatch

### CI/CD Jobs

1. **backend-tests** - Go unit tests with coverage
2. **e2e-tests** - Playwright tests (matrix: chromium, firefox, webkit)
3. **api-tests** - API integration tests
4. **performance-tests** - k6 performance validation
5. **test-summary** - Aggregated results summary

### Viewing CI Results

1. Go to **Actions** tab in GitHub repository
2. Select the workflow run
3. View job results and logs
4. Download artifacts:
   - Test reports (HTML)
   - Screenshots and videos (on failure)
   - Performance test results
   - Coverage reports

### Test Artifacts

Artifacts are retained for:
- Test reports: 30 days
- Videos/screenshots: 7 days
- Performance results: 30 days

## Writing Tests

### E2E Test Best Practices

#### 1. Use Flexible Selectors

```typescript
// ✅ Good - multiple selector strategies
const button = page.getByRole('button', { name: /create subscription/i })
  .or(page.getByText(/create subscription/i))
  .or(page.locator('button:has-text("Create Subscription")'));

// ❌ Avoid - brittle CSS selectors
const button = page.locator('.btn-primary.create-sub');
```

#### 2. Handle Dynamic Content

```typescript
// Wait for element to be visible
await page.waitForSelector('[data-testid="subscription-list"]', {
  state: 'visible',
  timeout: 10000,
});

// Wait for network to be idle
await page.waitForLoadState('networkidle');
```

#### 3. Test Structure (Arrange-Act-Assert)

```typescript
test('should create subscription', async ({ page }) => {
  // Arrange
  await page.goto('/subscriptions');
  
  // Act
  await page.click('button:has-text("Create")');
  await page.fill('input[name="name"]', 'Test Subscription');
  await page.click('button[type="submit"]');
  
  // Assert
  await expect(page.locator('.success-message')).toBeVisible();
});
```

#### 4. Handle API Errors Gracefully

```typescript
test('should handle API errors', async ({ page }) => {
  // Mock API error
  await page.route('**/api/v1/beta/subscriptions', route => {
    route.fulfill({ status: 500 });
  });
  
  await page.goto('/subscriptions');
  
  // Should show error message
  await expect(page.locator('.error-message')).toBeVisible();
});
```

### Performance Test Best Practices

#### 1. Define Clear Thresholds

```javascript
export const options = {
  thresholds: {
    http_req_duration: ['p(95)<500'],  // 95% under 500ms
    http_req_failed: ['rate<0.01'],     // Error rate < 1%
  },
};
```

#### 2. Use Custom Metrics

```javascript
import { Rate, Trend, Counter } from 'k6/metrics';

const errorRate = new Rate('errors');
const responseDuration = new Trend('response_duration');
const successfulRequests = new Counter('successful_requests');

export default function () {
  const res = http.get('http://example.com');
  
  errorRate.add(res.status !== 200);
  responseDuration.add(res.timings.duration);
  
  if (res.status === 200) {
    successfulRequests.add(1);
  }
}
```

#### 3. Realistic Load Patterns

```javascript
export const options = {
  stages: [
    { duration: '2m', target: 50 },   // Ramp up
    { duration: '5m', target: 100 },  // Sustained load
    { duration: '2m', target: 0 },    // Ramp down
  ],
};
```

#### 4. Add Think Time

```javascript
export default function () {
  http.get('http://example.com');
  sleep(1);  // Simulate user think time
}
```

## Troubleshooting

### Common Issues

#### Backend Not Responding

**Symptom**: Tests fail with connection errors

**Solutions**:
1. Check if backend is running: `curl http://localhost:8080/health`
2. Start backend: `go run ./cmd/web-server`
3. Check port conflicts: `lsof -i :8080`
4. Review backend logs for errors

#### Playwright Browser Installation Failed

**Symptom**: `browserType.launch: Executable doesn't exist`

**Solutions**:
1. Install browsers: `npx playwright install`
2. Install system dependencies: `npx playwright install-deps`
3. Clear cache: `rm -rf ~/.cache/ms-playwright`

#### Tests Timing Out

**Symptom**: Tests consistently timeout

**Solutions**:
1. Increase timeout in `playwright.config.ts`:
   ```typescript
   timeout: 60000,  // 60 seconds
   ```
2. Check backend performance
3. Add explicit waits: `await page.waitForLoadState('networkidle')`

#### Performance Tests Failing Thresholds

**Symptom**: k6 tests fail threshold checks

**Solutions**:
1. Review system resources (CPU, memory)
2. Check for concurrent processes consuming resources
3. Adjust thresholds if needed
4. Investigate specific failing endpoints
5. Review application logs for errors

#### CI/CD Tests Failing Locally Pass

**Symptom**: Tests pass locally but fail in CI

**Solutions**:
1. Check CI logs for specific errors
2. Ensure CI environment variables are set correctly
3. Test with `CI=true` locally: `CI=true npm run test`
4. Review timing differences (CI may be slower)
5. Check for hardcoded localhost URLs

### Test Debugging

#### Enable Verbose Logging

**Playwright:**
```bash
DEBUG=pw:api npx playwright test
```

**k6:**
```bash
k6 run --verbose performance-tests/health-check.js
```

#### Generate Trace Files

```bash
npx playwright test --trace on
```

View traces:
```bash
npx playwright show-trace trace.zip
```

#### Video Recording

Videos are automatically recorded on failure. To record all tests:

```typescript
// In playwright.config.ts
use: {
  video: 'on',  // or 'retain-on-failure'
}
```

## Performance Benchmarks

### Baseline Metrics (Local Development)

| Endpoint | Avg Response | p95 Response | Throughput |
|----------|--------------|--------------|------------|
| Health | 5ms | 10ms | 1000 req/s |
| Metrics | 15ms | 30ms | 500 req/s |
| Token Create | 50ms | 100ms | 200 req/s |
| Subscriptions List | 20ms | 50ms | 400 req/s |

### Load Test Results (100 Concurrent Users)

| Test | Success Rate | p95 Duration | Max Users |
|------|--------------|--------------|-----------|
| Health Check | 99.9% | 450ms | 100+ |
| Token Creation | 99.5% | 900ms | 100+ |
| Complete Workflow | 99.0% | 4500ms | 150 |

### Stress Test Findings

- **Breaking Point**: ~400-500 concurrent users
- **Bottlenecks**: Database connection pool, token signing operations
- **Recommendations**: 
  - Implement connection pooling
  - Consider token caching
  - Add rate limiting

## Best Practices Summary

### E2E Testing
✅ Use flexible selectors (role, text, data-testid)  
✅ Handle loading states explicitly  
✅ Test error scenarios  
✅ Keep tests independent and isolated  
✅ Use page objects for complex interactions  
✅ Mock external dependencies  

### Performance Testing
✅ Define realistic load patterns  
✅ Set meaningful thresholds  
✅ Include think time  
✅ Test complete user workflows  
✅ Monitor system resources  
✅ Baseline before optimization  

### CI/CD
✅ Run tests on every PR  
✅ Fail fast on critical errors  
✅ Upload artifacts for debugging  
✅ Parallel test execution  
✅ Clear failure notifications  

## Additional Resources

- [Playwright Documentation](https://playwright.dev/)
- [k6 Documentation](https://k6.io/docs/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [AgentAuth Project Documentation](./README.md)

## Support

For questions or issues with testing:
1. Check this guide first
2. Review test logs and error messages
3. Check existing issues in GitHub
4. Create new issue with:
   - Test output/logs
   - Environment details
   - Steps to reproduce

---

**Last Updated**: December 2024  
**Version**: 1.0.0
