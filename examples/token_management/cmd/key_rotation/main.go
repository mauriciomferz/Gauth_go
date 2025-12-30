package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/token"
)

// KeyRotator manages RSA key rotation and JWT signing
type KeyRotator struct {
	mu      sync.RWMutex
	keys    map[string][]byte
	current string
	signer  token.JWTSigner
}

func NewKeyRotator() *KeyRotator {
	kr := &KeyRotator{
		keys: make(map[string][]byte),
	}
	kr.rotateKey() // Generate initial key
	return kr
}

// rotateKey generates a new RSA key and sets it as current
func (kr *KeyRotator) rotateKey() {
	kr.mu.Lock()
	defer kr.mu.Unlock()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		log.Printf("Failed to generate key: %v", err)
		return
	}
	kid := kr.keyIDForKey(key)
	kr.keys[kid] = key
	kr.current = kid
	kr.signer = token.NewJWTSigner(key, token.RS256)
	log.Printf("Rotated key. New key ID: %s", kid)
}

// keyIDForKey generates a key ID for a given RSA key
func (kr *KeyRotator) keyIDForKey(key []byte) string {
	hash := sha256.Sum256(key)
	return base64.RawURLEncoding.EncodeToString(hash[:8])
}

// signToken issues a JWT for a given subject
func (kr *KeyRotator) signToken(subject string) (string, error) {
	kr.mu.RLock()
	defer kr.mu.RUnlock()
	t := &token.Token{
		ID:        fmt.Sprintf("tok-%d", time.Now().UnixNano()),
		Type:      token.Access,
		Issuer:    "key-rotation-demo",
		Subject:   subject,
		Audience:  []string{"demo"},
		IssuedAt:  time.Now(),
		NotBefore: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Scopes:    []string{"demo"},
	}
	return kr.signer.Sign(t)
}

// verifyToken validates a JWT and returns the key ID used
func (kr *KeyRotator) verifyToken(tokenStr string) (string, error) {
	kr.mu.RLock()
	defer kr.mu.RUnlock()
	// Try all known keys
	for kid, key := range kr.keys {
		signer := token.NewJWTSigner(key, token.RS256)
		_, err := signer.Verify(tokenStr)
		if err == nil {
			return kid, nil
		}
	}
	return "", fmt.Errorf("token invalid or signed with unknown key")
}

func main() {
	rotator := NewKeyRotator()

	http.HandleFunc("/rotate", func(w http.ResponseWriter, r *http.Request) {
		rotator.rotateKey()
		fmt.Fprintf(w, "Key rotated. Current key ID: %s\n", rotator.current)
	})

	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		subject := r.URL.Query().Get("sub")
		if subject == "" {
			subject = "demo-user"
		}
		tok, err := rotator.signToken(subject)
		if err != nil {
			http.Error(w, "Failed to sign token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Token: %s\n", tok)
	})

	http.HandleFunc("/validate", func(w http.ResponseWriter, r *http.Request) {
		tokenStr := r.URL.Query().Get("token")
		if tokenStr == "" {
			http.Error(w, "Token required", http.StatusBadRequest)
			return
		}
		kid, err := rotator.verifyToken(tokenStr)
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}
		fmt.Fprintf(w, "Token valid. Signed with key ID: %s\n", kid)
	})

	log.Println("Key rotation demo server running on :8082")
	log.Println("  GET /rotate - Rotate signing key")
	log.Println("  GET /token?sub=<user> - Issue token for user")
	log.Println("  GET /validate?token=<token> - Validate token")

	server := &http.Server{
		Addr:         ":8082",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}
