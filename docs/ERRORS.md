# Error Catalog

Central source of truth for public error codes exposed by GAuth APIs. Each code includes HTTP mapping, severity, and remediation guidance. This file is generated / maintained alongside the centralized registry in `internal/errors/catalog.go`.

| Code | HTTP | Category | Severity | Retryable | Description | Remediation |
|------|------|----------|----------|-----------|-------------|------------|
| UNAUTHENTICATED | 401 | auth | warning | true | Caller did not present valid authentication credentials. | Obtain and include a valid token or credentials. |
| UNAUTHORIZED | 403 | auth | warning | false | Caller lacks required permissions or scope. | Request elevated scope or adjust policy grants. |
| INVALID_TOKEN | 401 | auth | error | true | Presented token is malformed or fails cryptographic verification. | Reissue token and verify signing domain freshness. |
| EXPIRED_TOKEN | 401 | auth | info | true | Token has passed its expiration timestamp. | Acquire a new token before retrying. |
| VALIDATION_ERROR | 400 | validation | warning | true | Generic validation failure. | Correct the request fields per schema. |
| INVALID_REQUEST | 400 | validation | warning | true | Request structurally invalid (schema mismatch). | Consult OpenAPI schema and adjust payload format. |
| MISSING_FIELD | 400 | validation | warning | true | Required field absent in request payload. | Include all mandatory fields. |
| NOT_FOUND | 404 | system | info | false | Referenced resource does not exist. | Verify identifier correctness or create resource first. |
| CONFLICT | 409 | system | warning | true | Operation conflicts with current resource state. | Reload latest state and retry with updated preconditions. |
| RATE_LIMIT | 429 | rate_limit | warning | true | Caller exceeded allocated rate limit window. | Throttle requests; respect retry-after guidance. |
| TIMEOUT | 504 | network | warning | true | Upstream or internal operation timed out. | Retry with backoff; investigate latency metrics. |
| INTERNAL_ERROR | 500 | system | error | true | Unexpected internal server error. | Capture diagnostics (traceId) and escalate if persistent. |
| SERVICE_DOWN | 503 | network | critical | true | Dependent service unavailable or degraded. | Failover or wait for service restoration; monitor health endpoint. |
| NETWORK_ERROR | 502 | network | warning | true | Network path failure or upstream bad gateway. | Retry after transient network stabilization. |
| INSUFFICIENT_SCOPE | 403 | policy | warning | false | Token scope insufficient for requested operation. | Request broadened scope or adjust token issuance policy. |

## Error Envelope

All error responses SHOULD conform to a standard envelope to enable consistent client handling and correlation:

```json
{
  "code": "INVALID_REQUEST",
  "message": "request structurally invalid (schema mismatch)",
  "details": {
    "timestamp": "2025-10-28T12:34:56Z",
    "request_id": "abc-123",
    "http_method": "POST",
    "http_path": "/api/v1/token/issue",
    "additional_info": {
      "field_errors": [
        { "field": "subject", "message": "must be non-empty" }
      ]
    }
  }
}
```

Fields:

- `code`: Stable error code (see table above) for programmatic handling.
- `message`: Human-readable short description (avoid leaking sensitive details).
- `details.timestamp`: Time of error generation in RFC3339.
- `details.request_id`: Trace / correlation identifier from incoming context.
- `details.http_method` / `details.http_path`: Request metadata aiding debugging.
- `details.additional_info`: Arbitrary structured payload; MUST avoid PII.

## Versioning & Stability

Error codes are append-only. Deprecations follow the taxonomy ADR pattern: mark as deprecated in docs, retain mapping for at least one minor release, then gate behind compatibility flag before removal.

## Mapping Logic

Server handlers SHOULD derive HTTP status via `errorscatalog.HTTPStatusFor(code)`; unknown codes default to 500.

## Remediation & Observability

Severity drives logging level: info (debug-level), warning (structured warn), error (alert candidate), critical (page). Retry behavior informs client-level backoff strategies.
