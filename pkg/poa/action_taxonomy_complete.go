// Package poa - RFC-0115 Section B.4 Complete Authorization Actions Taxonomy
// This file provides comprehensive taxonomy completion including:
// - Action categorization and risk assessment
// - Compliance checking and validation
// - Hierarchical action relationships
// - Action scope and impact analysis
package poa

import (
	"fmt"
	"strings"
)

// ActionCategory represents the high-level category of an action
type ActionCategory string

const (
	CategoryFinancial     ActionCategory = "Financial"
	CategoryOperational   ActionCategory = "Operational"
	CategoryStrategic     ActionCategory = "Strategic"
	CategoryPhysical      ActionCategory = "Physical"
	CategoryDigital       ActionCategory = "Digital"
	CategoryAnalytical    ActionCategory = "Analytical"
	CategoryCommunication ActionCategory = "Communication"
)

// RiskLevel represents the risk assessment for actions
type RiskLevel string

const (
	RiskCritical RiskLevel = "critical" // Immediate financial/safety impact
	RiskHigh     RiskLevel = "high"     // Significant consequences
	RiskMedium   RiskLevel = "medium"   // Moderate impact
	RiskLow      RiskLevel = "low"      // Minimal impact
	RiskMinimal  RiskLevel = "minimal"  // Negligible impact
)

// ActionScope defines the scope of impact for an action
type ActionScope string

const (
	ScopeInternal     ActionScope = "internal"     // Within organization only
	ScopeExternal     ActionScope = "external"     // Affects external parties
	ScopePublic       ActionScope = "public"       // Public-facing actions
	ScopeConfidential ActionScope = "confidential" // Confidential/sensitive
)

// ActionImpact quantifies potential impact dimensions
type ActionImpact struct {
	Financial    RiskLevel `json:"financial"`
	Operational  RiskLevel `json:"operational"`
	Reputational RiskLevel `json:"reputational"`
	Legal        RiskLevel `json:"legal"`
	Safety       RiskLevel `json:"safety"`
}

// TransactionMetadata provides detailed transaction classification
type TransactionMetadata struct {
	Type             TransactionType
	Category         ActionCategory
	Risk             RiskLevel
	Scope            ActionScope
	RequiresApproval bool
	ImpactAnalysis   ActionImpact
	ComplianceReqs   []string
	Description      string
}

// DecisionMetadata provides detailed decision classification
type DecisionMetadata struct {
	Type             DecisionType
	Category         ActionCategory
	Risk             RiskLevel
	Scope            ActionScope
	RequiresApproval bool
	ImpactAnalysis   ActionImpact
	ComplianceReqs   []string
	Description      string
}

// PhysicalActionMetadata provides detailed physical action classification
type PhysicalActionMetadata struct {
	Type                ActionTypePhysical
	Category            ActionCategory
	Risk                RiskLevel
	Scope               ActionScope
	RequiresSafety      bool
	RequiresSupervision bool
	ImpactAnalysis      ActionImpact
	ComplianceReqs      []string
	Description         string
}

// NonPhysicalActionMetadata provides detailed non-physical action classification
type NonPhysicalActionMetadata struct {
	Type             ActionTypeNonPhysical
	Category         ActionCategory
	Risk             RiskLevel
	Scope            ActionScope
	RequiresApproval bool
	ImpactAnalysis   ActionImpact
	ComplianceReqs   []string
	Description      string
}

