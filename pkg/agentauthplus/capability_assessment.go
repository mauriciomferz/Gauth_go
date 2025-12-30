// Package agentauthplus - Capability Assessment Service
package agentauthplus

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mauriciomferz/AgentAuth/pkg/database"
)

// PostgreSQLCapabilityAssessmentService implements CapabilityAssessmentService (pgx)
type PostgreSQLCapabilityAssessmentService struct {
	db *database.DB
}

// NewPostgreSQLCapabilityAssessmentService creates a new capability assessment service
func NewPostgreSQLCapabilityAssessmentService(db *database.DB) *PostgreSQLCapabilityAssessmentService {
	return &PostgreSQLCapabilityAssessmentService{db: db}
}

// CreateAssessment creates a new capability assessment
func (s *PostgreSQLCapabilityAssessmentService) CreateAssessment(
	ctx context.Context,
	assessment *AICapabilityAssessment,
) error {
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	if assessment.ID == "" {
		assessment.ID = uuid.New().String()
	}

	assessment.CreatedAt = time.Now().UTC()
	assessment.UpdatedAt = time.Now().UTC()

	domainScoresJSON, err := json.Marshal(assessment.DomainScores)
	if err != nil {
		return fmt.Errorf("failed to marshal domain_scores: %w", err)
	}

	riskProfileJSON, err := json.Marshal(assessment.RiskProfile)
	if err != nil {
		return fmt.Errorf("failed to marshal risk_profile: %w", err)
	}

	limitationsJSON, err := json.Marshal(assessment.Limitations)
	if err != nil {
		return fmt.Errorf("failed to marshal limitations: %w", err)
	}

	certificationsJSON, err := json.Marshal(assessment.Certifications)
	if err != nil {
		return fmt.Errorf("failed to marshal certifications: %w", err)
	}

	_, err = s.db.Pool.Exec(ctx, `
		INSERT INTO ai_capability_assessments (
			id, agent_id, assessed_by, assessment_date,
			overall_level, domain_scores, risk_profile, limitations,
			certifications, valid_until, notes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, assessment.ID, assessment.AgentID, assessment.AssessedBy,
		assessment.AssessmentDate, assessment.OverallLevel, domainScoresJSON,
		riskProfileJSON, limitationsJSON, certificationsJSON,
		assessment.ValidUntil, assessment.Notes, assessment.CreatedAt,
		assessment.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create assessment: %w", err)
	}

	return nil
}

// GetLatestAssessment retrieves the most recent valid assessment
func (s *PostgreSQLCapabilityAssessmentService) GetLatestAssessment(
	ctx context.Context,
	agentID string,
) (*AICapabilityAssessment, error) {
	if s.db == nil {
		return nil, nil // No assessment found
	}

	assessment := &AICapabilityAssessment{}
	var domainScoresJSON, riskProfileJSON, limitationsJSON, certificationsJSON []byte
	var notes *string

	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, agent_id, assessed_by, assessment_date,
		       overall_level, domain_scores, risk_profile, limitations,
		       certifications, valid_until, notes, created_at, updated_at
		FROM ai_capability_assessments
		WHERE agent_id = $1
		  AND (valid_until IS NULL OR valid_until > NOW())
		ORDER BY assessment_date DESC
		LIMIT 1
	`, agentID).Scan(
		&assessment.ID, &assessment.AgentID, &assessment.AssessedBy,
		&assessment.AssessmentDate, &assessment.OverallLevel,
		&domainScoresJSON, &riskProfileJSON, &limitationsJSON,
		&certificationsJSON, &assessment.ValidUntil, &notes,
		&assessment.CreatedAt, &assessment.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil // No assessment found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest assessment: %w", err)
	}

	if err := json.Unmarshal(domainScoresJSON, &assessment.DomainScores); err != nil {
		return nil, fmt.Errorf("failed to unmarshal domain_scores: %w", err)
	}

	if err := json.Unmarshal(riskProfileJSON, &assessment.RiskProfile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal risk_profile: %w", err)
	}

	if err := json.Unmarshal(limitationsJSON, &assessment.Limitations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal limitations: %w", err)
	}

	if err := json.Unmarshal(certificationsJSON, &assessment.Certifications); err != nil {
		return nil, fmt.Errorf("failed to unmarshal certifications: %w", err)
	}

	if notes != nil {
		assessment.Notes = *notes
	}

	return assessment, nil
}

