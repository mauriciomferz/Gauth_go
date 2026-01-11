package aap

// This file provides high-level RFC compliance helper summaries and demo/validation helpers.
// Detailed configuration & data structures live in combined_config.go to avoid duplication.

import (
	"time"
)

// Summary types (lightweight – do not duplicate full config domain models)
type AAP001Summary struct {
	Version     string
	Compliance  bool
	Features    []string
	LastUpdated time.Time
}

type AAP002Summary struct {
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
	AAP001  AAP001Summary
	AAP002  AAP002Summary
	RFC0150 RFC0150Summary
}

func GetAAP001Compliance() AAP001Summary {
	return AAP001Summary{
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

func GetAAP002Compliance() AAP002Summary {
	return AAP002Summary{
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
		AAP001:  GetAAP001Compliance(),
		AAP002:  GetAAP002Compliance(),
		RFC0150: GetRFC0150Compliance(),
	}
}

// ValidateCompliance performs a simple supported-RFC check.
func ValidateCompliance(code string) bool {
	switch code {
	case "AAP-001", "AAP001", "001", "AAP-002", "AAP002", "002", "RFC-0150", "0150":
		return true
	default:
		return false
	}
}

func GetSupportedRFCs() []string { return []string{"AAP001", "AAP002", "RFC-0150"} }

// Methods ValidateAAP001Flow, TestAAP002Features, DemoAAP001PowerOfAttorney have been moved
// to pkg/aapdemo to avoid import cycles.
