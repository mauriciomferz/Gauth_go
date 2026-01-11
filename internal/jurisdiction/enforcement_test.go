package jurisdiction

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/compliance"
)

func TestNewEnforcementEngine(t *testing.T) {
	engine := NewEnforcementEngine()

	if engine == nil {
		t.Fatal("NewEnforcementEngine should return a non-nil engine")
	}

	if !engine.IsEnabled() { //nolint:staticcheck // nil checked above
		t.Error("Engine should be enabled by default")
	}

	if engine.validator == nil { //nolint:staticcheck // nil checked above
		t.Error("Engine should have a validator")
	}

	if len(engine.jurisdictionRules) == 0 {
		t.Error("Engine should have default jurisdiction rules initialized")
	}

	// Verify all expected jurisdictions are initialized
	expectedJurisdictions := []compliance.Jurisdiction{
		compliance.JurisdictionEU,
		compliance.JurisdictionUS,
		compliance.JurisdictionUK,
		compliance.JurisdictionCA,
		compliance.JurisdictionAU,
		compliance.JurisdictionJP,
	}

	for _, jurisdiction := range expectedJurisdictions {
		if _, exists := engine.jurisdictionRules[jurisdiction]; !exists {
			t.Errorf("Expected jurisdiction %s to be initialized", jurisdiction)
		}
	}
}

func TestEnforce_BasicValidation(t *testing.T) {
	engine := NewEnforcementEngine()
	ctx := context.Background()

	tests := []struct {
		name             string
		enfCtx           *EnforcementContext
		expectAllowed    bool
		expectViolations bool
	}{
		{
			name: "Valid US trade execution",
			enfCtx: &EnforcementContext{
				RequestID:    "test-1",
				Subject:      "user@example.com",
				Resource:     "trading-system",
				Action:       "trade_execution",
				Value:        5000000.0, // Within $10M limit
				EntityType:   compliance.EntityTypeCorporation,
				Jurisdiction: compliance.JurisdictionUS,
				Claims:       map[string]interface{}{},
				Timestamp:    time.Now(),
			},
			expectAllowed:    true,
			expectViolations: false,
		},
		{
			name: "US trade execution exceeds value limit",
			enfCtx: &EnforcementContext{
				RequestID:    "test-2",
				Subject:      "user@example.com",
				Resource:     "trading-system",
				Action:       "trade_execution",
				Value:        15000000.0, // Exceeds $10M limit
				EntityType:   compliance.EntityTypeCorporation,
				Jurisdiction: compliance.JurisdictionUS,
				Claims:       map[string]interface{}{},
				Timestamp:    time.Now(),
			},
			expectAllowed:    false,
			expectViolations: true,
		},
		{
			name: "Valid EU fund transfer",
			enfCtx: &EnforcementContext{
				RequestID:    "test-3",
				Subject:      "user@example.com",
				Resource:     "payment-system",
				Action:       "fund_transfer",
				Value:        2000000.0, // Within €3M limit
				EntityType:   compliance.EntityTypeCorporation,
				Jurisdiction: compliance.JurisdictionEU,
				Claims:       map[string]interface{}{},
				Timestamp:    time.Now(),
			},
			expectAllowed:    true,
			expectViolations: false,
		},
		{
			name: "Unsupported jurisdiction",
			enfCtx: &EnforcementContext{
				RequestID:    "test-4",
				Subject:      "user@example.com",
				Resource:     "system",
				Action:       "test_action",
				EntityType:   compliance.EntityTypeCorporation,
				Jurisdiction: compliance.Jurisdiction("INVALID"),
				Claims:       map[string]interface{}{},
				Timestamp:    time.Now(),
			},
			expectAllowed:    false,
			expectViolations: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := engine.Enforce(ctx, tt.enfCtx)
			if err != nil {
				t.Fatalf("Enforce returned error: %v", err)
			}

			if decision.Allowed != tt.expectAllowed {
				t.Errorf("Expected allowed=%v, got %v. Violations: %v",
					tt.expectAllowed, decision.Allowed, decision.Violations)
			}

			if tt.expectViolations && len(decision.Violations) == 0 {
				t.Error("Expected violations but got none")
			}

			if !tt.expectViolations && len(decision.Violations) > 0 {
				t.Errorf("Expected no violations but got: %v", decision.Violations)
			}
		})
	}
}

