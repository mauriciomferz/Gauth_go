package constraints

import (
	"fmt"
	"math/big"
	"strings"
)

// ⚠️ LEGAL DISCLAIMER ⚠️
//
// This package provides TECHNICAL authorization constraints ONLY.
//
// This does NOT claim to encode:
//   - Fiduciary duty
//   - "Best interest" judgments
//   - "Prudence" or "diligence"
//   - Context-dependent ethical reasoning
//
// Legal Reality:
//   - Fiduciary duty is a SUBJECTIVE ethical standard ("Act in client's best interest")
//   - Code can only encode OBJECTIVE logic (if amount < 1000)
//   - An AI can make a technically-compliant but imprudent decision
//   - Example: Investing in a pump-and-dump scam that meets risk_level=6 constraint
//
// Principal Liability:
//   - The Principal retains FULL LEGAL LIABILITY for all AI actions
//   - Compliance with these constraints does NOT constitute legal compliance
//   - These constraints do NOT satisfy fiduciary duty requirements
//   - Human oversight is REQUIRED for fiduciary decisions
//
// Recommendation:
//   - Use SemanticAllowList for TECHNICAL controls (address allow-lists, value limits)
//   - Use HUMAN OVERSIGHT for FIDUCIARY decisions (investment prudence, client best interest)
//
// See SQA_AUDIT_RESPONSE.md CRITICAL-3 for full analysis.

// SemanticAllowList replaces subjective "fiduciary duty" with explicit, verifiable permissions.
// DO NOT grant broad "investment power" - grant specific contract interactions only.
type SemanticAllowList struct {
	// AllowedContracts specifies the exact contract addresses the AI can interact with.
	// NO WILDCARDS allowed - each contract must be explicitly listed.
	AllowedContracts []ContractPermission

	// HardLimits specifies objective, verifiable thresholds (no subjective "risk levels").
	HardLimits HardLimits

	// Description provides human-readable context for auditing.
	Description string
}

// ContractPermission specifies permissions for a single smart contract.
type ContractPermission struct {
	// Address is the exact contract address (e.g., "0xE592427A0AEce92De3Edee1F18E0157C05861564").
	// NO WILDCARDS - must be exact 42-character hex address with 0x prefix.
	Address string

	// AllowedFunctions lists the function signatures the AI can call.
	// Must be full signatures: "swap(uint256,uint256,uint256,uint256)"
	AllowedFunctions []string

	// ParameterRules specifies objective constraints on function parameters.
	ParameterRules []ParameterRule

	// Description provides human-readable context (e.g., "Uniswap V3 Router").
	Description string
}

// ParameterRule specifies an objective constraint on a function parameter.
type ParameterRule struct {
	// ParameterIndex is the 0-based index of the parameter.
	ParameterIndex int

	// Constraint is a human-readable constraint description.
	// Example: "slippage <= 0.01" (1% max slippage)
	Constraint string
}

// HardLimits specifies absolute, objective thresholds (no subjective risk assessment).
type HardLimits struct {
	// MaxTransactionValue is the maximum value per transaction (in wei or smallest unit).
	MaxTransactionValue *big.Int

	// MaxDailyValue is the maximum total value per 24-hour period.
	MaxDailyValue *big.Int

	// MaxWeeklyValue is the maximum total value per 7-day period.
	MaxWeeklyValue *big.Int

	// MaxDailyLoss is the circuit breaker - halt if losses exceed this threshold.
	MaxDailyLoss *big.Int

	// RequireMultisig indicates if multi-signature approval is required.
	RequireMultisig bool

	// MultisigThreshold specifies the M-of-N multisig configuration.
	MultisigThreshold *MultisigConfig

	// RequirePrincipalApproval specifies when human Principal approval is required.
	RequirePrincipalApproval *Threshold

	// MaxGasPrice is the maximum gas price to prevent MEV attacks.
	MaxGasPrice *big.Int
}

// MultisigConfig specifies M-of-N multi-signature requirements.
type MultisigConfig struct {
	// RequiredApprovals is the number of signatures required (M in M-of-N).
	RequiredApprovals int

	// TotalSigners is the total number of authorized signers (N in M-of-N).
	TotalSigners int

	// AuthorizedSigners lists the public keys or addresses of authorized signers.
	AuthorizedSigners []string
}

// Threshold specifies when human Principal approval is required.
type Threshold struct {
	// Value is the threshold amount.
	Value *big.Int

	// ApprovalRequired specifies the type of approval ("principal", "multisig", "external").
	ApprovalRequired string
}

