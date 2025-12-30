// Package poa - AAP002 Section C.3 Rights and Obligations
// This implements rights and obligations framework for AI authorization
// as required by AAP002 Section C.3
package poa

import (
	"fmt"
	"time"
)

// RightsObligationSet represents comprehensive rights and obligations per AAP002 C.3
type RightsObligationSet struct {
	ReportingDuties   []ReportingDuty    `json:"reporting_duties,omitempty"`
	LiabilityRules    []LiabilityRule    `json:"liability_rules,omitempty"`
	CompensationRules []CompensationRule `json:"compensation_rules,omitempty"`
	AuditRequirements *AuditRequirements `json:"audit_requirements,omitempty"`
	ComplianceRules   []ComplianceRule   `json:"compliance_rules,omitempty"`
}

// ReportingDuty defines mandatory reporting obligations per AAP002 C.3.1
type ReportingDuty struct {
	// Type describes the report type
	Type ReportType `json:"type"`

	// Frequency defines reporting frequency
	Frequency ReportFrequency `json:"frequency"`

	// Recipients lists required report recipients
	Recipients []string `json:"recipients"`

	// Triggers defines event-based reporting triggers
	Triggers []ReportTrigger `json:"triggers,omitempty"`

	// Content specifies required report content
	Content []string `json:"content"`

	// Format specifies report format
	Format string `json:"format,omitempty"` // e.g., "JSON", "PDF", "HTML"

	// RetentionDays defines report retention period
	RetentionDays int `json:"retention_days,omitempty"`

	// Mandatory indicates if reporting is required
	Mandatory bool `json:"mandatory"`
}

// ReportType classifies report types
type ReportType string

const (
	ReportTypeActivity    ReportType = "activity"    // Activity logs
	ReportTypePerformance ReportType = "performance" // Performance metrics
	ReportTypeCompliance  ReportType = "compliance"  // Compliance status
	ReportTypeIncident    ReportType = "incident"    // Incident reports
	ReportTypeAudit       ReportType = "audit"       // Audit logs
	ReportTypeFinancial   ReportType = "financial"   // Financial reports
	ReportTypeDecision    ReportType = "decision"    // Decision rationale
	ReportTypeException   ReportType = "exception"   // Exception reports
	ReportTypeSecurity    ReportType = "security"    // Security events
	ReportTypeDataAccess  ReportType = "data_access" // Data access logs
)

// ReportFrequency defines reporting intervals
type ReportFrequency string

const (
	FrequencyRealtime ReportFrequency = "realtime" // Real-time streaming
	FrequencyHourly   ReportFrequency = "hourly"   // Every hour
	FrequencyDaily    ReportFrequency = "daily"    // Once per day
	FrequencyWeekly   ReportFrequency = "weekly"   // Once per week
	FrequencyMonthly  ReportFrequency = "monthly"  // Once per month
	FrequencyEvent    ReportFrequency = "event"    // Event-driven
)

// ReportTrigger defines event-based reporting conditions
type ReportTrigger struct {
	EventType string `json:"event_type"` // Event type triggering report
	Condition string `json:"condition"`  // Condition expression
	Severity  string `json:"severity"`   // Severity level
	Immediate bool   `json:"immediate"`  // Requires immediate reporting
}

// LiabilityRule defines liability attribution per AAP002 C.3.2
type LiabilityRule struct {
	// RuleID uniquely identifies the liability rule
	RuleID string `json:"rule_id"`

	// Scope defines when this rule applies
	Scope LiabilityScope `json:"scope"`

	// PrimaryParty identifies who bears primary liability
	PrimaryParty LiabilityParty `json:"primary_party"`

	// SecondaryParty identifies secondary liability
	SecondaryParty LiabilityParty `json:"secondary_party,omitempty"`

	// LiabilityType classifies liability type
	LiabilityType LiabilityType `json:"liability_type"`

	// MaxLiability defines liability cap
	MaxLiability *MonetaryAmount `json:"max_liability,omitempty"`

	// ExclusionsExemptions lists liability exemptions
	ExclusionsExemptions []string `json:"exclusions_exemptions,omitempty"`

	// InsuranceRequired indicates if insurance is required
	InsuranceRequired bool `json:"insurance_required,omitempty"`

	// MinInsuranceCoverage defines minimum coverage
	MinInsuranceCoverage *MonetaryAmount `json:"min_insurance_coverage,omitempty"`
}

