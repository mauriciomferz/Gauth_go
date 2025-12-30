package main

import (
	"context"
	"fmt"

	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

func main() {
	fmt.Println("Authorization Example")
	fmt.Println("====================")

	ctx := context.Background()

	// Create authorization service
	service := authz.NewMemoryAuthorizer()

	// 1. Define policies
	fmt.Println("\n1. Creating Policies")
	fmt.Println("-------------------")

	// Admin policy - allow all actions on users resource
	adminPolicy := authz.Policy{
		ID:       "admin-policy",
		Subject:  "admin1",
		Resource: "users",
		Actions:  []string{"read", "write", "delete"},
		Effect:   authz.Allow,
	}

	// User policy - allow read action on own profile
	userPolicy := authz.Policy{
		ID:       "user-policy",
		Subject:  "user123",
		Resource: "profile:user123",
		Actions:  []string{"read"},
		Effect:   authz.Allow,
	}

	// Add policies to service
	service.AddPolicy(adminPolicy)
	service.AddPolicy(userPolicy)

	fmt.Printf("Added admin policy: %s\n", adminPolicy.ID)
	fmt.Printf("Added user policy: %s\n", userPolicy.ID)

	// 2. Test authorization requests
	fmt.Println("\n2. Authorization Checks")
	fmt.Println("----------------------")

	// Test admin access
	adminRequest := authz.Request{
		Subject:  "admin1",
		Resource: "users",
		Action:   "read",
	}

	result, err := service.Authorize(ctx, adminRequest)
	if err != nil {
		fmt.Printf("Error checking admin authorization: %v\n", err)
	} else {
		fmt.Printf("Admin read users: Allow=%t\n", result.Allow)
	}

	// Test user access to own profile
	userRequest := authz.Request{
		Subject:  "user123",
		Resource: "profile:user123",
		Action:   "read",
	}

	result, err = service.Authorize(ctx, userRequest)
	if err != nil {
		fmt.Printf("Error checking user authorization: %v\n", err)
	} else {
		fmt.Printf("User read own profile: Allow=%t\n", result.Allow)
	}

	// Test user access to other profile (should be denied)
	userRequest2 := authz.Request{
		Subject:  "user123",
		Resource: "profile:admin1",
		Action:   "read",
	}

	result, err = service.Authorize(ctx, userRequest2)
	if err != nil {
		fmt.Printf("Error checking user authorization: %v\n", err)
	} else {
		fmt.Printf("User read other profile: Allow=%t\n", result.Allow)
	}

	fmt.Println("\nAuthorization example completed!")
}
