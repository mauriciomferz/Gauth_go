import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const authDuration = new Trend('auth_duration');
const poaDuration = new Trend('poa_duration');
const validationDuration = new Trend('validation_duration');
const requestsPerSecond = new Counter('requests_per_second');

// Test configuration
export const options = {
  scenarios: {
    // Scenario 1: Baseline load test
    baseline: {
      executor: 'constant-vus',
      vus: 10,
      duration: '2m',
      gracefulStop: '10s',
    },
    // Scenario 2: Ramp-up load test
    rampup: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '1m', target: 50 },
        { duration: '3m', target: 100 },
        { duration: '2m', target: 200 },
        { duration: '1m', target: 0 },
      ],
      gracefulStop: '30s',
      startTime: '2m',
    },
    // Scenario 3: Spike test
    spike: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 50 },
        { duration: '1m', target: 50 },
        { duration: '10s', target: 500 },
        { duration: '30s', target: 500 },
        { duration: '10s', target: 50 },
        { duration: '1m', target: 50 },
        { duration: '10s', target: 0 },
      ],
      gracefulStop: '30s',
      startTime: '9m',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'], // 95% < 500ms, 99% < 1s
    http_req_failed: ['rate<0.05'], // Error rate < 5%
    'errors': ['rate<0.1'], // Custom error rate < 10%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test data
const testData = {
  brazil: {
    cpf: '12345678909',
    name: 'Test User',
  },
  canada: {
    sin: '046454286',
    firstName: 'John',
    lastName: 'Doe',
    dateOfBirth: '1990-01-15',
  },
  mexico: {
    curp: 'GOTJ901015HDFRRL09',
    name: 'Test User',
  },
  southAfrica: {
    idNumber: '9001085800087',
    name: 'Test User',
  },
  nigeria: {
    nin: '12345678901',
    firstName: 'John',
    surname: 'Doe',
  },
  kenya: {
    idNumber: '12345678',
    firstName: 'John',
    surname: 'Doe',
  },
};

// Test functions
export function testHealthCheck() {
  const res = http.get(`${BASE_URL}/healthz`);
  check(res, {
    'health check status 200': (r) => r.status === 200,
  });
  requestsPerSecond.add(1);
}

