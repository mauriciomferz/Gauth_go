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
	*GAuthService // Embed existing service
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
	ID                    string                 `json:"id"`
	Name                  string                 `json:"name"`
	Type                  string                 `json:"type"` // individual, corporation, government
	RegisteredOffice      *RegisteredOffice      `json:"registered_office,omitempty"`
	AuthorizedRepresentative *AuthorizedRepresentative `json:"authorized_representative,omitempty"`
	LegalCapacity         *LegalCapacity         `json:"legal_capacity"`
	AuthorityLevel        string                 `json:"authority_level"` // primary, secondary, delegated
	VerificationStatus    string                 `json:"verification_status"`
	CreatedAt             time.Time              `json:"created_at"`
}

// RegisteredOffice represents the legal registered office of a company
type RegisteredOffice struct {
	Address      string `json:"address"`
	Jurisdiction string `json:"jurisdiction"`
	RegistrationNumber string `json:"registration_number"`
	TaxID        string `json:"tax_id"`
	LegalForm    string `json:"legal_form"` // LLC, Inc, GmbH, etc.
}

// AuthorizedRepresentative represents someone authorized to act on behalf of an entity
type AuthorizedRepresentative struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Position     string    `json:"position"`
	AuthorityScope []string `json:"authority_scope"`
	ValidFrom    time.Time `json:"valid_from"`
	ValidUntil   time.Time `json:"valid_until"`
	VerifiedBy   string    `json:"verified_by"`
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
	DualControlPrinciple *DualControlPrinciple     `json:"dual_control_principle"`
	AuthorizationCascade *AuthorizationCascade     `json:"authorization_cascade"`
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
	BasicPowers      []string            `json:"basic_powers"`
	DerivedPowers    []string            `json:"derived_powers"`
	PowerDerivation  map[string][]string `json:"power_derivation"` // basic power -> derived powers
	StandardPowers   *StandardPowers     `json:"standard_powers"`
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
	SigningAuthority       *SigningAuthority `json:"signing_authority,omitempty"`
	ApprovalLimits        *ApprovalLimits   `json:"approval_limits,omitempty"`
	InvestmentAuthority   []string          `json:"investment_authority,omitempty"`
	BankingOperations     []string          `json:"banking_operations,omitempty"`
	TreasuryManagement    []string          `json:"treasury_management,omitempty"`
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

// ContractualPowers defines contract-related authorities
type ContractualPowers struct {
	ContractTypes        []string          `json:"contract_types"`
	MaxContractValue     float64           `json:"max_contract_value"`
	RequiresApproval     bool              `json:"requires_approval"`
	ApprovalWorkflow     []string          `json:"approval_workflow"`
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
	RegulatoryReporting bool     `json:"regulatory_reporting"`
	ComplianceMonitoring bool    `json:"compliance_monitoring"`
	AuditCooperation    bool     `json:"audit_cooperation"`
	LegalRepresentation []string `json:"legal_representation"`
}

// DecisionAuthority defines what decisions the AI can make
type DecisionAuthority struct {
	AutonomousDecisions []string            `json:"autonomous_decisions"`
	ApprovalRequired    []string            `json:"approval_required"`
	DecisionMatrix      map[string]string   `json:"decision_matrix"` // decision type -> authority level
	EscalationRules     *EscalationRules    `json:"escalation_rules"`
}

// EscalationRules defines when decisions must be escalated
type EscalationRules struct {
	ThresholdTriggers  map[string]interface{} `json:"threshold_triggers"`
	EscalationPath     []string               `json:"escalation_path"`
	ResponseTimeReqs   map[string]string      `json:"response_time_requirements"`
	OverrideAuthority  []string               `json:"override_authority"`
}

// TransactionRights defines transaction permissions
type TransactionRights struct {
	AllowedTransactionTypes []string        `json:"allowed_transaction_types"`
	TransactionLimits       *TransactionLimits `json:"transaction_limits"`
	RequiredApprovals       map[string][]string `json:"required_approvals"` // transaction type -> approvers
	ProhibitedTransactions  []string        `json:"prohibited_transactions"`
}

// TransactionLimits defines transaction value and frequency limits
type TransactionLimits struct {
	PerTransaction *ApprovalLimits `json:"per_transaction"`
	Cumulative     *ApprovalLimits `json:"cumulative"`
	FrequencyLimits map[string]int `json:"frequency_limits"` // time period -> max count
}

// ActionPermissions defines specific actions the AI can perform
type ActionPermissions struct {
	ResourceActions  map[string][]string `json:"resource_actions"` // resource -> actions
	HumanInteractions *HumanInteractions `json:"human_interactions"`
	AgentInteractions *AgentInteractions `json:"agent_interactions"`
	SystemActions     []string           `json:"system_actions"`
}