// GetTransactionMetadata returns comprehensive metadata for transaction type
func GetTransactionMetadata(tt TransactionType) (*TransactionMetadata, error) {
	metadata := map[TransactionType]*TransactionMetadata{
		TransactionLoan: {
			Type:             TransactionLoan,
			Category:         CategoryFinancial,
			Risk:             RiskHigh,
			Scope:            ScopeExternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskHigh,
				Operational:  RiskMedium,
				Reputational: RiskMedium,
				Legal:        RiskHigh,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"KYC", "AML", "Credit_Assessment", "Regulatory_Approval"},
			Description:    "Loan agreements and credit facilities - requires comprehensive due diligence",
		},
		TransactionPurchase: {
			Type:             TransactionPurchase,
			Category:         CategoryFinancial,
			Risk:             RiskMedium,
			Scope:            ScopeExternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskMedium,
				Reputational: RiskLow,
				Legal:        RiskMedium,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Budget_Approval", "Vendor_Verification", "Contract_Review"},
			Description:    "Purchase of goods or services - requires budget and vendor approval",
		},
		TransactionSale: {
			Type:             TransactionSale,
			Category:         CategoryFinancial,
			Risk:             RiskMedium,
			Scope:            ScopeExternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskMedium,
				Reputational: RiskMedium,
				Legal:        RiskMedium,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Pricing_Authority", "Contract_Terms", "Customer_Verification"},
			Description:    "Sale of goods or services - requires pricing and terms authority",
		},
		TransactionLeasingRental: {
			Type:             TransactionLeasingRental,
			Category:         CategoryFinancial,
			Risk:             RiskMedium,
			Scope:            ScopeExternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskMedium,
				Reputational: RiskLow,
				Legal:        RiskHigh,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Lease_Terms", "Asset_Verification", "Contract_Duration"},
			Description:    "Leasing or rental agreements - requires long-term commitment approval",
		},
		TransactionInvestment: {
			Type:             TransactionInvestment,
			Category:         CategoryFinancial,
			Risk:             RiskHigh,
			Scope:            ScopeExternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskCritical,
				Operational:  RiskMedium,
				Reputational: RiskHigh,
				Legal:        RiskHigh,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Investment_Policy", "Risk_Assessment", "Due_Diligence", "Board_Approval"},
			Description:    "Investment transactions - requires comprehensive risk assessment and board approval",
		},
		TransactionPayment: {
			Type:             TransactionPayment,
			Category:         CategoryFinancial,
			Risk:             RiskMedium,
			Scope:            ScopeExternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskLow,
				Reputational: RiskLow,
				Legal:        RiskLow,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Payment_Limits", "Recipient_Verification", "AML_Check"},
			Description:    "Payment processing and transfers - subject to payment limits and verification",
		},
		TransactionContract: {
			Type:             TransactionContract,
			Category:         CategoryFinancial,
			Risk:             RiskHigh,
			Scope:            ScopeExternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskHigh,
				Operational:  RiskHigh,
				Reputational: RiskMedium,
				Legal:        RiskCritical,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Legal_Review", "Authority_Check", "Risk_Assessment"},
			Description:    "Contract execution and management - requires legal review and authority verification",
		},
		TransactionRefund: {
			Type:             TransactionRefund,
			Category:         CategoryFinancial,
			Risk:             RiskLow,
			Scope:            ScopeExternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskLow,
				Reputational: RiskMedium,
				Legal:        RiskLow,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Refund_Policy", "Amount_Limits"},
			Description:    "Refund processing - subject to refund policy and amount limits",
		},
		TransactionExchange: {
			Type:             TransactionExchange,
			Category:         CategoryFinancial,
			Risk:             RiskMedium,
			Scope:            ScopeExternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskLow,
				Reputational: RiskLow,
				Legal:        RiskMedium,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Exchange_Rate_Policy", "AML", "KYC"},
			Description:    "Currency or asset exchange - subject to exchange policies and AML requirements",
		},
		TransactionOther: {
			Type:             TransactionOther,
			Category:         CategoryFinancial,
			Risk:             RiskMedium,
			Scope:            ScopeExternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskMedium,
				Reputational: RiskMedium,
				Legal:        RiskMedium,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Case_By_Case_Review"},
			Description:    "Other transaction types - requires case-by-case review",
		},
	}

	meta, ok := metadata[tt]
	if !ok {
		return nil, fmt.Errorf("unknown transaction type: %s", tt)
	}
	return meta, nil
}

