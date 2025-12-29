# Phase 27: Extended Load Testing Task Log

## Objective
Enhance load testing capabilities to include coverage for Revocation and Admin Audit scenarios, and establish a framework for long-duration "soak" testing to verify system stability.

## Key Actions Taken

1.  **Script Enhancement (`tests/load/k6-load-test.js`)**:
    *   Added `testRevocation` scenario: Simulates the full lifecycle of creating a Power of Attorney (PoA) and immediately revoking it via the Admin API.
    *   Added `testAdminAudit` scenario: Tests the asynchronous Audit Export API (`/api/admin/audit/export`).
    *   Updated the execution mix to include these new scenarios (10% and 5% probability respectively).
    *   Added dynamic configuration: `DURATION`, `RAMP_DURATION`, and `STAGE_DURATION` can now be injected via environment variables for flexible testing (e.g., short verification runs vs. full stress tests).
    *   **Fix**: Updated `BASE_URL` logic to support IPv6 (`[::1]:8080`) correctly to resolve connectivity issues.
    *   **Fix**: Updated MCP endpoint path to `/api/v1/gauth/mcp/servers`.

2.  **Soak Test Creation (`tests/load/soak-test.js`)**:
    *   Created a new dedicated script for stability testing.
    *   Implements a "Mixed Workload" to stress database and memory consistently over time.
    *   Includes a `DEGRADED_MODE` toggle: When enabled (`DEGRADED_MODE=true`), the script gracefully downgrades to only perform health checks. This ensures the test suite remains valid and verifiable even in environments without a full database (e.g., CI/CD or local dev modes).

3.  **Environment Setup**:
    *   Verified backend health (`/healthz`).
    *   Installed `k6` via Homebrew (`brew install k6`).
    *   Resolved `connection refused` errors by identifying the server's IPv6 binding and adjusting test scripts accordingly.

## Verification Results

Tests were verified in **Degraded Mode** (Backend running without Database):

*   **Load Test**: `k6 run -e DURATION=10s tests/load/k6-load-test.js`
    *   **Result**: 0% Failure Rate.
    *   **Validated**: `/healthz` and `/api/v1/gauth/mcp/servers` endpoints.

*   **Soak Test**: `k6 run -e DURATION=10s -e DEGRADED_MODE=true tests/load/soak-test.js`
    *   **Result**: 0% Failure Rate.
    *   **Validated**: Script infrastructure, argument parsing, and basic connectivity.

## Next Steps
*   **Full Environment Test**: Executing these scripts against a Staging or Production environment with a live database will validate the database-dependent scenarios (`PoA Creation`, `Revocation`, `Audit Export`).
*   **CI Integration**: The test scripts are now ready to be integrated into a CI pipeline (using the `DEGRADED_MODE` flag where appropriate).
