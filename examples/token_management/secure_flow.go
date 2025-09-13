package tokenmanagement

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// SecureTokenManager combines multiple security features
type SecureTokenManager struct {
	store         *EncryptedStore
	blacklist     *token.Blacklist
	pasetoManager *PasetoManager
	jwtManager    *token.JWTManager
	keyManager    *KeyManager
	auditLog      *AuditLog
}

// KeyManager handles key rotation and storage
type KeyManager struct {
	mu           sync.RWMutex
	currentKeyID string
	keys         map[string]*KeyPair
	rotateEvery  time.Duration
	lastRotation time.Time
}

type KeyPair struct {
	RSAPrivate *rsa.PrivateKey
	RSAPublic  *rsa.PublicKey
	PasetoKey  []byte
	Created    time.Time
}

// AuditLog tracks security events
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

// NewSecureTokenManager creates a comprehensive token manager
func NewSecureTokenManager(ctx context.Context) (*SecureTokenManager, error) {
	// Create encrypted store
	baseStore := token.NewMemoryStore(24 * time.Hour)
	encKey := make([]byte, 32)
	if _, err := rand.Read(encKey); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	encStore, err := NewEncryptedStore(baseStore, encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create encrypted store: %w", err)
	}

	// Create key manager with initial keys
	keyMgr := &KeyManager{
		keys:        make(map[string]*KeyPair),
		rotateEvery: 24 * time.Hour,
	}
	if err := keyMgr.rotate(); err != nil {
		return nil, fmt.Errorf("failed to initialize keys: %w", err)
	}

	// Create PASETO manager
	pasetoMgr, err := NewPasetoManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create PASETO manager: %w", err)
	}

	// Create JWT manager with initial RSA key
	currentKey := keyMgr.getCurrentKey()
	jwtMgr := token.NewJWTManager(token.JWTConfig{
		SigningMethod: token.RS256,
		SigningKey:    currentKey.RSAPrivate,
		KeyID:         keyMgr.currentKeyID,
		MaxAge:        time.Hour,
	})

	mgr := &SecureTokenManager{
		store:         encStore,
		blacklist:     token.NewBlacklist(),
		pasetoManager: pasetoMgr,
		jwtManager:    jwtMgr,
		keyManager:    keyMgr,
		auditLog:      &AuditLog{events: make([]AuditEvent, 0)},
	}

	// Start key rotation goroutine
	go mgr.keyRotationWorker(ctx)

	return mgr, nil
}

func (km *KeyManager) rotate() error {
	// Generate new RSA key
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Generate new PASETO key
	pasetoKey := make([]byte, 32)
	if _, err := rand.Read(pasetoKey); err != nil {
		return fmt.Errorf("failed to generate PASETO key: %w", err)
	}

	// Create new key pair
	keyID := fmt.Sprintf("key-%d", time.Now().Unix())
	km.mu.Lock()
	defer km.mu.Unlock()

	km.keys[keyID] = &KeyPair{
		RSAPrivate: rsaKey,
		RSAPublic:  &rsaKey.PublicKey,
		PasetoKey:  pasetoKey,
		Created:    time.Now(),
	}
	km.currentKeyID = keyID
	km.lastRotation = time.Now()

	return nil
}

func (km *KeyManager) getCurrentKey() *KeyPair {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.keys[km.currentKeyID]
}

func (mgr *SecureTokenManager) keyRotationWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if time.Since(mgr.keyManager.lastRotation) >= mgr.keyManager.rotateEvery {
				if err := mgr.rotateKeys(ctx); err != nil {
					mgr.logAudit(AuditEvent{
						Timestamp: time.Now(),
						Action:    "key_rotation",
						Success:   false,
						Error:     err.Error(),
					})
				}
			}
		}
	}
}

func (mgr *SecureTokenManager) rotateKeys(ctx context.Context) error {
	// Rotate keys
	if err := mgr.keyManager.rotate(); err != nil {
		return err
	}

	// Update JWT manager with new key
	currentKey := mgr.keyManager.getCurrentKey()
	mgr.jwtManager = token.NewJWTManager(token.JWTConfig{
		SigningMethod: token.RS256,
		SigningKey:    currentKey.RSAPrivate,
		KeyID:         mgr.keyManager.currentKeyID,
		MaxAge:        time.Hour,
	})

	mgr.logAudit(AuditEvent{
		Timestamp: time.Now(),
		Action:    "key_rotation",
		Success:   true,
		Metadata: map[string]string{
			"new_key_id": mgr.keyManager.currentKeyID,
		},
	})

	return nil
}

func (mgr *SecureTokenManager) logAudit(event AuditEvent) {
	mgr.auditLog.mu.Lock()
	defer mgr.auditLog.mu.Unlock()
	mgr.auditLog.events = append(mgr.auditLog.events, event)
}

// ExportPublicKey exports the current public key in PEM format
func (mgr *SecureTokenManager) ExportPublicKey() string {
	currentKey := mgr.keyManager.getCurrentKey()
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(currentKey.RSAPublic)
	if err != nil {
		return ""
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubKeyBytes,
	})
	return string(pubKeyPEM)
}

// CreateToken creates a new secure token
func (mgr *SecureTokenManager) CreateToken(ctx context.Context, userID string, scopes []string) (string, error) {
	// Create token
	t := &token.Token{
		ID:        token.NewID(),
		Type:      token.Access,
		Subject:   userID,
		Issuer:    "secure-token-manager",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Scopes:    scopes,
		Metadata: map[string]string{
			"key_id": mgr.keyManager.currentKeyID,
		},
	}

	// Store encrypted token
	if err := mgr.store.Save(ctx, t); err != nil {
		mgr.logAudit(AuditEvent{
			Timestamp: time.Now(),
			Action:    "token_creation",
			TokenID:   t.ID,
			UserID:    userID,
			Success:   false,
			Error:     err.Error(),
		})
		return "", err
	}

	// Create both JWT and PASETO tokens
	jwt, err := mgr.jwtManager.SignToken(ctx, t)
	if err != nil {
		return "", err
	}

	paseto, err := mgr.pasetoManager.SignToken(ctx, t)
	if err != nil {
		return "", err
	}

	// Combine tokens
	combined := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%s.%s", jwt, paseto)))

	mgr.logAudit(AuditEvent{
		Timestamp: time.Now(),
		Action:    "token_creation",
		TokenID:   t.ID,
		UserID:    userID,
		Success:   true,
		Metadata: map[string]string{
			"key_id": mgr.keyManager.currentKeyID,
		},
	})

	return combined, nil
}

func main() {
	ctx := context.Background()

	// Create secure token manager
	mgr, err := NewSecureTokenManager(ctx)
	if err != nil {
		log.Fatalf("Failed to create secure token manager: %v", err)
	}

	// Export public key
	pubKey := mgr.ExportPublicKey()
	fmt.Printf("Current public key:\n%s\n\n", pubKey)

	// Create token
	token, err := mgr.CreateToken(ctx, "user123", []string{"read", "write"})
	if err != nil {
		log.Fatalf("Failed to create token: %v", err)
	}
	fmt.Printf("Created secure token:\n%s\n\n", token)

	// Show audit log
	fmt.Println("Audit Log:")
	for _, event := range mgr.auditLog.events {
		fmt.Printf("- %s: %s (Success: %v)\n",
			event.Timestamp.Format(time.RFC3339),
			event.Action,
			event.Success)
		if event.Error != "" {
			fmt.Printf("  Error: %s\n", event.Error)
		}
		if len(event.Metadata) > 0 {
			fmt.Printf("  Metadata: %v\n", event.Metadata)
		}
	}
}
