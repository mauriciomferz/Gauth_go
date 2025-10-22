// Package store provides token storage functionality
// This is a compatibility alias for the token package storage
package store

import (
	// no need for context or time
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/token"
)

// Re-export types from token package for compatibility
type (
	Token       = token.Token
	MemoryStore = token.MemoryStore
	Store       = token.Store
	Filter      = token.Filter
	TokenType   = token.TokenType
)
