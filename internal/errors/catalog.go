// Package errorscatalog provides a centralized registry of public error codes,
// their HTTP status mappings, severity, and remediation guidance. This enables
// consistent API responses and a single source of truth for documentation and
// OpenAPI error schemas.
package errorscatalog

import (
	"net/http"
	"sort"

	gerrs "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/errors"
)

// Category represents a coarse grouping for error codes to enable filtering
// in docs and metrics.
type Category string

const (
	CatAuth       Category = "auth"
	CatValidation Category = "validation"
	CatSystem     Category = "system"
	CatRateLimit  Category = "rate_limit"
	CatNetwork    Category = "network"
	CatPolicy     Category = "policy"
)

// Severity represents impact level used for prioritization in logs / alerts.
type Severity string

const (
	SevInfo     Severity = "info"
	SevWarning  Severity = "warning"
	SevError    Severity = "error"
	SevCritical Severity = "critical"
)

// Entry defines catalog metadata for a single error code.
type Entry struct {
	Code        gerrs.ErrorCode `json:"code"`
	HTTPStatus  int             `json:"http_status"`
	Category    Category        `json:"category"`
	Severity    Severity        `json:"severity"`
	Description string          `json:"description"`
	Remediation string          `json:"remediation"`
	Retryable   bool            `json:"retryable"`
}

// catalog holds all registered entries keyed by code.
var catalog = map[gerrs.ErrorCode]Entry{}

// init builds the catalog. Keep this ordered logically; docs will render sorted by code.
func init() {
	add(Entry{Code: gerrs.ErrCodeUnauthenticated, HTTPStatus: http.StatusUnauthorized, Category: CatAuth, Severity: SevWarning, Description: "Caller did not present valid authentication credentials.", Remediation: "Obtain and include a valid token or credentials.", Retryable: true})
	add(Entry{Code: gerrs.ErrCodeUnauthorized, HTTPStatus: http.StatusForbidden, Category: CatAuth, Severity: SevWarning, Description: "Caller lacks required permissions or scope.", Remediation: "Request elevated scope or adjust policy grants.", Retryable: false})
	add(Entry{Code: gerrs.ErrCodeInvalidToken, HTTPStatus: http.StatusUnauthorized, Category: CatAuth, Severity: SevError, Description: "Presented token is malformed or fails cryptographic verification.", Remediation: "Reissue token and verify signing domain freshness.", Retryable: true})
	add(Entry{Code: gerrs.ErrCodeExpiredToken, HTTPStatus: http.StatusUnauthorized, Category: CatAuth, Severity: SevInfo, Description: "Token has passed its expiration timestamp.", Remediation: "Acquire a new token before retrying.", Retryable: true})
	add(Entry{Code: gerrs.ErrCodeValidation, HTTPStatus: http.StatusBadRequest, Category: CatValidation, Severity: SevWarning, Description: "Generic validation failure.", Remediation: "Correct the request fields per schema.", Retryable: true})
	add(Entry{Code: gerrs.ErrCodeInvalidRequest, HTTPStatus: http.StatusBadRequest, Category: CatValidation, Severity: SevWarning, Description: "Request structurally invalid (schema mismatch).", Remediation: "Consult OpenAPI schema and adjust payload format.", Retryable: true})
	add(Entry{Code: gerrs.ErrCodeMissingField, HTTPStatus: http.StatusBadRequest, Category: CatValidation, Severity: SevWarning, Description: "Required field absent in request payload.", Remediation: "Include all mandatory fields.", Retryable: true})
	add(Entry{Code: gerrs.ErrCodeNotFound, HTTPStatus: http.StatusNotFound, Category: CatSystem, Severity: SevInfo, Description: "Referenced resource does not exist.", Remediation: "Verify identifier correctness or create resource first.", Retryable: false})
	add(Entry{Code: gerrs.ErrCodeConflict, HTTPStatus: http.StatusConflict, Category: CatSystem, Severity: SevWarning, Description: "Operation conflicts with current resource state.", Remediation: "Reload latest state and retry with updated preconditions.", Retryable: true})
	add(Entry{Code: gerrs.ErrCodeRateLimit, HTTPStatus: http.StatusTooManyRequests, Category: CatRateLimit, Severity: SevWarning, Description: "Caller exceeded allocated rate limit window.", Remediation: "Throttle requests; respect retry-after guidance.", Retryable: true})
	add(Entry{Code: gerrs.ErrCodeTimeout, HTTPStatus: http.StatusGatewayTimeout, Category: CatNetwork, Severity: SevWarning, Description: "Upstream or internal operation timed out.", Remediation: "Retry with backoff; investigate latency metrics.", Retryable: true})
	add(Entry{Code: gerrs.ErrCodeInternal, HTTPStatus: http.StatusInternalServerError, Category: CatSystem, Severity: SevError, Description: "Unexpected internal server error.", Remediation: "Capture diagnostics (traceId) and escalate if persistent.", Retryable: true})
	add(Entry{Code: gerrs.ErrCodeServiceDown, HTTPStatus: http.StatusServiceUnavailable, Category: CatNetwork, Severity: SevCritical, Description: "Dependent service unavailable or degraded.", Remediation: "Failover or wait for service restoration; monitor health endpoint.", Retryable: true})
	add(Entry{Code: gerrs.ErrCodeNetworkError, HTTPStatus: http.StatusBadGateway, Category: CatNetwork, Severity: SevWarning, Description: "Network path failure or upstream bad gateway.", Remediation: "Retry after transient network stabilization.", Retryable: true})
	add(Entry{Code: gerrs.ErrCodeInsufficientScope, HTTPStatus: http.StatusForbidden, Category: CatPolicy, Severity: SevWarning, Description: "Token scope insufficient for requested operation.", Remediation: "Request broadened scope or adjust token issuance policy.", Retryable: false})
}

// add inserts an entry into the catalog (internal helper).
func add(e Entry) { catalog[e.Code] = e }

// Lookup returns the catalog entry for a given error code and a boolean indicating presence.
func Lookup(code gerrs.ErrorCode) (Entry, bool) {
	e, ok := catalog[code]
	return e, ok
}

// All returns a sorted slice of all entries for rendering in docs / APIs.
func All() []Entry {
	out := make([]Entry, 0, len(catalog))
	for _, e := range catalog {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// HTTPStatusFor returns the mapped HTTP status for the provided error code,
// or 500 if unmapped.
func HTTPStatusFor(code gerrs.ErrorCode) int {
	if e, ok := Lookup(code); ok {
		return e.HTTPStatus
	}
	return http.StatusInternalServerError
}
