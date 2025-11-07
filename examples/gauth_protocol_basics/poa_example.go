// poa_example.go
// Title: Minimal POA (RFC 111) Flow
// Description: Basic Power of Attorney request/response illustrating core required fields.
package main // Unified for go vet ./... compatibility

import (
	"context"
	"fmt"

	auth "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/auth"
)

//nolint:unused // Example function for documentation purposes
func runPOAExample() {
	ctx := context.Background()
	svc := auth.NewRFCCompliantService()
	// Construct a minimal POA request
	poaReq := auth.PowerOfAttorneyRequest{
		ClientID:     "demo-client",
		ResponseType: "code",
		Scope:        "ai_power_of_attorney,financial_transactions",
		RedirectURI:  "https://app.example.com/callback",
		PowerType:    "financial_transactions",
		PrincipalID:  "user-123",
		AIAgentID:    "ai-agent-1",
		Jurisdiction: "US",
		LegalBasis:   "power_of_attorney_act_2025",
	}
	resp, err := svc.AuthorizePowerOfAttorney(ctx, poaReq)
	if err != nil {
		fmt.Println("POA request failed:", err)
		return
	}
	fmt.Println("POA granted! Authorization Code:", resp.AuthorizationCode)
	fmt.Println("Legal Compliance:", resp.LegalCompliance)
	fmt.Println("Audit Record ID:", resp.AuditRecordID)
}
