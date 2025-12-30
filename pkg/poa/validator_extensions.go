package poa

import (
	"fmt"

	"github.com/mauriciomferz/AgentAuth/pkg/poa/taxonomy"
)

func ActionCompatibilityCheck(actions *taxonomy.AuthorizedActionSet, client *AuthorizedClient) error {
	if actions == nil || client == nil {
		return fmt.Errorf("actions and client must not be nil")
	}

	// Physical actions require physical embodiment
	if actions.RequiresPhysicalCapability() && !client.IsPhysicalSystem() {
		return fmt.Errorf("physical actions authorized but client type %s is not a physical system", client.TypeEnum)
	}

	// High-autonomy physical actions require appropriate capability level
	if actions.RequiresPhysicalCapability() {
		if client.CapabilityLevel == CapabilityL0 || client.CapabilityLevel == CapabilityL1 {
			return fmt.Errorf("physical actions require capability level L2 or higher (client has %s)", client.CapabilityLevel)
		}
	}

	// Financial transactions require active operational status
	if actions.RequiresFinancialCapability() {
		if !client.CanOperate() {
			return fmt.Errorf("financial capabilities require active operational status (client status: %s)", client.StatusEnum)
		}
	}

	return nil
}