func TestEnforce_GDPR_DataProcessing(t *testing.T) {
	engine := NewEnforcementEngine()
	ctx := context.Background()

	tests := []struct {
		name          string
		claims        map[string]interface{}
		expectAllowed bool
	}{
		{
			name: "GDPR data processing with consent",
			claims: map[string]interface{}{
				"gdpr_consent": true,
			},
			expectAllowed: true,
		},
		{
			name: "GDPR data processing without consent",
			claims: map[string]interface{}{
				"gdpr_consent": false,
			},
			expectAllowed: false,
		},
		{
			name:          "GDPR data processing missing consent claim",
			claims:        map[string]interface{}{},
			expectAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enfCtx := &EnforcementContext{
				RequestID:    generateRequestID(),
				Subject:      "user@example.com",
				Resource:     "customer-data",
				Action:       "gdpr_data_processing",
				EntityType:   compliance.EntityTypeCorporation,
				Jurisdiction: compliance.JurisdictionEU,
				Claims:       tt.claims,
				Timestamp:    time.Now(),
			}

			decision, err := engine.Enforce(ctx, enfCtx)
			if err != nil {
				t.Fatalf("Enforce returned error: %v", err)
			}

			if decision.Allowed != tt.expectAllowed {
				t.Errorf("Expected allowed=%v, got %v. Violations: %v",
					tt.expectAllowed, decision.Allowed, decision.Violations)
			}
		})
	}
}

func TestEnforce_CCPA_DataProcessing(t *testing.T) {
	engine := NewEnforcementEngine()
	ctx := context.Background()

	tests := []struct {
		name          string
		claims        map[string]interface{}
		expectAllowed bool
	}{
		{
			name: "CCPA data processing without opt-out",
			claims: map[string]interface{}{
				"ccpa_opt_out": false,
			},
			expectAllowed: true,
		},
		{
			name: "CCPA data processing with opt-out",
			claims: map[string]interface{}{
				"ccpa_opt_out": true,
			},
			expectAllowed: false,
		},
		{
			name:          "CCPA data processing missing opt-out claim",
			claims:        map[string]interface{}{},
			expectAllowed: true, // Default is allowed if no opt-out
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enfCtx := &EnforcementContext{
				RequestID:    generateRequestID(),
				Subject:      "user@example.com",
				Resource:     "customer-data",
				Action:       "ccpa_data_processing",
				EntityType:   compliance.EntityTypeCorporation,
				Jurisdiction: compliance.JurisdictionUS,
				Claims:       tt.claims,
				Timestamp:    time.Now(),
			}

			decision, err := engine.Enforce(ctx, enfCtx)
			if err != nil {
				t.Fatalf("Enforce returned error: %v", err)
			}

			if decision.Allowed != tt.expectAllowed {
				t.Errorf("Expected allowed=%v, got %v. Violations: %v",
					tt.expectAllowed, decision.Allowed, decision.Violations)
			}
		})
	}
}

func TestEnforce_CrossBorderDataTransfer(t *testing.T) {
	engine := NewEnforcementEngine()
	ctx := context.Background()

	tests := []struct {
		name                    string
		sourceJurisdiction      compliance.Jurisdiction
		destinationJurisdiction string
		action                  string
		expectAllowed           bool
	}{
		{
			name:                    "EU to UK personal data transfer (allowed)",
			sourceJurisdiction:      compliance.JurisdictionEU,
			destinationJurisdiction: "UK",
			action:                  "personal_data_transfer",
			expectAllowed:           true,
		},
		{
			name:                    "EU to US personal data transfer (not allowed)",
			sourceJurisdiction:      compliance.JurisdictionEU,
			destinationJurisdiction: "US",
			action:                  "personal_data_transfer",
			expectAllowed:           false,
		},
		{
			name:                    "US to EU personal data transfer (allowed)",
			sourceJurisdiction:      compliance.JurisdictionUS,
			destinationJurisdiction: "EU",
			action:                  "personal_data_transfer",
			expectAllowed:           true,
		},
		{
			name:                    "Japan to EU personal data transfer (allowed)",
			sourceJurisdiction:      compliance.JurisdictionJP,
			destinationJurisdiction: "EU",
			action:                  "personal_data_transfer",
			expectAllowed:           true,
		},
		{
			name:                    "Japan to US personal data transfer (not allowed)",
			sourceJurisdiction:      compliance.JurisdictionJP,
			destinationJurisdiction: "US",
			action:                  "personal_data_transfer",
			expectAllowed:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enfCtx := &EnforcementContext{
				RequestID:    generateRequestID(),
				Subject:      "user@example.com",
				Resource:     "customer-data",
				Action:       tt.action,
				EntityType:   compliance.EntityTypeCorporation,
				Jurisdiction: tt.sourceJurisdiction,
				Claims: map[string]interface{}{
					"destination_jurisdiction": tt.destinationJurisdiction,
				},
				Timestamp: time.Now(),
			}

			decision, err := engine.Enforce(ctx, enfCtx)
			if err != nil {
				t.Fatalf("Enforce returned error: %v", err)
			}

			if decision.Allowed != tt.expectAllowed {
				t.Errorf("Expected allowed=%v, got %v. Violations: %v",
					tt.expectAllowed, decision.Allowed, decision.Violations)
			}

			// Verify cross-border metrics were updated
			metrics := engine.GetMetrics()
			if metrics.CrossBorderAttempts == 0 {
				t.Error("Expected cross-border attempts to be recorded")
			}
		})
	}
}

