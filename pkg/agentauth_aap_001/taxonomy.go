package agentauth_aap_001

import "fmt"

// RB2 Taxonomy enumeration and validation utilities.
// These sets are intentionally small initial seeds; expansion documented in compliance backlog.

var AllowedAgentTypes = []string{"human", "service", "team", "automation", "robot", "llm"}
var AllowedSectors = []string{"finance", "health", "legal", "it", "operations", "security", "research"}
var AllowedActionClasses = []string{"read_ops", "write_ops", "admin", "transfer", "decision", "audit"}

// membership builds a map for O(1) lookup; invoked once per slice via init.
var agentTypeSet, sectorSet, actionClassSet map[string]struct{}

//nolint:gochecknoinits // intentional package initialization
func init() {
	agentTypeSet = make(map[string]struct{}, len(AllowedAgentTypes))
	for _, v := range AllowedAgentTypes {
		agentTypeSet[v] = struct{}{}
	}
	sectorSet = make(map[string]struct{}, len(AllowedSectors))
	for _, v := range AllowedSectors {
		sectorSet[v] = struct{}{}
	}
	actionClassSet = make(map[string]struct{}, len(AllowedActionClasses))
	for _, v := range AllowedActionClasses {
		actionClassSet[v] = struct{}{}
	}
}

// ValidateTaxonomy returns error when any non-empty taxonomy field is not part of its allowed enumeration.
// For Version <3 callers SHOULD NOT invoke this (fields are ignored in canonical digest).
func ValidateTaxonomy(p *PowerOfAttorney) error {
	if p == nil {
		return fmt.Errorf("nil poa")
	}
	// Guard: only enforce when Version>=3 per canonical logic.
	if p.Version < 3 {
		return nil
	}
	if p.AgentType != "" {
		if _, ok := agentTypeSet[p.AgentType]; !ok {
			return fmt.Errorf("agent_type unsupported: %s", p.AgentType)
		}
	}
	if p.Sector != "" {
		if _, ok := sectorSet[p.Sector]; !ok {
			return fmt.Errorf("sector unsupported: %s", p.Sector)
		}
	}
	if p.ActionClass != "" {
		if _, ok := actionClassSet[p.ActionClass]; !ok {
			return fmt.Errorf("action_class unsupported: %s", p.ActionClass)
		}
	}
	return nil
}

// TaxonomyEnums returns a map of field->allowed slice for discovery enrichment.
func TaxonomyEnums() map[string][]string {
	return map[string][]string{
		"agent_type":   AllowedAgentTypes,
		"sector":       AllowedSectors,
		"action_class": AllowedActionClasses,
	}
}
