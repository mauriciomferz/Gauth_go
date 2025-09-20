// Minimal GAuth Server Example
// This example demonstrates how to set up a minimal GAuth authorization server and issue a grant and token.
// [GAuth] Only GAuth protocol logic is used here.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
)

func main() {
	// Initialize the GAuth authorization server
	config := &gauth.Config{
		ClientID:          "OrderService",
		AccessTokenExpiry: time.Hour,
	}
	gauthInstance, err := gauth.New(config, nil)
	if err != nil {
		panic(err)
	}

	// Issue a grant (authorization)
	grantReq := gauth.AuthorizationRequest{
		ClientID: "OrderService",
		Scopes:   []string{"order:process"},
	}
	grant, err := gauthInstance.InitiateAuthorization(grantReq)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Grant issued: %+v\n", grant)

	// Request a token using the grant
	tokenReq := gauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   []string{"order:process"},
		Context: context.Background(),
	}
	tokenResp, err := gauthInstance.RequestToken(tokenReq)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Token issued: %+v\n", tokenResp)
}
