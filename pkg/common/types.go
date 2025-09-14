package common

import "time"

// DelegationLink represents one link in the authorization chain
// Used by HumanVerification
//
type DelegationLink struct {
	FromID string
	ToID   string
	Type   string // "human-to-human", "human-to-ai", "ai-to-ai"
	Level  int
	Time   time.Time
}

// HumanVerification ensures human accountability in the authorization chain
type HumanVerification struct {
	UltimateHumanID           string
	Role                      string
	LegalCapacityVerified     bool
	CapacityVerificationTime  time.Time
	CapacityVerifier          string
	DelegationChain           []DelegationLink
}

// SecondLevelApproval represents secondary approval context
type SecondLevelApproval struct {
	PrimaryApprover        string
	PrimaryApprovalTime    time.Time
	PrimaryRole            string
	SecondaryApprover      string
	SecondaryApprovalTime  time.Time
	SecondaryRole          string
	ApprovalLevel          int
	ApprovalScope          []string
	ApprovalDuration       time.Duration
	JurisdictionRules      interface{}
}
