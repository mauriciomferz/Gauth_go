package cascade

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/config"
	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/agentauth_aap_001"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
)

// ProcessorResult holds the results of cascade processing
type ProcessorResult struct {
	ProcessedCount  int           // Total number of descendants processed
	SuccessCount    int           // Number of successfully updated descendants
	FailureCount    int           // Number of failed updates
	MaxDepthReached int           // Maximum depth reached during processing
	ProcessingTime  time.Duration // Total time taken for processing
	BatchCount      int           // Number of batches processed
	Errors          []error       // Errors encountered during processing
}

// Processor handles cascade revocation operations
type Processor struct {
	repo    agentauth_aap_001.POARepository
	auditor *audit.MemoryLogger
	config  config.CascadeConfig
	metrics metrics.Metrics
}

// NewProcessor creates a new cascade processor
func NewProcessor(repo agentauth_aap_001.POARepository, auditor *audit.MemoryLogger, cfg config.CascadeConfig, metricsImpl metrics.Metrics) *Processor {
	if metricsImpl == nil {
		metricsImpl = metrics.Noop
	}
	return &Processor{
		repo:    repo,
		auditor: auditor,
		config:  cfg,
		metrics: metricsImpl,
	}
}

// ProcessCascadeRevocation processes descendants of a revoked parent POA
func (p *Processor) ProcessCascadeRevocation(ctx context.Context, parentPoaID, revokedBy string) (*ProcessorResult, error) {
	if !p.config.ShouldProcessCascade() {
		return &ProcessorResult{}, fmt.Errorf("cascade processing disabled")
	}

	startTime := time.Now()
	result := &ProcessorResult{
		ProcessingTime: 0,
		Errors:         []error{},
	}

	// Increment cascade trigger metric
	p.metrics.IncCascadeRevocationTriggered()

	// Audit log the cascade initiation
	if p.auditor != nil {
		event := &audit.Event{
			Type:      audit.EventTypeResourceAccess,
			Action:    "cascade_start",
			Subject:   parentPoaID,
			Object:    "cascade_processor",
			Result:    "initiated",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"max_depth": p.config.MaxDepth,
				"operation": "revoke_parent",
			},
		}
		if err := p.auditor.Log(ctx, event); err != nil { //nolint:SA9003
			// Log the audit error but continue processing
			_ = err // Log the audit error but continue processing
		}
	}

	// Find all descendants
	descendants, err := p.repo.ListDescendants(parentPoaID, p.config.MaxDepth)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("failed to list descendants: %w", err))
		result.ProcessingTime = time.Since(startTime)
		return result, err
	}

	if len(descendants) == 0 {
		result.ProcessingTime = time.Since(startTime)
		return result, nil // No descendants to process
	}

	// Group descendants by depth for ordered processing
	depthGroups := p.groupByDepth(descendants)

	// Calculate max depth reached
	for depth := range depthGroups {
		if depth > result.MaxDepthReached {
			result.MaxDepthReached = depth
		}
	}

	// Update max depth metric
	if result.MaxDepthReached > 0 {
		p.metrics.SetCascadeMaxDepthReached(result.MaxDepthReached)
	}

	// Process each depth level in order (parents before children)
	for depth := 1; depth <= result.MaxDepthReached; depth++ {
		if depthDescendants, exists := depthGroups[depth]; exists {
			// Check if we're at max depth limit
			if depth >= p.config.MaxDepth {
				p.metrics.IncCascadeDepthLimitReached()
			}

			batchResult := p.processBatch(ctx, depthDescendants, depth, revokedBy)
			result.ProcessedCount += batchResult.ProcessedCount
			result.SuccessCount += batchResult.SuccessCount
			result.FailureCount += batchResult.FailureCount
			result.BatchCount += batchResult.BatchCount
			result.Errors = append(result.Errors, batchResult.Errors...)
		}
	}

	result.ProcessingTime = time.Since(startTime)

	// Record processing latency metric
	p.metrics.ObserveCascadeProcessingLatency(result.ProcessingTime)

	// Final audit log
	if p.auditor != nil {
		event := &audit.Event{
			Type:      audit.EventTypeAuthorization,
			Action:    "cascade_revocation_completed",
			Subject:   revokedBy,
			Object:    parentPoaID,
			Result:    p.getOutcome(result),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"processed_count":    result.ProcessedCount,
				"success_count":      result.SuccessCount,
				"failure_count":      result.FailureCount,
				"max_depth_reached":  result.MaxDepthReached,
				"processing_time_ms": result.ProcessingTime.Milliseconds(),
				"batch_count":        result.BatchCount,
				"error_count":        len(result.Errors),
			},
		}
		if err := p.auditor.Log(ctx, event); err != nil { //nolint:SA9003
			// Log the audit error but continue processing
			_ = err // Log the audit error but continue processing
		}
	}

	return result, nil
}

