import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time');
const requestsCounter = new Counter('total_requests');

const DURATION = __ENV.DURATION || '1h';

// Test configuration
export const options = {
    scenarios: {
        // Soak test: Constant load for extended duration
        soak: {
            executor: 'constant-vus',
            vus: 20, // Moderate load
            duration: DURATION, // Standard soak duration (can be shortened for dry-runs)
        },
    },
    thresholds: {
        http_req_duration: ['p(95)<800'], // Slightly happier threshold for long runs
        http_req_failed: ['rate<0.02'], // Max 2% errors allowed
        'errors': ['rate<0.05'],
    },
};

const BASE_URL = __ENV.BASE_URL || 'http://[::1]:8080';

// Simplified Test Functions (Reuse core logic)
function testHealthCheck() {
    const res = http.get(`${BASE_URL}/healthz`);
    check(res, { 'status 200': (r) => r.status === 200 });
    if (res.status !== 200) errorRate.add(1);
    responseTime.add(res.timings.duration);
    requestsCounter.add(1);
}

function testMixedWorkload() {
    // Combined create/revoke flow to stress DB consistency over time
    const start = Date.now();
    const payload = JSON.stringify({
        grantor: `soak-${Date.now()}@example.com`,
        grantee: 'grantee@example.com',
        scope: ['read'],
        valid_from: new Date().toISOString(),
        valid_until: new Date(Date.now() + 3600 * 1000).toISOString(),
    });

    const res = http.post(`${BASE_URL}/api/v1/beta/poa`, payload, { headers: { 'Content-Type': 'application/json' } });

    if (check(res, { 'created': (r) => r.status === 200 }) {
        const body = JSON.parse(res.body);
        const id = body.poa?.id;
        if (id) {
            // Revoke immediately to keep DB size management reasonable
            const revPayload = JSON.stringify({ reason: 'soak cleanup' });
            http.post(`${BASE_URL}/api/admin/poa/${id}/revoke`, revPayload, {
                headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'default', 'X-User-ID': 'soak-user' }
            });
        }
    } else {
        errorRate.add(1);
    }
    responseTime.add(Date.now() - start);
    requestsCounter.add(1);
}

const DEGRADED_MODE = __ENV.DEGRADED_MODE === 'true';

export default function () {
    const rand = Math.random();
    // Validates basic server stability if constrained
    if (DEGRADED_MODE || rand < 0.3) {
        testHealthCheck();
    } else {
        testMixedWorkload();
    }
    sleep(1); // Pace the soak test
}

export function handleSummary(data) {
    return {
        'soak-test-summary.json': JSON.stringify(data, null, 2),
        stdout: textSummary(data, { indent: '→', enableColors: true }),
    };
}

function textSummary(data, options) {
    // Minimal summary for stdout
    return `Soak Test Completed.
    Requests: ${data.metrics.http_reqs.values.count}
    Failures: ${data.metrics.http_req_failed.values.count}
    P95 Latency: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms
    `;
}
