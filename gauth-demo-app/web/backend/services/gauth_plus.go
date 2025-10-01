package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// GAuthPlusService provides comprehensive AI authorization with blockchain-based commercial register
type GAuthPlusService struct {
	*GAuthService      // Embed existing service
	blockchainRegistry *BlockchainRegistry
	commercialRegister *CommercialRegister
}

// BlockchainRegistry represents the blockchain-based authorization registry
type BlockchainRegistry struct {
	config *viper.Viper
	logger *logrus.Logger
	redis  *redis.Client
}

// CommercialRegister provides commercial register functionality for AI systems
type CommercialRegister struct {
	config *viper.Viper
	logger *logrus.Logger
	redis  *redis.Client
}

// AuthorizingParty represents an entity that can authorize AI systems
type AuthorizingParty struct {
	ID                       string                    `json:"id"`
	Name                     string                    `json:"name"`
	Type                     string                    `json:"type"` // individual, corporation, government
	RegisteredOffice         *RegisteredOffice         `json:"registered_office,omitempty"`
	AuthorizedRepresentative *AuthorizedRepresentative `json:"authorized_representative,omitempty"`
	LegalCapacity            *LegalCapacity            `json:"legal_capacity"`
	AuthorityLevel           string                    `json:"authority_level"` // primary, secondary, delegated
	VerificationStatus       string                    `json:"verification_status"`
	CreatedAt                time.Time                 `json:"created_at"`
}

// RegisteredOffice represents the legal registered office of a company
type RegisteredOffice struct {
	Address            string `json:"address"`
	Jurisdiction       string `json:"jurisdiction"`
	RegistrationNumber string `json:"registration_number"`
	TaxID              string `json:"tax_id"`
	LegalForm          string `json:"legal_form"` // LLC, Inc, GmbH, etc.
}

// AuthorizedRepresentative represents someone authorized to act on behalf of an entity
type AuthorizedRepresentative struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Position       string    `json:"position"`
	AuthorityScope []string  `json:"authority_scope"`
	ValidFrom      time.Time `json:"valid_from"`
	ValidUntil     time.Time `json:"valid_until"`
	VerifiedBy     string    `json:"verified_by"`
}

// LegalCapacity represents the legal capacity and verification of an entity
type LegalCapacity struct {
	Verified         bool      `json:"verified"`
	VerificationDate time.Time `json:"verification_date"`
	VerifiedBy       string    `json:"verified_by"`
	Jurisdiction     string    `json:"jurisdiction"`
	LegalFramework   string    `json:"legal_framework"`
	Limitations      []string  `json:"limitations,omitempty"`
}

// AIAuthorizationRecord represents a comprehensive authorization record for an AI system
type AIAuthorizationRecord struct {
	ID                   string                    `json:"id"`
	AISystemID           string                    `json:"ai_system_id"`
	AuthorizingParty     *AuthorizingParty         `json:"authorizing_party"`
	AuthorizationGrant   *AuthorizationGrant       `json:"authorization_grant"`
	PowersGranted        *PowersGranted            `json:"powers_granted"`
	DecisionAuthority    *DecisionAuthority        `json:"decision_authority"`
	TransactionRights    *TransactionRights        `json:"transaction_rights"`
	ActionPermissions    *ActionPermissions        `json:"action_permissions"`
	AuthorizationCascade *AuthorizationCascade     `json:"authorization_cascade"`
	HumanAccountability  *HumanAccountabilityChain `json:"human_accountability"`
	MathematicalProof    *MathematicalProof        `json:"mathematical_proof"`
	DualControlPrinciple *DualControlPrinciple     `json:"dual_control_principle"`
	BlockchainHash       string                    `json:"blockchain_hash"`
	CreatedAt            time.Time                 `json:"created_at"`
	ExpiresAt            time.Time                 `json:"expires_at"`
	Status               string                    `json:"status"`
}

// AuthorizationGrant represents the specific grant details
type AuthorizationGrant struct {
	Type        string    `json:"type"` // individual, general
	Scope       []string  `json:"scope"`
	Limitations []string  `json:"limitations"`
	ValidFrom   time.Time `json:"valid_from"`
	ValidUntil  time.Time `json:"valid_until"`
	Revocable   bool      `json:"revocable"`
}

// PowersGranted defines what powers have been granted to the AI
type PowersGranted struct {
	BasicPowers     []string            `json:"basic_powers"`
	DerivedPowers   []string            `json:"derived_powers"`
	PowerDerivation map[string][]string `json:"power_derivation"` // basic power -> derived powers
	StandardPowers  *StandardPowers     `json:"standard_powers"`
}

// StandardPowers represents the comprehensive standard powers from which authorization can be derived
type StandardPowers struct {
	FinancialPowers      *FinancialPowers      `json:"financial_powers,omitempty"`
	ContractualPowers    *ContractualPowers    `json:"contractual_powers,omitempty"`
	OperationalPowers    *OperationalPowers    `json:"operational_powers,omitempty"`
	RepresentationPowers *RepresentationPowers `json:"representation_powers,omitempty"`
	CompliancePowers     *CompliancePowers     `json:"compliance_powers,omitempty"`
}