// LiabilityScope defines when liability rule applies
type LiabilityScope struct {
	Actions       []string `json:"actions,omitempty"`       // Action types covered
	Sectors       []string `json:"sectors,omitempty"`       // Industry sectors
	Jurisdictions []string `json:"jurisdictions,omitempty"` // Legal jurisdictions
	DamageTypes   []string `json:"damage_types,omitempty"`  // Types of damage
}

// LiabilityParty identifies a liable party
type LiabilityParty string

const (
	LiabilityPrincipal      LiabilityParty = "principal"         // Principal organization
	LiabilityRepresentative LiabilityParty = "representative"    // Representative/owner
	LiabilityClient         LiabilityParty = "client"            // AI client/system
	LiabilityManufacturer   LiabilityParty = "manufacturer"      // AI manufacturer
	LiabilityOperator       LiabilityParty = "operator"          // System operator
	LiabilityShared         LiabilityParty = "shared"            // Shared liability
	LiabilityJointSeveral   LiabilityParty = "joint_and_several" // Joint and several
)

// LiabilityType classifies liability categories
type LiabilityType string

const (
	LiabilityStrict       LiabilityType = "strict"       // Strict liability
	LiabilityNegligence   LiabilityType = "negligence"   // Negligence-based
	LiabilityVicarious    LiabilityType = "vicarious"    // Vicarious liability
	LiabilityProduct      LiabilityType = "product"      // Product liability
	LiabilityProfessional LiabilityType = "professional" // Professional liability
	LiabilityContractual  LiabilityType = "contractual"  // Contractual liability
)

// MonetaryAmount represents a monetary value
type MonetaryAmount struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"` // ISO 4217 code
}

// CompensationRule defines compensation obligations per AAP002 C.3.3
type CompensationRule struct {
	// RuleID uniquely identifies the compensation rule
	RuleID string `json:"rule_id"`

	// TriggerConditions defines when compensation applies
	TriggerConditions []string `json:"trigger_conditions"`

	// CompensationType classifies compensation type
	CompensationType CompensationType `json:"compensation_type"`

	// Amount defines compensation amount
	Amount *MonetaryAmount `json:"amount,omitempty"`

	// CalculationMethod describes how to calculate compensation
	CalculationMethod string `json:"calculation_method,omitempty"`

	// PaymentSchedule defines payment terms
	PaymentSchedule *PaymentSchedule `json:"payment_schedule,omitempty"`

	// Beneficiaries lists who receives compensation
	Beneficiaries []string `json:"beneficiaries"`

	// MaxCompensation defines compensation cap
	MaxCompensation *MonetaryAmount `json:"max_compensation,omitempty"`

	// EscalationClauses define escalation conditions
	EscalationClauses []string `json:"escalation_clauses,omitempty"`
}

// CompensationType classifies compensation categories
type CompensationType string

const (
	CompensationFixed         CompensationType = "fixed"         // Fixed amount
	CompensationVariable      CompensationType = "variable"      // Variable/calculated
	CompensationPercentage    CompensationType = "percentage"    // Percentage-based
	CompensationActualLoss    CompensationType = "actual_loss"   // Actual loss reimbursement
	CompensationPunitive      CompensationType = "punitive"      // Punitive damages
	CompensationConsequential CompensationType = "consequential" // Consequential damages
)

// PaymentSchedule defines payment timing
type PaymentSchedule struct {
	Type         string        `json:"type"` // "immediate", "installment", "milestone"
	Frequency    string        `json:"frequency,omitempty"`
	Installments int           `json:"installments,omitempty"`
	DueDays      int           `json:"due_days,omitempty"`
	GracePeriod  time.Duration `json:"grace_period,omitempty"`
}

