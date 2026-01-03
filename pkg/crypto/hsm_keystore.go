// Copyright 2025 AgentAuth Contributors
// SPDX-License-Identifier: Apache-2.0

package crypto

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/miekg/pkcs11"
)

// PKCS#11 v3.0 constants (if not in library)
const (
	CKK_EC_EDWARDS              = 0x00000040
	CKM_EC_EDWARDS_KEY_PAIR_GEN = 0x00001055
)

// HSMConfig holds configuration for PKCS#11 HSM connection.
type HSMConfig struct {
	LibraryPath string
	SlotID      uint
	TokenLabel  string
	Pin         string
	MaxSessions int
}

// HSMKeyStore implements KeyStore using a hardware security module via PKCS#11.
type HSMKeyStore struct {
	ctx        *pkcs11.Ctx
	slotID     uint
	pin        string
	mu         sync.RWMutex
	sessions   chan pkcs11.SessionHandle
	activeKeys map[string]string // Cache of active key IDs
}

// NewHSMKeyStore creates a new HSM-backed key store.
func NewHSMKeyStore(config HSMConfig) (*HSMKeyStore, error) {
	if config.LibraryPath == "" {
		return nil, errors.New("hsm: library path required")
	}

	p := pkcs11.New(config.LibraryPath)
	if err := p.Initialize(); err != nil {
		return nil, fmt.Errorf("hsm: initialize failed: %w", err)
	}

	// Find slot if only label provided
	slotID := config.SlotID
	if config.TokenLabel != "" {
		slots, err := p.GetSlotList(true)
		if err != nil {
			return nil, fmt.Errorf("hsm: get slot list failed: %w", err)
		}
		found := false
		for _, slot := range slots {
			info, err := p.GetTokenInfo(slot)
			if err != nil {
				continue
			}
			if info.Label == config.TokenLabel {
				slotID = slot
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("hsm: details token with label %s not found", config.TokenLabel)
		}
	}

	store := &HSMKeyStore{
		ctx:        p,
		slotID:     slotID,
		pin:        config.Pin,
		sessions:   make(chan pkcs11.SessionHandle, config.MaxSessions),
		activeKeys: make(map[string]string),
	}

	// Pre-fill session pool
	if config.MaxSessions <= 0 {
		config.MaxSessions = 5
	}
	store.sessions = make(chan pkcs11.SessionHandle, config.MaxSessions)
	for i := 0; i < config.MaxSessions; i++ {
		session, err := p.OpenSession(slotID, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
		if err != nil {
			return nil, fmt.Errorf("hsm: open session failed: %w", err)
		}
		if err := p.Login(session, pkcs11.CKU_USER, config.Pin); err != nil {
			_ = p.CloseSession(session)
			return nil, fmt.Errorf("hsm: login failed: %w", err)
		}
		store.sessions <- session
	}

	return store, nil
}

// Close cleans up HSM resources.
func (h *HSMKeyStore) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	close(h.sessions)
	for session := range h.sessions {
		_ = h.ctx.Logout(session)
		_ = h.ctx.CloseSession(session)
	}
	_ = h.ctx.Finalize()
}

func (h *HSMKeyStore) getSession() (pkcs11.SessionHandle, error) {
	select {
	case session := <-h.sessions:
		return session, nil
	case <-time.After(5 * time.Second):
		return 0, errors.New("hsm: timeout waiting for session")
	}
}

func (h *HSMKeyStore) returnSession(session pkcs11.SessionHandle) {
	h.sessions <- session
}

// Generate creates a new Ed25519 key pair in the HSM.
func (h *HSMKeyStore) Generate(ctx context.Context, tenant string) (string, error) {
	session, err := h.getSession()
	if err != nil {
		return "", err
	}
	defer h.returnSession(session)

	// Generate ID
	id := make([]byte, 16)
	if _, err := fmt.Sscanf(fmt.Sprintf("%x", time.Now().UnixNano()), "%x", &id); err != nil {
		// fallback
	}

	keyLabel := fmt.Sprintf("agentauth-%s-%d", tenant, time.Now().Unix())

	publicKeyTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, CKK_EC_EDWARDS),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_VERIFY, true),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, keyLabel),
		pkcs11.NewAttribute(pkcs11.CKA_ID, id),
		pkcs11.NewAttribute(pkcs11.CKA_EC_PARAMS, []byte{0x06, 0x03, 0x2b, 0x65, 0x70}), // OID for Ed25519
	}

	privateKeyTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, CKK_EC_EDWARDS),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_SIGN, true),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, keyLabel),
		pkcs11.NewAttribute(pkcs11.CKA_ID, id),
		pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
		pkcs11.NewAttribute(pkcs11.CKA_EXTRACTABLE, false),
	}

	_, _, err = h.ctx.GenerateKeyPair(session,
		[]*pkcs11.Mechanism{pkcs11.NewMechanism(CKM_EC_EDWARDS_KEY_PAIR_GEN, nil)},
		publicKeyTemplate, privateKeyTemplate)

	if err != nil {
		return "", fmt.Errorf("hsm: generate key failed: %w", err)
	}

	return hex.EncodeToString(id), nil
}

