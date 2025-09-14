package main

import (
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/gauth"
)

// ...existing code...

// processTransaction simulates a transaction and checks for token validity.
func processTransaction(auth *gauth.GAuth, token string) error {
	// Simulate token expiry
	time.Sleep(2 * time.Second)
	_, err := auth.ValidateToken(token)
	if err != nil {
		return err
	}
	// Simulate business logic
	return nil
}

// isTokenExpired checks if the error is a token expiration error.
func isTokenExpired(err error) bool {
	return err != nil && (err.Error() == "token has expired" || err.Error() == "token not found")
}
