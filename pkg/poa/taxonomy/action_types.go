// Package poa - AAP002 Section B.4 Authorized Actions Classification
// This implements transaction types, decision types, and action types
// as required by AAP002 Section B.4 (Authorized Actions)
package taxonomy

import (
	"fmt"
	"strings"
)

const (
	// Risk level string constants for action sets
	// Note: RiskHigh, RiskMedium, RiskLow are defined as RiskLevel type in action_taxonomy_complete.go
	riskMediumHigh = "medium-high"
	riskMediumLow  = "medium-low"
)

// TransactionType represents financial/commercial transaction types
// per AAP002 Section B.4.1
type TransactionType string

const (
	// TransactionLoan - Loan agreements and credit facilities
	TransactionLoan TransactionType = "Loan"

	// TransactionPurchase - Purchase of goods or services
	TransactionPurchase TransactionType = "Purchase"

	// TransactionSale - Sale of goods or services
	TransactionSale TransactionType = "Sale"

	// TransactionLeasingRental - Leasing or rental agreements
	TransactionLeasingRental TransactionType = "LeasingRental"

	// TransactionInvestment - Investment transactions
	TransactionInvestment TransactionType = "Investment"

	// TransactionPayment - Payment processing and transfers
	TransactionPayment TransactionType = "Payment"

	// TransactionContract - Contract execution and management
	TransactionContract TransactionType = "Contract"

	// TransactionRefund - Refund processing
	TransactionRefund TransactionType = "Refund"

	// TransactionExchange - Currency or asset exchange
	TransactionExchange TransactionType = "Exchange"

	// TransactionOther - Other transaction types
	TransactionOther TransactionType = "Other"
)

// DecisionType represents decision-making categories
// per AAP002 Section B.4.2
type DecisionType string

const (
	// DecisionPersonnel - Personnel and HR decisions
	DecisionPersonnel DecisionType = "Personnel"

	// DecisionFinancial - Financial decisions (budgets, investments)
	DecisionFinancial DecisionType = "Financial"

	// DecisionBuySell - Buy/sell decisions for assets or services
	DecisionBuySell DecisionType = "BuySell"

	// DecisionConceptual - Conceptual and strategic planning
	DecisionConceptual DecisionType = "Conceptual"

	// DecisionDesign - Design and engineering decisions
	DecisionDesign DecisionType = "Design"

	// DecisionInfoSharing - Information sharing and disclosure
	DecisionInfoSharing DecisionType = "InfoSharing"

	// DecisionStrategic - Strategic business decisions
	DecisionStrategic DecisionType = "Strategic"

	// DecisionLegal - Legal and compliance decisions
	DecisionLegal DecisionType = "Legal"

	// DecisionAssetMgmt - Asset management decisions
	DecisionAssetMgmt DecisionType = "AssetMgmt"

	// DecisionOperational - Day-to-day operational decisions
	DecisionOperational DecisionType = "Operational"

	// DecisionRisk - Risk management decisions
	DecisionRisk DecisionType = "Risk"

	// DecisionCompliance - Compliance and regulatory decisions
	DecisionCompliance DecisionType = "Compliance"

	// DecisionOther - Other decision types
	DecisionOther DecisionType = "Other"
)

// ActionTypePhysical represents physical action categories
// per AAP002 Section B.4.3
type ActionTypePhysical string

const (
	// ActionPhysicalManufacturing - Manufacturing and production
	ActionPhysicalManufacturing ActionTypePhysical = "Manufacturing"

	// ActionPhysicalAssembly - Assembly and construction
	ActionPhysicalAssembly ActionTypePhysical = "Assembly"

	// ActionPhysicalTransport - Transportation and logistics
	ActionPhysicalTransport ActionTypePhysical = "Transport"

	// ActionPhysicalMaintenance - Maintenance and repair
	ActionPhysicalMaintenance ActionTypePhysical = "Maintenance"

	// ActionPhysicalInspection - Physical inspection and testing
	ActionPhysicalInspection ActionTypePhysical = "Inspection"

	// ActionPhysicalHandling - Material handling and movement
	ActionPhysicalHandling ActionTypePhysical = "Handling"

	// ActionPhysicalInstallation - Installation and deployment
	ActionPhysicalInstallation ActionTypePhysical = "Installation"

	// ActionPhysicalOperation - Physical operation of equipment
	ActionPhysicalOperation ActionTypePhysical = "Operation"

	// ActionPhysicalSurgery - Medical/surgical procedures
	ActionPhysicalSurgery ActionTypePhysical = "Surgery"

	// ActionPhysicalDelivery - Delivery and distribution
	ActionPhysicalDelivery ActionTypePhysical = "Delivery"

	// ActionPhysicalStorage - Storage and warehousing
	// AAP002 B.4.3: Required for physical asset management
	ActionPhysicalStorage ActionTypePhysical = "Storage"

	// ActionPhysicalPackaging - Packaging and wrapping
	// AAP002 B.4.3: Required for product preparation and logistics
	ActionPhysicalPackaging ActionTypePhysical = "Packaging"

	// ActionPhysicalCleaning - Cleaning and sanitation
	// AAP002 B.4.3: Required for maintenance and facility management
	ActionPhysicalCleaning ActionTypePhysical = "Cleaning"

	// ActionPhysicalRecycling - Recycling and waste management
	// AAP002 B.4.3: Required for environmental compliance and sustainability
	ActionPhysicalRecycling ActionTypePhysical = "Recycling"

	// ActionPhysicalCustomization - Customization and modification
	// AAP002 B.4.3: Required for bespoke manufacturing and adaptation
	ActionPhysicalCustomization ActionTypePhysical = "Customization"

	// ActionPhysicalOther - Other physical actions
	ActionPhysicalOther ActionTypePhysical = "Other"
)