// Validate checks if the SemanticAllowList is well-formed.
func (s *SemanticAllowList) Validate() error {
	if len(s.AllowedContracts) == 0 {
		return fmt.Errorf("semantic allow-list must specify at least one contract")
	}

	for i, contract := range s.AllowedContracts {
		if err := contract.Validate(); err != nil {
			return fmt.Errorf("contract %d validation failed: %w", i, err)
		}
	}

	// Validate hard limits consistency
	if s.HardLimits.MaxDailyValue != nil && s.HardLimits.MaxWeeklyValue != nil {
		if s.HardLimits.MaxDailyValue.Cmp(s.HardLimits.MaxWeeklyValue) > 0 {
			return fmt.Errorf("max_daily_value (%s) cannot exceed max_weekly_value (%s)",
				s.HardLimits.MaxDailyValue.String(), s.HardLimits.MaxWeeklyValue.String())
		}
	}

	// Validate multisig configuration if required
	if s.HardLimits.RequireMultisig && s.HardLimits.MultisigThreshold == nil {
		return fmt.Errorf("require_multisig is true but multisig_threshold is not specified")
	}

	if s.HardLimits.MultisigThreshold != nil {
		if err := s.HardLimits.MultisigThreshold.Validate(); err != nil {
			return fmt.Errorf("multisig_threshold validation failed: %w", err)
		}
	}

	return nil
}

// Validate checks if the ContractPermission is well-formed.
func (c *ContractPermission) Validate() error {
	// Validate address format
	if c.Address == "" {
		return fmt.Errorf("contract address cannot be empty")
	}

	if !strings.HasPrefix(c.Address, "0x") || len(c.Address) != 42 {
		return fmt.Errorf("invalid contract address format: %s (must be 42-char hex with 0x prefix)", c.Address)
	}

	// Validate function signatures
	if len(c.AllowedFunctions) == 0 {
		return fmt.Errorf("contract %s must specify at least one allowed function", c.Address)
	}

	for _, fn := range c.AllowedFunctions {
		if !strings.Contains(fn, "(") || !strings.Contains(fn, ")") {
			return fmt.Errorf("invalid function signature format: %s (must include parentheses)", fn)
		}
	}

	return nil
}

// Validate checks if the MultisigConfig is well-formed.
func (m *MultisigConfig) Validate() error {
	if m.RequiredApprovals <= 0 {
		return fmt.Errorf("required_approvals must be positive, got %d", m.RequiredApprovals)
	}

	if m.RequiredApprovals > m.TotalSigners {
		return fmt.Errorf("required_approvals (%d) cannot exceed total_signers (%d)",
			m.RequiredApprovals, m.TotalSigners)
	}

	if len(m.AuthorizedSigners) != m.TotalSigners {
		return fmt.Errorf("authorized_signers count (%d) does not match total_signers (%d)",
			len(m.AuthorizedSigners), m.TotalSigners)
	}

	// Check for duplicate signers
	seen := make(map[string]bool)
	for _, signer := range m.AuthorizedSigners {
		if seen[signer] {
			return fmt.Errorf("duplicate signer in authorized_signers: %s", signer)
		}
		seen[signer] = true
	}

	return nil
}

// IsContractAllowed checks if a contract address is in the allow-list.
// Address comparison is case-insensitive for Ethereum addresses.
func (s *SemanticAllowList) IsContractAllowed(address string) bool {
	address = strings.ToLower(address)
	for _, contract := range s.AllowedContracts {
		if strings.ToLower(contract.Address) == address {
			return true
		}
	}
	return false
}

// IsFunctionAllowed checks if a function call is allowed on the given contract.
func (s *SemanticAllowList) IsFunctionAllowed(contractAddress, functionSignature string) bool {
	permission := s.GetContractPermission(contractAddress)
	if permission == nil {
		return false
	}

	for _, allowedFn := range permission.AllowedFunctions {
		if allowedFn == functionSignature {
			return true
		}
	}

	return false
}

// GetContractPermission returns the permission struct for a contract, or nil if not found.
func (s *SemanticAllowList) GetContractPermission(address string) *ContractPermission {
	address = strings.ToLower(address)
	for _, contract := range s.AllowedContracts {
		if strings.ToLower(contract.Address) == address {
			return &contract
		}
	}
	return nil
}

// CheckTransactionValue checks if a transaction value is within limits.
func (h *HardLimits) CheckTransactionValue(value *big.Int) (bool, error) {
	if h.MaxTransactionValue == nil {
		return true, nil // No limit set
	}

	if value.Cmp(h.MaxTransactionValue) > 0 {
		return false, fmt.Errorf("transaction value %s exceeds max_transaction_value %s",
			value.String(), h.MaxTransactionValue.String())
	}

	return true, nil
}