func TestEnforce_DataResidency(t *testing.T) {
	engine := NewEnforcementEngine()
	ctx := context.Background()

	tests := []struct {
		name                    string
		sourceJurisdiction      compliance.Jurisdiction
		destinationJurisdiction string
		dataType                string
		expectAllowed           bool
	}{
		{
			name:                    "EU personal data must stay in EU",
			sourceJurisdiction:      compliance.JurisdictionEU,
			destinationJurisdiction: "EU",
			dataType:                "personal_data",
			expectAllowed:           true,
		},
		{
			name:                    "EU personal data leaving EU (violation)",
			sourceJurisdiction:      compliance.JurisdictionEU,
			destinationJurisdiction: "US",
			dataType:                "personal_data",
			expectAllowed:           false,
		},
		{
			name:                    "EU health data leaving EU (violation)",
			sourceJurisdiction:      compliance.JurisdictionEU,
			destinationJurisdiction: "US",
			dataType:                "health_data",
			expectAllowed:           false,
		},
		{
			name:                    "US personal data to EU (allowed - no residency requirement)",
			sourceJurisdiction:      compliance.JurisdictionUS,
			destinationJurisdiction: "EU",
			dataType:                "personal_data",
			expectAllowed:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enfCtx := &EnforcementContext{
				RequestID:    generateRequestID(),
				Subject:      "user@example.com",
				Resource:     "data-store",
				Action:       "data_export",
				EntityType:   compliance.EntityTypeCorporation,
				Jurisdiction: tt.sourceJurisdiction,
				Claims: map[string]interface{}{
					"destination_jurisdiction": tt.destinationJurisdiction,
					"data_type":                tt.dataType,
				},
				Timestamp: time.Now(),
			}

			decision, err := engine.Enforce(ctx, enfCtx)
			if err != nil {
				t.Fatalf("Enforce returned error: %v", err)
			}

			if decision.Allowed != tt.expectAllowed {
				t.Errorf("Expected allowed=%v, got %v. Violations: %v",
					tt.expectAllowed, decision.Allowed, decision.Violations)
			}

			// Check data residency violations metric
			if !tt.expectAllowed && tt.destinationJurisdiction != string(tt.sourceJurisdiction) {
				metrics := engine.GetMetrics()
				if metrics.DataResidencyViolations == 0 {
					t.Error("Expected data residency violation to be recorded")
				}
			}
		})
	}
}