// GetDecisionMetadata returns comprehensive metadata for decision type
func GetDecisionMetadata(dt DecisionType) (*DecisionMetadata, error) {
	metadata := map[DecisionType]*DecisionMetadata{
		DecisionPersonnel: {
			Type:             DecisionPersonnel,
			Category:         CategoryOperational,
			Risk:             RiskHigh,
			Scope:            ScopeInternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskHigh,
				Reputational: RiskHigh,
				Legal:        RiskHigh,
				Safety:       RiskLow,
			},
			ComplianceReqs: []string{"HR_Policy", "Labor_Law", "Anti_Discrimination", "Privacy"},
			Description:    "Personnel and HR decisions - requires HR policy compliance and legal review",
		},
		DecisionFinancial: {
			Type:             DecisionFinancial,
			Category:         CategoryFinancial,
			Risk:             RiskCritical,
			Scope:            ScopeInternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskCritical,
				Operational:  RiskHigh,
				Reputational: RiskHigh,
				Legal:        RiskHigh,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Budget_Authority", "Financial_Policy", "Board_Approval", "Audit_Trail"},
			Description:    "Financial decisions (budgets, investments) - requires financial authority and audit trail",
		},
		DecisionBuySell: {
			Type:             DecisionBuySell,
			Category:         CategoryFinancial,
			Risk:             RiskHigh,
			Scope:            ScopeExternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskHigh,
				Operational:  RiskMedium,
				Reputational: RiskMedium,
				Legal:        RiskHigh,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Valuation", "Due_Diligence", "Contract_Terms", "Approval_Authority"},
			Description:    "Buy/sell decisions for assets or services - requires valuation and due diligence",
		},
		DecisionConceptual: {
			Type:             DecisionConceptual,
			Category:         CategoryStrategic,
			Risk:             RiskMedium,
			Scope:            ScopeInternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskMedium,
				Reputational: RiskLow,
				Legal:        RiskMinimal,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Strategic_Alignment"},
			Description:    "Conceptual and strategic planning - subject to strategic alignment review",
		},
		DecisionDesign: {
			Type:             DecisionDesign,
			Category:         CategoryOperational,
			Risk:             RiskMedium,
			Scope:            ScopeInternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskHigh,
				Reputational: RiskMedium,
				Legal:        RiskLow,
				Safety:       RiskMedium,
			},
			ComplianceReqs: []string{"Design_Standards", "Safety_Requirements", "Regulatory_Compliance"},
			Description:    "Design and engineering decisions - requires standards and safety compliance",
		},
		DecisionInfoSharing: {
			Type:             DecisionInfoSharing,
			Category:         CategoryCommunication,
			Risk:             RiskHigh,
			Scope:            ScopeConfidential,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskMedium,
				Reputational: RiskCritical,
				Legal:        RiskCritical,
				Safety:       RiskLow,
			},
			ComplianceReqs: []string{"Data_Privacy", "Confidentiality", "GDPR", "Information_Classification"},
			Description:    "Information sharing and disclosure - requires privacy and confidentiality review",
		},
		DecisionStrategic: {
			Type:             DecisionStrategic,
			Category:         CategoryStrategic,
			Risk:             RiskCritical,
			Scope:            ScopeInternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskCritical,
				Operational:  RiskCritical,
				Reputational: RiskHigh,
				Legal:        RiskHigh,
				Safety:       RiskLow,
			},
			ComplianceReqs: []string{"Board_Approval", "Strategic_Review", "Stakeholder_Consultation"},
			Description:    "Strategic business decisions - requires board approval and stakeholder consultation",
		},
		DecisionLegal: {
			Type:             DecisionLegal,
			Category:         CategoryStrategic,
			Risk:             RiskCritical,
			Scope:            ScopeExternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskHigh,
				Operational:  RiskHigh,
				Reputational: RiskCritical,
				Legal:        RiskCritical,
				Safety:       RiskLow,
			},
			ComplianceReqs: []string{"Legal_Counsel", "Authority_Check", "Regulatory_Compliance", "Risk_Assessment"},
			Description:    "Legal and compliance decisions - requires legal counsel and authority verification",
		},
		DecisionAssetMgmt: {
			Type:             DecisionAssetMgmt,
			Category:         CategoryFinancial,
			Risk:             RiskHigh,
			Scope:            ScopeInternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskHigh,
				Operational:  RiskMedium,
				Reputational: RiskMedium,
				Legal:        RiskMedium,
				Safety:       RiskLow,
			},
			ComplianceReqs: []string{"Asset_Policy", "Valuation", "Depreciation_Rules"},
			Description:    "Asset management decisions - requires asset policy and valuation compliance",
		},
		DecisionOperational: {
			Type:             DecisionOperational,
			Category:         CategoryOperational,
			Risk:             RiskMedium,
			Scope:            ScopeInternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskMedium,
				Reputational: RiskLow,
				Legal:        RiskLow,
				Safety:       RiskMedium,
			},
			ComplianceReqs: []string{"Operational_Procedures", "Safety_Guidelines"},
			Description:    "Day-to-day operational decisions - subject to operational procedures",
		},
		DecisionRisk: {
			Type:             DecisionRisk,
			Category:         CategoryStrategic,
			Risk:             RiskHigh,
			Scope:            ScopeInternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskHigh,
				Operational:  RiskHigh,
				Reputational: RiskHigh,
				Legal:        RiskHigh,
				Safety:       RiskHigh,
			},
			ComplianceReqs: []string{"Risk_Framework", "Risk_Assessment", "Mitigation_Plan", "Approval_Authority"},
			Description:    "Risk management decisions - requires comprehensive risk assessment and approval",
		},
		DecisionCompliance: {
			Type:             DecisionCompliance,
			Category:         CategoryStrategic,
			Risk:             RiskCritical,
			Scope:            ScopeInternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskHigh,
				Operational:  RiskHigh,
				Reputational: RiskCritical,
				Legal:        RiskCritical,
				Safety:       RiskMedium,
			},
			ComplianceReqs: []string{"Regulatory_Framework", "Compliance_Officer", "Legal_Review", "Audit_Trail"},
			Description:    "Compliance and regulatory decisions - requires compliance officer and legal review",
		},
		DecisionOther: {
			Type:             DecisionOther,
			Category:         CategoryOperational,
			Risk:             RiskMedium,
			Scope:            ScopeInternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskMedium,
				Reputational: RiskMedium,
				Legal:        RiskMedium,
				Safety:       RiskLow,
			},
			ComplianceReqs: []string{"Case_By_Case_Review"},
			Description:    "Other decision types - requires case-by-case review",
		},
	}

	meta, ok := metadata[dt]
	if !ok {
		return nil, fmt.Errorf("unknown decision type: %s", dt)
	}
	return meta, nil
}

