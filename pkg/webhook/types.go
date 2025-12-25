package webhook

import (
	"time"
)

// EventType represents the type of webhook event
type EventType string

const (
	// PoA Events
	EventPoACreated  EventType = "poa.created"
	EventPoAUpdated  EventType = "poa.updated"
	EventPoARevoked  EventType = "poa.revoked"
	EventPoAVerified EventType = "poa.verified"
	EventPoAExpired  EventType = "poa.expired"

	// Successor Events
	EventSuccessorActivated EventType = "successor.activated"

	// Delegation Events
	EventDelegationCreated EventType = "delegation.created"
	EventDelegationRevoked EventType = "delegation.revoked"

	// Dual Control Events
	EventDualControlApprovalRequired EventType = "dual_control.approval_required"
	EventDualControlApproved         EventType = "dual_control.approved"
	EventDualControlRejected         EventType = "dual_control.rejected"

	// Blockchain Events
	EventBlockchainSyncCompleted EventType = "blockchain.sync_completed"
	EventBlockchainSyncFailed    EventType = "blockchain.sync_failed"
)

// DeliveryStatus represents the status of a webhook delivery
type DeliveryStatus string

const (
	DeliveryStatusPending  DeliveryStatus = "pending"
	DeliveryStatusSuccess  DeliveryStatus = "success"
	DeliveryStatusFailed   DeliveryStatus = "failed"
	DeliveryStatusRetrying DeliveryStatus = "retrying"
)

// Webhook represents a webhook registration
type Webhook struct {
	ID              string            `json:"id" db:"id"`
	Name            string            `json:"name" db:"name"`
	URL             string            `json:"url" db:"url"`
	Secret          string            `json:"-" db:"secret"` // Never expose in JSON
	UserID          string            `json:"user_id" db:"user_id"`
	Enabled         bool              `json:"enabled" db:"enabled"`
	Events          []string          `json:"events" db:"events"`
	RetryCount      int               `json:"retry_count" db:"retry_count"`
	Timeout         int               `json:"timeout_seconds" db:"timeout_seconds"`
	Description     string            `json:"description,omitempty" db:"description"`
	Headers         map[string]string `json:"headers,omitempty" db:"headers"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
	LastTriggeredAt *time.Time        `json:"last_triggered_at,omitempty" db:"last_triggered_at"`
}

// WebhookDelivery represents a webhook delivery attempt
type WebhookDelivery struct {
	ID              string                 `json:"id" db:"id"`
	WebhookID       string                 `json:"webhook_id" db:"webhook_id"`
	EventID         string                 `json:"event_id" db:"event_id"`
	EventType       string                 `json:"event_type" db:"event_type"`
	URL             string                 `json:"url" db:"url"`
	HTTPMethod      string                 `json:"http_method" db:"http_method"`
	Headers         map[string]string      `json:"headers,omitempty" db:"headers"`
	Payload         map[string]interface{} `json:"payload" db:"payload"`
	StatusCode      *int                   `json:"status_code,omitempty" db:"status_code"`
	ResponseBody    string                 `json:"response_body,omitempty" db:"response_body"`
	ResponseHeaders map[string]string      `json:"response_headers,omitempty" db:"response_headers"`
	Status          DeliveryStatus         `json:"status" db:"status"`
	AttemptNumber   int                    `json:"attempt_number" db:"attempt_number"`
	MaxAttempts     int                    `json:"max_attempts" db:"max_attempts"`
	ErrorMessage    string                 `json:"error_message,omitempty" db:"error_message"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	SentAt          *time.Time             `json:"sent_at,omitempty" db:"sent_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	NextRetryAt     *time.Time             `json:"next_retry_at,omitempty" db:"next_retry_at"`
	DurationMs      *int                   `json:"duration_ms,omitempty" db:"duration_ms"`
}

// Event represents a webhook event to be dispatched
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    string                 `json:"user_id"`
	Data      map[string]interface{} `json:"data"`
}

// WebhookStats represents delivery statistics for a webhook
type WebhookStats struct {
	WebhookID            string     `json:"webhook_id" db:"webhook_id"`
	Name                 string     `json:"name" db:"name"`
	URL                  string     `json:"url" db:"url"`
	Enabled              bool       `json:"enabled" db:"enabled"`
	TotalDeliveries      int        `json:"total_deliveries" db:"total_deliveries"`
	SuccessfulDeliveries int        `json:"successful_deliveries" db:"successful_deliveries"`
	FailedDeliveries     int        `json:"failed_deliveries" db:"failed_deliveries"`
	RetryingDeliveries   int        `json:"retrying_deliveries" db:"retrying_deliveries"`
	AvgDurationMs        *float64   `json:"avg_duration_ms,omitempty" db:"avg_duration_ms"`
	LastDeliveryAt       *time.Time `json:"last_delivery_at,omitempty" db:"last_delivery_at"`
	SuccessRatePercent   *float64   `json:"success_rate_percent,omitempty" db:"success_rate_percent"`
}

// CreateWebhookRequest represents a request to create a webhook
type CreateWebhookRequest struct {
	Name        string            `json:"name" binding:"required"`
	URL         string            `json:"url" binding:"required,url"`
	Events      []string          `json:"events" binding:"required,min=1"`
	Description string            `json:"description,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	RetryCount  int               `json:"retry_count,omitempty"`     // Default: 3
	Timeout     int               `json:"timeout_seconds,omitempty"` // Default: 30
}

// UpdateWebhookRequest represents a request to update a webhook
type UpdateWebhookRequest struct {
	Name        *string           `json:"name,omitempty"`
	URL         *string           `json:"url,omitempty"`
	Events      []string          `json:"events,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"`
	Description *string           `json:"description,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	RetryCount  *int              `json:"retry_count,omitempty"`
	Timeout     *int              `json:"timeout_seconds,omitempty"`
}

// WebhookResponse represents a webhook in API responses (includes secret on creation)
type WebhookResponse struct {
	Webhook
	Secret string `json:"secret,omitempty"` // Only included on creation
}

// ListWebhooksQuery represents query parameters for listing webhooks
type ListWebhooksQuery struct {
	UserID  string `form:"user_id"`
	Enabled *bool  `form:"enabled"`
	Limit   int    `form:"limit"`
	Offset  int    `form:"offset"`
}

// ListDeliveriesQuery represents query parameters for listing deliveries
type ListDeliveriesQuery struct {
	WebhookID string         `form:"webhook_id"`
	EventType string         `form:"event_type"`
	Status    DeliveryStatus `form:"status"`
	Limit     int            `form:"limit"`
	Offset    int            `form:"offset"`
}
