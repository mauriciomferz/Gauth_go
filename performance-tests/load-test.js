/**
 * K6 Load Test - Complete API Workflow
 * Simulates realistic user journey through the API
 */

import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const workflowDuration = new Trend('workflow_duration');
const completedWorkflows = new Counter('completed_workflows');

// Load test configuration
export const options = {
  stages: [
    { duration: '2m', target: 50 },    // Ramp up to 50 concurrent users
    { duration: '5m', target: 100 },   // Stay at 100 users for 5 minutes
    { duration: '2m', target: 150 },   // Spike to 150 users
    { duration: '3m', target: 100 },   // Back to 100 users
    { duration: '2m', target: 0 },     // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    http_req_failed: ['rate<0.05'],
    errors: ['rate<0.05'],
    'workflow_duration': ['p(95)<5000'],
  },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8080';

export default function () {
  const workflowStart = Date.now();
  let workflowSuccess = true;
  
  // Step 1: Health Check
  group('Health Check', () => {
    const res = http.get(`${BASE_URL}/health`);
    workflowSuccess = workflowSuccess && check(res, {
      'health check ok': (r) => r.status === 200,
    });
  });
  
  sleep(0.5);
  
  // Step 2: Create Token
  let token;
  group('Create Token', () => {
    const clientId = `load-test-${__VU}-${__ITER}`;
    const payload = JSON.stringify({
      clientId: clientId,
      subject: 'load-test-subject',
      scopes: ['read', 'write'],
    });
    
    const res = http.post(`${BASE_URL}/api/v1/beta/tokens`, payload, {
      headers: { 'Content-Type': 'application/json' },
    });
    
    const success = check(res, {
      'token created': (r) => r.status === 200 || r.status === 201,
    });
    
    if (success) {
      try {
        const body = JSON.parse(res.body);
        token = body.token;
      } catch (e) {}
    }
    
    workflowSuccess = workflowSuccess && success;
  });
  
  sleep(0.5);
  
  // Step 3: Get Metrics
  group('Fetch Metrics', () => {
    const res = http.get(`${BASE_URL}/api/v1/beta/metrics`);
    workflowSuccess = workflowSuccess && check(res, {
      'metrics fetched': (r) => r.status === 200,
    });
  });
  
  sleep(0.5);
  
  // Step 4: List Subscriptions
  group('List Subscriptions', () => {
    const res = http.get(`${BASE_URL}/api/v1/beta/subscriptions`);
    workflowSuccess = workflowSuccess && check(res, {
      'subscriptions listed': (r) => r.status === 200 || r.status === 404,
    });
  });
  
  sleep(0.5);
  
  // Step 5: Check PIP Cache
  group('PIP Cache Stats', () => {
    const res = http.get(`${BASE_URL}/api/v1/beta/pip/cache/stats`);
    workflowSuccess = workflowSuccess && check(res, {
      'cache stats ok': (r) => r.status === 200 || r.status === 404,
    });
  });
  
  // Record workflow completion
  const workflowTime = Date.now() - workflowStart;
  workflowDuration.add(workflowTime);
  errorRate.add(!workflowSuccess);
  
  if (workflowSuccess) {
    completedWorkflows.add(1);
  }
  
  sleep(1);
}

export function handleSummary(data) {
  const passed = data.metrics.http_req_failed.values.rate < 0.05;
  
  return {
    'performance-tests/results/load-test-summary.json': JSON.stringify(data),
    'performance-tests/results/load-test-summary.html': htmlReport(data, passed),
    stdout: textSummary(data, passed),
  };
}

function textSummary(data, passed) {
  const passSymbol = passed ? '✓' : '✗';
  let summary = '\n';
  summary += `${passSymbol} Load Test Summary\n`;
  summary += `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`;
  summary += `  Total Requests:     ${data.metrics.http_reqs.values.count}\n`;
  summary += `  Failed Requests:    ${data.metrics.http_req_failed.values.rate * 100}%\n`;
  summary += `  Avg Duration:       ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms\n`;
  summary += `  P95 Duration:       ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms\n`;
  summary += `  Workflows Completed: ${data.metrics.completed_workflows?.values.count || 0}\n`;
  summary += `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`;
  return summary;
}

function htmlReport(data, passed) {
  return `
<!DOCTYPE html>
<html>
<head>
  <title>Load Test Report</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 40px; }
    .header { background: ${passed ? '#4CAF50' : '#f44336'}; color: white; padding: 20px; }
    .metrics { margin: 20px 0; }
    .metric { padding: 10px; border-bottom: 1px solid #ddd; }
    .label { font-weight: bold; }
  </style>
</head>
<body>
  <div class="header">
    <h1>${passed ? '✓' : '✗'} Load Test Report</h1>
    <p>Generated: ${new Date().toISOString()}</p>
  </div>
  <div class="metrics">
    <div class="metric">
      <span class="label">Total Requests:</span> ${data.metrics.http_reqs.values.count}
    </div>
    <div class="metric">
      <span class="label">Failed Requests:</span> ${(data.metrics.http_req_failed.values.rate * 100).toFixed(2)}%
    </div>
    <div class="metric">
      <span class="label">Avg Duration:</span> ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms
    </div>
    <div class="metric">
      <span class="label">P95 Duration:</span> ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms
    </div>
  </div>
</body>
</html>
  `;
}
