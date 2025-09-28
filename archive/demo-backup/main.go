// Moved from secure_flow.go
package main

import (
	"sync"
	"time"
)

type SecureTokenManager struct {
	// store, blacklist, pasetoManager, jwtManager, keyManager, auditLog omitted for brevity
}

type KeyManager struct {
	mu           sync.RWMutex
	currentKeyID string
	keys         map[string]*KeyPair
	rotateEvery  time.Duration
	lastRotation time.Time
}

type KeyPair struct {
	// RSAPrivate, RSAPublic omitted
	PasetoKey []byte
	Created   time.Time
}

type AuditLog struct {
	mu     sync.Mutex
	events []AuditEvent
}

type AuditEvent struct {
	Timestamp time.Time
	Action    string
	TokenID   string
	UserID    string
	Success   bool
	Error     string
	Metadata  map[string]string
}
