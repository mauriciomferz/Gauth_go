/**
 * K6 Stress Test - Find Breaking Point
 * Gradually increases load to find system limits
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const requestDuration = new Trend('request_duration');
const activeUsers = new Gauge('active_users');
const successfulRequests = new Counter('successful_requests');

// Stress test configuration
export const options = {
  stages: [
    { duration: '2m', target: 100 },    // Normal load
    { duration: '5m', target: 200 },    // Increased load
    { duration: '5m', target: 300 },    // High load
    { duration: '5m', target: 400 },    // Very high load
    { duration: '5m', target: 500 },    // Extreme load
    { duration: '10m', target: 0 },     // Recovery
  ],
  thresholds: {
    http_req_duration: ['p(99)<3000'],  // 99% under 3 seconds
    http_req_failed: ['rate<0.1'],       // Error rate < 10%
  },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8080';

export default function () {
  activeUsers.add(__VU);
  
  const res = http.get(`${BASE_URL}/health`);
  
  const checkResult = check(res, {
    'status is 200': (r) => r.status === 200,
    'response time acceptable': (r) => r.timings.duration < 3000,
  });
  
  errorRate.add(!checkResult);
  requestDuration.add(res.timings.duration);
  
  if (checkResult) {
    successfulRequests.add(1);
  }
  
  sleep(1);
}

export function handleSummary(data) {
  const summary = generateStressSummary(data);
  
  return {
    'performance-tests/results/stress-test-summary.json': JSON.stringify(data),
    'performance-tests/results/stress-test-summary.txt': summary,
    stdout: summary,
  };
}

function generateStressSummary(data) {
  let summary = '\n';
  summary += '╔════════════════════════════════════════╗\n';
  summary += '║        STRESS TEST RESULTS             ║\n';
  summary += '╠════════════════════════════════════════╣\n';
  summary += `║ Max VUs:           ${data.metrics.vus_max.values.max.toFixed(0).padStart(18)} ║\n`;
  summary += `║ Total Requests:    ${data.metrics.http_reqs.values.count.toString().padStart(18)} ║\n`;
  summary += `║ Failed Rate:       ${(data.metrics.http_req_failed.values.rate * 100).toFixed(2).padStart(15)}% ║\n`;
  summary += `║ Avg Duration:      ${data.metrics.http_req_duration.values.avg.toFixed(0).padStart(13)}ms ║\n`;
  summary += `║ P95 Duration:      ${data.metrics.http_req_duration.values['p(95)'].toFixed(0).padStart(13)}ms ║\n`;
  summary += `║ P99 Duration:      ${data.metrics.http_req_duration.values['p(99)'].toFixed(0).padStart(13)}ms ║\n`;
  summary += '╚════════════════════════════════════════╝\n';
  
  // Determine system limit
  const failRate = data.metrics.http_req_failed.values.rate;
  const p99Duration = data.metrics.http_req_duration.values['p(99)'];
  
  if (failRate < 0.01 && p99Duration < 1000) {
    summary += '\n✅ System handled stress well - consider higher loads\n';
  } else if (failRate < 0.05 && p99Duration < 2000) {
    summary += '\n⚠️  System showing signs of stress - approaching limits\n';
  } else {
    summary += '\n❌ System under significant stress - limits reached\n';
  }
  
  return summary;
}
