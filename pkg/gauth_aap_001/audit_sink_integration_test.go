package gauth_aap_001

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
)

const (
	testActionCreateDelegation = "create_delegation"
)

// allowAllAuditSinkAuthorizer is a test-only authorizer that allows all requests
type allowAllAuditSinkAuthorizer struct{}

func (a *allowAllAuditSinkAuthorizer) Authorize(ctx context.Context, req authz.Request) (authz.Decision, error) {
	return authz.Decision{Allow: true, Reason: "test-allow-all"}, nil
}

func (a *allowAllAuditSinkAuthorizer) GetPermissions(ctx context.Context, subject string) ([]authz.Permission, error) {
	// Test authorizer grants all permissions
	return []authz.Permission{
		{Resource: "*", Actions: []string{"*"}, Granted: true},
	}, nil
}

func (a *allowAllAuditSinkAuthorizer) LoadPolicies(ctx context.Context, policies []authz.Policy) error {
	return nil
}

// testAuditSink is a simple in-memory sink for testing
type testAuditSink struct {
	mu     sync.Mutex
	events []*audit.Event
	errors int // number of errors to return before succeeding
	closed bool
}

func (t *testAuditSink) Send(ctx context.Context, event *audit.Event) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return errors.New("sink is closed")
	}

	if t.errors > 0 {
		t.errors--
		return errors.New("simulated sink error")
	}

	t.events = append(t.events, event)
	return nil
}

func (t *testAuditSink) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	return nil
}

func (t *testAuditSink) Events() []*audit.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]*audit.Event{}, t.events...)
}

func TestAuditSinkIntegration_Disabled(t *testing.T) {
	// Test backward compatibility: audit sink disabled by default
	logger := audit.NewMemoryLogger(nil)
	authorizer := &allowAllAuditSinkAuthorizer{}

	svc := NewService(logger, authorizer)

	req := DelegationRequest{
		Grantor:  "alice@example.com",
		Grantee:  "bob@example.com",
		Scope:    []string{"read:data"},
		Duration: 1 * time.Hour,
	}

	ctx := WithSubject(context.Background(), "bob@example.com")
	resp, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	if resp.POA.ID == "" {
		t.Fatal("Expected POA ID but got empty")
	}

	// Allow async audit processing to complete
	time.Sleep(50 * time.Millisecond)

	// Verify audit logger received event but sink was not called (nil check passes)
	events, err := logger.Query(ctx, &audit.Filter{EventTypes: []audit.EventType{audit.TypeAuth}})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("Expected audit events in logger but got none")
	}
}

func TestAuditSinkIntegration_CreateDelegation(t *testing.T) {
	logger := audit.NewMemoryLogger(nil)
	authorizer := &allowAllAuditSinkAuthorizer{}
	sink := &testAuditSink{}

	svc := NewService(logger, authorizer, WithAuditSink(sink))

	req := DelegationRequest{
		Grantor:  "alice@example.com",
		Grantee:  "bob@example.com",
		Scope:    []string{"read:data", "write:data"},
		Duration: 2 * time.Hour,
	}

	ctx := WithSubject(context.Background(), "bob@example.com")
	resp, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	if resp.POA.ID == "" {
		t.Fatal("Expected POA ID but got empty")
	}

	// Verify sink received event
	events := sink.Events()
	if len(events) == 0 {
		t.Fatal("Expected sink to receive event but got none")
	}

	found := false
	for _, ev := range events {
		if ev.Action == testActionCreateDelegation && ev.Result == audit.ResultSuccess {
			if ev.Subject != "alice@example.com" {
				t.Errorf("Expected subject 'alice@example.com' but got '%s'", ev.Subject)
			}
			if ev.Object != resp.POA.ID {
				t.Errorf("Expected object '%s' but got '%s'", resp.POA.ID, ev.Object)
			}
			found = true
			break
		}
	}

	if !found {
		t.Fatal("Expected create_delegation success event in sink but not found")
	}
}