func TestEnforce_BlockedActions(t *testing.T) {
	engine := NewEnforcementEngine()
	ctx := context.Background()

	tests := []struct {
		name          string
		jurisdiction  compliance.Jurisdiction
		action        string
		expectAllowed bool
	}{
		{
			name:          "EU unrestricted data export (blocked)",
			jurisdiction:  compliance.JurisdictionEU,
			action:        "unrestricted_data_export",
			expectAllowed: false,
		},
		{
			name:          "EU automated profiling (blocked)",
			jurisdiction:  compliance.JurisdictionEU,
			action:        "automated_profiling",
			expectAllowed: false,
		},
		{
			name:          "US autonomous high risk decision (blocked)",
			jurisdiction:  compliance.JurisdictionUS,
			action:        "autonomous_high_risk_decision",
			expectAllowed: false,
		},
		{
			name:          "UK unrestricted data export (blocked)",
			jurisdiction:  compliance.JurisdictionUK,
			action:        "unrestricted_data_export",
			expectAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enfCtx := &EnforcementContext{
				RequestID:    generateRequestID(),
				Subject:      "user@example.com",
				Resource:     "system",
				Action:       tt.action,
				EntityType:   compliance.EntityTypeCorporation,
				Jurisdiction: tt.jurisdiction,
				Claims:       map[string]interface{}{},
				Timestamp:    time.Now(),
			}

			decision, err := engine.Enforce(ctx, enfCtx)
			if err != nil {
				t.Fatalf("Enforce returned error: %v", err)
			}

			if decision.Allowed != tt.expectAllowed {
				t.Errorf("Expected allowed=%v, got %v. Violations: %v",
					tt.expectAllowed, decision.Allowed, decision.Violations)
			}

			if !tt.expectAllowed && len(decision.Violations) == 0 {
				t.Error("Expected violations for blocked action")
			}
		})
	}
}

func TestEnforce_Metrics(t *testing.T) {
	engine := NewEnforcementEngine()
	ctx := context.Background()

	// Perform multiple enforcements
	for i := 0; i < 10; i++ {
		enfCtx := &EnforcementContext{
			RequestID:    generateRequestID(),
			Subject:      "user@example.com",
			Resource:     "system",
			Action:       "test_action",
			EntityType:   compliance.EntityTypeCorporation,
			Jurisdiction: compliance.JurisdictionUS,
			Claims:       map[string]interface{}{},
			Timestamp:    time.Now(),
		}

		_, _ = engine.Enforce(ctx, enfCtx)
	}

	metrics := engine.GetMetrics()

	if metrics.TotalEnforcements != 10 {
		t.Errorf("Expected 10 total enforcements, got %d", metrics.TotalEnforcements)
	}

	if metrics.AllowedCount+metrics.DeniedCount != metrics.TotalEnforcements {
		t.Error("Allowed + Denied should equal Total")
	}

	if metrics.JurisdictionBreakdown[compliance.JurisdictionUS] != 10 {
		t.Errorf("Expected 10 US jurisdiction enforcements, got %d",
			metrics.JurisdictionBreakdown[compliance.JurisdictionUS])
	}

	if metrics.AverageLatencyMs == 0 {
		t.Error("Expected average latency to be recorded")
	}
}

func TestEnforce_AuditCallback(t *testing.T) {
	engine := NewEnforcementEngine()
	ctx := context.Background()

	var auditedDecisions []EnforcementDecision
	var mu sync.Mutex

	// Set audit callback
	engine.SetAuditCallback(func(decision EnforcementDecision) {
		mu.Lock()
		defer mu.Unlock()
		auditedDecisions = append(auditedDecisions, decision)
	})

	// Perform enforcement
	enfCtx := &EnforcementContext{
		RequestID:    "test-audit",
		Subject:      "user@example.com",
		Resource:     "system",
		Action:       "test_action",
		EntityType:   compliance.EntityTypeCorporation,
		Jurisdiction: compliance.JurisdictionUS,
		Claims:       map[string]interface{}{},
		Timestamp:    time.Now(),
	}

	_, err := engine.Enforce(ctx, enfCtx)
	if err != nil {
		t.Fatalf("Enforce returned error: %v", err)
	}

	// Wait for audit callback
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(auditedDecisions) != 1 {
		t.Errorf("Expected 1 audited decision, got %d", len(auditedDecisions))
	}

	if len(auditedDecisions) > 0 && auditedDecisions[0].RequestID != "test-audit" {
		t.Errorf("Expected request ID 'test-audit', got '%s'", auditedDecisions[0].RequestID)
	}
}