// HumanInteractions defines how AI can interact with humans
type HumanInteractions struct {
	CanInitiateContact   bool     `json:"can_initiate_contact"`
	AuthorizedChannels   []string `json:"authorized_channels"`
	RequiresNotification bool     `json:"requires_notification"`
	InteractionLimits    *InteractionLimits `json:"interaction_limits"`
}

// AgentInteractions defines how AI can interact with other agents
type AgentInteractions struct {
	CanAuthorizeAgents  bool                `json:"can_authorize_agents"`
	DelegationLimits    *DelegationLimits   `json:"delegation_limits"`
	InterAgentProtocols []string            `json:"inter_agent_protocols"`
	AuthorityValidation *AuthorityValidation `json:"authority_validation"`
}

// InteractionLimits defines limits on interactions
type InteractionLimits struct {
	DailyContacts   int      `json:"daily_contacts"`
	MaxDuration     int      `json:"max_duration"` // in minutes
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

// DualControlPrinciple implements dual control for sensitive operations
type DualControlPrinciple struct {
	Enabled                bool                    `json:"enabled"`
	SecondLevelApprovers   []string                `json:"second_level_approvers"`
	RequiresDualControl    []string                `json:"requires_dual_control"`
	ApprovalMatrix         map[string][]string     `json:"approval_matrix"`
	ControlMechanisms      *ControlMechanisms      `json:"control_mechanisms"`
}

// ControlMechanisms defines the dual control mechanisms
type ControlMechanisms struct {
	MultiSignatureRequired  bool     `json:"multi_signature_required"`
	TimeDelayedOperations   map[string]int `json:"time_delayed_operations"` // operation -> delay in minutes
	WitnessRequirements     map[string]int `json:"witness_requirements"`    // operation -> number of witnesses
	AuditTrailRequirements  []string `json:"audit_trail_requirements"`
}

// AuthorizationCascade tracks the chain of authorization
type AuthorizationCascade struct {
	HumanAuthority   *HumanAuthority   `json:"human_authority"`
	CascadeChain     []*CascadeLevel   `json:"cascade_chain"`
	UltimateHuman    *UltimateHuman    `json:"ultimate_human"`
	AccountabilityChain []string       `json:"accountability_chain"`
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
	PersonID          string    `json:"person_id"`
	Name              string    `json:"name"`
	LegalAuthority    string    `json:"legal_authority"`
	AccountabilityLevel string  `json:"accountability_level"`
	VerificationProof string    `json:"verification_proof"`
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

	// Check derived permissions from powers
	if record.PowersGranted != nil {
		for _, basicPower := range record.PowersGranted.BasicPowers {
			if derivedActions, exists := record.PowersGranted.PowerDerivation[basicPower]; exists {
				for _, derivedAction := range derivedActions {
					if derivedAction == action {
						return true, "action derived from granted power"
					}
				}
			}
		}
	}

	return false, "action not permitted"
}

// requiresDualControl checks if an action requires dual control
func (s *GAuthPlusService) requiresDualControl(record *AIAuthorizationRecord, action string) bool {
	if record.DualControlPrinciple == nil || !record.DualControlPrinciple.Enabled {
		return false
	}

	for _, controlledAction := range record.DualControlPrinciple.RequiresDualControl {
		if controlledAction == action {
			return true
		}
	}

	return false
}

// verifyDualControlApproval verifies dual control approval for an action
func (s *GAuthPlusService) verifyDualControlApproval(ctx context.Context, record *AIAuthorizationRecord, action string, context map[string]interface{}) (bool, []string) {
	// In a real implementation, this would check for actual approvals
	// For now, return the required approvers
	if approvers, exists := record.DualControlPrinciple.ApprovalMatrix[action]; exists {
		return false, approvers // Simulate requiring approval
	}

	return false, record.DualControlPrinciple.SecondLevelApprovers
}

// StoreAuthorizationRecord stores an authorization record in the blockchain registry
func (br *BlockchainRegistry) StoreAuthorizationRecord(ctx context.Context, record *AIAuthorizationRecord) error {
	key := fmt.Sprintf("blockchain:ai_auth:%s", record.ID)
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal authorization record: %w", err)
	}

	if br.redis != nil {
		expiration := time.Until(record.ExpiresAt)
		if err := br.redis.Set(ctx, key, data, expiration).Err(); err != nil {
			br.logger.Warnf("Failed to store in Redis: %v", err)
		}
	}

	br.logger.WithFields(logrus.Fields{
		"record_id":       record.ID,
		"blockchain_hash": record.BlockchainHash,
	}).Info("Authorization record stored in blockchain registry")

	return nil
}

