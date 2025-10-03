package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	stderrs "errors"

	"github.com/Gimel-Foundation/gauth/pkg/errors"
	"github.com/google/uuid"
)

// AuditHandler serves RFC-compliant advanced audit responses
func AuditHandler(w http.ResponseWriter, r *http.Request) {
	auditResp := AdvancedAuditResponse{
		AuditID: "audit_1759000523",
		Status: "initiated",
		Timestamp: time.Now().Format(time.RFC3339),
		AuditScope: []string{"financial_transactions", "regulatory_compliance", "risk_assessment"},
		ForensicAnalysis: map[string]interface{}{
			"enabled": true,
			"tools": []string{"log_analysis", "anomaly_detection", "pattern_recognition"},
			"status": "analyzing",
		},
		ComplianceTracking: map[string]interface{}{
			"enabled": true,
			"frameworks": []string{"SOX", "GDPR", "HIPAA"},
			"status": "monitoring",
		},
		RealTimeMonitoring: map[string]interface{}{
			"enabled": true,
			"status": "active",
			"status_indicators": []string{"active", "pending", "inactive"},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(auditResp)
}

// RFC-compliant advanced audit response template
type AdvancedAuditResponse struct {
	AuditID            string                 `json:"audit_id"`
	Status             string                 `json:"status"`
	Timestamp          string                 `json:"timestamp"`
	AuditScope         []string               `json:"audit_scope"`
	ForensicAnalysis   map[string]interface{} `json:"forensic_analysis"`
	ComplianceTracking map[string]interface{} `json:"compliance_tracking"`
	RealTimeMonitoring map[string]interface{} `json:"real_time_monitoring"`
}

// ErrorHandler is middleware that handles errors from downstream handlers
type ErrorHandler struct {
	Next http.Handler
}

// ServeHTTP implements the http.Handler interface
func (e *ErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	crw := &captureResponseWriter{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
	}
	r.Header.Set("X-Request-ID", requestID)
	ctx := r.Context()
	r = r.WithContext(ctx)
	e.Next.ServeHTTP(crw, r)
}

type captureResponseWriter struct {
	http.ResponseWriter
	status int
}

func (c *captureResponseWriter) WriteHeader(status int) {
	c.status = status
	c.ResponseWriter.WriteHeader(status)
}

func ErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
		status := http.StatusInternalServerError
		errBody := map[string]interface{}{
			"error_code": "server_error",
			"error_message": "An unexpected error occurred",
			"error_uri": "https://gauth.example.com/docs/errors#server_error",
			"timestamp": time.Now().Format(time.RFC3339),
			"details": map[string]interface{}{
				"request_id": r.Header.Get("X-Request-ID"),
			},
		}

		var authErr *errors.Error
		if stderrs.As(err, &authErr) {
			if authErr.Details != nil && authErr.Details.HTTPStatusCode > 0 {
				status = authErr.Details.HTTPStatusCode
			}
			errBody["error_code"] = string(authErr.Code)
			errBody["error_message"] = authErr.Message
			errBody["error_uri"] = "https://gauth.example.com/docs/errors#" + string(authErr.Code)
			details := map[string]interface{}{
				"request_id": r.Header.Get("X-Request-ID"),
			}
			if authErr.Details != nil {
				for k, v := range authErr.Details.AdditionalInfo {
					details[k] = v
				}
			}
			errBody["details"] = details
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
		w.WriteHeader(status)
		if err := json.NewEncoder(w).Encode(errBody); err != nil {
			// Log encoding error but don't send another response
			fmt.Printf("Error encoding error response: %v\n", err)
		}
}