func TestEnforce_EnableDisable(t *testing.T) {
	engine := NewEnforcementEngine()
	ctx := context.Background()

	enfCtx := &EnforcementContext{
		RequestID:    generateRequestID(),
		Subject:      "user@example.com",
		Resource:     "system",
		Action:       "test_action",
		EntityType:   compliance.EntityTypeCorporation,
		Jurisdiction: compliance.Jurisdiction("INVALID"), // Would fail if enabled
		Claims:       map[string]interface{}{},
		Timestamp:    time.Now(),
	}

	// Disable enforcement
	engine.SetEnabled(false)

	decision, err := engine.Enforce(ctx, enfCtx)
	if err != nil {
		t.Fatalf("Enforce returned error: %v", err)
	}

	if !decision.Allowed {
		t.Error("Expected enforcement to be bypassed when disabled")
	}

	if len(decision.Warnings) == 0 {
		t.Error("Expected warning when enforcement is disabled")
	}

	// Re-enable enforcement
	engine.SetEnabled(true)

	decision, err = engine.Enforce(ctx, enfCtx)
	if err != nil {
		t.Fatalf("Enforce returned error: %v", err)
	}

	if decision.Allowed {
		t.Error("Expected enforcement to deny invalid jurisdiction when enabled")
	}
}

func TestExtractJurisdictionFromClaims(t *testing.T) {
	tests := []struct {
		name                 string
		claims               map[string]interface{}
		expectedJurisdiction compliance.Jurisdiction
	}{
		{
			name: "Direct jurisdiction claim",
			claims: map[string]interface{}{
				"jurisdiction": "EU",
			},
			expectedJurisdiction: compliance.JurisdictionEU,
		},
		{
			name: "Location claim - Europe",
			claims: map[string]interface{}{
				"location": "Europe",
			},
			expectedJurisdiction: compliance.JurisdictionEU,
		},
		{
			name: "Location claim - USA",
			claims: map[string]interface{}{
				"location": "USA",
			},
			expectedJurisdiction: compliance.JurisdictionUS,
		},
		{
			name: "Location claim - UK",
			claims: map[string]interface{}{
				"location": "United Kingdom",
			},
			expectedJurisdiction: compliance.JurisdictionUK,
		},
		{
			name:                 "No jurisdiction claim - defaults to US",
			claims:               map[string]interface{}{},
			expectedJurisdiction: compliance.JurisdictionUS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jurisdiction := ExtractJurisdictionFromClaims(tt.claims)
			if jurisdiction != tt.expectedJurisdiction {
				t.Errorf("Expected jurisdiction %s, got %s", tt.expectedJurisdiction, jurisdiction)
			}
		})
	}
}

func TestConcurrentEnforcement(t *testing.T) {
	engine := NewEnforcementEngine()
	ctx := context.Background()

	var wg sync.WaitGroup
	concurrency := 100

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			enfCtx := &EnforcementContext{
				RequestID:    generateRequestID(),
				Subject:      "user@example.com",
				Resource:     "system",
				Action:       "test_action",
				EntityType:   compliance.EntityTypeCorporation,
				Jurisdiction: compliance.JurisdictionUS,
				Claims:       map[string]interface{}{},
				Timestamp:    time.Now(),
			}

			_, err := engine.Enforce(ctx, enfCtx)
			if err != nil {
				t.Errorf("Concurrent enforcement %d failed: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	metrics := engine.GetMetrics()
	if metrics.TotalEnforcements != int64(concurrency) {
		t.Errorf("Expected %d total enforcements, got %d", concurrency, metrics.TotalEnforcements)
	}
}

func BenchmarkEnforce(b *testing.B) {
	engine := NewEnforcementEngine()
	ctx := context.Background()

	enfCtx := &EnforcementContext{
		RequestID:    "bench-test",
		Subject:      "user@example.com",
		Resource:     "system",
		Action:       "test_action",
		EntityType:   compliance.EntityTypeCorporation,
		Jurisdiction: compliance.JurisdictionUS,
		Claims:       map[string]interface{}{},
		Timestamp:    time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Enforce(ctx, enfCtx)
	}
}

func BenchmarkEnforce_CrossBorder(b *testing.B) {
	engine := NewEnforcementEngine()
	ctx := context.Background()

	enfCtx := &EnforcementContext{
		RequestID:    "bench-test",
		Subject:      "user@example.com",
		Resource:     "customer-data",
		Action:       "personal_data_transfer",
		EntityType:   compliance.EntityTypeCorporation,
		Jurisdiction: compliance.JurisdictionEU,
		Claims: map[string]interface{}{
			"destination_jurisdiction": "UK",
		},
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Enforce(ctx, enfCtx)
	}
}
