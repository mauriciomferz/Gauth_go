// delegation_example.go
// Title: Minimal Delegation (RFC 115)
// Description: Basic delegation request showing principal -> delegate with attestation requirement and validity period.
package main

import (
	"context"
	"fmt"

	auth "github.com/mauriciomferz/AgentAuth/pkg/auth"
)

//nolint:unused // Example function for documentation purposes
func runDelegationExample() {
	ctx := context.Background()
	svc := auth.NewRFCCompliantService()
	// Construct a minimal Delegation request
	delegReq := auth.DelegationRequest{
		PrincipalID:    "user-123",
		DelegateID:     "service-456",
		ValidityPeriod: auth.ValidityPeriod{Days: 7},
		AttestationRequirement: auth.AttestationRequirement{
			Attesters: []string{"compliance_officer"},
		},
	}
	resp, err := svc.CreateAdvancedDelegation(ctx, delegReq)
	if err != nil {
		fmt.Println("Delegation request failed:", err)
		return
	}
	fmt.Println("Delegation granted! Delegation ID:", resp.DelegationID)
	fmt.Println("Status:", resp.Status)
	fmt.Println("Valid Until:", resp.ValidUntil.Format("2006-01-02"))
	fmt.Println("Attestations:", resp.Attestations)
}
