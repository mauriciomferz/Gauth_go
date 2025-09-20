package main

import (
	"time"
)

// This example demonstrates using the enhanced typed metadata system
// to create, manage and access strongly typed event data.

type UserInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	IsActive  bool      `json:"is_active"`
	Score     float64   `json:"score"`
}

// ...rest of the code from the original enhanced_typed_metadata.go...
