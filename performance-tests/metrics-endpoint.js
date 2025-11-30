/**
 * K6 Performance Test - Metrics Endpoint
 * Tests Prometheus metrics endpoint under load
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const metricsFetchDuration = new Trend('metrics_fetch_duration');
const successfulFetches = new Counter('successful_fetches');
const metricsSize = new Trend('metrics_response_size');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '1m', target: 30 },
    { duration: '2m', target: 50 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<800'],
    http_req_failed: ['rate<0.02'],
    errors: ['rate<0.02'],
  },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8080';

export default function () {
  const res = http.get(`${BASE_URL}/api/v1/beta/metrics`);
  
  const checkResult = check(res, {
    'status is 200': (r) => r.status === 200,
    'contains Prometheus format': (r) => r.body && r.body.includes('# HELP') && r.body.includes('# TYPE'),
    'response time < 800ms': (r) => r.timings.duration < 800,
    'response size reasonable': (r) => r.body && r.body.length < 1000000, // Less than 1MB
  });
  
  errorRate.add(!checkResult);
  metricsFetchDuration.add(res.timings.duration);
  
  if (res.body) {
    metricsSize.add(res.body.length);
  }
  
  if (checkResult) {
    successfulFetches.add(1);
  }
  
  sleep(2); // Metrics fetching typically has longer intervals
}

export function handleSummary(data) {
  return {
    'performance-tests/results/metrics-endpoint-summary.json': JSON.stringify(data),
  };
}
