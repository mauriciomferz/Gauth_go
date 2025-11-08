// Package arbitration provides webhook support for external dispute resolution systems (RFC 0111 sec4.item3).
package arbitration

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// WebhookConfig configures webhook delivery for arbitration events.
type WebhookConfig struct {
	// URL is the endpoint to send webhook payloads to
	URL string
	// Secret is used for HMAC signature verification (optional)
	Secret string
	// Timeout for HTTP requests (default: 10s)
	Timeout time.Duration
	// MaxRetries for failed deliveries (default: 3)
	MaxRetries int
	// RetryBackoff base duration (default: 1s, exponential backoff)
	RetryBackoff time.Duration
	// Events to subscribe to (default: all events)
	Events []WebhookEventType
}

// WebhookEventType identifies types of arbitration events.
type WebhookEventType string

const (
	EventDisputeCreated   WebhookEventType = "dispute.created"
	EventDisputeResolved  WebhookEventType = "dispute.resolved"
	EventDisputeEscalated WebhookEventType = "dispute.escalated"
	EventRuleRegistered   WebhookEventType = "rule.registered"
)

// WebhookPayload is sent to external arbitration systems.
type WebhookPayload struct {
	Event     WebhookEventType       `json:"event"`
	Timestamp time.Time              `json:"timestamp"`
	Dispute   *Dispute               `json:"dispute,omitempty"`
	Result    *ArbitrationResult     `json:"result,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// WebhookDelivery tracks webhook delivery attempts.
type WebhookDelivery struct {
	ID          string
	Payload     WebhookPayload
	Attempts    int
	LastAttempt time.Time
	Status      string // "pending", "delivered", "failed"
	Error       string
}

// WebhookClient handles webhook delivery with retries and signatures.
type WebhookClient struct {
	mu         sync.RWMutex
	config     WebhookConfig
	client     *http.Client
	deliveries map[string]*WebhookDelivery // Track delivery status
	metrics    *WebhookMetrics
}

// WebhookMetrics tracks webhook delivery statistics.
type WebhookMetrics struct {
	mu                 sync.RWMutex
	TotalSent          int64
	SuccessfulDelivery int64
	FailedDelivery     int64
	AverageLatencyMs   float64
	EventBreakdown     map[WebhookEventType]int64
}

// NewWebhookClient creates a webhook client from configuration.
// If config.URL is empty, webhook delivery is disabled (all Send operations are no-ops).
func NewWebhookClient(config WebhookConfig) *WebhookClient {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryBackoff == 0 {
		config.RetryBackoff = time.Second
	}
	if len(config.Events) == 0 {
		// Subscribe to all events by default
		config.Events = []WebhookEventType{
			EventDisputeCreated,
			EventDisputeResolved,
			EventDisputeEscalated,
			EventRuleRegistered,
		}
	}

	return &WebhookClient{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		deliveries: make(map[string]*WebhookDelivery),
		metrics: &WebhookMetrics{
			EventBreakdown: make(map[WebhookEventType]int64),
		},
	}
}

// NewWebhookClientFromEnv creates a webhook client from environment variables:
// - GAUTH_ARBITRATION_WEBHOOK_URL: Webhook endpoint URL
// - GAUTH_ARBITRATION_WEBHOOK_SECRET: HMAC secret for signature verification
// - GAUTH_ARBITRATION_WEBHOOK_TIMEOUT: Request timeout (default: 10s)
// - GAUTH_ARBITRATION_WEBHOOK_MAX_RETRIES: Max retry attempts (default: 3)
func NewWebhookClientFromEnv() *WebhookClient {
	url := os.Getenv("GAUTH_ARBITRATION_WEBHOOK_URL")
	secret := os.Getenv("GAUTH_ARBITRATION_WEBHOOK_SECRET")
	timeout := 10 * time.Second
	if t := os.Getenv("GAUTH_ARBITRATION_WEBHOOK_TIMEOUT"); t != "" {
		if parsed, err := time.ParseDuration(t); err == nil {
			timeout = parsed
		}
	}
	maxRetries := 3
	if r := os.Getenv("GAUTH_ARBITRATION_WEBHOOK_MAX_RETRIES"); r != "" {
		if _, err := fmt.Sscanf(r, "%d", &maxRetries); err == nil && maxRetries > 0 {
			_ = err // Use parsed value
		}
	}

	return NewWebhookClient(WebhookConfig{
		URL:          url,
		Secret:       secret,
		Timeout:      timeout,
		MaxRetries:   maxRetries,
		RetryBackoff: time.Second,
	})
}

// Send delivers a webhook payload with retries and signature.
// Returns nil if webhook delivery is disabled (empty URL).
func (w *WebhookClient) Send(ctx context.Context, payload WebhookPayload) error {
	if w.config.URL == "" {
		return nil // Webhook disabled, skip delivery
	}

	// Check if subscribed to this event
	if !w.isSubscribed(payload.Event) {
		return nil // Not subscribed to this event
	}

	// Generate delivery ID
	deliveryID := fmt.Sprintf("%s-%d", payload.Event, time.Now().UnixNano())

	// Track delivery
	delivery := &WebhookDelivery{
		ID:      deliveryID,
		Payload: payload,
		Status:  "pending",
	}
	w.mu.Lock()
	w.deliveries[deliveryID] = delivery
	w.mu.Unlock()

	// Update metrics
	w.metrics.mu.Lock()
	w.metrics.TotalSent++
	w.metrics.EventBreakdown[payload.Event]++
	w.metrics.mu.Unlock()

	// Attempt delivery with retries
	var lastErr error
	for attempt := 0; attempt <= w.config.MaxRetries; attempt++ {
		delivery.Attempts = attempt + 1
		delivery.LastAttempt = time.Now()

		start := time.Now()
		err := w.deliver(ctx, payload)
		latency := time.Since(start)

		if err == nil {
			// Success
			delivery.Status = "delivered"
			w.mu.Lock()
			w.deliveries[deliveryID] = delivery
			w.mu.Unlock()

			w.metrics.mu.Lock()
			w.metrics.SuccessfulDelivery++
			// Update average latency (simple moving average)
			if w.metrics.AverageLatencyMs == 0 {
				w.metrics.AverageLatencyMs = float64(latency.Milliseconds())
			} else {
				w.metrics.AverageLatencyMs = (w.metrics.AverageLatencyMs + float64(latency.Milliseconds())) / 2
			}
			w.metrics.mu.Unlock()

			return nil
		}

		lastErr = err

		// Exponential backoff before retry
		if attempt < w.config.MaxRetries {
			//nolint:gosec // G115: small retry attempt value, safe conversion
			backoff := w.config.RetryBackoff * time.Duration(1<<uint(attempt))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				// Continue to next retry
			}
		}
	}

	// All retries failed
	delivery.Status = "failed"
	delivery.Error = lastErr.Error()
	w.mu.Lock()
	w.deliveries[deliveryID] = delivery
	w.mu.Unlock()

	w.metrics.mu.Lock()
	w.metrics.FailedDelivery++
	w.metrics.mu.Unlock()

	return fmt.Errorf("webhook delivery failed after %d attempts: %w", w.config.MaxRetries+1, lastErr)
}

// deliver performs a single webhook HTTP POST with signature.
func (w *WebhookClient) deliver(ctx context.Context, payload WebhookPayload) error {
	// Serialize payload
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", w.config.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GAuth-Arbitration-Webhook/1.0")
	req.Header.Set("X-GAuth-Event", string(payload.Event))
	req.Header.Set("X-GAuth-Timestamp", payload.Timestamp.Format(time.RFC3339))

	// Generate HMAC signature if secret is configured
	if w.config.Secret != "" {
		signature := w.generateSignature(body)
		req.Header.Set("X-GAuth-Signature", signature)
	}

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// generateSignature computes HMAC-SHA256 signature for webhook payload.
func (w *WebhookClient) generateSignature(body []byte) string {
	mac := hmac.New(sha256.New, []byte(w.config.Secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature validates an incoming webhook signature (for bidirectional webhooks).
func (w *WebhookClient) VerifySignature(body []byte, signature string) bool {
	if w.config.Secret == "" {
		return true // No secret configured, skip verification
	}
	expected := w.generateSignature(body)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// isSubscribed checks if the client is subscribed to a specific event type.
func (w *WebhookClient) isSubscribed(event WebhookEventType) bool {
	for _, subscribed := range w.config.Events {
		if subscribed == event {
			return true
		}
	}
	return false
}

// GetMetrics returns current webhook delivery metrics.
func (w *WebhookClient) GetMetrics() WebhookMetrics {
	w.metrics.mu.RLock()
	defer w.metrics.mu.RUnlock()

	// Deep copy event breakdown
	eventBreakdown := make(map[WebhookEventType]int64)
	for k, v := range w.metrics.EventBreakdown {
		eventBreakdown[k] = v
	}

	return WebhookMetrics{
		TotalSent:          w.metrics.TotalSent,
		SuccessfulDelivery: w.metrics.SuccessfulDelivery,
		FailedDelivery:     w.metrics.FailedDelivery,
		AverageLatencyMs:   w.metrics.AverageLatencyMs,
		EventBreakdown:     eventBreakdown,
	}
}

// GetDelivery retrieves delivery status for a specific delivery ID.
func (w *WebhookClient) GetDelivery(deliveryID string) (*WebhookDelivery, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	delivery, ok := w.deliveries[deliveryID]
	return delivery, ok
}

// ListDeliveries returns all tracked webhook deliveries.
func (w *WebhookClient) ListDeliveries() []*WebhookDelivery {
	w.mu.RLock()
	defer w.mu.RUnlock()

	deliveries := make([]*WebhookDelivery, 0, len(w.deliveries))
	for _, d := range w.deliveries {
		deliveries = append(deliveries, d)
	}
	return deliveries
}

// WebhookEnabledArbiter wraps an Arbiter with webhook notifications.
type WebhookEnabledArbiter struct {
	arbiter Arbiter
	webhook *WebhookClient
}

// NewWebhookEnabledArbiter wraps an arbiter with webhook support.
func NewWebhookEnabledArbiter(arbiter Arbiter, webhook *WebhookClient) *WebhookEnabledArbiter {
	return &WebhookEnabledArbiter{
		arbiter: arbiter,
		webhook: webhook,
	}
}

// Arbitrate resolves a dispute and sends webhook notification.
func (w *WebhookEnabledArbiter) Arbitrate(ctx context.Context, dispute *Dispute) (*ArbitrationResult, error) {
	// Send dispute created webhook
	_ = w.webhook.Send(ctx, WebhookPayload{
		Event:     EventDisputeCreated,
		Timestamp: time.Now(),
		Dispute:   dispute,
	})

	// Perform arbitration
	result, err := w.arbiter.Arbitrate(ctx, dispute)
	if err != nil {
		return nil, err
	}

	// Send resolution webhook
	event := EventDisputeResolved
	if result.RequiresEscalation {
		event = EventDisputeEscalated
	}

	_ = w.webhook.Send(ctx, WebhookPayload{
		Event:     event,
		Timestamp: time.Now(),
		Dispute:   dispute,
		Result:    result,
	})

	return result, nil
}

// RegisterRule registers a rule and sends webhook notification.
func (w *WebhookEnabledArbiter) RegisterRule(name string, priority int, handler RuleHandler) error {
	err := w.arbiter.RegisterRule(name, priority, handler)
	if err != nil {
		return err
	}

	// Send rule registered webhook
	_ = w.webhook.Send(context.Background(), WebhookPayload{
		Event:     EventRuleRegistered,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"rule_name": name,
			"priority":  priority,
		},
	})

	return nil
}

// GetDispute delegates to wrapped arbiter.
func (w *WebhookEnabledArbiter) GetDispute(ctx context.Context, disputeID string) (*Dispute, error) {
	return w.arbiter.GetDispute(ctx, disputeID)
}

// ListDisputes delegates to wrapped arbiter.
func (w *WebhookEnabledArbiter) ListDisputes(ctx context.Context, filter DisputeFilter) ([]*Dispute, error) {
	return w.arbiter.ListDisputes(ctx, filter)
}
