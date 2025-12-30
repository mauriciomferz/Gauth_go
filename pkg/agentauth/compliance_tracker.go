package agentauth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/poa"
)

// ComplianceTracker monitors ongoing compliance for active authorizations
// This implements RFC-0111 Step (i): Compliance Tracking
type ComplianceTracker interface {
	// StartTracking begins compliance monitoring for an extended token
	StartTracking(ctx context.Context, req *ComplianceTrackingRequest) error

	// CheckCompliance performs a compliance check for a specific token
	CheckCompliance(ctx context.Context, tokenID string) (*ComplianceStatus, error)

	// StopTracking stops compliance monitoring for a token
	StopTracking(ctx context.Context, tokenID string) error

	// GetTrackingStatus returns the current tracking status
	GetTrackingStatus(ctx context.Context, tokenID string) (*ComplianceTrackingStatus, error)

	// ListActiveTracking returns all tokens currently being tracked
	ListActiveTracking(ctx context.Context) ([]string, error)
}

// ComplianceTrackingRequest contains information needed to start tracking
type ComplianceTrackingRequest struct {
	ExtendedTokenID  string
	ClientID         string
	ResourceOwnerID  string
	PoACredential    *poa.PoADefinition
	MonitoringPeriod time.Duration
	CheckInterval    time.Duration // How often to check compliance
}

// ComplianceTrackingStatus represents the current tracking state
type ComplianceTrackingStatus struct {
	TokenID          string
	StartedAt        time.Time
	LastChecked      time.Time
	NextCheck        time.Time
	CheckCount       int
	ComplianceStatus *ComplianceStatus
	Active           bool
}

// MemoryComplianceTracker is an in-memory implementation of ComplianceTracker
type MemoryComplianceTracker struct {
	mu           sync.RWMutex
	tracking     map[string]*ComplianceTrackingStatus
	stopChannels map[string]chan struct{}

	// Dependencies for compliance checks
	complianceValidator *ComplianceValidator
}

// NewMemoryComplianceTracker creates a new in-memory compliance tracker
func NewMemoryComplianceTracker(validator *ComplianceValidator) *MemoryComplianceTracker {
	return &MemoryComplianceTracker{
		tracking:            make(map[string]*ComplianceTrackingStatus),
		stopChannels:        make(map[string]chan struct{}),
		complianceValidator: validator,
	}
}

// StartTracking begins compliance monitoring for an extended token
func (t *MemoryComplianceTracker) StartTracking(ctx context.Context, req *ComplianceTrackingRequest) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check if already tracking
	if _, exists := t.tracking[req.ExtendedTokenID]; exists {
		return fmt.Errorf("already tracking token: %s", req.ExtendedTokenID)
	}

	// Set default check interval if not provided
	checkInterval := req.CheckInterval
	if checkInterval == 0 {
		checkInterval = 1 * time.Hour // Default: check every hour
	}

	// Create tracking status
	now := time.Now()
	status := &ComplianceTrackingStatus{
		TokenID:     req.ExtendedTokenID,
		StartedAt:   now,
		LastChecked: now,
		NextCheck:   now.Add(checkInterval),
		CheckCount:  0,
		ComplianceStatus: &ComplianceStatus{
			Compliant:   true,
			Violations:  []string{},
			LastChecked: now,
			NextCheck:   now.Add(checkInterval),
		},
		Active: true,
	}

	t.tracking[req.ExtendedTokenID] = status

	// Create stop channel for this tracking
	stopChan := make(chan struct{})
	t.stopChannels[req.ExtendedTokenID] = stopChan

	// Start background monitoring goroutine
	go t.monitorCompliance(req, status, stopChan, checkInterval)

	return nil
}

