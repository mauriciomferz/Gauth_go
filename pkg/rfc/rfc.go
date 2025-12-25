package rfc

// This file provides high-level RFC compliance helper summaries and demo/validation helpers.
// Detailed configuration & data structures live in combined_config.go to avoid duplication.

import (
	"time"
)

// Summary types (lightweight – do not duplicate full config domain models)
type RFC0111Summary struct {
	Version     string
	Compliance  bool
	Features    []string
	LastUpdated time.Time
}

type RFC0115Summary struct {
	Version     string
	Compliance  bool
	Features    []string
	LastUpdated time.Time
}

type RFC0150Summary struct {
	Version     string
	Compliance  bool
	Features    []string
	LastUpdated time.Time
}

type ComplianceInfo struct {
	RFC0111 RFC0111Summary
	RFC0115 RFC0115Summary
	RFC0150 RFC0150Summary
}

func GetRFC0111Compliance() RFC0111Summary {
	return RFC0111Summary{
		Version:    "1.0",
		Compliance: true,
		Features: []string{
			"token_validation",
			"proof_of_authorization",
			"delegation_attestation",
			"resource_access_control",
		},
		LastUpdated: time.Now(),
	}
}

func GetRFC0115Compliance() RFC0115Summary {
	return RFC0115Summary{
		Version:    "1.0",
		Compliance: true,
		Features: []string{
			"enhanced_security",
			"multi_factor_authentication",
			"advanced_token_management",
			"audit_logging",
		},
		LastUpdated: time.Now(),
	}
}

func GetRFC0150Compliance() RFC0150Summary {
	return RFC0150Summary{
		Version:    "1.0",
		Compliance: true,
		Features: []string{
			"go_implementation",
			"performance_optimized",
			"scalable_architecture",
			"production_ready",
		},
		LastUpdated: time.Now(),
	}
}

func GetComplianceInfo() ComplianceInfo {
	return ComplianceInfo{
		RFC0111: GetRFC0111Compliance(),
		RFC0115: GetRFC0115Compliance(),
		RFC0150: GetRFC0150Compliance(),
	}
}

// ValidateCompliance performs a simple supported-RFC check.
func ValidateCompliance(code string) bool {
	switch code {
	case "RFC-0111", "0111", "RFC-0115", "0115", "RFC-0150", "0150":
		return true
	default:
		return false
	}
}

func GetSupportedRFCs() []string { return []string{"RFC-0111", "RFC-0115", "RFC-0150"} }

// Methods ValidateRFC0111Flow, TestRFC0115Features, DemoRFC0111PowerOfAttorney have been moved to pkg/rfcdemo to avoid import cycles.