// GetPhysicalActionMetadata returns comprehensive metadata for physical action
func GetPhysicalActionMetadata(pa ActionTypePhysical) (*PhysicalActionMetadata, error) {
	metadata := map[ActionTypePhysical]*PhysicalActionMetadata{
		ActionPhysicalManufacturing: {
			Type:                ActionPhysicalManufacturing,
			Category:            CategoryPhysical,
			Risk:                RiskHigh,
			Scope:               ScopeInternal,
			RequiresSafety:      true,
			RequiresSupervision: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskHigh,
				Operational:  RiskHigh,
				Reputational: RiskMedium,
				Legal:        RiskMedium,
				Safety:       RiskHigh,
			},
			ComplianceReqs: []string{"Safety_Standards", "Quality_Control", "Environmental_Compliance", "Worker_Safety"},
			Description:    "Manufacturing and production - requires safety standards and quality control",
		},
		ActionPhysicalAssembly: {
			Type:                ActionPhysicalAssembly,
			Category:            CategoryPhysical,
			Risk:                RiskMedium,
			Scope:               ScopeInternal,
			RequiresSafety:      true,
			RequiresSupervision: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskHigh,
				Reputational: RiskMedium,
				Legal:        RiskLow,
				Safety:       RiskMedium,
			},
			ComplianceReqs: []string{"Assembly_Procedures", "Safety_Guidelines", "Quality_Check"},
			Description:    "Assembly and construction - requires assembly procedures and safety guidelines",
		},
		ActionPhysicalTransport: {
			Type:                ActionPhysicalTransport,
			Category:            CategoryPhysical,
			Risk:                RiskHigh,
			Scope:               ScopeExternal,
			RequiresSafety:      true,
			RequiresSupervision: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskHigh,
				Reputational: RiskHigh,
				Legal:        RiskHigh,
				Safety:       RiskCritical,
			},
			ComplianceReqs: []string{"Transport_License", "Safety_Regulations", "Insurance", "Route_Planning"},
			Description:    "Transportation and logistics - requires transport license and safety compliance",
		},
		ActionPhysicalMaintenance: {
			Type:                ActionPhysicalMaintenance,
			Category:            CategoryPhysical,
			Risk:                RiskMedium,
			Scope:               ScopeInternal,
			RequiresSafety:      true,
			RequiresSupervision: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskHigh,
				Reputational: RiskLow,
				Legal:        RiskLow,
				Safety:       RiskMedium,
			},
			ComplianceReqs: []string{"Maintenance_Schedule", "Safety_Procedures", "Equipment_Standards"},
			Description:    "Maintenance and repair - requires maintenance schedule and safety procedures",
		},
		ActionPhysicalInspection: {
			Type:                ActionPhysicalInspection,
			Category:            CategoryPhysical,
			Risk:                RiskLow,
			Scope:               ScopeInternal,
			RequiresSafety:      false,
			RequiresSupervision: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskMedium,
				Reputational: RiskLow,
				Legal:        RiskLow,
				Safety:       RiskLow,
			},
			ComplianceReqs: []string{"Inspection_Standards", "Documentation"},
			Description:    "Physical inspection and testing - requires inspection standards and documentation",
		},
		ActionPhysicalHandling: {
			Type:                ActionPhysicalHandling,
			Category:            CategoryPhysical,
			Risk:                RiskMedium,
			Scope:               ScopeInternal,
			RequiresSafety:      true,
			RequiresSupervision: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskMedium,
				Reputational: RiskLow,
				Legal:        RiskLow,
				Safety:       RiskMedium,
			},
			ComplianceReqs: []string{"Handling_Procedures", "Safety_Equipment", "Weight_Limits"},
			Description:    "Material handling and movement - requires handling procedures and safety equipment",
		},
		ActionPhysicalInstallation: {
			Type:                ActionPhysicalInstallation,
			Category:            CategoryPhysical,
			Risk:                RiskHigh,
			Scope:               ScopeExternal,
			RequiresSafety:      true,
			RequiresSupervision: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskHigh,
				Operational:  RiskHigh,
				Reputational: RiskHigh,
				Legal:        RiskMedium,
				Safety:       RiskHigh,
			},
			ComplianceReqs: []string{"Installation_Standards", "Safety_Procedures", "Testing_Protocol", "Customer_Approval"},
			Description:    "Installation and deployment - requires installation standards and testing protocol",
		},
		ActionPhysicalOperation: {
			Type:                ActionPhysicalOperation,
			Category:            CategoryPhysical,
			Risk:                RiskHigh,
			Scope:               ScopeInternal,
			RequiresSafety:      true,
			RequiresSupervision: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskHigh,
				Reputational: RiskMedium,
				Legal:        RiskMedium,
				Safety:       RiskCritical,
			},
			ComplianceReqs: []string{"Operating_Procedures", "Safety_Certification", "Training_Requirements", "Emergency_Protocols"},
			Description:    "Physical operation of equipment - requires operating procedures and safety certification",
		},
		ActionPhysicalSurgery: {
			Type:                ActionPhysicalSurgery,
			Category:            CategoryPhysical,
			Risk:                RiskCritical,
			Scope:               ScopeExternal,
			RequiresSafety:      true,
			RequiresSupervision: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskCritical,
				Operational:  RiskCritical,
				Reputational: RiskCritical,
				Legal:        RiskCritical,
				Safety:       RiskCritical,
			},
			ComplianceReqs: []string{"Medical_License", "Surgical_Protocols", "Patient_Consent", "Emergency_Procedures", "Insurance", "Regulatory_Approval"},
			Description:    "Medical/surgical procedures - requires medical license and comprehensive approvals",
		},
		ActionPhysicalDelivery: {
			Type:                ActionPhysicalDelivery,
			Category:            CategoryPhysical,
			Risk:                RiskMedium,
			Scope:               ScopeExternal,
			RequiresSafety:      false,
			RequiresSupervision: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskMedium,
				Reputational: RiskMedium,
				Legal:        RiskLow,
				Safety:       RiskLow,
			},
			ComplianceReqs: []string{"Delivery_Procedures", "Customer_Verification", "Proof_Of_Delivery"},
			Description:    "Delivery and distribution - requires delivery procedures and proof of delivery",
		},
		ActionPhysicalOther: {
			Type:                ActionPhysicalOther,
			Category:            CategoryPhysical,
			Risk:                RiskHigh,
			Scope:               ScopeInternal,
			RequiresSafety:      true,
			RequiresSupervision: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskMedium,
				Reputational: RiskMedium,
				Legal:        RiskMedium,
				Safety:       RiskHigh,
			},
			ComplianceReqs: []string{"Case_By_Case_Review", "Safety_Assessment"},
			Description:    "Other physical actions - requires case-by-case review and safety assessment",
		},
	}

	meta, ok := metadata[pa]
	if !ok {
		return nil, fmt.Errorf("unknown physical action type: %s", pa)
	}
	return meta, nil
}