export function testPoACreation() {
  const start = Date.now();
  const payload = JSON.stringify({
    grantor: 'test@example.com',
    grantee: 'grantee@example.com',
    scope: ['read', 'write'],
    valid_from: new Date().toISOString(),
    valid_until: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString(),
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const res = http.post(`${BASE_URL}/api/v1/beta/poa`, payload, params);
  const duration = Date.now() - start;
  
  const success = check(res, {
    'PoA creation status 200': (r) => r.status === 200,
    'PoA has ID': (r) => {
      const body = JSON.parse(r.body);
      return body.poa && body.poa.id;
    },
  });

  if (!success) {
    errorRate.add(1);
  }
  
  poaDuration.add(duration);
  requestsPerSecond.add(1);
}

export function testAuthorization() {
  const start = Date.now();
  const subscriptionID = `sub_${Date.now()}`;
  const payload = JSON.stringify({
    client_id: 'test-client',
    subscription_id: subscriptionID,
    resource_owner_id: 'owner@example.com',
    scope: 'read write',
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const res = http.post(`${BASE_URL}/api/v1/rfc0111/authorize`, payload, params);
  const duration = Date.now() - start;
  
  const success = check(res, {
    'Authorization status 200': (r) => r.status === 200,
    'Has authorization code': (r) => {
      const body = JSON.parse(r.body);
      return body.code || body.authorization_code;
    },
  });

  if (!success) {
    errorRate.add(1);
  }
  
  authDuration.add(duration);
  requestsPerSecond.add(1);
}

export function testBrazilCPFValidation() {
  const start = Date.now();
  const payload = JSON.stringify({
    cpf: testData.brazil.cpf,
    name: testData.brazil.name,
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const res = http.post(`${BASE_URL}/api/v1/external/brazil/validate-cpf`, payload, params);
  const duration = Date.now() - start;
  
  const success = check(res, {
    'Brazil CPF validation status 200': (r) => r.status === 200,
    'Has validation result': (r) => {
      const body = JSON.parse(r.body);
      return body.valid !== undefined;
    },
  });

  if (!success) {
    errorRate.add(1);
  }
  
  validationDuration.add(duration);
  requestsPerSecond.add(1);
}

export function testCanadaSINValidation() {
  const start = Date.now();
  const payload = JSON.stringify({
    sin: testData.canada.sin,
    first_name: testData.canada.firstName,
    last_name: testData.canada.lastName,
    date_of_birth: testData.canada.dateOfBirth,
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const res = http.post(`${BASE_URL}/api/v1/external/canada/validate-sin`, payload, params);
  const duration = Date.now() - start;
  
  const success = check(res, {
    'Canada SIN validation status 200': (r) => r.status === 200,
  });

  if (!success) {
    errorRate.add(1);
  }
  
  validationDuration.add(duration);
  requestsPerSecond.add(1);
}

export function testMexicoCURPValidation() {
  const start = Date.now();
  const payload = JSON.stringify({
    curp: testData.mexico.curp,
    name: testData.mexico.name,
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const res = http.post(`${BASE_URL}/api/v1/external/mexico/validate-curp`, payload, params);
  const duration = Date.now() - start;
  
  const success = check(res, {
    'Mexico CURP validation status 200': (r) => r.status === 200,
  });

  if (!success) {
    errorRate.add(1);
  }
  
  validationDuration.add(duration);
  requestsPerSecond.add(1);
}

export function testMCPResourcesList() {
  const start = Date.now();
  const res = http.get(`${BASE_URL}/api/v1/beta/mcp/servers`);
  const duration = Date.now() - start;
  
  const success = check(res, {
    'MCP servers list status 200': (r) => r.status === 200,
  });

  if (!success) {
    errorRate.add(1);
  }
  
  requestsPerSecond.add(1);
  sleep(0.1);
}

// Main test execution
export default function () {
  // Randomly execute different test scenarios
  const rand = Math.random();
  
  if (rand < 0.2) {
    testHealthCheck();
  } else if (rand < 0.35) {
    testPoACreation();
  } else if (rand < 0.5) {
    testAuthorization();
  } else if (rand < 0.6) {
    testBrazilCPFValidation();
  } else if (rand < 0.7) {
    testCanadaSINValidation();
  } else if (rand < 0.8) {
    testMexicoCURPValidation();
  } else {
    testMCPResourcesList();
  }
  
  sleep(1);
}

// Handle test summary
export function handleSummary(data) {
  return {
    'load-test-summary.json': JSON.stringify(data, null, 2),
    stdout: textSummary(data, { indent: '→', enableColors: true }),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';
  const colors = options.enableColors || false;
  
  let summary = '\n' + indent + '═══════════════════════════════════════\n';
  summary += indent + '  Load Test Summary\n';
  summary += indent + '═══════════════════════════════════════\n\n';
  
  // Overall metrics
  summary += indent + 'Overall Metrics:\n';
  summary += indent + '  Total Requests: ' + data.metrics.http_reqs.values.count + '\n';
  summary += indent + '  Failed Requests: ' + data.metrics.http_req_failed.values.count + '\n';
  summary += indent + '  Request Rate: ' + data.metrics.http_reqs.values.rate.toFixed(2) + ' req/s\n';
  summary += indent + '  Data Received: ' + (data.metrics.data_received.values.count / 1024 / 1024).toFixed(2) + ' MB\n';
  summary += indent + '  Data Sent: ' + (data.metrics.data_sent.values.count / 1024 / 1024).toFixed(2) + ' MB\n\n';
  
  // Response times
  summary += indent + 'Response Times:\n';
  summary += indent + '  Avg: ' + data.metrics.http_req_duration.values.avg.toFixed(2) + ' ms\n';
  summary += indent + '  Min: ' + data.metrics.http_req_duration.values.min.toFixed(2) + ' ms\n';
  summary += indent + '  Max: ' + data.metrics.http_req_duration.values.max.toFixed(2) + ' ms\n';
  summary += indent + '  P95: ' + data.metrics.http_req_duration.values['p(95)'].toFixed(2) + ' ms\n';
  summary += indent + '  P99: ' + data.metrics.http_req_duration.values['p(99)'].toFixed(2) + ' ms\n\n';
  
  return summary;
}
