/**
 * K6 Performance Test - Health Check Endpoint
 * Tests backend health endpoint performance under load
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const healthCheckDuration = new Trend('health_check_duration');
const successfulChecks = new Counter('successful_checks');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 10 },   // Ramp up to 10 users
    { duration: '1m', target: 50 },    // Ramp up to 50 users
    { duration: '2m', target: 100 },   // Ramp up to 100 users
    { duration: '1m', target: 50 },    // Ramp down to 50 users
    { duration: '30s', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],  // 95% of requests should be below 500ms
    http_req_failed: ['rate<0.01'],     // Error rate should be less than 1%
    errors: ['rate<0.01'],
  },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8080';

export default function () {
  // Test health endpoint
  const healthRes = http.get(`${BASE_URL}/health`);
  
  // Check response
  const checkResult = check(healthRes, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
  
  // Record metrics
  errorRate.add(!checkResult);
  healthCheckDuration.add(healthRes.timings.duration);
  
  if (checkResult) {
    successfulChecks.add(1);
  }
  
  // Small delay between requests
  sleep(1);
}

export function handleSummary(data) {
  return {
    'performance-tests/results/health-check-summary.json': JSON.stringify(data),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}

function textSummary(data, options = {}) {
  const indent = options.indent || '';
  const enableColors = options.enableColors !== false;
  
  let summary = '\n';
  summary += `${indent}✓ checks........................: ${(data.metrics.checks.values.rate * 100).toFixed(2)}%\n`;
  summary += `${indent}✓ http_req_duration............: avg=${data.metrics.http_req_duration.values.avg.toFixed(2)}ms\n`;
  summary += `${indent}  ├─ min......................: ${data.metrics.http_req_duration.values.min.toFixed(2)}ms\n`;
  summary += `${indent}  ├─ med......................: ${data.metrics.http_req_duration.values.med.toFixed(2)}ms\n`;
  summary += `${indent}  ├─ max......................: ${data.metrics.http_req_duration.values.max.toFixed(2)}ms\n`;
  summary += `${indent}  ├─ p(95)....................: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms\n`;
  summary += `${indent}  └─ p(99)....................: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms\n`;
  summary += `${indent}✓ http_reqs....................: ${data.metrics.http_reqs.values.count} (${data.metrics.http_reqs.values.rate.toFixed(2)}/s)\n`;
  
  return summary;
}