// ActionTypeNonPhysical represents non-physical action categories
// per AAP002 Section B.4.4
type ActionTypeNonPhysical string

const (
	// ActionNonPhysicalResearching - Research and investigation
	ActionNonPhysicalResearching ActionTypeNonPhysical = "Researching"

	// ActionNonPhysicalBrainstorming - Brainstorming and ideation
	ActionNonPhysicalBrainstorming ActionTypeNonPhysical = "Brainstorming"

	// ActionNonPhysicalAnalyzing - Data analysis and interpretation
	ActionNonPhysicalAnalyzing ActionTypeNonPhysical = "Analyzing"

	// ActionNonPhysicalPlanning - Planning and strategy development
	ActionNonPhysicalPlanning ActionTypeNonPhysical = "Planning"

	// ActionNonPhysicalDocumenting - Documentation and reporting
	ActionNonPhysicalDocumenting ActionTypeNonPhysical = "Documenting"

	// ActionNonPhysicalCommunicating - Communication and messaging
	ActionNonPhysicalCommunicating ActionTypeNonPhysical = "Communicating"

	// ActionNonPhysicalNegotiating - Negotiation and mediation
	ActionNonPhysicalNegotiating ActionTypeNonPhysical = "Negotiating"

	// ActionNonPhysicalMonitoring - Monitoring and surveillance
	ActionNonPhysicalMonitoring ActionTypeNonPhysical = "Monitoring"

	// ActionNonPhysicalModeling - Modeling and simulation
	ActionNonPhysicalModeling ActionTypeNonPhysical = "Modeling"

	// ActionNonPhysicalTraining - Training and education
	ActionNonPhysicalTraining ActionTypeNonPhysical = "Training"

	// ActionNonPhysicalAdvising - Advising and consulting
	ActionNonPhysicalAdvising ActionTypeNonPhysical = "Advising"

	// ActionNonPhysicalApproving - Approval and authorization
	ActionNonPhysicalApproving ActionTypeNonPhysical = "Approving"

	// ActionNonPhysicalReviewing - Review and evaluation
	ActionNonPhysicalReviewing ActionTypeNonPhysical = "Reviewing"

	// ActionNonPhysicalDesigning - Design and architecture
	ActionNonPhysicalDesigning ActionTypeNonPhysical = "Designing"

	// ActionNonPhysicalDataAggregation - Data aggregation and consolidation
	// AAP002 B.4.4: Required for AI data processing operations
	ActionNonPhysicalDataAggregation ActionTypeNonPhysical = "DataAggregation"

	// ActionNonPhysicalVisualization - Data visualization and reporting
	// AAP002 B.4.4: Required for AI reporting and presentation
	ActionNonPhysicalVisualization ActionTypeNonPhysical = "Visualization"

	// ActionNonPhysicalNotification - Notification and alerting
	// AAP002 B.4.4: Required for AI event-driven communications
	ActionNonPhysicalNotification ActionTypeNonPhysical = "Notification"

	// ActionNonPhysicalRAG - Retrieval-Augmented Generation (RAG) operations
	// AAP002 B.4.4: Explicit RAG support as specified in "Researching (e.g., RAG)"
	ActionNonPhysicalRAG ActionTypeNonPhysical = "RAG"

	// ActionNonPhysicalPresenting - Sharing and presenting information
	// AAP002 B.4.4: "Sharing / presenting" from specification
	ActionNonPhysicalPresenting ActionTypeNonPhysical = "Presenting"

	// ActionNonPhysicalOther - Other non-physical actions
	ActionNonPhysicalOther ActionTypeNonPhysical = "Other"
)