// GetNonPhysicalActionMetadata returns comprehensive metadata for non-physical action
func GetNonPhysicalActionMetadata(npa ActionTypeNonPhysical) (*NonPhysicalActionMetadata, error) {
	metadata := map[ActionTypeNonPhysical]*NonPhysicalActionMetadata{
		ActionNonPhysicalResearching: {
			Type:             ActionNonPhysicalResearching,
			Category:         CategoryAnalytical,
			Risk:             RiskLow,
			Scope:            ScopeInternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskLow,
				Reputational: RiskLow,
				Legal:        RiskLow,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Data_Access_Policy", "Privacy"},
			Description:    "Research and investigation - subject to data access policies",
		},
		ActionNonPhysicalBrainstorming: {
			Type:             ActionNonPhysicalBrainstorming,
			Category:         CategoryAnalytical,
			Risk:             RiskMinimal,
			Scope:            ScopeInternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMinimal,
				Operational:  RiskMinimal,
				Reputational: RiskMinimal,
				Legal:        RiskMinimal,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{},
			Description:    "Brainstorming and ideation - minimal compliance requirements",
		},
		ActionNonPhysicalAnalyzing: {
			Type:             ActionNonPhysicalAnalyzing,
			Category:         CategoryAnalytical,
			Risk:             RiskLow,
			Scope:            ScopeInternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskMedium,
				Reputational: RiskLow,
				Legal:        RiskLow,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Data_Quality", "Analysis_Standards"},
			Description:    "Data analysis and interpretation - requires data quality standards",
		},
		ActionNonPhysicalPlanning: {
			Type:             ActionNonPhysicalPlanning,
			Category:         CategoryStrategic,
			Risk:             RiskMedium,
			Scope:            ScopeInternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskHigh,
				Reputational: RiskLow,
				Legal:        RiskLow,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Strategic_Alignment", "Resource_Availability"},
			Description:    "Planning and strategy development - subject to strategic alignment",
		},
		ActionNonPhysicalDocumenting: {
			Type:             ActionNonPhysicalDocumenting,
			Category:         CategoryOperational,
			Risk:             RiskLow,
			Scope:            ScopeInternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskLow,
				Reputational: RiskLow,
				Legal:        RiskMedium,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Documentation_Standards", "Record_Retention", "Privacy"},
			Description:    "Documentation and reporting - requires documentation standards and record retention",
		},
		ActionNonPhysicalCommunicating: {
			Type:             ActionNonPhysicalCommunicating,
			Category:         CategoryCommunication,
			Risk:             RiskMedium,
			Scope:            ScopeExternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskMedium,
				Reputational: RiskHigh,
				Legal:        RiskMedium,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Communication_Policy", "Confidentiality", "Brand_Guidelines"},
			Description:    "Communication and messaging - subject to communication policy and confidentiality",
		},
		ActionNonPhysicalNegotiating: {
			Type:             ActionNonPhysicalNegotiating,
			Category:         CategoryOperational,
			Risk:             RiskHigh,
			Scope:            ScopeExternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskHigh,
				Operational:  RiskHigh,
				Reputational: RiskHigh,
				Legal:        RiskHigh,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Negotiation_Authority", "Approval_Limits", "Legal_Review"},
			Description:    "Negotiation and mediation - requires negotiation authority and approval limits",
		},
		ActionNonPhysicalMonitoring: {
			Type:             ActionNonPhysicalMonitoring,
			Category:         CategoryOperational,
			Risk:             RiskLow,
			Scope:            ScopeInternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskMedium,
				Reputational: RiskLow,
				Legal:        RiskMedium,
				Safety:       RiskLow,
			},
			ComplianceReqs: []string{"Privacy", "Data_Protection", "Monitoring_Policy"},
			Description:    "Monitoring and surveillance - subject to privacy and data protection requirements",
		},
		ActionNonPhysicalModeling: {
			Type:             ActionNonPhysicalModeling,
			Category:         CategoryAnalytical,
			Risk:             RiskLow,
			Scope:            ScopeInternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskMedium,
				Reputational: RiskLow,
				Legal:        RiskLow,
				Safety:       RiskLow,
			},
			ComplianceReqs: []string{"Modeling_Standards", "Validation"},
			Description:    "Modeling and simulation - requires modeling standards and validation",
		},
		ActionNonPhysicalTraining: {
			Type:             ActionNonPhysicalTraining,
			Category:         CategoryOperational,
			Risk:             RiskMedium,
			Scope:            ScopeInternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskMedium,
				Reputational: RiskMedium,
				Legal:        RiskMedium,
				Safety:       RiskLow,
			},
			ComplianceReqs: []string{"Training_Standards", "Certification", "Record_Keeping"},
			Description:    "Training and education - requires training standards and certification",
		},
		ActionNonPhysicalAdvising: {
			Type:             ActionNonPhysicalAdvising,
			Category:         CategoryAnalytical,
			Risk:             RiskHigh,
			Scope:            ScopeExternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskHigh,
				Operational:  RiskMedium,
				Reputational: RiskHigh,
				Legal:        RiskHigh,
				Safety:       RiskLow,
			},
			ComplianceReqs: []string{"Professional_License", "Liability_Insurance", "Disclosure_Requirements"},
			Description:    "Advising and consulting - requires professional license and liability insurance",
		},
		ActionNonPhysicalApproving: {
			Type:             ActionNonPhysicalApproving,
			Category:         CategoryOperational,
			Risk:             RiskHigh,
			Scope:            ScopeInternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskHigh,
				Operational:  RiskHigh,
				Reputational: RiskMedium,
				Legal:        RiskHigh,
				Safety:       RiskMedium,
			},
			ComplianceReqs: []string{"Approval_Authority", "Signature_Rights", "Audit_Trail"},
			Description:    "Approval and authorization - requires approval authority and audit trail",
		},
		ActionNonPhysicalReviewing: {
			Type:             ActionNonPhysicalReviewing,
			Category:         CategoryOperational,
			Risk:             RiskMedium,
			Scope:            ScopeInternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskMedium,
				Reputational: RiskMedium,
				Legal:        RiskMedium,
				Safety:       RiskLow,
			},
			ComplianceReqs: []string{"Review_Standards", "Quality_Criteria"},
			Description:    "Review and evaluation - requires review standards and quality criteria",
		},
		ActionNonPhysicalDesigning: {
			Type:             ActionNonPhysicalDesigning,
			Category:         CategoryAnalytical,
			Risk:             RiskMedium,
			Scope:            ScopeInternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskHigh,
				Reputational: RiskMedium,
				Legal:        RiskLow,
				Safety:       RiskMedium,
			},
			ComplianceReqs: []string{"Design_Standards", "Safety_Requirements"},
			Description:    "Design and architecture - requires design standards and safety requirements",
		},
		ActionNonPhysicalDataAggregation: {
			Type:             ActionNonPhysicalDataAggregation,
			Category:         CategoryDigital,
			Risk:             RiskMedium,
			Scope:            ScopeInternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskMedium,
				Reputational: RiskMedium,
				Legal:        RiskHigh,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Data_Privacy", "GDPR", "Data_Quality", "Aggregation_Rules"},
			Description:    "Data aggregation and consolidation - requires data privacy and GDPR compliance",
		},
		ActionNonPhysicalVisualization: {
			Type:             ActionNonPhysicalVisualization,
			Category:         CategoryDigital,
			Risk:             RiskLow,
			Scope:            ScopeInternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskLow,
				Reputational: RiskMedium,
				Legal:        RiskLow,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Presentation_Standards", "Data_Privacy"},
			Description:    "Data visualization and reporting - requires presentation standards",
		},
		ActionNonPhysicalNotification: {
			Type:             ActionNonPhysicalNotification,
			Category:         CategoryCommunication,
			Risk:             RiskMedium,
			Scope:            ScopeExternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskMedium,
				Reputational: RiskMedium,
				Legal:        RiskMedium,
				Safety:       RiskLow,
			},
			ComplianceReqs: []string{"Notification_Policy", "Privacy", "Communication_Standards"},
			Description:    "Notification and alerting - subject to notification policy and privacy",
		},
		ActionNonPhysicalRAG: {
			Type:             ActionNonPhysicalRAG,
			Category:         CategoryDigital,
			Risk:             RiskMedium,
			Scope:            ScopeInternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskMedium,
				Reputational: RiskMedium,
				Legal:        RiskHigh,
				Safety:       RiskLow,
			},
			ComplianceReqs: []string{"Data_Access_Policy", "Privacy", "IP_Rights", "Source_Verification"},
			Description:    "Retrieval-Augmented Generation (RAG) - requires data access policy and IP rights compliance",
		},
		ActionNonPhysicalPresenting: {
			Type:             ActionNonPhysicalPresenting,
			Category:         CategoryCommunication,
			Risk:             RiskMedium,
			Scope:            ScopeExternal,
			RequiresApproval: false,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskLow,
				Operational:  RiskMedium,
				Reputational: RiskHigh,
				Legal:        RiskMedium,
				Safety:       RiskMinimal,
			},
			ComplianceReqs: []string{"Presentation_Policy", "Brand_Guidelines", "Confidentiality"},
			Description:    "Sharing and presenting information - subject to presentation policy and brand guidelines",
		},
		ActionNonPhysicalOther: {
			Type:             ActionNonPhysicalOther,
			Category:         CategoryOperational,
			Risk:             RiskMedium,
			Scope:            ScopeInternal,
			RequiresApproval: true,
			ImpactAnalysis: ActionImpact{
				Financial:    RiskMedium,
				Operational:  RiskMedium,
				Reputational: RiskMedium,
				Legal:        RiskMedium,
				Safety:       RiskLow,
			},
			ComplianceReqs: []string{"Case_By_Case_Review"},
			Description:    "Other non-physical actions - requires case-by-case review",
		},
	}

	meta, ok := metadata[npa]
	if !ok {
		return nil, fmt.Errorf("unknown non-physical action type: %s", npa)
	}
	return meta, nil
}

