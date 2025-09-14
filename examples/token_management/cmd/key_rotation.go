package main

import (
	"crypto/rsa"
	"sync"
)

// KeyRotator manages RSA key rotation
type KeyRotator struct {
	mu      sync.RWMutex
	keys    map[string]*rsa.PrivateKey
	current string
}

// ...existing code from original key_rotation.go...
