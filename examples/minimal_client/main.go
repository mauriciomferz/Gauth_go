// Minimal GAuth Client Example
// This example demonstrates how a client might request authorization from a GAuth server.
// [GAuth] Only GAuth protocol logic is used here.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
)

func main() {
	// Simulate a client requesting authorization for an action
	config := &gauth.Config{
		ClientID:          "PaymentService",
		AccessTokenExpiry: 2 * time.Hour,
	}
	gauthInstance, err := gauth.New(config, nil)
	if err != nil {
		panic(err)
	}

	// Request authorization (grant)
	grantReq := gauth.AuthorizationRequest{
		ClientID: "PaymentService",
		Scopes:   []string{"payment:process"},
	}
	grant, err := gauthInstance.InitiateAuthorization(grantReq)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Grant issued: %+v\n", grant)

	// Request a token using the grant
	tokenReq := gauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   []string{"payment:process"},
		Context: context.Background(),
	}
	tokenResp, err := gauthInstance.RequestToken(tokenReq)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Token issued: %+v\n", tokenResp)

	// The principal grants power of attorney to the agent

	// The agent requests authorization to perform the action
}