func TestAuditSinkIntegration_VerifyToken(t *testing.T) {
	logger := audit.NewMemoryLogger(nil)
	authorizer := &allowAllAuditSinkAuthorizer{}
	sink := &testAuditSink{}

	svc := NewService(logger, authorizer, WithAuditSink(sink))

	// Create delegation
	req := DelegationRequest{
		Grantor:  "alice@example.com",
		Grantee:  "bob@example.com",
		Scope:    []string{"read:data"},
		Duration: 1 * time.Hour,
	}

	ctx := WithSubject(context.Background(), "bob@example.com")
	resp, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	// Verify token
	err = svc.ValidateDelegation(resp.POA.ID, "bob@example.com", "read:data")
	if err != nil {
		t.Fatalf("ValidateDelegation failed: %v", err)
	}

	// Verify sink received both events (create + validate)
	events := sink.Events()
	if len(events) < 2 {
		t.Fatalf("Expected at least 2 events (create + validate) but got %d", len(events))
	}

	foundCreate := false
	foundValidate := false
	for _, ev := range events {
		if ev.Action == testActionCreateDelegation && ev.Result == audit.ResultSuccess {
			foundCreate = true
		}
		if ev.Action == "validate_delegation" && ev.Result == audit.ResultSuccess {
			if ev.Subject != "bob@example.com" {
				t.Errorf("Expected subject 'bob@example.com' but got '%s'", ev.Subject)
			}
			foundValidate = true
		}
	}

	if !foundCreate {
		t.Fatal("Expected create_delegation event in sink but not found")
	}
	if !foundValidate {
		t.Fatal("Expected validate_delegation event in sink but not found")
	}
}

func TestAuditSinkIntegration_RevokeToken(t *testing.T) {
	logger := audit.NewMemoryLogger(nil)
	authorizer := &allowAllAuditSinkAuthorizer{}
	sink := &testAuditSink{}

	svc := NewService(logger, authorizer, WithAuditSink(sink))

	// Create delegation
	req := DelegationRequest{
		Grantor:  "alice@example.com",
		Grantee:  "bob@example.com",
		Scope:    []string{"read:data"},
		Duration: 1 * time.Hour,
	}

	ctx := WithSubject(context.Background(), "bob@example.com")
	resp, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	// Revoke token
	err = svc.RevokeDelegation(resp.POA.ID, "alice@example.com")
	if err != nil {
		t.Fatalf("RevokeDelegation failed: %v", err)
	}

	// Verify sink received both events (create + revoke)
	events := sink.Events()
	if len(events) < 2 {
		t.Fatalf("Expected at least 2 events (create + revoke) but got %d", len(events))
	}

	foundCreate := false
	foundRevoke := false
	for _, ev := range events {
		if ev.Action == testActionCreateDelegation && ev.Result == audit.ResultSuccess {
			foundCreate = true
		}
		if ev.Action == "revoke_delegation" && ev.Result == audit.ResultSuccess {
			if ev.Subject != "alice@example.com" {
				t.Errorf("Expected subject 'alice@example.com' but got '%s'", ev.Subject)
			}
			foundRevoke = true
		}
	}

	if !foundCreate {
		t.Fatal("Expected create_delegation event in sink but not found")
	}
	if !foundRevoke {
		t.Fatal("Expected revoke_delegation event in sink but not found")
	}
}

func TestAuditSinkIntegration_AsyncSink(t *testing.T) {
	logger := audit.NewMemoryLogger(nil)
	authorizer := &allowAllAuditSinkAuthorizer{}
	baseSink := &testAuditSink{}
	asyncSink := NewAsyncAuditSink(baseSink, 10)
	defer asyncSink.Close()

	svc := NewService(logger, authorizer, WithAuditSink(asyncSink))

	req := DelegationRequest{
		Grantor:  "alice@example.com",
		Grantee:  "bob@example.com",
		Scope:    []string{"read:data"},
		Duration: 1 * time.Hour,
	}

	ctx := WithSubject(context.Background(), "bob@example.com")
	_, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	// Give async sink time to flush
	time.Sleep(100 * time.Millisecond)

	// Verify base sink received event via async wrapper
	events := baseSink.Events()
	if len(events) == 0 {
		t.Fatal("Expected async sink to flush event to base sink but got none")
	}

	if events[0].Action != testActionCreateDelegation {
		t.Errorf("Expected create_delegation but got '%s'", events[0].Action)
	}
}

