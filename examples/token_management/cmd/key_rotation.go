package main
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// KeyRotator manages RSA key rotation
type KeyRotator struct {
	mu      sync.RWMutex
	keys    map[string]*rsa.PrivateKey
	current string
	jwtMgr  *token.JWTManager
}

// ...existing code from original key_rotation.go...
