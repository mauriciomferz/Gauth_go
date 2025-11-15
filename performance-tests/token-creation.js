/**
 * K6 Performance Test - Token Creation Endpoint
 * Tests token creation performance under load
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const tokenCreationDuration = new Trend('token_creation_duration');
const successfulTokens = new Counter('successful_tokens');

// Test configuration
export const options = {
  stages: [
    { duration: '1m', target: 20 },    // Ramp up to 20 users
    { duration: '3m', target: 50 },    // Stay at 50 users
    { duration: '1m', target: 0 },     // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'], // 95% under 1 second
    http_req_failed: ['rate<0.05'],     // Error rate < 5%
    errors: ['rate<0.05'],
  },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8080';

export default function () {
  const clientId = `perf-test-${__VU}-${__ITER}`;
  
  const payload = JSON.stringify({
    clientId: clientId,
    subject: 'performance-test-subject',
    scopes: ['read', 'write'],
    audience: ['https://api.example.com'],
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };
  
  const res = http.post(`${BASE_URL}/api/v1/beta/tokens`, payload, params);
  
  const checkResult = check(res, {
    'status is 200 or 201': (r) => r.status === 200 || r.status === 201,
    'has token property': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.token !== undefined;
      } catch {
        return false;
      }
    },
    'response time < 1000ms': (r) => r.timings.duration < 1000,
  });
  
  errorRate.add(!checkResult);
  tokenCreationDuration.add(res.timings.duration);
  
  if (checkResult) {
    successfulTokens.add(1);
  }
  
  sleep(1);
}

export function handleSummary(data) {
  return {
    'performance-tests/results/token-creation-summary.json': JSON.stringify(data),
  };
}