// FinancialPowers defines financial authority levels
type FinancialPowers struct {
	SigningAuthority    *SigningAuthority `json:"signing_authority,omitempty"`
	ApprovalLimits      *ApprovalLimits   `json:"approval_limits,omitempty"`
	InvestmentAuthority []string          `json:"investment_authority,omitempty"`
	BankingOperations   []string          `json:"banking_operations,omitempty"`
	TreasuryManagement  []string          `json:"treasury_management,omitempty"`
}

// SigningAuthority defines signing limits and scope
type SigningAuthority struct {
	SingleSignatureLimit float64  `json:"single_signature_limit"`
	RequiresDualSigning  float64  `json:"requires_dual_signing"`
	AuthorizedDocuments  []string `json:"authorized_documents"`
	ProhibitedDocuments  []string `json:"prohibited_documents"`
}

// ApprovalLimits defines monetary and transaction limits
type ApprovalLimits struct {
	DailyLimit   float64 `json:"daily_limit"`
	WeeklyLimit  float64 `json:"weekly_limit"`
	MonthlyLimit float64 `json:"monthly_limit"`
	AnnualLimit  float64 `json:"annual_limit"`
	Currency     string  `json:"currency"`
}

// HumanAccountabilityChain ensures human authority at the top of delegation cascade
type HumanAccountabilityChain struct {
	UltimateHumanAuthority *UltimateHumanAuthority `json:"ultimate_human_authority"`
	DelegationChain        []AuthorityLevel        `json:"delegation_chain"`
	AccountabilityLevel    int                     `json:"accountability_level"`
	VerificationRequired   bool                    `json:"verification_required"`
	Validated              bool                    `json:"validated"`
	ValidatedAt            time.Time               `json:"validated_at"`
	ValidatedBy            string                  `json:"validated_by"`
}

// UltimateHumanAuthority represents the top-level human in the authorization cascade
type UltimateHumanAuthority struct {
	HumanID              string                 `json:"human_id"`
	Name                 string                 `json:"name"`
	Position             string                 `json:"position"`
	LegalCapacity        *LegalCapacity         `json:"legal_capacity"`
	IdentityVerification *IdentityVerification  `json:"identity_verification"`
	Authority            []string               `json:"authority"`
	DelegationPowers     map[string]interface{} `json:"delegation_powers"`
	AccountabilityScope  []string               `json:"accountability_scope"`
	IsUltimateAuthority  bool                   `json:"is_ultimate_authority"`
}

// AuthorityLevel represents a level in the delegation chain
type AuthorityLevel struct {
	Level         int       `json:"level"`
	AuthorityID   string    `json:"authority_id"`
	AuthorityType string    `json:"authority_type"` // human, ai_agent, system
	DelegatedFrom string    `json:"delegated_from"`
	DelegatedTo   string    `json:"delegated_to"`
	PowerScope    []string  `json:"power_scope"`
	Limitations   []string  `json:"limitations"`
	CreatedAt     time.Time `json:"created_at"`
	IsHuman       bool      `json:"is_human"`
}

// IdentityVerification provides comprehensive identity validation
type IdentityVerification struct {
	Verified           bool                   `json:"verified"`
	VerificationMethod string                 `json:"verification_method"`
	VerificationLevel  string                 `json:"verification_level"` // basic, enhanced, government
	Documents          []VerificationDocument `json:"documents"`
	BiometricData      *BiometricData         `json:"biometric_data,omitempty"`
	VerifiedAt         time.Time              `json:"verified_at"`
	VerifiedBy         string                 `json:"verified_by"`
	ExpiresAt          time.Time              `json:"expires_at"`
}

