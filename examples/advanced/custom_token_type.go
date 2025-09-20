// Advanced Example: Custom Token Type
// This example demonstrates how to extend GAuth with a custom token type using the current API.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
)

// CustomToken embeds the standard TokenResponse and adds a custom field
type CustomToken struct {
	gauth.TokenResponse
	CustomField string
}

func main() {
	// Create a GAuth instance
	config := &gauth.Config{
		ClientID:          "InventoryService",
		AccessTokenExpiry: time.Hour,
	}
	gauthInstance, err := gauth.New(config, nil)
	if err != nil {
		panic(err)
	}

	// Initiate authorization to get a grant
	grantReq := gauth.AuthorizationRequest{
		ClientID: "InventoryService",
		Scopes:   []string{"inventory:manage"},
	}
	grant, err := gauthInstance.InitiateAuthorization(grantReq)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Grant issued: %+v\n", grant)

	// Request a token using the grant
	tokenReq := gauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   []string{"inventory:manage"},
		Context: context.Background(),
	}
	tokenResp, err := gauthInstance.RequestToken(tokenReq)
	if err != nil {
		panic(err)
	}

	// Create a custom token by embedding the TokenResponse
	customToken := CustomToken{
		TokenResponse: *tokenResp,
		CustomField:   "custom-claim-value",
	}
	fmt.Printf("Custom token issued: %+v\n", customToken)
}
