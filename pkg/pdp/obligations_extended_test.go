package pdp

import (
	"context"
	"testing"
	"time"
)

// TestBufferedAdviceChannel_EmitAndReceive verifies advice event emission and reception.
func TestBufferedAdviceChannel_EmitAndReceive(t *testing.T) {
	ch := NewBufferedAdviceChannel(10)
	defer ch.Close()

	ctx := context.Background()
	event := AdviceEvent{
		Timestamp:  time.Now(),
		Subject:    "alice",
		Action:     "read",
		Resource:   "doc:123",
		AdviceID:   "adv1",
		AdviceType: "rate_limit",
		Message:    "Consider rate limiting this user",
		Metadata:   map[string]string{"reason": "high_frequency"},
	}

	// Emit event
	err := ch.Emit(ctx, event)
	if err != nil {
		t.Fatalf("failed to emit advice: %v", err)
	}

	// Receive event
	select {
	case received := <-ch.AdviceEvents():
		if received.AdviceID != "adv1" {
			t.Errorf("expected advice ID 'adv1', got '%s'", received.AdviceID)
		}
		if received.Subject != "alice" {
			t.Errorf("expected subject 'alice', got '%s'", received.Subject)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for advice event")
	}
}

// TestBufferedAdviceChannel_BufferFull verifies drop behavior when buffer is full.
func TestBufferedAdviceChannel_BufferFull(t *testing.T) {
	ch := NewBufferedAdviceChannel(2) // Small buffer
	defer ch.Close()

	ctx := context.Background()

	// Fill buffer
	for i := 0; i < 2; i++ {
		event := AdviceEvent{
			AdviceID: "adv" + string(rune('0'+i)),
			Subject:  "user",
		}
		err := ch.Emit(ctx, event)
		if err != nil {
			t.Fatalf("failed to emit event %d: %v", i, err)
		}
	}

	// Next emit should fail (buffer full)
	event := AdviceEvent{AdviceID: "adv_overflow", Subject: "user"}
	err := ch.Emit(ctx, event)
	if err == nil {
		t.Error("expected error when buffer full, got nil")
	}
}

// TestBufferedAdviceChannel_ClosedChannel verifies emission fails after close.
func TestBufferedAdviceChannel_ClosedChannel(t *testing.T) {
	ch := NewBufferedAdviceChannel(10)
	ch.Close()

	ctx := context.Background()
	event := AdviceEvent{AdviceID: "adv1", Subject: "user"}
	err := ch.Emit(ctx, event)
	if err == nil {
		t.Error("expected error when emitting to closed channel, got nil")
	}
}

// TestExtendedObligationExecutor_LogHandler verifies log obligation execution.
func TestExtendedObligationExecutor_LogHandler(t *testing.T) {
	exec := NewExtendedObligationExecutor()
	ctx := context.Background()

	results := exec.Execute(ctx, []string{"log:user_action"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if !results[0].Success {
		t.Errorf("expected success, got failure: %v", results[0].Error)
	}
	if results[0].Name != "log:user_action" {
		t.Errorf("expected name 'log:user_action', got '%s'", results[0].Name)
	}
}

// TestExtendedObligationExecutor_NotifyHandler verifies notify obligation execution.
func TestExtendedObligationExecutor_NotifyHandler(t *testing.T) {
	exec := NewExtendedObligationExecutor()
	ctx := context.Background()

	results := exec.Execute(ctx, []string{"notify:admin_alert"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if !results[0].Success {
		t.Errorf("expected success, got failure: %v", results[0].Error)
	}
}

// TestExtendedObligationExecutor_RateLimitHandler verifies rate_limit obligation execution.
func TestExtendedObligationExecutor_RateLimitHandler(t *testing.T) {
	exec := NewExtendedObligationExecutor()
	ctx := context.Background()

	results := exec.Execute(ctx, []string{"rate_limit:api_calls"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if !results[0].Success {
		t.Errorf("expected success, got failure: %v", results[0].Error)
	}
}

// TestExtendedObligationExecutor_UnknownHandler verifies error on unknown obligation type.
func TestExtendedObligationExecutor_UnknownHandler(t *testing.T) {
	exec := NewExtendedObligationExecutor()
	ctx := context.Background()

	results := exec.Execute(ctx, []string{"unknown_type:action"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Success {
		t.Error("expected failure for unknown obligation type, got success")
	}
	if results[0].Error == nil {
		t.Error("expected error for unknown obligation type, got nil")
	}
}

// TestExtendedObligationExecutor_MultipleObligations verifies batch execution.
func TestExtendedObligationExecutor_MultipleObligations(t *testing.T) {
	exec := NewExtendedObligationExecutor()
	ctx := context.Background()

	obligations := []string{
		"log:user_login",
		"notify:security_alert",
		"rate_limit:api_calls",
	}

	results := exec.Execute(ctx, obligations)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for i, result := range results {
		if !result.Success {
			t.Errorf("obligation %d failed: %v", i, result.Error)
		}
		if result.Name != obligations[i] {
			t.Errorf("expected name '%s', got '%s'", obligations[i], result.Name)
		}
	}
}

// TestExtendedObligationExecutor_WithAuditSink verifies audit sink integration.
func TestExtendedObligationExecutor_WithAuditSink(t *testing.T) {
	auditSink := &mockObligationAuditSink{records: make([]ObligationAuditRecord, 0)}
	exec := NewExtendedObligationExecutor(WithObligationAuditSink(auditSink))
	ctx := context.Background()

	results := exec.Execute(ctx, []string{"log:user_action"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Wait briefly for audit to complete (fire-and-forget)
	time.Sleep(50 * time.Millisecond)

	if len(auditSink.records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(auditSink.records))
	}

	record := auditSink.records[0]
	if record.ObligationID != "user_action" {
		t.Errorf("expected obligation ID 'user_action', got '%s'", record.ObligationID)
	}
	if record.ObligationType != "log" {
		t.Errorf("expected obligation type 'log', got '%s'", record.ObligationType)
	}
	if !record.Success {
		t.Error("expected success in audit record")
	}
}

// TestExtendedObligationExecutor_WithAdviceChannel verifies advice emission integration.
func TestExtendedObligationExecutor_WithAdviceChannel(t *testing.T) {
	adviceChannel := NewBufferedAdviceChannel(10)
	defer adviceChannel.Close()

	exec := NewExtendedObligationExecutor(WithAdviceChannel(adviceChannel))
	ctx := context.Background()

	// Emit advice
	event := AdviceEvent{
		Timestamp:  time.Now(),
		AdviceID:   "adv1",
		AdviceType: "performance",
		Subject:    "alice",
		Action:     "read",
		Resource:   "doc:123",
		Message:    "Consider caching this resource",
	}

	err := exec.EmitAdvice(ctx, event)
	if err != nil {
		t.Fatalf("failed to emit advice: %v", err)
	}

	// Verify advice received
	select {
	case received := <-adviceChannel.AdviceEvents():
		if received.AdviceID != "adv1" {
			t.Errorf("expected advice ID 'adv1', got '%s'", received.AdviceID)
		}
		if received.Message != "Consider caching this resource" {
			t.Errorf("unexpected advice message: %s", received.Message)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for advice event")
	}
}

// TestExtendedObligationExecutor_CustomHandler verifies custom handler registration.
func TestExtendedObligationExecutor_CustomHandler(t *testing.T) {
	exec := NewExtendedObligationExecutor()
	customHandler := &mockObligationHandler{
		handlerType: "custom",
		executed:    false,
	}
	exec.RegisterHandler(customHandler)

	ctx := context.Background()
	results := exec.Execute(ctx, []string{"custom:test_action"})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if !results[0].Success {
		t.Errorf("expected success, got failure: %v", results[0].Error)
	}

	if !customHandler.executed {
		t.Error("custom handler was not executed")
	}
}

// TestObligationNameParsing verifies obligation name parsing logic.
func TestObligationNameParsing(t *testing.T) {
	exec := NewExtendedObligationExecutor()

	tests := []struct {
		name         string
		input        string
		expectedType string
		expectedID   string
	}{
		{
			name:         "colon-separated",
			input:        "log:user_action",
			expectedType: "log",
			expectedID:   "user_action",
		},
		{
			name:         "no-colon-defaults-to-log",
			input:        "simple_obligation",
			expectedType: "log",
			expectedID:   "simple_obligation",
		},
		{
			name:         "multiple-colons",
			input:        "notify:admin:alert",
			expectedType: "notify",
			expectedID:   "admin:alert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obligationType, obligationID := exec.parseObligationName(tt.input)
			if obligationType != tt.expectedType {
				t.Errorf("expected type '%s', got '%s'", tt.expectedType, obligationType)
			}
			if obligationID != tt.expectedID {
				t.Errorf("expected ID '%s', got '%s'", tt.expectedID, obligationID)
			}
		})
	}
}

// Mock implementations for testing

type mockObligationAuditSink struct {
	records []ObligationAuditRecord
}

func (m *mockObligationAuditSink) RecordObligationExecution(ctx context.Context, record ObligationAuditRecord) error {
	m.records = append(m.records, record)
	return nil
}

func (m *mockObligationAuditSink) Close() error {
	return nil
}

type mockObligationHandler struct {
	handlerType string
	executed    bool
}

func (m *mockObligationHandler) Type() string {
	return m.handlerType
}

func (m *mockObligationHandler) Execute(ctx context.Context, obligationID string, params map[string]string) error {
	m.executed = true
	return nil
}

type mockRateLimitChecker struct {
	shouldAllow bool
	err         error
	calledKey   string
}

func (m *mockRateLimitChecker) Allow(ctx context.Context, key string) (bool, error) {
	m.calledKey = key
	return m.shouldAllow, m.err
}

// TestExtendedObligationExecutor_RateLimitHandler_WithChecker verifies integration with external checker.
func TestExtendedObligationExecutor_RateLimitHandler_WithChecker(t *testing.T) {
	mockChecker := &mockRateLimitChecker{shouldAllow: true}
	handler := NewRateLimitObligationHandler(0, 0, mockChecker)

	exec := NewExtendedObligationExecutor()
	exec.RegisterHandler(handler)

	ctx := context.Background()
	results := exec.Execute(ctx, []string{"rate_limit:api_user_123"})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Success {
		t.Errorf("expected success with allow=true, got failure: %v", results[0].Error)
	}
	if mockChecker.calledKey != "api_user_123" {
		t.Errorf("expected called key 'api_user_123', got '%s'", mockChecker.calledKey)
	}

	// Test denied case
	mockChecker.shouldAllow = false
	results = exec.Execute(ctx, []string{"rate_limit:spammer"})
	if results[0].Success {
		t.Error("expected failure with allow=false, got success")
	}
}