// processBatch processes a batch of descendants at the same depth level
func (p *Processor) processBatch(ctx context.Context, descendants []*agentauth_aap_001.PowerOfAttorney, depth int, revokedBy string) *ProcessorResult {
	result := &ProcessorResult{
		Errors: []error{},
	}

	// Process in batches based on configuration
	batchSize := p.config.BatchSize
	if batchSize <= 0 {
		batchSize = len(descendants) // Process all at once
	}

	for i := 0; i < len(descendants); i += batchSize {
		end := i + batchSize
		if end > len(descendants) {
			end = len(descendants)
		}

		batch := descendants[i:end]
		result.BatchCount++

		for _, poa := range batch {
			select {
			case <-ctx.Done():
				result.Errors = append(result.Errors, ctx.Err())
				return result
			default:
			}

			err := p.processDescendant(poa, depth, revokedBy)
			result.ProcessedCount++
			p.metrics.IncCascadeDescendantsProcessed()

			if err != nil {
				result.FailureCount++
				result.Errors = append(result.Errors, fmt.Errorf("failed to process POA %s: %w", poa.ID, err))
				p.metrics.IncCascadeProcessingErrors()
			} else {
				result.SuccessCount++
			}
		}

		// Increment batch processed metric
		p.metrics.IncCascadeBatchProcessed()
	}

	return result
}

// processDescendant processes a single descendant POA based on cascade mode
func (p *Processor) processDescendant(poa *agentauth_aap_001.PowerOfAttorney, depth int, revokedBy string) error {
	originalStatus := poa.Status
	now := time.Now()

	switch p.config.Mode {
	case config.CascadeModeRevoke:
		poa.Status = agentauth_aap_001.POAStatusRevoked
		poa.RevokedAt = &now
		poa.RevocationReason = fmt.Sprintf("cascade_revocation:parent_revoked:depth_%d", depth)

	case config.CascadeModeSuspend:
		poa.Status = agentauth_aap_001.POAStatusSuspended
		poa.RevocationReason = fmt.Sprintf("cascade_suspension:parent_revoked:depth_%d", depth)

	case config.CascadeModeNotify:
		// Only log, don't change status
		if p.auditor != nil {
			event := &audit.Event{
				Type:      audit.EventTypeAuthorization,
				Action:    "cascade_notification",
				Subject:   revokedBy,
				Object:    poa.ID,
				Result:    "notified",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"depth":           depth,
					"parent_revoked":  true,
					"current_status":  string(poa.Status),
					"would_change_to": "revoked", // What would happen in real mode
				},
			}
			if err := p.auditor.Log(context.Background(), event); err != nil { //nolint:SA9003
				// Log the audit error but continue processing
				_ = err // Log the audit error but continue processing
			}
		}
		return nil // Don't actually update the POA

	default:
		return fmt.Errorf("invalid cascade mode: %s", p.config.Mode)
	}

	poa.UpdatedAt = now

	// Update the POA in repository
	if err := p.repo.Update(poa); err != nil {
		return fmt.Errorf("failed to update POA in repository: %w", err)
	}

	// Audit log the change
	if p.auditor != nil {
		event := &audit.Event{
			Type:      audit.EventTypeAuthorization,
			Action:    fmt.Sprintf("cascade_%s", p.config.Mode),
			Subject:   revokedBy,
			Object:    poa.ID,
			Result:    "success",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"depth":             depth,
				"original_status":   string(originalStatus),
				"new_status":        string(poa.Status),
				"revocation_reason": poa.RevocationReason,
				"cascade_mode":      string(p.config.Mode),
			},
		}
		if err := p.auditor.Log(context.Background(), event); err != nil { //nolint:SA9003
			// Log error but continue processing
			_ = err
		}
	}

	return nil
}

// groupByDepth organizes descendants by their depth level
func (p *Processor) groupByDepth(descendants []*agentauth_aap_001.PowerOfAttorney) map[int][]*agentauth_aap_001.PowerOfAttorney {
	groups := make(map[int][]*agentauth_aap_001.PowerOfAttorney)

	for _, poa := range descendants {
		depth := poa.Depth
		if depth <= 0 {
			depth = 1 // Default depth for descendants
		}
		groups[depth] = append(groups[depth], poa)
	}

	return groups
}

// getOutcome determines the overall outcome based on the result
func (p *Processor) getOutcome(result *ProcessorResult) string {
	switch {
	case result.FailureCount == 0:
		return "success"
	case result.SuccessCount > 0:
		return "partial_success"
	default:
		return "failure"
	}
}
