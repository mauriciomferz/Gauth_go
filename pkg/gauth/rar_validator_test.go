package gauth

import (
	"testing"

	"github.com/mauriciomferz/Gauth_go/pkg/poa"
	"github.com/stretchr/testify/assert"
)

func TestValidateAuthorizationDetails_Success(t *testing.T) {
	validator := NewRARValidator()

	poaDef := &poa.PoADefinition{
		Authorization: poa.AuthorizationScope{
			AuthorizedActions: poa.AuthorizedActions{
				Transactions: []poa.TransactionType{"transfer", "payment"},
			},
		},
		Requirements: poa.Requirements{
			PowerLimits: poa.PowerLimits{
				InteractionBounds: []string{"https://api.example.com/*"},
				PowerLevels:       []string{"max_amount:USD:1000.00"},
			},
		},
	}

	details := []AuthorizationDetail{
		{
			Type:      "payment_initiation",
			Actions:   []string{"payment"},
			Locations: []string{"https://api.example.com/payments"},
			InstructedAmount: &Amount{
				Currency: "USD",
				Amount:   "500.00",
			},
		},
	}

	err := validator.ValidateAuthorizationDetails(poaDef, details)
	assert.NoError(t, err)
}

func TestValidateAuthorizationDetails_UnauthorizedAction(t *testing.T) {
	validator := NewRARValidator()

	poaDef := &poa.PoADefinition{
		Authorization: poa.AuthorizationScope{
			AuthorizedActions: poa.AuthorizedActions{
				Transactions: []poa.TransactionType{"read_only"},
			},
		},
	}

	details := []AuthorizationDetail{
		{
			Type:    "payment_initiation",
			Actions: []string{"payment"}, // Not authorized
		},
	}

	err := validator.ValidateAuthorizationDetails(poaDef, details)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "action 'payment' not authorized")
}

func TestValidateAuthorizationDetails_UnauthorizedLocation(t *testing.T) {
	validator := NewRARValidator()

	poaDef := &poa.PoADefinition{
		Requirements: poa.Requirements{
			PowerLimits: poa.PowerLimits{
				InteractionBounds: []string{"https://api.example.com/readonly/*"},
			},
		},
	}

	details := []AuthorizationDetail{
		{
			Type:      "file_access",
			Locations: []string{"https://api.example.com/admin/delete"}, // Mismatch
		},
	}

	err := validator.ValidateAuthorizationDetails(poaDef, details)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "location")
}

func TestValidateAuthorizationDetails_ExceedsLimit(t *testing.T) {
	validator := NewRARValidator()

	poaDef := &poa.PoADefinition{
		Authorization: poa.AuthorizationScope{
			AuthorizedActions: poa.AuthorizedActions{
				Transactions: []poa.TransactionType{"payment"},
			},
		},
		Requirements: poa.Requirements{
			PowerLimits: poa.PowerLimits{
				PowerLevels: []string{"max_amount:USD:100.00"},
			},
		},
	}

	details := []AuthorizationDetail{
		{
			Type:    "payment_initiation",
			Actions: []string{"payment"},
			InstructedAmount: &Amount{
				Currency: "USD",
				Amount:   "150.00", // Exceeds 100.00
			},
		},
	}

	err := validator.ValidateAuthorizationDetails(poaDef, details)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds PoA limit")
}