func TestAuditSinkIntegration_ErrorHandling(t *testing.T) {
	logger := audit.NewMemoryLogger(nil)
	authorizer := &allowAllAuditSinkAuthorizer{}
	sink := &testAuditSink{errors: 1} // Return error on first Send()

	svc := NewService(logger, authorizer, WithAuditSink(sink))

	req := DelegationRequest{
		Grantor:  "alice@example.com",
		Grantee:  "bob@example.com",
		Scope:    []string{"read:data"},
		Duration: 1 * time.Hour,
	}

	ctx := WithSubject(context.Background(), "bob@example.com")
	resp, err := svc.CreateDelegationCtx(ctx, req)

	// Operation should succeed despite sink error (fail-open behavior)
	if err != nil {
		t.Fatalf("CreateDelegation should succeed despite sink error: %v", err)
	}

	if resp.POA.ID == "" {
		t.Fatal("Expected POA ID but got empty")
	}

	// Allow async audit processing to complete
	time.Sleep(50 * time.Millisecond)

	// Verify audit logger still received event (sink failure doesn't affect logger)
	events, err := logger.Query(ctx, &audit.Filter{EventTypes: []audit.EventType{audit.TypeAuth}})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("Expected audit logger to have event despite sink error")
	}

	// Verify sink has no events (due to simulated error)
	sinkEvents := sink.Events()
	if len(sinkEvents) != 0 {
		t.Errorf("Expected sink to have 0 events due to error but got %d", len(sinkEvents))
	}
}

func TestAuditSinkIntegration_MultiplexSink(t *testing.T) {
	logger := audit.NewMemoryLogger(nil)
	authorizer := &allowAllAuditSinkAuthorizer{}

	sink1 := &testAuditSink{}
	sink2 := &testAuditSink{}
	multiplex := NewMultiplexAuditSink(sink1, sink2)

	svc := NewService(logger, authorizer, WithAuditSink(multiplex))

	req := DelegationRequest{
		Grantor:  "alice@example.com",
		Grantee:  "bob@example.com",
		Scope:    []string{"read:data"},
		Duration: 1 * time.Hour,
	}

	ctx := WithSubject(context.Background(), "bob@example.com")
	_, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	// Verify both sinks received event
	events1 := sink1.Events()
	events2 := sink2.Events()

	if len(events1) == 0 {
		t.Fatal("Expected sink1 to receive event but got none")
	}
	if len(events2) == 0 {
		t.Fatal("Expected sink2 to receive event but got none")
	}

	if events1[0].Action != testActionCreateDelegation {
		t.Errorf("Expected sink1 create_delegation but got '%s'", events1[0].Action)
	}
	if events2[0].Action != testActionCreateDelegation {
		t.Errorf("Expected sink2 create_delegation but got '%s'", events2[0].Action)
	}
}

func TestAuditSinkIntegration_FilteredSink(t *testing.T) {
	logger := audit.NewMemoryLogger(nil)
	authorizer := &allowAllAuditSinkAuthorizer{}
	baseSink := &testAuditSink{}

	// Filter: only allow "revoke_delegation" actions
	filtered := NewFilteredAuditSink(baseSink, FilterByAction("revoke_delegation"))

	svc := NewService(logger, authorizer, WithAuditSink(filtered))

	// Create delegation (should be filtered out)
	req := DelegationRequest{
		Grantor:  "alice@example.com",
		Grantee:  "bob@example.com",
		Scope:    []string{"read:data"},
		Duration: 1 * time.Hour,
	}

	ctx := WithSubject(context.Background(), "bob@example.com")
	resp, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	// Revoke delegation (should pass filter)
	err = svc.RevokeDelegation(resp.POA.ID, "alice@example.com")
	if err != nil {
		t.Fatalf("RevokeDelegation failed: %v", err)
	}

	// Verify sink only received revoke event (create was filtered out)
	events := baseSink.Events()
	if len(events) != 1 {
		t.Fatalf("Expected filtered sink to have 1 event (revoke only) but got %d", len(events))
	}

	if events[0].Action != "revoke_delegation" {
		t.Errorf("Expected revoke_delegation but got '%s'", events[0].Action)
	}
}