// AuditRequirements defines audit obligations per AAP002 C.3.4
type AuditRequirements struct {
	// AuditFrequency defines audit schedule
	AuditFrequency string `json:"audit_frequency"` // "quarterly", "annual", "biennial"

	// AuditScope defines what is audited
	AuditScope []string `json:"audit_scope"`

	// ExternalAuditorRequired indicates if external auditor needed
	ExternalAuditorRequired bool `json:"external_auditor_required"`

	// AccreditedAuditors lists approved auditors
	AccreditedAuditors []string `json:"accredited_auditors,omitempty"`

	// AuditStandards lists compliance standards
	AuditStandards []string `json:"audit_standards"`

	// RetentionYears defines audit record retention
	RetentionYears int `json:"retention_years"`

	// PublicDisclosure indicates if audit results are public
	PublicDisclosure bool `json:"public_disclosure"`

	// CertificationRequired indicates if certification needed
	CertificationRequired bool `json:"certification_required"`
}

// ComplianceRule defines regulatory compliance requirements per AAP002 C.3.5
type ComplianceRule struct {
	// RuleID uniquely identifies the compliance rule
	RuleID string `json:"rule_id"`

	// Regulation identifies the regulation
	Regulation string `json:"regulation"` // e.g., "GDPR", "CCPA", "HIPAA"

	// Jurisdiction identifies applicable jurisdiction
	Jurisdiction string `json:"jurisdiction"`

	// Requirements lists specific requirements
	Requirements []string `json:"requirements"`

	// ComplianceLevel defines required compliance level
	ComplianceLevel ComplianceLevel `json:"compliance_level"`

	// VerificationMethod describes how to verify compliance
	VerificationMethod string `json:"verification_method,omitempty"`

	// ReportingFrequency defines compliance reporting frequency
	ReportingFrequency ReportFrequency `json:"reporting_frequency"`

	// Penalties describes non-compliance penalties
	Penalties []Penalty `json:"penalties,omitempty"`
}

// ComplianceLevel defines compliance strictness
type ComplianceLevel string

const (
	ComplianceMandatory   ComplianceLevel = "mandatory"   // Legally required
	ComplianceRequired    ComplianceLevel = "required"    // Contractually required
	ComplianceRecommended ComplianceLevel = "recommended" // Best practice
	ComplianceOptional    ComplianceLevel = "optional"    // Optional
)

// Penalty describes non-compliance consequences
type Penalty struct {
	Type        string          `json:"type"` // "monetary", "suspension", "termination"
	Description string          `json:"description"`
	Amount      *MonetaryAmount `json:"amount,omitempty"`
	Duration    time.Duration   `json:"duration,omitempty"`
	Severity    string          `json:"severity"` // "minor", "major", "critical"
}

// Validate performs complete validation of rights/obligations set
func (ros *RightsObligationSet) Validate() error {
	for i, rd := range ros.ReportingDuties {
		if err := rd.Validate(); err != nil {
			return fmt.Errorf("reporting duty %d: %w", i, err)
		}
	}

	for i, lr := range ros.LiabilityRules {
		if err := lr.Validate(); err != nil {
			return fmt.Errorf("liability rule %d: %w", i, err)
		}
	}

	for i, cr := range ros.CompensationRules {
		if err := cr.Validate(); err != nil {
			return fmt.Errorf("compensation rule %d: %w", i, err)
		}
	}

	if ros.AuditRequirements != nil {
		if err := ros.AuditRequirements.Validate(); err != nil {
			return fmt.Errorf("audit requirements: %w", err)
		}
	}

	for i, cr := range ros.ComplianceRules {
		if err := cr.Validate(); err != nil {
			return fmt.Errorf("compliance rule %d: %w", i, err)
		}
	}

	return nil
}

// Validate validates reporting duty
func (rd *ReportingDuty) Validate() error {
	if rd.Type == "" {
		return fmt.Errorf("report type required")
	}

	if rd.Frequency == "" {
		return fmt.Errorf("report frequency required")
	}

	if len(rd.Recipients) == 0 {
		return fmt.Errorf("at least one recipient required")
	}

	if len(rd.Content) == 0 {
		return fmt.Errorf("report content specification required")
	}

	if rd.RetentionDays < 0 {
		return fmt.Errorf("retention days cannot be negative")
	}

	return nil
}

