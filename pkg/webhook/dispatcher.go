package webhook

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
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Dispatcher handles webhook event dispatching and delivery
type Dispatcher struct {
	db         *sqlx.DB
	manager    *Manager
	httpClient *http.Client
	workers    int
	queue      chan *deliveryJob
	stopCh     chan struct{}
}

type deliveryJob struct {
	webhook  *Webhook
	event    *Event
	delivery *WebhookDelivery
}

// NewDispatcher creates a new webhook dispatcher
func NewDispatcher(db *sqlx.DB, manager *Manager, workers int) *Dispatcher {
	if workers <= 0 {
		workers = 3 // Default to 3 workers
	}

	return &Dispatcher{
		db:      db,
		manager: manager,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		workers: workers,
		queue:   make(chan *deliveryJob, 100),
		stopCh:  make(chan struct{}),
	}
}

// Start starts the dispatcher workers
func (d *Dispatcher) Start(ctx context.Context) {
	for i := 0; i < d.workers; i++ {
		go d.worker(ctx, i)
	}

	// Start retry processor
	go d.retryProcessor(ctx)
}

// Stop stops the dispatcher
func (d *Dispatcher) Stop() {
	close(d.stopCh)
	close(d.queue)
}

// Dispatch dispatches an event to all matching webhooks
func (d *Dispatcher) Dispatch(ctx context.Context, event *Event) error {
	// Find all webhooks subscribed to this event type
	query := `
		SELECT id, name, url, secret, user_id, enabled, events,
			   retry_count, timeout_seconds, description, headers,
			   created_at, updated_at, last_triggered_at
		FROM webhooks
		WHERE enabled = true
		  AND $1 = ANY(events)
		  AND (user_id = $2 OR user_id = 'all')
	`

	webhooks := []Webhook{}
	err := d.db.SelectContext(ctx, &webhooks, query, string(event.Type), event.UserID)
	if err != nil {
		return fmt.Errorf("failed to find webhooks: %w", err)
	}

	// Create delivery records and queue jobs
	for i := range webhooks {
		webhook := &webhooks[i]

		delivery, err := d.createDelivery(ctx, webhook, event)
		if err != nil {
			// Log error but continue with other webhooks
			fmt.Printf("Failed to create delivery for webhook %s: %v\n", webhook.ID, err)
			continue
		}

		// Queue delivery job
		select {
		case d.queue <- &deliveryJob{
			webhook:  webhook,
			event:    event,
			delivery: delivery,
		}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// worker processes delivery jobs
func (d *Dispatcher) worker(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopCh:
			return
		case job, ok := <-d.queue:
			if !ok {
				return
			}
			d.deliverWebhook(ctx, job)
		}
	}
}

// deliverWebhook delivers a webhook to its endpoint
func (d *Dispatcher) deliverWebhook(ctx context.Context, job *deliveryJob) {
	startTime := time.Now()

	// Prepare payload
	payload, err := json.Marshal(job.event)
	if err != nil {
		d.updateDeliveryFailed(ctx, job.delivery, fmt.Sprintf("failed to marshal payload: %v", err), 0, 0)
		return
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", job.webhook.URL, bytes.NewReader(payload))
	if err != nil {
		d.updateDeliveryFailed(ctx, job.delivery, fmt.Sprintf("failed to create request: %v", err), 0, 0)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GAuth-Webhook/1.0")
	req.Header.Set("X-Webhook-ID", job.webhook.ID)
	req.Header.Set("X-Event-Type", string(job.event.Type))
	req.Header.Set("X-Event-ID", job.event.ID)
	req.Header.Set("X-Webhook-Signature", d.generateSignature(payload, job.webhook.Secret))

	// Add custom headers
	for key, value := range job.webhook.Headers {
		req.Header.Set(key, value)
	}

	// Update delivery status to sending
	now := time.Now()
	job.delivery.SentAt = &now

	// Send request
	resp, err := d.httpClient.Do(req)
	duration := int(time.Since(startTime).Milliseconds())

	if err != nil {
		d.handleDeliveryError(ctx, job, err, duration)
		return
	}
	defer resp.Body.Close()

	// Read response
	body, _ := io.ReadAll(resp.Body)
	responseBody := string(body)
	if len(responseBody) > 1000 {
		responseBody = responseBody[:1000] + "..." // Truncate large responses
	}

	// Update delivery based on status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		d.updateDeliverySuccess(ctx, job.delivery, resp.StatusCode, responseBody, duration)
		d.updateWebhookLastTriggered(ctx, job.webhook.ID)
	} else {
		d.handleDeliveryError(ctx, job, fmt.Errorf("HTTP %d: %s", resp.StatusCode, responseBody), duration)
	}
}

// createDelivery creates a new delivery record
func (d *Dispatcher) createDelivery(ctx context.Context, webhook *Webhook, event *Event) (*WebhookDelivery, error) {
	delivery := &WebhookDelivery{
		ID:            uuid.New().String(),
		WebhookID:     webhook.ID,
		EventID:       event.ID,
		EventType:     string(event.Type),
		URL:           webhook.URL,
		HTTPMethod:    "POST",
		Headers:       webhook.Headers,
		Payload:       event.Data,
		Status:        DeliveryStatusPending,
		AttemptNumber: 1,
		MaxAttempts:   webhook.RetryCount + 1,
		CreatedAt:     time.Now(),
	}

	query := `
		INSERT INTO webhook_deliveries (
			id, webhook_id, event_id, event_type, url, http_method,
			headers, payload, status, attempt_number, max_attempts, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := d.db.ExecContext(ctx, query,
		delivery.ID, delivery.WebhookID, delivery.EventID, delivery.EventType,
		delivery.URL, delivery.HTTPMethod, delivery.Headers, delivery.Payload,
		delivery.Status, delivery.AttemptNumber, delivery.MaxAttempts, delivery.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create delivery: %w", err)
	}

	return delivery, nil
}

// updateDeliverySuccess updates delivery as successful
func (d *Dispatcher) updateDeliverySuccess(ctx context.Context, delivery *WebhookDelivery, statusCode int, responseBody string, durationMs int) {
	now := time.Now()
	query := `
		UPDATE webhook_deliveries
		SET status = $1, status_code = $2, response_body = $3,
			completed_at = $4, duration_ms = $5, sent_at = $6
		WHERE id = $7
	`

	_, err := d.db.ExecContext(ctx, query,
		DeliveryStatusSuccess, statusCode, responseBody,
		now, durationMs, delivery.SentAt, delivery.ID,
	)
	if err != nil {
		fmt.Printf("Failed to update delivery success: %v\n", err)
	}
}

// updateDeliveryFailed updates delivery as failed
func (d *Dispatcher) updateDeliveryFailed(ctx context.Context, delivery *WebhookDelivery, errorMsg string, statusCode int, durationMs int) {
	now := time.Now()
	query := `
		UPDATE webhook_deliveries
		SET status = $1, error_message = $2, status_code = $3,
			completed_at = $4, duration_ms = $5
		WHERE id = $6
	`

	_, err := d.db.ExecContext(ctx, query,
		DeliveryStatusFailed, errorMsg, statusCode,
		now, durationMs, delivery.ID,
	)
	if err != nil {
		fmt.Printf("Failed to update delivery failure: %v\n", err)
	}
}

// handleDeliveryError handles delivery errors and schedules retries
func (d *Dispatcher) handleDeliveryError(ctx context.Context, job *deliveryJob, err error, durationMs int) {
	if job.delivery.AttemptNumber >= job.delivery.MaxAttempts {
		// Max retries reached, mark as failed
		d.updateDeliveryFailed(ctx, job.delivery, err.Error(), 0, durationMs)
		return
	}

	// Schedule retry with exponential backoff
	retryDelay := time.Duration(1<<uint(job.delivery.AttemptNumber-1)) * time.Minute
	nextRetry := time.Now().Add(retryDelay)

	query := `
		UPDATE webhook_deliveries
		SET status = $1, error_message = $2, next_retry_at = $3,
			attempt_number = attempt_number + 1, duration_ms = $4
		WHERE id = $5
	`

	_, dbErr := d.db.ExecContext(ctx, query,
		DeliveryStatusRetrying, err.Error(), nextRetry,
		durationMs, job.delivery.ID,
	)
	if dbErr != nil {
		fmt.Printf("Failed to schedule retry: %v\n", dbErr)
	}
}

// retryProcessor processes pending retries
func (d *Dispatcher) retryProcessor(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopCh:
			return
		case <-ticker.C:
			d.processRetries(ctx)
		}
	}
}

// processRetries processes deliveries ready for retry
func (d *Dispatcher) processRetries(ctx context.Context) {
	query := `
		SELECT wd.id, wd.webhook_id, wd.event_id, wd.event_type, wd.url, 
			   wd.http_method, wd.headers, wd.payload, wd.status,
			   wd.attempt_number, wd.max_attempts, wd.created_at, wd.next_retry_at
		FROM webhook_deliveries wd
		JOIN webhooks w ON wd.webhook_id = w.id
		WHERE wd.status = $1 
		  AND wd.next_retry_at <= CURRENT_TIMESTAMP
		  AND w.enabled = true
		LIMIT 100
	`

	deliveries := []WebhookDelivery{}
	err := d.db.SelectContext(ctx, &deliveries, query, DeliveryStatusRetrying)
	if err != nil {
		fmt.Printf("Failed to fetch retries: %v\n", err)
		return
	}

	for _, delivery := range deliveries {
		// Get webhook and recreate event
		webhook, err := d.manager.GetWebhook(ctx, delivery.WebhookID)
		if err != nil {
			fmt.Printf("Failed to get webhook for retry: %v\n", err)
			continue
		}

		event := &Event{
			ID:        delivery.EventID,
			Type:      EventType(delivery.EventType),
			Timestamp: delivery.CreatedAt,
			Data:      delivery.Payload,
		}

		// Queue retry
		select {
		case d.queue <- &deliveryJob{
			webhook:  webhook,
			event:    event,
			delivery: &delivery,
		}:
		case <-ctx.Done():
			return
		}
	}
}

// updateWebhookLastTriggered updates the last triggered timestamp
func (d *Dispatcher) updateWebhookLastTriggered(ctx context.Context, webhookID string) {
	query := `UPDATE webhooks SET last_triggered_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := d.db.ExecContext(ctx, query, webhookID)
	if err != nil {
		fmt.Printf("Failed to update last_triggered_at: %v\n", err)
	}
}

// generateSignature generates HMAC-SHA256 signature
func (d *Dispatcher) generateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// GetDelivery retrieves a delivery by ID
func (d *Dispatcher) GetDelivery(ctx context.Context, id string) (*WebhookDelivery, error) {
	delivery := &WebhookDelivery{}
	query := `
		SELECT id, webhook_id, event_id, event_type, url, http_method,
			   headers, payload, status_code, response_body, status,
			   attempt_number, max_attempts, error_message, created_at,
			   sent_at, completed_at, next_retry_at, duration_ms
		FROM webhook_deliveries
		WHERE id = $1
	`

	err := d.db.GetContext(ctx, delivery, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery: %w", err)
	}

	return delivery, nil
}

// ListDeliveries lists deliveries with optional filtering
func (d *Dispatcher) ListDeliveries(ctx context.Context, query *ListDeliveriesQuery) ([]WebhookDelivery, error) {
	deliveries := []WebhookDelivery{}

	sqlQuery := `
		SELECT id, webhook_id, event_id, event_type, url, http_method,
			   headers, payload, status_code, response_body, status,
			   attempt_number, max_attempts, error_message, created_at,
			   sent_at, completed_at, next_retry_at, duration_ms
		FROM webhook_deliveries
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if query.WebhookID != "" {
		sqlQuery += fmt.Sprintf(" AND webhook_id = $%d", argPos)
		args = append(args, query.WebhookID)
		argPos++
	}

	if query.EventType != "" {
		sqlQuery += fmt.Sprintf(" AND event_type = $%d", argPos)
		args = append(args, query.EventType)
		argPos++
	}

	if query.Status != "" {
		sqlQuery += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, query.Status)
		argPos++
	}

	sqlQuery += " ORDER BY created_at DESC"

	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, query.Limit)
		argPos++
	}

	if query.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, query.Offset)
	}

	err := d.db.SelectContext(ctx, &deliveries, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list deliveries: %w", err)
	}

	return deliveries, nil
}
