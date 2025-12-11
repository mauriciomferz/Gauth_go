package device

import (
	"errors"
	"sync"
	"time"
)

// DeviceCodeStore manages device authorization codes.
type DeviceCodeStore interface {
	// Create stores a new device code
	Create(req *DeviceAuthRequest, expiresIn, interval int) (*DeviceCode, error)

	// GetByDeviceCode retrieves by device code
	GetByDeviceCode(deviceCode string) (*DeviceCode, error)

	// GetByUserCode retrieves by user code
	GetByUserCode(userCode string) (*DeviceCode, error)

	// Authorize marks the device code as authorized
	Authorize(userCode, userID string) error

	// Deny marks the device code as denied
	Deny(userCode string) error

	// UpdateLastPoll updates the last poll time
	UpdateLastPoll(deviceCode string) error
}

// MemoryDeviceCodeStore implements DeviceCodeStore in-memory.
type MemoryDeviceCodeStore struct {
	mu           sync.RWMutex
	byDeviceCode map[string]*DeviceCode
	byUserCode   map[string]*DeviceCode
}

// NewMemoryDeviceCodeStore creates an in-memory device code store.
func NewMemoryDeviceCodeStore() *MemoryDeviceCodeStore {
	return &MemoryDeviceCodeStore{
		byDeviceCode: make(map[string]*DeviceCode),
		byUserCode:   make(map[string]*DeviceCode),
	}
}

// Create stores a new device code.
func (s *MemoryDeviceCodeStore) Create(req *DeviceAuthRequest, expiresIn, interval int) (*DeviceCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	dc := &DeviceCode{
		DeviceCode: GenerateDeviceCode(),
		UserCode:   GenerateUserCode(),
		ClientID:   req.ClientID,
		Scope:      req.Scope,
		Status:     StatusPending,
		CreatedAt:  now,
		ExpiresAt:  now.Add(time.Duration(expiresIn) * time.Second),
		Interval:   interval,
	}

	s.byDeviceCode[dc.DeviceCode] = dc
	s.byUserCode[dc.UserCode] = dc

	return dc, nil
}

// GetByDeviceCode retrieves by device code.
func (s *MemoryDeviceCodeStore) GetByDeviceCode(deviceCode string) (*DeviceCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dc, ok := s.byDeviceCode[deviceCode]
	if !ok {
		return nil, errors.New("device code not found")
	}
	return dc, nil
}

// GetByUserCode retrieves by user code.
func (s *MemoryDeviceCodeStore) GetByUserCode(userCode string) (*DeviceCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dc, ok := s.byUserCode[userCode]
	if !ok {
		return nil, errors.New("user code not found")
	}
	return dc, nil
}

// Authorize marks the device code as authorized.
func (s *MemoryDeviceCodeStore) Authorize(userCode, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dc, ok := s.byUserCode[userCode]
	if !ok {
		return errors.New("user code not found")
	}
	if dc.IsExpired() {
		dc.Status = StatusExpired
		return errors.New("device code expired")
	}

	dc.Status = StatusAuthorized
	dc.UserID = userID
	return nil
}

// Deny marks the device code as denied.
func (s *MemoryDeviceCodeStore) Deny(userCode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dc, ok := s.byUserCode[userCode]
	if !ok {
		return errors.New("user code not found")
	}

	dc.Status = StatusDenied
	return nil
}

// UpdateLastPoll updates the last poll time.
func (s *MemoryDeviceCodeStore) UpdateLastPoll(deviceCode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dc, ok := s.byDeviceCode[deviceCode]
	if !ok {
		return errors.New("device code not found")
	}

	dc.LastPollAt = time.Now().UTC()
	return nil
}

// Cleanup removes expired device codes.
func (s *MemoryDeviceCodeStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for code, dc := range s.byDeviceCode {
		if now.After(dc.ExpiresAt) {
			delete(s.byDeviceCode, code)
			delete(s.byUserCode, dc.UserCode)
		}
	}
}