// Validate validates liability rule
func (lr *LiabilityRule) Validate() error {
	if lr.RuleID == "" {
		return fmt.Errorf("rule ID required")
	}

	if lr.PrimaryParty == "" {
		return fmt.Errorf("primary party required")
	}

	if lr.LiabilityType == "" {
		return fmt.Errorf("liability type required")
	}

	if lr.InsuranceRequired && lr.MinInsuranceCoverage == nil {
		return fmt.Errorf("insurance required but min coverage not specified")
	}

	return nil
}

// Validate validates compensation rule
func (cr *CompensationRule) Validate() error {
	if cr.RuleID == "" {
		return fmt.Errorf("rule ID required")
	}

	if len(cr.TriggerConditions) == 0 {
		return fmt.Errorf("trigger conditions required")
	}

	if cr.CompensationType == "" {
		return fmt.Errorf("compensation type required")
	}

	if len(cr.Beneficiaries) == 0 {
		return fmt.Errorf("at least one beneficiary required")
	}

	if cr.CompensationType == CompensationFixed && cr.Amount == nil {
		return fmt.Errorf("fixed compensation requires amount")
	}

	return nil
}

// Validate validates audit requirements
func (ar *AuditRequirements) Validate() error {
	if ar.AuditFrequency == "" {
		return fmt.Errorf("audit frequency required")
	}

	if len(ar.AuditScope) == 0 {
		return fmt.Errorf("audit scope required")
	}

	if len(ar.AuditStandards) == 0 {
		return fmt.Errorf("at least one audit standard required")
	}

	if ar.RetentionYears <= 0 {
		return fmt.Errorf("retention years must be positive")
	}

	if ar.ExternalAuditorRequired && len(ar.AccreditedAuditors) == 0 {
		return fmt.Errorf("external auditor required but no accredited auditors specified")
	}

	return nil
}

// Validate validates compliance rule
func (cr *ComplianceRule) Validate() error {
	if cr.RuleID == "" {
		return fmt.Errorf("rule ID required")
	}

	if cr.Regulation == "" {
		return fmt.Errorf("regulation required")
	}

	if cr.Jurisdiction == "" {
		return fmt.Errorf("jurisdiction required")
	}

	if len(cr.Requirements) == 0 {
		return fmt.Errorf("at least one requirement must be specified")
	}

	if cr.ComplianceLevel == "" {
		return fmt.Errorf("compliance level required")
	}

	return nil
}

// EnforceReportingCompliance checks if reporting obligations are met
func EnforceReportingCompliance(lastReport time.Time, duty ReportingDuty) error {
	if !duty.Mandatory {
		return nil
	}

	now := time.Now()
	var requiredInterval time.Duration

	switch duty.Frequency {
	case FrequencyHourly:
		requiredInterval = time.Hour
	case FrequencyDaily:
		requiredInterval = 24 * time.Hour
	case FrequencyWeekly:
		requiredInterval = 7 * 24 * time.Hour
	case FrequencyMonthly:
		requiredInterval = 30 * 24 * time.Hour
	case FrequencyEvent:
		return nil // Event-driven, not time-based
	default:
		return fmt.Errorf("unknown report frequency: %s", duty.Frequency)
	}

	if now.Sub(lastReport) > requiredInterval {
		return fmt.Errorf("reporting duty violated: %s report overdue by %v",
			duty.Type, now.Sub(lastReport)-requiredInterval)
	}

	return nil
}

// String returns human-readable representation
func (ros *RightsObligationSet) String() string {
	return fmt.Sprintf("Reporting: %d duties | Liability: %d rules | Compensation: %d rules | Compliance: %d rules",
		len(ros.ReportingDuties), len(ros.LiabilityRules),
		len(ros.CompensationRules), len(ros.ComplianceRules))
}
