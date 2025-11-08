package arbitration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestWebhookClient_Send verifies basic webhook delivery.
func TestWebhookClient_Send(t *testing.T) {
	var receivedPayload WebhookPayload
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		_ = json.Unmarshal(body, &receivedPayload) //nolint:errcheck
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWebhookClient(WebhookConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	})

	payload := WebhookPayload{
		Event:     EventDisputeCreated,
		Timestamp: time.Now(),
		Dispute: &Dispute{
			ID:      "dispute-1",
			Type:    "policy_conflict",
			Subject: "alice",
			Status:  "pending",
		},
	}

	err := client.Send(context.Background(), payload)
	if err != nil {
		t.Fatalf("webhook send failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if receivedPayload.Event != EventDisputeCreated {
		t.Errorf("expected event %s, got %s", EventDisputeCreated, receivedPayload.Event)
	}
	if receivedPayload.Dispute.ID != "dispute-1" {
		t.Errorf("expected dispute ID dispute-1, got %s", receivedPayload.Dispute.ID)
	}
}

// TestWebhookClient_SignatureVerification validates HMAC signature generation.
func TestWebhookClient_SignatureVerification(t *testing.T) {
	secret := "test-secret-key"
	var receivedSignature string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedSignature = r.Header.Get("X-GAuth-Signature")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWebhookClient(WebhookConfig{
		URL:     server.URL,
		Secret:  secret,
		Timeout: 5 * time.Second,
	})

	payload := WebhookPayload{
		Event:     EventDisputeResolved,
		Timestamp: time.Now(),
	}

	err := client.Send(context.Background(), payload)
	if err != nil {
		t.Fatalf("webhook send failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if receivedSignature == "" {
		t.Fatalf("expected signature header, got empty string")
	}

	// Verify signature locally
	body, _ := json.Marshal(payload)
	if !client.VerifySignature(body, receivedSignature) {
		t.Errorf("signature verification failed")
	}
}

// TestWebhookClient_Retry verifies retry logic on failures.
func TestWebhookClient_Retry(t *testing.T) {
	attempts := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		currentAttempts := attempts
		mu.Unlock()

		// Fail first 2 attempts, succeed on 3rd
		if currentAttempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := NewWebhookClient(WebhookConfig{
		URL:          server.URL,
		Timeout:      2 * time.Second,
		MaxRetries:   3,
		RetryBackoff: 50 * time.Millisecond,
	})

	payload := WebhookPayload{
		Event:     EventDisputeCreated,
		Timestamp: time.Now(),
	}

	err := client.Send(context.Background(), payload)
	if err != nil {
		t.Fatalf("webhook send failed after retries: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}

	// Check metrics
	metrics := client.GetMetrics()
	if metrics.SuccessfulDelivery != 1 {
		t.Errorf("expected 1 successful delivery, got %d", metrics.SuccessfulDelivery)
	}
}

// TestWebhookClient_MaxRetriesExceeded verifies failure after max retries.
func TestWebhookClient_MaxRetriesExceeded(t *testing.T) {
	attempts := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		mu.Unlock()
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewWebhookClient(WebhookConfig{
		URL:          server.URL,
		Timeout:      2 * time.Second,
		MaxRetries:   2,
		RetryBackoff: 10 * time.Millisecond,
	})

	payload := WebhookPayload{
		Event:     EventDisputeCreated,
		Timestamp: time.Now(),
	}

	err := client.Send(context.Background(), payload)
	if err == nil {
		t.Fatalf("expected error after max retries, got nil")
	}
	if !strings.Contains(err.Error(), "webhook delivery failed after") {
		t.Errorf("expected delivery failure message, got: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if attempts != 3 { // Initial attempt + 2 retries
		t.Errorf("expected 3 attempts, got %d", attempts)
	}

	// Check metrics
	metrics := client.GetMetrics()
	if metrics.FailedDelivery != 1 {
		t.Errorf("expected 1 failed delivery, got %d", metrics.FailedDelivery)
	}
}

// TestWebhookClient_DisabledWebhook verifies no-op when URL is empty.
func TestWebhookClient_DisabledWebhook(t *testing.T) {
	client := NewWebhookClient(WebhookConfig{
		URL: "", // Empty URL = disabled
	})

	payload := WebhookPayload{
		Event:     EventDisputeCreated,
		Timestamp: time.Now(),
	}

	err := client.Send(context.Background(), payload)
	if err != nil {
		t.Fatalf("disabled webhook should not error, got: %v", err)
	}

	metrics := client.GetMetrics()
	if metrics.TotalSent != 0 {
		t.Errorf("expected 0 total sent for disabled webhook, got %d", metrics.TotalSent)
	}
}

// TestWebhookClient_EventFiltering verifies event subscription filtering.
func TestWebhookClient_EventFiltering(t *testing.T) {
	sent := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		sent++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWebhookClient(WebhookConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
		Events:  []WebhookEventType{EventDisputeCreated}, // Only subscribe to created events
	})

	// Send subscribed event
	client.Send(context.Background(), WebhookPayload{
		Event:     EventDisputeCreated,
		Timestamp: time.Now(),
	})

	// Send unsubscribed event
	client.Send(context.Background(), WebhookPayload{
		Event:     EventDisputeResolved,
		Timestamp: time.Now(),
	})

	time.Sleep(100 * time.Millisecond) // Allow async delivery

	mu.Lock()
	defer mu.Unlock()
	if sent != 1 {
		t.Errorf("expected 1 webhook sent (filtered), got %d", sent)
	}
}

// TestWebhookEnabledArbiter_Integration verifies webhook integration with arbiter.
func TestWebhookEnabledArbiter_Integration(t *testing.T) {
	receivedEvents := make(map[WebhookEventType]int)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload WebhookPayload
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &payload)

		mu.Lock()
		receivedEvents[payload.Event]++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	webhook := NewWebhookClient(WebhookConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	})

	baseArbiter := NewDefaultArbiter()
	arbiter := NewWebhookEnabledArbiter(baseArbiter, webhook)

	// Test arbitration with webhooks
	dispute := &Dispute{
		ID:        "test-dispute",
		Type:      "policy_conflict",
		Subject:   "alice",
		Resource:  "/api/data",
		Action:    "read",
		Policies:  []string{"policy-1", "policy-2"},
		Decisions: []string{"permit", "deny"},
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	result, err := arbiter.Arbitrate(context.Background(), dispute)
	if err != nil {
		t.Fatalf("arbitration failed: %v", err)
	}
	if result.Decision == "" {
		t.Fatalf("expected decision, got empty string")
	}

	time.Sleep(200 * time.Millisecond) // Allow async webhook delivery

	mu.Lock()
	defer mu.Unlock()

	if receivedEvents[EventDisputeCreated] != 1 {
		t.Errorf("expected 1 dispute.created event, got %d", receivedEvents[EventDisputeCreated])
	}
	if receivedEvents[EventDisputeResolved] != 1 {
		t.Errorf("expected 1 dispute.resolved event, got %d", receivedEvents[EventDisputeResolved])
	}
}

// TestWebhookClient_HeadersPresent verifies required headers are sent.
func TestWebhookClient_HeadersPresent(t *testing.T) {
	var headers http.Header
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		headers = r.Header.Clone()
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWebhookClient(WebhookConfig{
		URL:     server.URL,
		Secret:  "test-secret",
		Timeout: 5 * time.Second,
	})

	payload := WebhookPayload{
		Event:     EventDisputeCreated,
		Timestamp: time.Now(),
	}

	client.Send(context.Background(), payload)

	mu.Lock()
	defer mu.Unlock()

	requiredHeaders := []string{
		"Content-Type",
		"User-Agent",
		"X-GAuth-Event",
		"X-GAuth-Timestamp",
		"X-GAuth-Signature",
	}

	for _, header := range requiredHeaders {
		if headers.Get(header) == "" {
			t.Errorf("expected header %s to be present", header)
		}
	}

	if headers.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", headers.Get("Content-Type"))
	}
	if headers.Get("X-GAuth-Event") != string(EventDisputeCreated) {
		t.Errorf("expected X-GAuth-Event %s, got %s", EventDisputeCreated, headers.Get("X-GAuth-Event"))
	}
}

// TestWebhookClient_ContextCancellation verifies context cancellation stops delivery.
func TestWebhookClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Simulate slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWebhookClient(WebhookConfig{
		URL:          server.URL,
		Timeout:      5 * time.Second,
		MaxRetries:   1,
		RetryBackoff: 100 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	payload := WebhookPayload{
		Event:     EventDisputeCreated,
		Timestamp: time.Now(),
	}

	err := client.Send(ctx, payload)
	if err == nil {
		t.Fatalf("expected error from context cancellation, got nil")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "webhook delivery failed") {
		t.Errorf("expected context cancellation error, got: %v", err)
	}
}

// TestWebhookClient_Metrics verifies metrics tracking accuracy.
func TestWebhookClient_Metrics(t *testing.T) {
	successCount := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		successCount++
		current := successCount
		mu.Unlock()

		if current <= 2 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	client := NewWebhookClient(WebhookConfig{
		URL:          server.URL,
		Timeout:      2 * time.Second,
		MaxRetries:   0, // No retries
		RetryBackoff: 10 * time.Millisecond,
	})

	// Send 2 successful webhooks
	client.Send(context.Background(), WebhookPayload{Event: EventDisputeCreated, Timestamp: time.Now()})
	client.Send(context.Background(), WebhookPayload{Event: EventDisputeResolved, Timestamp: time.Now()})

	// Send 1 failing webhook
	client.Send(context.Background(), WebhookPayload{Event: EventDisputeEscalated, Timestamp: time.Now()})

	metrics := client.GetMetrics()
	if metrics.TotalSent != 3 {
		t.Errorf("expected 3 total sent, got %d", metrics.TotalSent)
	}
	if metrics.SuccessfulDelivery != 2 {
		t.Errorf("expected 2 successful deliveries, got %d", metrics.SuccessfulDelivery)
	}
	if metrics.FailedDelivery != 1 {
		t.Errorf("expected 1 failed delivery, got %d", metrics.FailedDelivery)
	}
	if metrics.EventBreakdown[EventDisputeCreated] != 1 {
		t.Errorf("expected 1 dispute.created event, got %d", metrics.EventBreakdown[EventDisputeCreated])
	}
}

// TestWebhookClient_DeliveryTracking verifies delivery status tracking.
func TestWebhookClient_DeliveryTracking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWebhookClient(WebhookConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	})

	payload := WebhookPayload{
		Event:     EventDisputeCreated,
		Timestamp: time.Now(),
	}

	err := client.Send(context.Background(), payload)
	if err != nil {
		t.Fatalf("webhook send failed: %v", err)
	}

	deliveries := client.ListDeliveries()
	if len(deliveries) != 1 {
		t.Fatalf("expected 1 delivery tracked, got %d", len(deliveries))
	}

	delivery := deliveries[0]
	if delivery.Status != "delivered" {
		t.Errorf("expected status delivered, got %s", delivery.Status)
	}
	if delivery.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", delivery.Attempts)
	}
}