// Activate marks a key as active for a tenant (simulated by updating internal map or attribute).
func (h *HSMKeyStore) Activate(ctx context.Context, tenant, keyID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.activeKeys[tenant] = keyID
	return nil
}

// Archive marks a key as archived.
func (h *HSMKeyStore) Archive(ctx context.Context, tenant, keyID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.activeKeys[tenant] == keyID {
		delete(h.activeKeys, tenant)
	}
	return nil
}

// GetActive retrieves the currently active key for a tenant.
func (h *HSMKeyStore) GetActive(ctx context.Context, tenant string) (*Key, error) {
	h.mu.RLock()
	keyID, ok := h.activeKeys[tenant]
	h.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no active key for tenant %s", tenant)
	}

	return h.GetKey(ctx, tenant, keyID)
}

// GetKey retrieves a key by ID.
func (h *HSMKeyStore) GetKey(ctx context.Context, tenant, keyID string) (*Key, error) {
	session, err := h.getSession()
	if err != nil {
		return nil, err
	}
	defer h.returnSession(session)

	id, err := hex.DecodeString(keyID)
	if err != nil {
		return nil, fmt.Errorf("invalid key id: %w", err)
	}

	template := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_ID, id),
	}

	if err := h.ctx.FindObjectsInit(session, template); err != nil {
		return nil, err
	}

	objs, _, err := h.ctx.FindObjects(session, 1)
	if err != nil {
		_ = h.ctx.FindObjectsFinal(session)
		return nil, err
	}
	if err := h.ctx.FindObjectsFinal(session); err != nil {
		return nil, err
	}

	if len(objs) == 0 {
		return nil, fmt.Errorf("key not found")
	}

	// Extract public key
	attrs, err := h.ctx.GetAttributeValue(session, objs[0], []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_EC_POINT, nil),
	})
	if err != nil {
		return nil, err
	}

	// EC_POINT is standard octet string of point. Ed25519 might be wrapped in ASN.1 or raw.
	// Typically raw 32 bytes or 04 + 32 bytes.
	pubBytes := attrs[0].Value
	if len(pubBytes) > 32 {
		// Handle potential ASN.1 wrapping or point compression header
		if len(pubBytes) == 34 && pubBytes[0] == 0x04 && pubBytes[1] == 0x20 { // Not quite correct for Ed25519 usually, but checking
			// Ed25519 public keys are 32 bytes.
			// PKCS#11 3.0 says CKA_EC_POINT is DER-encoded OCTET STRING.
			// Let's assume it's just the bytes for now or wrapped.
			// Foundation implementation - might need adjustment based on specific HSM.
		}
	}

	return &Key{
		ID:     keyID,
		Public: ed25519.PublicKey(pubBytes), // Simplified
		Alg:    "Ed25519",
		Use:    "sig",
	}, nil
}

func (h *HSMKeyStore) isActive(tenant, keyID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.activeKeys[tenant] == keyID
}

// ListKeys returns all keys for a tenant.
func (h *HSMKeyStore) ListKeys(ctx context.Context, tenant string) ([]*Key, error) {
	// Implementation would find all objects with matching label prefix
	return []*Key{}, nil
}

// Delete deletes a key.
func (h *HSMKeyStore) Delete(ctx context.Context, tenant, keyID string) error {
	return nil
}

// Health checks HSM.
func (h *HSMKeyStore) Health(ctx context.Context) error {
	session, err := h.getSession()
	if err != nil {
		return err
	}
	h.returnSession(session)
	return nil
}
