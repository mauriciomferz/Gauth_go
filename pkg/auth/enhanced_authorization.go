package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/common"
	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// ApprovalLevel represents the level of approval required
type ApprovalLevel int

const (
	SingleApproval ApprovalLevel = iota
	DualApproval
	MultiLevelApproval
)

// JurisdictionRules defines jurisdiction-specific authorization rules
type JurisdictionRules struct {
	// Country code for the jurisdiction
	Country string

	// Required approval levels for different action types
	RequiredApprovals map[string]ApprovalLevel

	// Special requirements for this jurisdiction
	FiduciaryDuties       []FiduciaryDuty
	IntegrityRequirements []string
	ComplianceRules       []string

	// Jurisdiction-specific value limits
	ValueLimits map[string]float64

	// Required roles/positions for authorization
	RequiredRoles []string
}

// AuthorizationEnforcer handles enhanced authorization rules
type AuthorizationEnforcer interface {
	// VerifyHumanInChain ensures human accountability
	VerifyHumanInChain(ctx context.Context, token *token.EnhancedToken) error

	// EnforceSecondLevelApproval implements dual control
	EnforceSecondLevelApproval(ctx context.Context, token *token.EnhancedToken, action string) error

	// ValidateJurisdictionRules checks jurisdiction compliance
	ValidateJurisdictionRules(ctx context.Context, token *token.EnhancedToken, rules *JurisdictionRules) error
}

// StandardAuthorizationEnforcer implements AuthorizationEnforcer
type StandardAuthorizationEnforcer struct {
	store    token.EnhancedStore
	verifier token.VerificationSystem
	registry RegistryVerifier
}

func NewStandardAuthorizationEnforcer(
	store token.EnhancedStore,
	verifier token.VerificationSystem,
	registry RegistryVerifier,
) *StandardAuthorizationEnforcer {
	return &StandardAuthorizationEnforcer{
		store:    store,
		verifier: verifier,
		registry: registry,
	}
}

// VerifyHumanInChain ensures there's always a human at the top of the authorization chain
func (e *StandardAuthorizationEnforcer) VerifyHumanInChain(ctx context.Context, token *token.EnhancedToken) error {
	verification, err := e.store.GetHumanVerification(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to get human verification: %w", err)
	}

	// Verify human exists and has legal capacity
	if !verification.LegalCapacityVerified {
		return fmt.Errorf("human legal capacity not verified")
	}

	// Verify delegation chain integrity
	if err := e.verifyDelegationChain(ctx, verification.DelegationChain); err != nil {
		return fmt.Errorf("invalid delegation chain: %w", err)
	}

	// Ensure first link is human-to-human or human-to-ai
	firstLink := verification.DelegationChain[0]
	if firstLink.Type != "human-to-human" && firstLink.Type != "human-to-ai" {
		return fmt.Errorf("first delegation must be from human")
	}

	return nil
}

// EnforceSecondLevelApproval implements the dual control principle
func (e *StandardAuthorizationEnforcer) EnforceSecondLevelApproval(
	ctx context.Context, token *token.EnhancedToken, action string,
) error {
	rules, err := e.getJurisdictionRules(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to get jurisdiction rules: %w", err)
	}

	// Determine required approval level
	requiredLevel := rules.RequiredApprovals[action]
	if requiredLevel >= DualApproval {
		approval, err := e.store.GetSecondLevelApproval(ctx, token)
		if err != nil {
			return fmt.Errorf("failed to get second level approval: %w", err)
		}

		// Verify both approvers exist
		if approval.SecondaryApprover == "" {
			return fmt.Errorf("secondary approval required")
		}

		// Verify roles meet requirements
		if !e.verifyApproverRoles(ctx, approval, rules.RequiredRoles) {
			return fmt.Errorf("approver roles do not meet requirements")
		}

		// Verify approval is still valid
		if time.Since(approval.SecondaryApprovalTime) > approval.ApprovalDuration {
			return fmt.Errorf("second level approval has expired")
		}
	}

	return nil
}

// ValidateJurisdictionRules checks compliance with jurisdiction-specific requirements
func (e *StandardAuthorizationEnforcer) ValidateJurisdictionRules(
	ctx context.Context, token *token.EnhancedToken, rules *JurisdictionRules,
) error {
	// Verify fiduciary duties
	if err := e.verifyFiduciaryDuties(ctx, token, rules.FiduciaryDuties); err != nil {
		return fmt.Errorf("fiduciary duties not met: %w", err)
	}

	// Check integrity requirements
	if err := e.verifyIntegrityRequirements(ctx, token, rules.IntegrityRequirements); err != nil {
		return fmt.Errorf("integrity requirements not met: %w", err)
	}

	// Validate value limits
	if err := e.verifyValueLimits(ctx, token, rules.ValueLimits); err != nil {
		return fmt.Errorf("value limits exceeded: %w", err)
	}

	return nil
}

// Helper methods

func (e *StandardAuthorizationEnforcer) verifyDelegationChain(_ context.Context, chain []common.DelegationLink) error {
	if len(chain) == 0 {
		return fmt.Errorf("empty delegation chain")
	}

	// Verify chain links are properly connected
	for i := 1; i < len(chain); i++ {
		if chain[i].FromID != chain[i-1].ToID {
			return fmt.Errorf("broken delegation chain")
		}
		if chain[i].Level <= chain[i-1].Level {
			return fmt.Errorf("invalid delegation level")
		}
	}

	return nil
}

func (e *StandardAuthorizationEnforcer) verifyApproverRoles(_ context.Context, _ *common.SecondLevelApproval, _ []string) bool {
	// Implementation would verify that approvers have required roles
	return true
}

func (e *StandardAuthorizationEnforcer) getJurisdictionRules(
	_ context.Context, _ *token.EnhancedToken,
) (*JurisdictionRules, error) {
	// Implementation would get rules for token's jurisdiction
	return nil, nil
}

func (e *StandardAuthorizationEnforcer) verifyFiduciaryDuties(
	_ context.Context, _ *token.EnhancedToken, _ []FiduciaryDuty,
) error {
	// Implementation would verify fiduciary duties are met
	return nil
}

func (e *StandardAuthorizationEnforcer) verifyIntegrityRequirements(_ context.Context, _ *token.EnhancedToken, _ []string) error {
	// Implementation would verify integrity requirements
	return nil
}

func (e *StandardAuthorizationEnforcer) verifyValueLimits(_ context.Context, _ *token.EnhancedToken, _ map[string]float64) error {
	// Implementation would verify value limits
	return nil
}
