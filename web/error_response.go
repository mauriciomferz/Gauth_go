package web

import (
	"time"

	"github.com/gin-gonic/gin"
	errorscatalog "github.com/mauriciomferz/AgentAuth/internal/errors"
	gerrs "github.com/mauriciomferz/AgentAuth/pkg/errors"
)

// ErrorResponse standardizes API error payloads and maps them to RFC references.
// Fields:
//
//	Success: always false for errors.
//	Code: machine-readable stable error code (snake_case).
//	Error: legacy short error string retained for backward compatibility.
//	Message: optional human-readable expansion.
//	RFCRef: optional RFC clause reference (e.g. "AAP-001:replay_protection").
//	Detail: arbitrary extra structured data.
//
// ErrorEnvelope aligns with OpenAPI components.schemas.ErrorEnvelope.
// "details" is expanded with standard fields + user-provided detail (additional_info).
type ErrorEnvelope struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Source    string                 `json:"source,omitempty"`
	Retryable bool                   `json:"retryable,omitempty"`
	// Legacy compatibility fields
	Error   string `json:"error,omitempty"`
	Success bool   `json:"success"`
}

// Backward compatibility alias for older tests referencing ErrorResponse.
type ErrorResponse = ErrorEnvelope

// respondError centralizes error rendering so refactors (e.g. adding trace IDs) require changing only one place.
// "errStr" keeps backward compatibility with older clients expecting the short field while "code" is the new stable taxonomy.
func respondError(c *gin.Context, status int, code, errStr, msg, rfcRef string, detail interface{}) {
	// Lookup retryable hint from catalog if exists.
	retryable := false
	if e, ok := errorscatalog.Lookup(gerrs.ErrorCode(code)); ok {
		retryable = e.Retryable
	}
	// Build details map
	det := map[string]interface{}{
		"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
		"http_method": c.Request.Method,
		"http_path":   c.Request.URL.Path,
	}
	if rid := c.GetString("request_id"); rid != "" {
		det["request_id"] = rid
	}
	if rfcRef != "" {
		det["rfc_ref"] = rfcRef
	}
	if detail != nil {
		det["additional_info"] = detail
	}
	envelope := ErrorEnvelope{Code: code, Message: msg, Details: det, Retryable: retryable, Source: "agentauth", Error: errStr, Success: false}
	c.JSON(status, envelope)
}
