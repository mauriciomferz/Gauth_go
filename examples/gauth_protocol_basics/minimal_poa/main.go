// Title: Minimal POA (RFC 111) Flow
// Description: Basic Power of Attorney request/response illustrating core required fields.
package main

import (
	"context"
	"fmt"

	auth "github.com/mauriciomferz/Gauth_go/pkg/auth"
)

func main() {
	ctx := context.Background()
	svc := auth.NewRFCCompliantService()
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