// monitorCompliance runs in the background to periodically check compliance
func (t *MemoryComplianceTracker) monitorCompliance(
	req *ComplianceTrackingRequest,
	status *ComplianceTrackingStatus,
	stopChan chan struct{},
	checkInterval time.Duration,
) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			// Stop monitoring
			return

		case <-ticker.C:
			// Perform compliance check
			ctx := context.Background()
			complianceStatus, err := t.performComplianceCheck(ctx, req)
			if err != nil {
				// Log error but continue monitoring
				fmt.Printf("Compliance check failed for token %s: %v\n", req.ExtendedTokenID, err)
				continue
			}

			// Update tracking status
			t.mu.Lock()
			status.LastChecked = time.Now()
			status.NextCheck = time.Now().Add(checkInterval)
			status.CheckCount++
			status.ComplianceStatus = complianceStatus
			t.mu.Unlock()

			// If non-compliant, log violation
			if !complianceStatus.Compliant {
				fmt.Printf("COMPLIANCE VIOLATION for token %s: %v\n",
					req.ExtendedTokenID, complianceStatus.Violations)
			}
		}
	}
}

// performComplianceCheck performs the actual compliance validation
func (t *MemoryComplianceTracker) performComplianceCheck(
	ctx context.Context,
	req *ComplianceTrackingRequest,
) (*ComplianceStatus, error) {
	// In a real implementation, this would:
	// 1. Check if PoA credential is still valid
	// 2. Verify authorization chain integrity
	// 3. Check if any restrictions have been violated
	// 4. Verify resource owner/server still authorize the client
	// 5. Check for any revocations

	violations := []string{}

	// Check PoA validity period
	if req.PoACredential != nil {
		validityPeriod := req.PoACredential.Requirements.ValidityPeriod
		if !validityPeriod.StartTime.IsZero() && time.Now().Before(validityPeriod.StartTime) {
			violations = append(violations, "PoA not yet valid")
		}
		if !validityPeriod.EndTime.IsZero() && time.Now().After(validityPeriod.EndTime) {
			violations = append(violations, "PoA has expired")
		}
	}

	// Additional checks can be added here:
	// - Transaction limits exceeded
	// - Geographic restrictions violated
	// - Time-based restrictions violated
	// - Revocation list checks

	now := time.Now()
	return &ComplianceStatus{
		Compliant:   len(violations) == 0,
		Violations:  violations,
		LastChecked: now,
		NextCheck:   now.Add(1 * time.Hour),
	}, nil
}

// CheckCompliance performs an immediate compliance check
func (t *MemoryComplianceTracker) CheckCompliance(ctx context.Context, tokenID string) (*ComplianceStatus, error) {
	t.mu.RLock()
	status, exists := t.tracking[tokenID]
	t.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("token not being tracked: %s", tokenID)
	}

	if !status.Active {
		return status.ComplianceStatus, nil
	}

	// Return current compliance status
	return status.ComplianceStatus, nil
}

// StopTracking stops compliance monitoring for a token
func (t *MemoryComplianceTracker) StopTracking(ctx context.Context, tokenID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	status, exists := t.tracking[tokenID]
	if !exists {
		return fmt.Errorf("token not being tracked: %s", tokenID)
	}

	// Mark as inactive
	status.Active = false

	// Signal the monitoring goroutine to stop
	if stopChan, exists := t.stopChannels[tokenID]; exists {
		close(stopChan)
		delete(t.stopChannels, tokenID)
	}

	return nil
}

// GetTrackingStatus returns the current tracking status
func (t *MemoryComplianceTracker) GetTrackingStatus(ctx context.Context, tokenID string) (*ComplianceTrackingStatus, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	status, exists := t.tracking[tokenID]
	if !exists {
		return nil, fmt.Errorf("token not being tracked: %s", tokenID)
	}

	return status, nil
}

// ListActiveTracking returns all tokens currently being tracked
func (t *MemoryComplianceTracker) ListActiveTracking(ctx context.Context) ([]string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var activeTokens []string
	for tokenID, status := range t.tracking {
		if status.Active {
			activeTokens = append(activeTokens, tokenID)
		}
	}

	return activeTokens, nil
}

// GetStats returns statistics about compliance tracking (useful for monitoring)
func (t *MemoryComplianceTracker) GetStats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_tracked"] = len(t.tracking)

	activeCount := 0
	compliantCount := 0
	violationCount := 0

	for _, status := range t.tracking {
		if status.Active {
			activeCount++
		}
		if status.ComplianceStatus != nil {
			if status.ComplianceStatus.Compliant {
				compliantCount++
			} else {
				violationCount++
			}
		}
	}

	stats["active"] = activeCount
	stats["compliant"] = compliantCount
	stats["violations"] = violationCount

	return stats
}