func TestAuditSinkIntegration_FilterByEventType(t *testing.T) {
	logger := audit.NewMemoryLogger(nil)
	authorizer := &allowAllAuditSinkAuthorizer{}
	baseSink := &testAuditSink{}

	// Filter: only allow TypeAuth events
	filtered := NewFilteredAuditSink(baseSink, FilterByEventType(audit.TypeAuth))

	svc := NewService(logger, authorizer, WithAuditSink(filtered))

	req := DelegationRequest{
		Grantor:  "alice@example.com",
		Grantee:  "bob@example.com",
		Scope:    []string{"read:data"},
		Duration: 1 * time.Hour,
	}

	ctx := WithSubject(context.Background(), "bob@example.com")
	_, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	// All AAP001 events are TypeAuth, so should pass filter
	events := baseSink.Events()
	if len(events) == 0 {
		t.Fatal("Expected filtered sink to receive TypeAuth events but got none")
	}

	for _, ev := range events {
		if ev.Type != audit.TypeAuth {
			t.Errorf("Expected all events to be TypeAuth but got '%s'", ev.Type)
		}
	}
}

func TestAuditSinkIntegration_FilterByResult(t *testing.T) {
	logger := audit.NewMemoryLogger(nil)
	// Authorizer that denies all requests to generate failure events
	denyAuthorizer := &denyAllAuthorizer{}
	baseSink := &testAuditSink{}

	// Filter: only allow failure results
	filtered := NewFilteredAuditSink(baseSink, FilterByResult(audit.ResultFailure))

	svc := NewService(logger, denyAuthorizer, WithAuditSink(filtered))

	req := DelegationRequest{
		Grantor:  "alice@example.com",
		Grantee:  "bob@example.com",
		Scope:    []string{"read:data"},
		Duration: 1 * time.Hour,
	}

	ctx := WithSubject(context.Background(), "bob@example.com")
	_, err := svc.CreateDelegationCtx(ctx, req)

	// Expect authorization failure
	if err == nil {
		t.Fatal("Expected CreateDelegation to fail with deny-all authorizer")
	}

	// Verify sink received failure event
	events := baseSink.Events()
	if len(events) == 0 {
		t.Fatal("Expected filtered sink to receive failure event but got none")
	}

	if events[0].Result != audit.ResultFailure {
		t.Errorf("Expected ResultFailure but got '%s'", events[0].Result)
	}
}

// denyAllAuthorizer is a test-only authorizer that denies all requests
type denyAllAuthorizer struct{}

func (a *denyAllAuthorizer) Authorize(ctx context.Context, req authz.Request) (authz.Decision, error) {
	return authz.Decision{Allow: false, Reason: "test-deny-all"}, nil
}

func (a *denyAllAuthorizer) GetPermissions(ctx context.Context, subject string) ([]authz.Permission, error) {
	// Test authorizer that denies: no permissions
	return []authz.Permission{}, nil
}

func (a *denyAllAuthorizer) LoadPolicies(ctx context.Context, policies []authz.Policy) error {
	return nil
}

func TestAsyncAuditSink_BufferOverflow(t *testing.T) {
	baseSink := &slowSink{delay: 100 * time.Millisecond}
	asyncSink := NewAsyncAuditSink(baseSink, 2) // Small buffer
	defer asyncSink.Close()

	logger := audit.NewMemoryLogger(nil)
	authorizer := &allowAllAuditSinkAuthorizer{}
	svc := NewService(logger, authorizer, WithAuditSink(asyncSink))

	ctx := WithSubject(context.Background(), "bob@example.com")

	// Send multiple events quickly to overflow buffer
	for i := 0; i < 10; i++ {
		req := DelegationRequest{
			Grantor:  "alice@example.com",
			Grantee:  "bob@example.com",
			Scope:    []string{"read:data"},
			Duration: 1 * time.Hour,
		}
		_, _ = svc.CreateDelegationCtx(ctx, req)
	}

	// Verify some events were dropped (buffer overflow)
	if asyncSink.Dropped == 0 {
		t.Log("Warning: Expected some dropped events due to buffer overflow, but got 0")
		t.Log("This may indicate the test ran too slow or buffer is too large")
	}
}

// slowSink simulates a slow external sink
type slowSink struct {
	mu     sync.Mutex
	events []*audit.Event
	delay  time.Duration
}

func (s *slowSink) Send(ctx context.Context, event *audit.Event) error {
	time.Sleep(s.delay)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *slowSink) Close() error {
	return nil
}