// ComprehensiveTaxonomyReport generates complete taxonomy report for action set
type ComprehensiveTaxonomyReport struct {
	OverallRisk         RiskLevel                   `json:"overall_risk"`
	TotalActions        int                         `json:"total_actions"`
	ActionsByCategory   map[ActionCategory]int      `json:"actions_by_category"`
	ActionsByRisk       map[RiskLevel]int           `json:"actions_by_risk"`
	RequiresApproval    int                         `json:"requires_approval"`
	RequiresSafety      int                         `json:"requires_safety"`
	RequiresSupervision int                         `json:"requires_supervision"`
	ComplianceReqsSet   []string                    `json:"compliance_requirements"`
	TransactionDetails  []TransactionMetadata       `json:"transaction_details"`
	DecisionDetails     []DecisionMetadata          `json:"decision_details"`
	PhysicalDetails     []PhysicalActionMetadata    `json:"physical_details"`
	NonPhysicalDetails  []NonPhysicalActionMetadata `json:"non_physical_details"`
	Summary             string                      `json:"summary"`
	Recommendations     []string                    `json:"recommendations"`
}

// GenerateComprehensiveTaxonomyReport creates detailed taxonomy report for action set
func GenerateComprehensiveTaxonomyReport(actions *AuthorizedActionSet) (*ComprehensiveTaxonomyReport, error) {
	if actions == nil {
		return nil, fmt.Errorf("actions cannot be nil")
	}

	report := &ComprehensiveTaxonomyReport{
		ActionsByCategory: make(map[ActionCategory]int),
		ActionsByRisk:     make(map[RiskLevel]int),
		ComplianceReqsSet: []string{},
	}

	complianceMap := make(map[string]bool)
	maxRisk := RiskMinimal

	// Process transactions
	for _, tt := range actions.Transactions {
		meta, err := GetTransactionMetadata(tt)
		if err != nil {
			return nil, err
		}
		report.TransactionDetails = append(report.TransactionDetails, *meta)
		report.TotalActions++
		report.ActionsByCategory[meta.Category]++
		report.ActionsByRisk[meta.Risk]++
		if meta.RequiresApproval {
			report.RequiresApproval++
		}
		for _, req := range meta.ComplianceReqs {
			complianceMap[req] = true
		}
		if riskLevel(meta.Risk) > riskLevel(maxRisk) {
			maxRisk = meta.Risk
		}
	}

	// Process decisions
	for _, dt := range actions.Decisions {
		meta, err := GetDecisionMetadata(dt)
		if err != nil {
			return nil, err
		}
		report.DecisionDetails = append(report.DecisionDetails, *meta)
		report.TotalActions++
		report.ActionsByCategory[meta.Category]++
		report.ActionsByRisk[meta.Risk]++
		if meta.RequiresApproval {
			report.RequiresApproval++
		}
		for _, req := range meta.ComplianceReqs {
			complianceMap[req] = true
		}
		if riskLevel(meta.Risk) > riskLevel(maxRisk) {
			maxRisk = meta.Risk
		}
	}

	// Process physical actions
	for _, pa := range actions.PhysicalActions {
		meta, err := GetPhysicalActionMetadata(pa)
		if err != nil {
			return nil, err
		}
		report.PhysicalDetails = append(report.PhysicalDetails, *meta)
		report.TotalActions++
		report.ActionsByCategory[meta.Category]++
		report.ActionsByRisk[meta.Risk]++
		if meta.RequiresSafety {
			report.RequiresSafety++
		}
		if meta.RequiresSupervision {
			report.RequiresSupervision++
		}
		for _, req := range meta.ComplianceReqs {
			complianceMap[req] = true
		}
		if riskLevel(meta.Risk) > riskLevel(maxRisk) {
			maxRisk = meta.Risk
		}
	}

	// Process non-physical actions
	for _, npa := range actions.NonPhysicalActions {
		meta, err := GetNonPhysicalActionMetadata(npa)
		if err != nil {
			return nil, err
		}
		report.NonPhysicalDetails = append(report.NonPhysicalDetails, *meta)
		report.TotalActions++
		report.ActionsByCategory[meta.Category]++
		report.ActionsByRisk[meta.Risk]++
		if meta.RequiresApproval {
			report.RequiresApproval++
		}
		for _, req := range meta.ComplianceReqs {
			complianceMap[req] = true
		}
		if riskLevel(meta.Risk) > riskLevel(maxRisk) {
			maxRisk = meta.Risk
		}
	}

	report.OverallRisk = maxRisk

	// Convert compliance map to slice
	for req := range complianceMap {
		report.ComplianceReqsSet = append(report.ComplianceReqsSet, req)
	}

	// Generate summary
	report.Summary = generateTaxonomySummary(report)

	// Generate recommendations
	report.Recommendations = generateRecommendations(report)

	return report, nil
}

