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
const DURATION = __ENV.DURATION || '2m';
const RAMP_DURATION = __ENV.DURATION || '1m';
const STAGE_DURATION = __ENV.DURATION || '10s';

export const options = {
  scenarios: {
    // Scenario 1: Baseline load test
    baseline: {
      executor: 'constant-vus',
      vus: 5,
      duration: DURATION, // Use env var
      gracefulStop: '5s',
    },
    // Scenario 2: Ramp-up load test (Simplified for verification)
    rampup: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: STAGE_DURATION, target: 10 },
        { duration: STAGE_DURATION, target: 0 },
      ],
      gracefulStop: '5s',
      startTime: DURATION, // Start after baseline
    }
  },
  thresholds: {
    http_req_duration: ['p(95)<1000'],
    http_req_failed: ['rate<0.05'],
    'errors': ['rate<0.1'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://[::1]:8080';
const DEGRADED_MODE = __ENV.DEGRADED_MODE === 'true';

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
  const res = http.get(`${BASE_URL}/api/v1/agentauth/mcp/servers`);
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

export function testRevocation() {
  const start = Date.now();
  // 1. Create PoA to Revoke
  const payloadCreate = JSON.stringify({
    grantor: 'test-revocation@example.com',
    grantee: 'grantee-revocation@example.com',
    scope: ['read'],
    valid_from: new Date().toISOString(),
    valid_until: new Date(Date.now() + 3600 * 1000).toISOString(),
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const resCreate = http.post(`${BASE_URL}/api/v1/beta/poa`, payloadCreate, params);

  const checkCreate = check(resCreate, {
    'Revocation setup: PoA created': (r) => r.status === 200,
  });

  if (!checkCreate) {
    errorRate.add(1);
    return; // Stop if creation failed
  }

  const bodyCreate = JSON.parse(resCreate.body);
  const poaID = bodyCreate.poa.id;

  // 2. Revoke PoA
  const payloadRevoke = JSON.stringify({
    reason: 'k6 load test revocation',
  });

  // Note: Admin API usually requires auth/tenant headers. 
  // For load testing against a dev/local setup without strict auth enforcement (or if using the dev bypass),
  // we might need to add headers. Assuming non-strict or basic auth for now based on existing tests.
  // Adding standard tenant header just in case.
  const paramsRevoke = {
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': 'default-tenant',
      'X-User-ID': 'load-test-admin'
    },
  };

  const resRevoke = http.post(`${BASE_URL}/api/admin/poa/${poaID}/revoke`, payloadRevoke, paramsRevoke);
  const duration = Date.now() - start;

  const success = check(resRevoke, {
    'Revocation status 200': (r) => r.status === 200,
    'Revocation successful message': (r) => r.body.includes('revoked successfully'),
  });

  if (!success) {
    errorRate.add(1);
  }

  // We can reuse a custom metric or add a new one. Using requestsPerSecond for now.
  requestsPerSecond.add(1);
}

export function testAdminAudit() {
  const start = Date.now();
  const payload = JSON.stringify({
    format: 'json',
    dateRange: 'last-1h',
    compressed: false
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': 'default-tenant'
    },
  };

  const res = http.post(`${BASE_URL}/api/admin/audit/export`, payload, params);
  const duration = Date.now() - start;

  const success = check(res, {
    'Audit export accepted (202) or created': (r) => r.status === 202 || r.status === 200 || r.status === 201,
  });

  if (!success) {
    errorRate.add(1);
  }

  requestsPerSecond.add(1);
}


export function testAuditPersistence() {
  // 1. Get initial metric count
  const resBefore = http.get(`${BASE_URL}/api/admin/metrics/prometheus`);

  let initialCount = 0;
  if (resBefore.status === 200) {
    const match = resBefore.body.match(/agentauth_audit_events_total\s+(\d+)/);
    if (match) {
      initialCount = parseInt(match[1], 10);
    }
  }

  // 2. Perform action to generate audit log (Create PoA)
  testPoACreation();

  // 3. Get new metric count (allow delay for async write/scrape)
  sleep(1);
  const resAfter = http.get(`${BASE_URL}/api/admin/metrics/prometheus`);

  let finalCount = 0;
  if (resAfter.status === 200) {
    const match = resAfter.body.match(/agentauth_audit_events_total\s+(\d+)/);
    if (match) {
      finalCount = parseInt(match[1], 10);
    }
  }

  // 4. Verification
  const success = check(resAfter, {
    'Audit metric increased': () => finalCount > initialCount,
  });

  if (!success) {
    console.warn(`Audit Persistence Failed: ${initialCount} -> ${finalCount}`);
    errorRate.add(1);
  }
}

// Main test execution
export default function () {
  const rand = Math.random();

  if (DEGRADED_MODE) {
    if (rand < 0.5) {
      testHealthCheck();
    } else {
      testMCPResourcesList();
    }
  } else {
    // Full Persistence Mode
    if (rand < 0.10) {
      testHealthCheck();
    } else if (rand < 0.25) {
      testPoACreation();
    } else if (rand < 0.35) {
      testAuthorization();
    } else if (rand < 0.45) {
      testRevocation();
    } else if (rand < 0.50) {
      testAdminAudit();
    } else if (rand < 0.55) {
      testAuditPersistence(); // Verification of DB write
    } else if (rand < 0.65) {
      testBrazilCPFValidation();
    } else if (rand < 0.75) {
      testCanadaSINValidation();
    } else if (rand < 0.85) {
      testMexicoCURPValidation();
    } else {
      testMCPResourcesList();
    }
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