// GetAssessmentHistory returns all assessments for agent
func (s *PostgreSQLCapabilityAssessmentService) GetAssessmentHistory(
	ctx context.Context,
	agentID string,
) ([]*AICapabilityAssessment, error) {
	if s.db == nil {
		return []*AICapabilityAssessment{}, nil
	}

	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, agent_id, assessed_by, assessment_date,
		       overall_level, domain_scores, risk_profile, limitations,
		       certifications, valid_until, notes, created_at, updated_at
		FROM ai_capability_assessments
		WHERE agent_id = $1
		ORDER BY assessment_date DESC
	`, agentID)

	if err != nil {
		return nil, fmt.Errorf("failed to get assessment history: %w", err)
	}
	defer rows.Close()

	var assessments []*AICapabilityAssessment
	for rows.Next() {
		assessment := &AICapabilityAssessment{}
		var domainScoresJSON, riskProfileJSON, limitationsJSON, certificationsJSON []byte
		var notes *string

		err := rows.Scan(
			&assessment.ID, &assessment.AgentID, &assessment.AssessedBy,
			&assessment.AssessmentDate, &assessment.OverallLevel,
			&domainScoresJSON, &riskProfileJSON, &limitationsJSON,
			&certificationsJSON, &assessment.ValidUntil, &notes,
			&assessment.CreatedAt, &assessment.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan assessment: %w", err)
		}

		if err := json.Unmarshal(domainScoresJSON, &assessment.DomainScores); err != nil {
			return nil, fmt.Errorf("failed to unmarshal domain_scores: %w", err)
		}

		if err := json.Unmarshal(riskProfileJSON, &assessment.RiskProfile); err != nil {
			return nil, fmt.Errorf("failed to unmarshal risk_profile: %w", err)
		}

		if err := json.Unmarshal(limitationsJSON, &assessment.Limitations); err != nil {
			return nil, fmt.Errorf("failed to unmarshal limitations: %w", err)
		}

		if err := json.Unmarshal(certificationsJSON, &assessment.Certifications); err != nil {
			return nil, fmt.Errorf("failed to unmarshal certifications: %w", err)
		}

		if notes != nil {
			assessment.Notes = *notes
		}

		assessments = append(assessments, assessment)
	}

	return assessments, rows.Err()
}

// MatchCapabilities checks if agent meets capability requirements
func (s *PostgreSQLCapabilityAssessmentService) CheckCapabilityMatch(
	ctx context.Context,
	agentID string,
	requirements *CapabilityRequirements,
) (bool, []string, error) {
	assessment, err := s.GetLatestAssessment(ctx, agentID)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get assessment: %w", err)
	}

	if assessment == nil {
		return false, []string{"No capability assessment found for agent"}, nil
	}

	// Check expiration
	if !assessment.ValidUntil.IsZero() && time.Now().UTC().After(assessment.ValidUntil) {
		return false, []string{"Capability assessment expired"}, nil
	}

	// Collect all reasons for mismatch
	var reasons []string

	// Check minimum overall level
	levelOrder := map[string]int{
		"L0": 0, "L1": 1, "L2": 2, "L3": 3, "L4": 4, "L5": 5,
	}

	assessmentLevel := levelOrder[assessment.OverallLevel]
	requiredLevel := levelOrder[requirements.MinimumLevel]

	if assessmentLevel < requiredLevel {
		reasons = append(reasons, fmt.Sprintf(
			"Agent level %s below required %s",
			assessment.OverallLevel,
			requirements.MinimumLevel,
		))
	}

	// Check domain-specific requirements
	for domain, minScore := range requirements.DomainScores {
		actualScore, exists := assessment.DomainScores[domain]
		if !exists {
			reasons = append(reasons, fmt.Sprintf("Agent not assessed in domain: %s", domain))
		} else if actualScore < minScore {
			reasons = append(reasons, fmt.Sprintf(
				"Agent score %.2f in domain %s below required %.2f",
				actualScore, domain, minScore,
			))
		}
	}

	// Check risk thresholds
	for riskType, maxThreshold := range requirements.RiskThresholds {
		actualRiskInterface, exists := assessment.RiskProfile[riskType]
		if !exists {
			reasons = append(reasons, fmt.Sprintf("Agent risk profile missing: %s", riskType))
			continue
		}

		actualRisk, ok := actualRiskInterface.(float64)
		if !ok {
			reasons = append(reasons, fmt.Sprintf("Invalid risk value type for %s", riskType))
			continue
		}

		if actualRisk > maxThreshold {
			reasons = append(reasons, fmt.Sprintf(
				"Agent %s risk %.2f exceeds threshold %.2f",
				riskType, actualRisk, maxThreshold,
			))
		}
	}

	// Check required certifications
	assessmentCertSet := make(map[string]bool)
	for _, cert := range assessment.Certifications {
		assessmentCertSet[cert] = true
	}

	for _, requiredCert := range requirements.RequiredCertifications {
		if !assessmentCertSet[requiredCert] {
			reasons = append(reasons, fmt.Sprintf("Agent missing certification: %s", requiredCert))
		}
	}

	if len(reasons) > 0 {
		return false, reasons, nil
	}

	return true, []string{"Agent meets all capability requirements"}, nil
}

// GetExpiringAssessments returns assessments expiring within specified days
func (s *PostgreSQLCapabilityAssessmentService) GetExpiringAssessments(
	ctx context.Context, daysUntilExpiry int,
) ([]*AICapabilityAssessment, error) {
	if s.db == nil {
		return []*AICapabilityAssessment{}, nil
	}

	query := `
		SELECT id, agent_id, assessment_date, overall_level,
			   domain_scores, risk_profile, limitations, certifications,
			   valid_until, notes, created_at, updated_at
		FROM ai_capability_assessments
		WHERE valid_until <= $1 AND valid_until > NOW()
		ORDER BY valid_until ASC`

	expiryDate := time.Now().AddDate(0, 0, daysUntilExpiry)
	rows, err := s.db.Pool.Query(ctx, query, expiryDate)
	if err != nil {
		return nil, fmt.Errorf("query expiring assessments: %w", err)
	}
	defer rows.Close()

	var assessments []*AICapabilityAssessment
	for rows.Next() {
		assessment := &AICapabilityAssessment{}
		var domainScoresJSON, riskProfileJSON, limitationsJSON, certificationsJSON []byte
		var notes sql.NullString

		err := rows.Scan(
			&assessment.ID, &assessment.AgentID,
			&assessment.AssessmentDate, &assessment.OverallLevel,
			&domainScoresJSON, &riskProfileJSON, &limitationsJSON,
			&certificationsJSON, &assessment.ValidUntil,
			&notes, &assessment.CreatedAt, &assessment.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan assessment: %w", err)
		}

		if err := json.Unmarshal(domainScoresJSON, &assessment.DomainScores); err != nil {
			return nil, fmt.Errorf("unmarshal domain scores: %w", err)
		}

		if err := json.Unmarshal(riskProfileJSON, &assessment.RiskProfile); err != nil {
			return nil, fmt.Errorf("unmarshal risk profile: %w", err)
		}

		if err := json.Unmarshal(limitationsJSON, &assessment.Limitations); err != nil {
			return nil, fmt.Errorf("unmarshal limitations: %w", err)
		}

		if err := json.Unmarshal(certificationsJSON, &assessment.Certifications); err != nil {
			return nil, fmt.Errorf("unmarshal certifications: %w", err)
		}

		if notes.Valid {
			assessment.Notes = notes.String
		}

		assessments = append(assessments, assessment)
	}

	return assessments, rows.Err()
}

// Capability level scoring helpers

// CalculateOverallLevel computes overall level from domain scores
func CalculateOverallLevel(domainScores map[string]float64) string {
	if len(domainScores) == 0 {
		return "L0"
	}

	// Calculate average score
	total := 0.0
	for _, score := range domainScores {
		total += score
	}
	avg := total / float64(len(domainScores))

	// Map average to level
	if avg >= 0.95 {
		return "L5"
	} else if avg >= 0.85 {
		return "L4"
	} else if avg >= 0.70 {
		return "L3"
	} else if avg >= 0.50 {
		return "L2"
	} else if avg >= 0.30 {
		return "L1"
	}
	return "L0"
}

// StandardDomains returns common capability domains
func StandardDomains() []string {
	return []string{
		"reasoning",
		"knowledge",
		"communication",
		"decision_making",
		"risk_assessment",
		"regulatory_compliance",
		"data_handling",
		"error_recovery",
		"explainability",
	}
}

// StandardRiskCategories returns common risk types
func StandardRiskCategories() []string {
	return []string{
		"data_breach",
		"unauthorized_access",
		"bias_discrimination",
		"decision_error",
		"compliance_violation",
		"system_failure",
		"manipulation",
	}
}