// GetAuthorizationRecord retrieves an authorization record from the blockchain registry
func (br *BlockchainRegistry) GetAuthorizationRecord(ctx context.Context, aiSystemID string) (*AIAuthorizationRecord, error) {
	// In a real implementation, this would query the blockchain
	// For now, simulate with Redis lookup
	keys := []string{
		fmt.Sprintf("blockchain:ai_auth:%s", aiSystemID),
		fmt.Sprintf("blockchain:ai_system:%s", aiSystemID),
	}

	for _, key := range keys {
		if br.redis != nil {
			data, err := br.redis.Get(ctx, key).Result()
			if err == nil {
				var record AIAuthorizationRecord
				if err := json.Unmarshal([]byte(data), &record); err == nil {
					return &record, nil
				}
			}
		}
	}

	// Return mock record for demo
	return &AIAuthorizationRecord{
		ID:         generateID("demo_auth"),
		AISystemID: aiSystemID,
		Status:     "active",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		AuthorizingParty: &AuthorizingParty{
			ID:                 "demo_party",
			Name:               "Demo Authorizing Party",
			Type:               "corporation",
			VerificationStatus: "verified",
			LegalCapacity: &LegalCapacity{
				Verified: true,
			},
		},
		PowersGranted: &PowersGranted{
			BasicPowers:   []string{"financial_operations", "contract_management"},
			DerivedPowers: []string{"sign_contracts", "approve_payments"},
			PowerDerivation: map[string][]string{
				"financial_operations": {"approve_payments", "authorize_transfers"},
				"contract_management":  {"sign_contracts", "modify_terms"},
			},
		},
		ActionPermissions: &ActionPermissions{
			SystemActions: []string{"sign_contracts", "approve_payments", "authorize_transfers"},
		},
		AuthorizationCascade: &AuthorizationCascade{
			HumanAuthority: &HumanAuthority{
				PersonID:   "human_001",
				Name:       "John Smith",
				Position:   "CEO",
				IsUltimate: true,
			},
			UltimateHuman: &UltimateHuman{
				PersonID:       "human_001",
				Name:           "John Smith",
				LegalAuthority: "corporate_ceo",
			},
		},
		DualControlPrinciple: &DualControlPrinciple{
			Enabled:             true,
			SecondLevelApprovers: []string{"cfo", "legal_counsel"},
			RequiresDualControl:  []string{"high_value_transactions", "contract_modifications"},
		},
	}, nil
}

// RegisterAISystem registers an AI system in the commercial register
func (cr *CommercialRegister) RegisterAISystem(ctx context.Context, record *AIAuthorizationRecord) error {
	cr.logger.WithFields(logrus.Fields{
		"ai_system_id": record.AISystemID,
		"record_id":    record.ID,
	}).Info("Registering AI system in commercial register")

	// Create commercial register entry
	entry := map[string]interface{}{
		"ai_system_id":        record.AISystemID,
		"authorization_id":    record.ID,
		"authorizing_party":   record.AuthorizingParty,
		"powers_summary":      cr.summarizePowers(record.PowersGranted),
		"decision_authority":  record.DecisionAuthority,
		"transaction_rights":  record.TransactionRights,
		"blockchain_hash":     record.BlockchainHash,
		"registration_date":   time.Now(),
		"status":             "active",
	}

	// Store in commercial register (Redis for demo)
	if cr.redis != nil {
		key := fmt.Sprintf("commercial_register:ai:%s", record.AISystemID)
		data, _ := json.Marshal(entry)
		expiration := time.Until(record.ExpiresAt)
		cr.redis.Set(ctx, key, data, expiration)
	}

	return nil
}

// summarizePowers creates a summary of granted powers for the commercial register
func (cr *CommercialRegister) summarizePowers(powers *PowersGranted) map[string]interface{} {
	if powers == nil {
		return nil
	}

	summary := map[string]interface{}{
		"basic_powers_count":   len(powers.BasicPowers),
		"derived_powers_count": len(powers.DerivedPowers),
		"power_categories":     []string{},
	}

	// Categorize powers
	categories := make(map[string]bool)
	for _, power := range powers.BasicPowers {
		if strings.Contains(power, "financial") {
			categories["financial"] = true
		}
		if strings.Contains(power, "contract") {
			categories["contractual"] = true
		}
		if strings.Contains(power, "operational") {
			categories["operational"] = true
		}
	}

	for category := range categories {
		summary["power_categories"] = append(summary["power_categories"].([]string), category)
	}

	return summary
}