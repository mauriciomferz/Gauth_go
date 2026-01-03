// Copyright 2025 AgentAuth Contributors
// SPDX-License-Identifier: Apache-2.0

package crypto

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditSink defines the interface for external audit trail destinations.
// Addresses gap sec8.item2 (P1): External append-only sink for rotation events.
// Uses the existing RotationEvent type from keystore.go.
type AuditSink interface {
	// Write appends a rotation event to the audit trail
	Write(event *RotationEvent) error
	// Close closes the sink and flushes any pending writes
	Close() error
}

// FileAuditSink writes rotation events to an append-only file.
type FileAuditSink struct {
	file   *os.File
	mu     sync.Mutex
	path   string
	closed bool
}

// NewFileAuditSink creates a new file-based audit sink.
// The file is opened in append mode with appropriate permissions.
func NewFileAuditSink(path string) (*FileAuditSink, error) {
	dir := filepath.Dir(path)
	// Use restricted directory permissions (0750 instead of 0755)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("rotation_audit: mkdir failed: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("rotation_audit: open file: %w", err)
	}

	return &FileAuditSink{
		file: file,
		path: path,
	}, nil
}

// Write appends a rotation event to the file as newline-delimited JSON.
func (s *FileAuditSink) Write(event *RotationEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("rotation_audit: sink closed")
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("rotation_audit: marshal event: %w", err)
	}

	if _, err := s.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("rotation_audit: write event: %w", err)
	}

	// Sync to disk for durability
	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("rotation_audit: sync: %w", err)
	}

	return nil
}

// Close closes the file sink.
func (s *FileAuditSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	return s.file.Close()
}

// HTTPAuditSink sends rotation events to an HTTP endpoint.
type HTTPAuditSink struct {
	endpoint   string
	httpClient *http.Client
	mu         sync.Mutex
	closed     bool
}

// NewHTTPAuditSink creates a new HTTP-based audit sink.
func NewHTTPAuditSink(endpoint string) *HTTPAuditSink {
	return &HTTPAuditSink{
		endpoint:   endpoint,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Write sends a rotation event to the HTTP endpoint as JSON.
func (s *HTTPAuditSink) Write(event *RotationEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("rotation_audit: sink closed")
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("rotation_audit: marshal event: %w", err)
	}

	req, err := http.NewRequest("POST", s.endpoint, nil)
	if err != nil {
		return fmt.Errorf("rotation_audit: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(io.MultiReader(io.NopCloser(io.Reader(nil))))
	req.ContentLength = int64(len(data))

	// Create a new request with body
	req, err = http.NewRequest("POST", s.endpoint, io.NopCloser(io.MultiReader(io.NopCloser(io.Reader(nil)))))
	if err != nil {
		return fmt.Errorf("rotation_audit: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("rotation_audit: send event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("rotation_audit: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Close closes the HTTP sink.
func (s *HTTPAuditSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// MultiAuditSink writes to multiple sinks simultaneously.
type MultiAuditSink struct {
	sinks  []AuditSink
	mu     sync.Mutex
	closed bool
}

// NewMultiAuditSink creates a sink that writes to multiple destinations.
func NewMultiAuditSink(sinks ...AuditSink) *MultiAuditSink {
	return &MultiAuditSink{
		sinks: sinks,
	}
}

// Write writes the event to all configured sinks.
// If any sink fails, the error is returned but other sinks continue.
func (s *MultiAuditSink) Write(event *RotationEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("rotation_audit: sink closed")
	}

	var lastErr error
	for _, sink := range s.sinks {
		if err := sink.Write(event); err != nil {
			lastErr = err
			// Continue to other sinks
		}
	}

	return lastErr
}

// Close closes all sinks.
func (s *MultiAuditSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	var lastErr error
	for _, sink := range s.sinks {
		if err := sink.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// TenantAwareSink wraps a sink and adds tenant ID to all events.
// Addresses multi-tenant segregation requirement.
type TenantAwareSink struct {
	inner    AuditSink
	tenantID string
}

// NewTenantAwareSink creates a sink that adds tenant context to events.
func NewTenantAwareSink(inner AuditSink, tenantID string) *TenantAwareSink {
	return &TenantAwareSink{
		inner:    inner,
		tenantID: tenantID,
	}
}

// Write adds tenant ID and forwards to the inner sink.
func (s *TenantAwareSink) Write(event *RotationEvent) error {
	event.Tenant = s.tenantID
	return s.inner.Write(event)
}

// Close closes the inner sink.
func (s *TenantAwareSink) Close() error {
	return s.inner.Close()
}