// AuthorizedActionSet represents a complete set of authorized actions
// per AAP002 Section B.4
type AuthorizedActionSet struct {
	Transactions       []TransactionType       `json:"transactions,omitempty"`
	Decisions          []DecisionType          `json:"decisions,omitempty"`
	PhysicalActions    []ActionTypePhysical    `json:"physical_actions,omitempty"`
	NonPhysicalActions []ActionTypeNonPhysical `json:"non_physical_actions,omitempty"`
	// AllowAll indicates all actions of a type are authorized (use with caution)
	AllowAllTransactions       bool `json:"allow_all_transactions,omitempty"`
	AllowAllDecisions          bool `json:"allow_all_decisions,omitempty"`
	AllowAllPhysicalActions    bool `json:"allow_all_physical_actions,omitempty"`
	AllowAllNonPhysicalActions bool `json:"allow_all_non_physical_actions,omitempty"`
}

// ValidateTransactionType validates AAP002 transaction type
func ValidateTransactionType(tt TransactionType) error {
	validTypes := []TransactionType{
		TransactionLoan, TransactionPurchase, TransactionSale,
		TransactionLeasingRental, TransactionInvestment, TransactionPayment,
		TransactionContract, TransactionRefund, TransactionExchange, TransactionOther,
	}
	for _, valid := range validTypes {
		if tt == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid transaction type: %s", tt)
}

// ValidateDecisionType validates AAP002 decision type
func ValidateDecisionType(dt DecisionType) error {
	validTypes := []DecisionType{
		DecisionPersonnel, DecisionFinancial, DecisionBuySell,
		DecisionConceptual, DecisionDesign, DecisionInfoSharing,
		DecisionStrategic, DecisionLegal, DecisionAssetMgmt,
		DecisionOperational, DecisionRisk, DecisionCompliance, DecisionOther,
	}
	for _, valid := range validTypes {
		if dt == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid decision type: %s", dt)
}

// ValidateActionTypePhysical validates AAP002 physical action type
func ValidateActionTypePhysical(at ActionTypePhysical) error {
	validTypes := []ActionTypePhysical{
		ActionPhysicalManufacturing, ActionPhysicalAssembly,
		ActionPhysicalTransport, ActionPhysicalMaintenance,
		ActionPhysicalInspection, ActionPhysicalHandling,
		ActionPhysicalInstallation, ActionPhysicalOperation,
		ActionPhysicalSurgery, ActionPhysicalDelivery,
		ActionPhysicalStorage, ActionPhysicalPackaging,
		ActionPhysicalCleaning, ActionPhysicalRecycling,
		ActionPhysicalCustomization, ActionPhysicalOther,
	}
	for _, valid := range validTypes {
		if at == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid physical action type: %s", at)
}

// ValidateActionTypeNonPhysical validates AAP002 non-physical action type
func ValidateActionTypeNonPhysical(at ActionTypeNonPhysical) error {
	validTypes := []ActionTypeNonPhysical{
		ActionNonPhysicalResearching, ActionNonPhysicalBrainstorming,
		ActionNonPhysicalAnalyzing, ActionNonPhysicalPlanning,
		ActionNonPhysicalDocumenting, ActionNonPhysicalCommunicating,
		ActionNonPhysicalNegotiating, ActionNonPhysicalMonitoring,
		ActionNonPhysicalModeling, ActionNonPhysicalTraining,
		ActionNonPhysicalAdvising, ActionNonPhysicalApproving,
		ActionNonPhysicalReviewing, ActionNonPhysicalDesigning,
		ActionNonPhysicalDataAggregation, ActionNonPhysicalVisualization,
		ActionNonPhysicalNotification, ActionNonPhysicalRAG,
		ActionNonPhysicalPresenting, ActionNonPhysicalOther,
	}
	for _, valid := range validTypes {
		if at == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid non-physical action type: %s", at)
}

// Validate performs complete validation of authorized action set
func (aas *AuthorizedActionSet) Validate() error {
	// Validate all transactions
	for _, tt := range aas.Transactions {
		if err := ValidateTransactionType(tt); err != nil {
			return fmt.Errorf("transaction validation: %w", err)
		}
	}

	// Validate all decisions
	for _, dt := range aas.Decisions {
		if err := ValidateDecisionType(dt); err != nil {
			return fmt.Errorf("decision validation: %w", err)
		}
	}

	// Validate all physical actions
	for _, pa := range aas.PhysicalActions {
		if err := ValidateActionTypePhysical(pa); err != nil {
			return fmt.Errorf("physical action validation: %w", err)
		}
	}

	// Validate all non-physical actions
	for _, npa := range aas.NonPhysicalActions {
		if err := ValidateActionTypeNonPhysical(npa); err != nil {
			return fmt.Errorf("non-physical action validation: %w", err)
		}
	}

	// Ensure at least one action type is specified
	if len(aas.Transactions) == 0 && !aas.AllowAllTransactions &&
		len(aas.Decisions) == 0 && !aas.AllowAllDecisions &&
		len(aas.PhysicalActions) == 0 && !aas.AllowAllPhysicalActions &&
		len(aas.NonPhysicalActions) == 0 && !aas.AllowAllNonPhysicalActions {
		return fmt.Errorf("at least one action type must be specified")
	}

	return nil
}

// IsTransactionAuthorized checks if a transaction type is authorized
func (aas *AuthorizedActionSet) IsTransactionAuthorized(tt TransactionType) bool {
	if aas.AllowAllTransactions {
		return true
	}
	for _, t := range aas.Transactions {
		if t == tt {
			return true
		}
	}
	return false
}

// IsDecisionAuthorized checks if a decision type is authorized
func (aas *AuthorizedActionSet) IsDecisionAuthorized(dt DecisionType) bool {
	if aas.AllowAllDecisions {
		return true
	}
	for _, d := range aas.Decisions {
		if d == dt {
			return true
		}
	}
	return false
}

// IsPhysicalActionAuthorized checks if a physical action is authorized
func (aas *AuthorizedActionSet) IsPhysicalActionAuthorized(pa ActionTypePhysical) bool {
	if aas.AllowAllPhysicalActions {
		return true
	}
	for _, a := range aas.PhysicalActions {
		if a == pa {
			return true
		}
	}
	return false
}

// IsNonPhysicalActionAuthorized checks if a non-physical action is authorized
func (aas *AuthorizedActionSet) IsNonPhysicalActionAuthorized(npa ActionTypeNonPhysical) bool {
	if aas.AllowAllNonPhysicalActions {
		return true
	}
	for _, a := range aas.NonPhysicalActions {
		if a == npa {
			return true
		}
	}
	return false
}

// GetRiskLevel returns risk assessment for the action set
func (aas *AuthorizedActionSet) GetRiskLevel() string {
	// High risk if physical actions or financial transactions allowed
	if len(aas.PhysicalActions) > 0 || aas.AllowAllPhysicalActions {
		return string(RiskHigh)
	}
	if len(aas.Transactions) > 0 || aas.AllowAllTransactions {
		return riskMediumHigh
	}
	// Medium risk for strategic/legal decisions
	for _, dt := range aas.Decisions {
		if dt == DecisionStrategic || dt == DecisionLegal || dt == DecisionFinancial {
			return string(RiskMedium)
		}
	}
	// Low-medium for non-physical actions only
	if len(aas.NonPhysicalActions) > 0 || aas.AllowAllNonPhysicalActions {
		return riskMediumLow
	}
	return string(RiskLow)
}

// String returns human-readable representation
func (aas *AuthorizedActionSet) String() string {
	parts := []string{}

	if aas.AllowAllTransactions {
		parts = append(parts, "All Transactions")
	} else if len(aas.Transactions) > 0 {
		parts = append(parts, fmt.Sprintf("Transactions: %v", aas.Transactions))
	}

	if aas.AllowAllDecisions {
		parts = append(parts, "All Decisions")
	} else if len(aas.Decisions) > 0 {
		parts = append(parts, fmt.Sprintf("Decisions: %v", aas.Decisions))
	}

	if aas.AllowAllPhysicalActions {
		parts = append(parts, "All Physical Actions")
	} else if len(aas.PhysicalActions) > 0 {
		parts = append(parts, fmt.Sprintf("Physical: %v", aas.PhysicalActions))
	}

	if aas.AllowAllNonPhysicalActions {
		parts = append(parts, "All Non-Physical Actions")
	} else if len(aas.NonPhysicalActions) > 0 {
		parts = append(parts, fmt.Sprintf("Non-Physical: %v", aas.NonPhysicalActions))
	}

	return strings.Join(parts, " | ")
}

// RequiresPhysicalCapability checks if action set requires physical embodiment
func (aas *AuthorizedActionSet) RequiresPhysicalCapability() bool {
	return len(aas.PhysicalActions) > 0 || aas.AllowAllPhysicalActions
}

// RequiresFinancialCapability checks if action set requires financial authority
func (aas *AuthorizedActionSet) RequiresFinancialCapability() bool {
	if len(aas.Transactions) > 0 || aas.AllowAllTransactions {
		return true
	}
	for _, dt := range aas.Decisions {
		if dt == DecisionFinancial || dt == DecisionBuySell {
			return true
		}
	}
	return false
}