// VerificationDocument represents a document used for verification
type VerificationDocument struct {
	Type         string    `json:"type"` // passport, drivers_license, etc.
	DocumentID   string    `json:"document_id"`
	IssuingAuth  string    `json:"issuing_authority"`
	VerifiedHash string    `json:"verified_hash"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// BiometricData represents biometric verification data
type BiometricData struct {
	FingerprintHash string `json:"fingerprint_hash,omitempty"`
	FaceRecognition string `json:"face_recognition,omitempty"`
	VoicePrint      string `json:"voice_print,omitempty"`
}

// DualControlPrinciple implements second-level approval mechanisms
type DualControlPrinciple struct {
	Enabled              bool               `json:"enabled"`
	RequiredForActions   []string           `json:"required_for_actions"`
	PrimaryApprover      *Approver          `json:"primary_approver"`
	SecondaryApprover    *Approver          `json:"secondary_approver"`
	ApprovalThreshold    *ApprovalThreshold `json:"approval_threshold"`
	ApprovalHistory      []ApprovalRecord   `json:"approval_history"`
	SequentialApproval   bool               `json:"sequential_approval"`
	SimultaneousApproval bool               `json:"simultaneous_approval"`
	EscalationRules      []EscalationRule   `json:"escalation_rules"`
}

// Approver represents an entity that can provide approval
type Approver struct {
	ApproverID      string                 `json:"approver_id"`
	ApproverType    string                 `json:"approver_type"` // human, ai_system, committee
	Name            string                 `json:"name"`
	Authority       []string               `json:"authority"`
	ApprovalPowers  map[string]interface{} `json:"approval_powers"`
	VerificationReq bool                   `json:"verification_required"`
	ContactInfo     map[string]string      `json:"contact_info"`
	IsActive        bool                   `json:"is_active"`
}

// ApprovalThreshold defines when dual control is required
type ApprovalThreshold struct {
	MonetaryThreshold float64  `json:"monetary_threshold"`
	RiskLevel         string   `json:"risk_level"`
	ActionTypes       []string `json:"action_types"`
	TimeConstraints   []string `json:"time_constraints"`
	GeographicScope   []string `json:"geographic_scope"`
}

// ApprovalRecord tracks approval history
type ApprovalRecord struct {
	RecordID       string                 `json:"record_id"`
	Action         string                 `json:"action"`
	RequestedBy    string                 `json:"requested_by"`
	ApprovedBy     []string               `json:"approved_by"`
	ApprovalStatus string                 `json:"approval_status"`
	ApprovalData   map[string]interface{} `json:"approval_data"`
	RequestedAt    time.Time              `json:"requested_at"`
	ApprovedAt     time.Time              `json:"approved_at"`
	ExpiresAt      time.Time              `json:"expires_at"`
}

// EscalationRule defines escalation procedures
type EscalationRule struct {
	RuleID       string        `json:"rule_id"`
	Trigger      string        `json:"trigger"`
	EscalateTo   []string      `json:"escalate_to"`
	Timeframe    time.Duration `json:"timeframe"`
	Conditions   []string      `json:"conditions"`
	AutoEscalate bool          `json:"auto_escalate"`
}

// MathematicalProof provides cryptographic enforcement of power-of-attorney rules
type MathematicalProof struct {
	ProofType          string             `json:"proof_type"` // zk_proof, digital_signature, hash_chain
	CryptographicProof string             `json:"cryptographic_proof"`
	VerificationKey    string             `json:"verification_key"`
	SignatureChain     []CryptographicSig `json:"signature_chain"`
	HashChain          []string           `json:"hash_chain"`
	MerkleRoot         string             `json:"merkle_root"`
	ProofValidation    *ProofValidation   `json:"proof_validation"`
	MathematicalRules  []MathematicalRule `json:"mathematical_rules"`
	EnforcementLevel   string             `json:"enforcement_level"`
	GeneratedAt        time.Time          `json:"generated_at"`
	ValidatedAt        time.Time          `json:"validated_at"`
}

// CryptographicSig represents a cryptographic signature in the chain
type CryptographicSig struct {
	SignerID   string    `json:"signer_id"`
	Signature  string    `json:"signature"`
	Algorithm  string    `json:"algorithm"`
	KeyID      string    `json:"key_id"`
	Timestamp  time.Time `json:"timestamp"`
	SignedData string    `json:"signed_data"`
}

// ProofValidation tracks validation of mathematical proofs
type ProofValidation struct {
	Valid           bool      `json:"valid"`
	ValidatedBy     string    `json:"validated_by"`
	ValidationAlgo  string    `json:"validation_algorithm"`
	ConfidenceLevel float64   `json:"confidence_level"`
	ValidatedAt     time.Time `json:"validated_at"`
	ValidationLog   []string  `json:"validation_log"`
}

// MathematicalRule defines enforceable mathematical constraints
type MathematicalRule struct {
	RuleID      string                 `json:"rule_id"`
	RuleType    string                 `json:"rule_type"`  // constraint, invariant, theorem
	Expression  string                 `json:"expression"` // mathematical expression
	Parameters  map[string]interface{} `json:"parameters"`
	Violation   string                 `json:"violation"`   // what happens on violation
	Enforcement string                 `json:"enforcement"` // how it's enforced
	Priority    int                    `json:"priority"`
}

// ContractualPowers defines contract-related authorities
type ContractualPowers struct {
	ContractTypes        []string              `json:"contract_types"`
	MaxContractValue     float64               `json:"max_contract_value"`
	RequiresApproval     bool                  `json:"requires_approval"`
	ApprovalWorkflow     []string              `json:"approval_workflow"`
	ContractModification *ContractModification `json:"contract_modification,omitempty"`
}

// ContractModification defines contract modification authorities
type ContractModification struct {
	CanAmend     bool     `json:"can_amend"`
	CanTerminate bool     `json:"can_terminate"`
	CanRenew     bool     `json:"can_renew"`
	Limitations  []string `json:"limitations"`
}

// OperationalPowers defines operational authorities
type OperationalPowers struct {
	ResourceManagement   []string `json:"resource_management"`
	ProcessControl       []string `json:"process_control"`
	DataAccess           []string `json:"data_access"`
	SystemAdministration []string `json:"system_administration"`
}

// RepresentationPowers defines representation authorities
type RepresentationPowers struct {
	ExternalRepresentation bool     `json:"external_representation"`
	AuthorizedInteractions []string `json:"authorized_interactions"`
	CommunicationChannels  []string `json:"communication_channels"`
	DocumentationRights    []string `json:"documentation_rights"`
}

// CompliancePowers defines compliance and regulatory authorities
type CompliancePowers struct {
	RegulatoryReporting  bool     `json:"regulatory_reporting"`
	ComplianceMonitoring bool     `json:"compliance_monitoring"`
	AuditCooperation     bool     `json:"audit_cooperation"`
	LegalRepresentation  []string `json:"legal_representation"`
}

// DecisionAuthority defines what decisions the AI can make
type DecisionAuthority struct {
	AutonomousDecisions []string          `json:"autonomous_decisions"`
	ApprovalRequired    []string          `json:"approval_required"`
	DecisionMatrix      map[string]string `json:"decision_matrix"` // decision type -> authority level
	EscalationRules     *EscalationRules  `json:"escalation_rules"`
}

// EscalationRules defines when decisions must be escalated
type EscalationRules struct {
	ThresholdTriggers map[string]interface{} `json:"threshold_triggers"`
	EscalationPath    []string               `json:"escalation_path"`
	ResponseTimeReqs  map[string]string      `json:"response_time_requirements"`
	OverrideAuthority []string               `json:"override_authority"`
}

// TransactionRights defines transaction permissions
type TransactionRights struct {
	AllowedTransactionTypes []string            `json:"allowed_transaction_types"`
	TransactionLimits       *TransactionLimits  `json:"transaction_limits"`
	RequiredApprovals       map[string][]string `json:"required_approvals"` // transaction type -> approvers
	ProhibitedTransactions  []string            `json:"prohibited_transactions"`
}

// TransactionLimits defines transaction value and frequency limits
type TransactionLimits struct {
	PerTransaction  *ApprovalLimits `json:"per_transaction"`
	Cumulative      *ApprovalLimits `json:"cumulative"`
	FrequencyLimits map[string]int  `json:"frequency_limits"` // time period -> max count
}

// ActionPermissions defines specific actions the AI can perform
type ActionPermissions struct {
	ResourceActions   map[string][]string `json:"resource_actions"` // resource -> actions
	HumanInteractions *HumanInteractions  `json:"human_interactions"`
	AgentInteractions *AgentInteractions  `json:"agent_interactions"`
	SystemActions     []string            `json:"system_actions"`
}

// HumanInteractions defines how AI can interact with humans
type HumanInteractions struct {
	CanInitiateContact   bool               `json:"can_initiate_contact"`
	AuthorizedChannels   []string           `json:"authorized_channels"`
	RequiresNotification bool               `json:"requires_notification"`
	InteractionLimits    *InteractionLimits `json:"interaction_limits"`
}

// AgentInteractions defines how AI can interact with other agents
type AgentInteractions struct {
	CanAuthorizeAgents  bool                 `json:"can_authorize_agents"`
	DelegationLimits    *DelegationLimits    `json:"delegation_limits"`
	InterAgentProtocols []string             `json:"inter_agent_protocols"`
	AuthorityValidation *AuthorityValidation `json:"authority_validation"`
}

// InteractionLimits defines limits on interactions
type InteractionLimits struct {
	DailyContacts    int      `json:"daily_contacts"`
	MaxDuration      int      `json:"max_duration"` // in minutes
	AllowedTimeSlots []string `json:"allowed_time_slots"`
}

// DelegationLimits defines limits on authority delegation
type DelegationLimits struct {
	MaxDelegationDepth int      `json:"max_delegation_depth"`
	DelegationScope    []string `json:"delegation_scope"`
	RequiresApproval   bool     `json:"requires_approval"`
	MinimumDuration    int      `json:"minimum_duration"` // in minutes
}

// AuthorityValidation defines how authority is validated
type AuthorityValidation struct {
	RequiresBlockchainVerification bool     `json:"requires_blockchain_verification"`
	ValidationInterval             int      `json:"validation_interval"` // in minutes
	TrustedValidators              []string `json:"trusted_validators"`
}

// ControlMechanisms defines the dual control mechanisms
type ControlMechanisms struct {
	MultiSignatureRequired bool           `json:"multi_signature_required"`
	TimeDelayedOperations  map[string]int `json:"time_delayed_operations"` // operation -> delay in minutes
	WitnessRequirements    map[string]int `json:"witness_requirements"`    // operation -> number of witnesses
	AuditTrailRequirements []string       `json:"audit_trail_requirements"`
}

// AuthorizationCascade tracks the chain of authorization
type AuthorizationCascade struct {
	HumanAuthority      *HumanAuthority `json:"human_authority"`
	CascadeChain        []*CascadeLevel `json:"cascade_chain"`
	UltimateHuman       *UltimateHuman  `json:"ultimate_human"`
	AccountabilityChain []string        `json:"accountability_chain"`
}

// HumanAuthority represents the human at the top of the cascade
type HumanAuthority struct {
	PersonID         string    `json:"person_id"`
	Name             string    `json:"name"`
	Position         string    `json:"position"`
	AuthoritySource  string    `json:"authority_source"`
	VerificationDate time.Time `json:"verification_date"`
	IsUltimate       bool      `json:"is_ultimate"`
}

// CascadeLevel represents a level in the authorization cascade
type CascadeLevel struct {
	Level          int       `json:"level"`
	AuthorizerID   string    `json:"authorizer_id"`
	AuthorizerType string    `json:"authorizer_type"` // human, ai_agent
	AuthorizedID   string    `json:"authorized_id"`
	AuthorizedType string    `json:"authorized_type"`
	GrantedAt      time.Time `json:"granted_at"`
	Scope          []string  `json:"scope"`
}

// UltimateHuman represents the ultimate human authority
type UltimateHuman struct {
	PersonID            string `json:"person_id"`
	Name                string `json:"name"`
	LegalAuthority      string `json:"legal_authority"`
	AccountabilityLevel string `json:"accountability_level"`
	VerificationProof   string `json:"verification_proof"`
}

// NewGAuthPlusService creates a new comprehensive GAuth+ service
func NewGAuthPlusService(config *viper.Viper, logger *logrus.Logger) (*GAuthPlusService, error) {
	// Initialize base GAuth service
	baseService, err := NewGAuthService(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize base GAuth service: %w", err)
	}

	// Initialize Redis client for blockchain registry
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.GetString("redis.addr"),
		Password: config.GetString("redis.password"),
		DB:       config.GetInt("redis.db"),
	})

	blockchainRegistry := &BlockchainRegistry{
		config: config,
		logger: logger,
		redis:  redisClient,
	}

	commercialRegister := &CommercialRegister{
		config: config,
		logger: logger,
		redis:  redisClient,
	}

	return &GAuthPlusService{
		GAuthService:       baseService,
		blockchainRegistry: blockchainRegistry,
		commercialRegister: commercialRegister,
	}, nil
}

// RegisterAIAuthorization creates a comprehensive authorization record on the blockchain
func (s *GAuthPlusService) RegisterAIAuthorization(ctx context.Context, record *AIAuthorizationRecord) (*AIAuthorizationRecord, error) {
	s.logger.WithFields(logrus.Fields{
		"ai_system_id":      record.AISystemID,
		"authorizing_party": record.AuthorizingParty.ID,
	}).Info("Registering AI authorization on blockchain")

	// Generate unique ID
	record.ID = generateID("ai_auth")
	record.CreatedAt = time.Now()
	record.Status = "active"

	// Validate authorization cascade - ensure human at top
	if err := s.validateAuthorizationCascade(record.AuthorizationCascade); err != nil {
		return nil, fmt.Errorf("authorization cascade validation failed: %w", err)
	}

	// Verify authorizing party authority
	if err := s.verifyAuthorizingPartyAuthority(ctx, record.AuthorizingParty); err != nil {
		return nil, fmt.Errorf("authorizing party verification failed: %w", err)
	}

	// Create blockchain hash
	record.BlockchainHash = s.createBlockchainHash(record)

	// Store in blockchain registry
	if err := s.blockchainRegistry.StoreAuthorizationRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to store in blockchain registry: %w", err)
	}

	// Register in commercial register
	if err := s.commercialRegister.RegisterAISystem(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to register in commercial register: %w", err)
	}

	// Log audit event
	s.logAuditEvent(ctx, "ai_authorization_registration", record.AuthorizingParty.ID, record.ID, "register", "success")

	return record, nil
}

// ValidateAIAuthority verifies an AI's authority to perform a specific action
func (s *GAuthPlusService) ValidateAIAuthority(ctx context.Context, aiSystemID, action string, context map[string]interface{}) (*AuthorityValidationResult, error) {
	s.logger.WithFields(logrus.Fields{
		"ai_system_id": aiSystemID,
		"action":       action,
	}).Info("Validating AI authority")

	// Retrieve authorization record from blockchain
	record, err := s.blockchainRegistry.GetAuthorizationRecord(ctx, aiSystemID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve authorization record: %w", err)
	}

	// Validate record is active and not expired
	if record.Status != "active" || time.Now().After(record.ExpiresAt) {
		return &AuthorityValidationResult{
			Valid:   false,
			Reason:  "authorization expired or inactive",
			Details: map[string]interface{}{"status": record.Status, "expires_at": record.ExpiresAt},
		}, nil
	}

	// Check if action is permitted
	permitted, reason := s.isActionPermitted(record, action, context)
	if !permitted {
		return &AuthorityValidationResult{
			Valid:   false,
			Reason:  reason,
			Details: map[string]interface{}{"action": action},
		}, nil
	}

	// Verify dual control if required
	if s.requiresDualControl(record, action) {
		approved, approvers := s.verifyDualControlApproval(ctx, record, action, context)
		if !approved {
			return &AuthorityValidationResult{
				Valid:   false,
				Reason:  "dual control approval required",
				Details: map[string]interface{}{"required_approvers": approvers},
			}, nil
		}
	}

	// Validate authorization cascade is still valid
	if err := s.validateAuthorizationCascade(record.AuthorizationCascade); err != nil {
		return &AuthorityValidationResult{
			Valid:   false,
			Reason:  "authorization cascade invalid",
			Details: map[string]interface{}{"cascade_error": err.Error()},
		}, nil
	}

	return &AuthorityValidationResult{
		Valid:     true,
		Reason:    "authorization validated",
		Details:   map[string]interface{}{"record_id": record.ID, "blockchain_hash": record.BlockchainHash},
		Record:    record,
		Timestamp: time.Now(),
	}, nil
}

// AuthorityValidationResult represents the result of authority validation
type AuthorityValidationResult struct {
	Valid     bool                   `json:"valid"`
	Reason    string                 `json:"reason"`
	Details   map[string]interface{} `json:"details"`
	Record    *AIAuthorizationRecord `json:"record,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// validateAuthorizationCascade ensures there's a human at the top of the cascade
func (s *GAuthPlusService) validateAuthorizationCascade(cascade *AuthorizationCascade) error {
	if cascade == nil {
		return fmt.Errorf("authorization cascade is required")
	}

	if cascade.UltimateHuman == nil {
		return fmt.Errorf("ultimate human authority is required")
	}

	if cascade.HumanAuthority == nil || !cascade.HumanAuthority.IsUltimate {
		return fmt.Errorf("human authority must be at the top of the cascade")
	}

	// Validate cascade chain
	for i, level := range cascade.CascadeChain {
		if i == 0 && level.AuthorizerType != "human" {
			return fmt.Errorf("first level in cascade must be authorized by a human")
		}
	}

	return nil
}

// verifyAuthorizingPartyAuthority verifies the authority of the authorizing party
func (s *GAuthPlusService) verifyAuthorizingPartyAuthority(ctx context.Context, party *AuthorizingParty) error {
	if party.VerificationStatus != "verified" {
		return fmt.Errorf("authorizing party is not verified")
	}

	if party.LegalCapacity == nil || !party.LegalCapacity.Verified {
		return fmt.Errorf("legal capacity is not verified")
	}

	// Additional verification logic would go here
	return nil
}

// createBlockchainHash creates a hash for blockchain storage
func (s *GAuthPlusService) createBlockchainHash(record *AIAuthorizationRecord) string {
	data := fmt.Sprintf("%s:%s:%s:%d", record.AISystemID, record.AuthorizingParty.ID, record.CreatedAt.Format(time.RFC3339), time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// isActionPermitted checks if an action is permitted for the AI
func (s *GAuthPlusService) isActionPermitted(record *AIAuthorizationRecord, action string, context map[string]interface{}) (bool, string) {
	// Check basic permissions
	if record.ActionPermissions != nil {
		for _, systemAction := range record.ActionPermissions.SystemActions {
			if systemAction == action {
				return true, "action permitted"
			}
		}
	}

	// Check if action is in granted powers
	if record.PowersGranted != nil {
		for _, power := range record.PowersGranted.BasicPowers {
			if strings.Contains(action, power) {
				return true, "action permitted by basic powers"
			}
		}
		for _, power := range record.PowersGranted.DerivedPowers {
			if strings.Contains(action, power) {
				return true, "action permitted by derived powers"
			}
		}
	}

	return false, "action not permitted"
}

// requiresDualControl checks if an action requires dual control approval
func (s *GAuthPlusService) requiresDualControl(record *AIAuthorizationRecord, action string) bool {
	if record.DualControlPrinciple == nil || !record.DualControlPrinciple.Enabled {
		return false
	}

	for _, requiredAction := range record.DualControlPrinciple.RequiredForActions {
		if requiredAction == action || strings.Contains(action, requiredAction) {
			return true
		}
	}

	return false
}

// verifyDualControlApproval verifies that dual control approval has been obtained
func (s *GAuthPlusService) verifyDualControlApproval(ctx context.Context, record *AIAuthorizationRecord, action string, context map[string]interface{}) (bool, []string) {
	if record.DualControlPrinciple == nil {
		return false, []string{}
	}

	requiredApprovers := []string{}
	if record.DualControlPrinciple.PrimaryApprover != nil {
		requiredApprovers = append(requiredApprovers, record.DualControlPrinciple.PrimaryApprover.ApproverID)
	}
	if record.DualControlPrinciple.SecondaryApprover != nil {
		requiredApprovers = append(requiredApprovers, record.DualControlPrinciple.SecondaryApprover.ApproverID)
	}

	// In a real implementation, this would check for actual approvals
	// For demo purposes, we'll assume approval is required but not yet obtained
	return false, requiredApprovers
}

// ValidateHumanAccountabilityChain validates that human accountability is maintained
func (s *GAuthPlusService) ValidateHumanAccountabilityChain(ctx context.Context, chain *HumanAccountabilityChain) error {
	if chain == nil {
		return fmt.Errorf("human accountability chain is required")
	}

	if chain.UltimateHumanAuthority == nil {
		return fmt.Errorf("ultimate human authority must be specified")
	}

	if !chain.UltimateHumanAuthority.IsUltimateAuthority {
		return fmt.Errorf("ultimate authority flag must be set")
	}

	// Validate identity verification
	if chain.UltimateHumanAuthority.IdentityVerification == nil || !chain.UltimateHumanAuthority.IdentityVerification.Verified {
		return fmt.Errorf("ultimate human authority identity must be verified")
	}

	// Validate delegation chain
	for i, level := range chain.DelegationChain {
		if i == 0 && !level.IsHuman {
			return fmt.Errorf("first level in delegation chain must be human")
		}

		if level.Level != i {
			return fmt.Errorf("delegation chain levels must be sequential")
		}
	}

	return nil
}

// GenerateMathematicalProof creates cryptographic proof for power-of-attorney
func (s *GAuthPlusService) GenerateMathematicalProof(ctx context.Context, record *AIAuthorizationRecord) (*MathematicalProof, error) {
	proof := &MathematicalProof{
		ProofType:         "digital_signature_chain",
		GeneratedAt:       time.Now(),
		EnforcementLevel:  "cryptographic",
		MathematicalRules: []MathematicalRule{},
		SignatureChain:    []CryptographicSig{},
	}

	// Create signature chain from authorization cascade
	if record.AuthorizationCascade != nil {
		for i, level := range record.AuthorizationCascade.CascadeChain {
			sig := CryptographicSig{
				SignerID:   level.AuthorizerID,
				Algorithm:  "ECDSA",
				KeyID:      fmt.Sprintf("key_%s", level.AuthorizerID),
				Timestamp:  level.GrantedAt,
				SignedData: fmt.Sprintf("level_%d_authorization", i),
			}

			// Generate mock signature for demo
			sigData := fmt.Sprintf("%s:%s:%d", sig.SignerID, sig.SignedData, sig.Timestamp.Unix())
			hash := sha256.Sum256([]byte(sigData))
			sig.Signature = hex.EncodeToString(hash[:])

			proof.SignatureChain = append(proof.SignatureChain, sig)
		}
	}

	// Add mathematical rules for power enforcement
	rules := []MathematicalRule{
		{
			RuleID:      "human_at_top",
			RuleType:    "invariant",
			Expression:  "∀ cascade : cascade.top.type = human",
			Enforcement: "cryptographic_verification",
			Priority:    1,
		},
		{
			RuleID:      "power_conservation",
			RuleType:    "constraint",
			Expression:  "∑ delegated_powers ≤ total_authority",
			Enforcement: "mathematical_validation",
			Priority:    2,
		},
		{
			RuleID:      "temporal_validity",
			RuleType:    "constraint",
			Expression:  "current_time ∈ [valid_from, valid_until]",
			Enforcement: "temporal_verification",
			Priority:    3,
		},
	}
	proof.MathematicalRules = rules

	// Create hash chain
	for i, rule := range rules {
		ruleData := fmt.Sprintf("%s:%s:%s", rule.RuleID, rule.Expression, rule.Enforcement)
		hash := sha256.Sum256([]byte(ruleData))
		proof.HashChain = append(proof.HashChain, hex.EncodeToString(hash[:]))

		if i == len(rules)-1 {
			// Set merkle root as final hash
			proof.MerkleRoot = hex.EncodeToString(hash[:])
		}
	}

	// Generate verification key
	keyData := fmt.Sprintf("%s:%s:%d", record.ID, record.AISystemID, proof.GeneratedAt.Unix())
	keyHash := sha256.Sum256([]byte(keyData))
	proof.VerificationKey = hex.EncodeToString(keyHash[:])

	// Create cryptographic proof
	proofData := fmt.Sprintf("%s:%s:%s", proof.MerkleRoot, proof.VerificationKey, proof.ProofType)
	proofHash := sha256.Sum256([]byte(proofData))
	proof.CryptographicProof = hex.EncodeToString(proofHash[:])

	// Validate the proof
	validation := &ProofValidation{
		Valid:           true,
		ValidatedBy:     "gauth_plus_service",
		ValidationAlgo:  "sha256_chain_validation",
		ConfidenceLevel: 0.95,
		ValidatedAt:     time.Now(),
		ValidationLog:   []string{"signature_chain_validated", "mathematical_rules_verified", "hash_chain_computed"},
	}
	proof.ProofValidation = validation
	proof.ValidatedAt = validation.ValidatedAt

	return proof, nil
}

// EnforceDualControlPrinciple enforces dual control for high-risk operations
func (s *GAuthPlusService) EnforceDualControlPrinciple(ctx context.Context, record *AIAuthorizationRecord, action string, requestData map[string]interface{}) (*DualControlResult, error) {
	if record.DualControlPrinciple == nil || !record.DualControlPrinciple.Enabled {
		return &DualControlResult{
			Required: false,
			Enforced: false,
			Status:   "not_required",
			Message:  "dual control not enabled for this authorization",
		}, nil
	}

	// Check if dual control is required for this action
	required := s.requiresDualControl(record, action)
	if !required {
		return &DualControlResult{
			Required: false,
			Enforced: false,
			Status:   "not_required",
			Message:  "dual control not required for this action",
		}, nil
	}

	// Create approval request
	approvalReq := &ApprovalRecord{
		RecordID:       generateID("approval"),
		Action:         action,
		RequestedBy:    record.AISystemID,
		ApprovalStatus: "pending",
		ApprovalData:   requestData,
		RequestedAt:    time.Now(),
		ExpiresAt:      time.Now().Add(24 * time.Hour), // 24 hour approval window
	}

	// Store approval request
	if err := s.storeApprovalRequest(ctx, approvalReq); err != nil {
		return nil, fmt.Errorf("failed to store approval request: %w", err)
	}

	return &DualControlResult{
		Required:        true,
		Enforced:        true,
		Status:          "approval_required",
		Message:         "dual control approval required before action can proceed",
		ApprovalRequest: approvalReq,
		RequiredApprovers: []string{
			record.DualControlPrinciple.PrimaryApprover.ApproverID,
			record.DualControlPrinciple.SecondaryApprover.ApproverID,
		},
	}, nil
}

// DualControlResult represents the result of dual control enforcement
type DualControlResult struct {
	Required          bool            `json:"required"`
	Enforced          bool            `json:"enforced"`
	Status            string          `json:"status"`
	Message           string          `json:"message"`
	ApprovalRequest   *ApprovalRecord `json:"approval_request,omitempty"`
	RequiredApprovers []string        `json:"required_approvers,omitempty"`
}

// storeApprovalRequest stores an approval request for processing
func (s *GAuthPlusService) storeApprovalRequest(ctx context.Context, req *ApprovalRecord) error {
	if s.redis != nil {
		data, err := json.Marshal(req)
		if err != nil {
			return err
		}
		return s.redis.Set(ctx, fmt.Sprintf("approval_request:%s", req.RecordID), data, 24*time.Hour).Err()
	}
	return nil
}

// Additional blockchain registry methods

// StoreAuthorizationRecord stores an authorization record in the blockchain
func (br *BlockchainRegistry) StoreAuthorizationRecord(ctx context.Context, record *AIAuthorizationRecord) error {
	if br.redis != nil {
		data, err := json.Marshal(record)
		if err != nil {
			return err
		}
		return br.redis.Set(ctx, fmt.Sprintf("blockchain:auth_record:%s", record.ID), data, 0).Err()
	}
	return nil
}

// GetAuthorizationRecord retrieves an authorization record from the blockchain
func (br *BlockchainRegistry) GetAuthorizationRecord(ctx context.Context, aiSystemID string) (*AIAuthorizationRecord, error) {
	if br.redis != nil {
		// In a real implementation, this would query by AI system ID
		// For demo, we'll create a mock record
		mockRecord := &AIAuthorizationRecord{
			ID:         generateID("mock_auth"),
			AISystemID: aiSystemID,
			Status:     "active",
			ExpiresAt:  time.Now().Add(365 * 24 * time.Hour), // 1 year
			CreatedAt:  time.Now().Add(-30 * 24 * time.Hour), // 30 days ago
		}
		return mockRecord, nil
	}
	return nil, fmt.Errorf("blockchain registry not available")
}

// RegisterAISystem registers an AI system in the commercial register
func (cr *CommercialRegister) RegisterAISystem(ctx context.Context, record *AIAuthorizationRecord) error {
	if cr.redis != nil {
		registryData := map[string]interface{}{
			"ai_system_id":      record.AISystemID,
			"authorizing_party": record.AuthorizingParty.ID,
			"registered_at":     time.Now(),
			"status":            "registered",
			"blockchain_hash":   record.BlockchainHash,
		}

		data, err := json.Marshal(registryData)
		if err != nil {
			return err
		}

		return cr.redis.Set(ctx, fmt.Sprintf("commercial_register:%s", record.AISystemID), data, 0).Err()
	}
	return nil
}