// riskLevel converts risk level to numeric value for comparison
func riskLevel(r RiskLevel) int {
	levels := map[RiskLevel]int{
		RiskMinimal:  0,
		RiskLow:      1,
		RiskMedium:   2,
		RiskHigh:     3,
		RiskCritical: 4,
	}
	return levels[r]
}

// generateTaxonomySummary creates human-readable summary
func generateTaxonomySummary(report *ComprehensiveTaxonomyReport) string {
	parts := []string{
		fmt.Sprintf("Total Actions: %d", report.TotalActions),
		fmt.Sprintf("Overall Risk Level: %s", report.OverallRisk),
		fmt.Sprintf("Actions Requiring Approval: %d", report.RequiresApproval),
	}

	if report.RequiresSafety > 0 {
		parts = append(parts, fmt.Sprintf("Actions Requiring Safety Measures: %d", report.RequiresSafety))
	}

	if report.RequiresSupervision > 0 {
		parts = append(parts, fmt.Sprintf("Actions Requiring Supervision: %d", report.RequiresSupervision))
	}

	parts = append(parts, fmt.Sprintf("Compliance Requirements: %d unique requirements", len(report.ComplianceReqsSet)))

	return strings.Join(parts, " | ")
}

// generateRecommendations creates actionable recommendations
func generateRecommendations(report *ComprehensiveTaxonomyReport) []string {
	recommendations := []string{}

	if report.OverallRisk == RiskCritical {
		recommendations = append(recommendations, "CRITICAL: This action set contains critical risk actions. Comprehensive approval workflow and continuous monitoring required.")
	} else if report.OverallRisk == RiskHigh {
		recommendations = append(recommendations, "HIGH RISK: This action set contains high risk actions. Approval workflow and regular monitoring strongly recommended.")
	}

	if report.RequiresSafety > 0 {
		recommendations = append(recommendations, fmt.Sprintf("%d actions require safety measures. Ensure appropriate safety protocols and equipment are in place.", report.RequiresSafety))
	}

	if report.RequiresSupervision > 0 {
		recommendations = append(recommendations, fmt.Sprintf("%d actions require supervision. Ensure qualified supervisors are assigned.", report.RequiresSupervision))
	}

	if len(report.PhysicalDetails) > 0 {
		recommendations = append(recommendations, "Physical actions authorized. Verify client has appropriate physical capabilities and certifications.")
	}

	if len(report.TransactionDetails) > 0 {
		recommendations = append(recommendations, "Financial transactions authorized. Implement transaction limits and comprehensive audit trail.")
	}

	if report.RequiresApproval > 0 {
		recommendations = append(recommendations, fmt.Sprintf("%d actions require approval. Establish clear approval workflows and authority matrices.", report.RequiresApproval))
	}

	if len(report.ComplianceReqsSet) > 5 {
		recommendations = append(recommendations, fmt.Sprintf("Complex compliance requirements (%d unique requirements). Consider compliance management system.", len(report.ComplianceReqsSet)))
	}

	return recommendations
}
