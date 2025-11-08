// Copyright 2025 Gimel Foundation
// SPDX-License-Identifier: Apache-2.0

package crypto

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileAuditSink_Write(t *testing.T) {
	tmpDir := t.TempDir()
	sinkPath := filepath.Join(tmpDir, "rotation_audit.log")

	sink, err := NewFileAuditSink(sinkPath)
	require.NoError(t, err)
	defer sink.Close()

	event := &RotationEvent{
		ID:        "rot-123",
		Timestamp: time.Now(),
		Tenant:    "test-tenant",
		Type:      "manual",
		NewKeyID:  "key-456",
		Backend:   "inmemory",
		Success:   true,
	}

	err = sink.Write(event)
	assert.NoError(t, err)

	// Verify file exists and has content
	data, err := os.ReadFile(sinkPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "rot-123")
	assert.Contains(t, string(data), "test-tenant")
}

func TestFileAuditSink_MultipleWrites(t *testing.T) {
	tmpDir := t.TempDir()
	sinkPath := filepath.Join(tmpDir, "rotation_audit.log")

	sink, err := NewFileAuditSink(sinkPath)
	require.NoError(t, err)
	defer sink.Close()

	for i := 0; i < 5; i++ {
		event := &RotationEvent{
			ID:        fmt.Sprintf("rot-%d", i),
			Timestamp: time.Now(),
			NewKeyID:  fmt.Sprintf("key-%d", i),
			Success:   true,
		}
		err = sink.Write(event)
		require.NoError(t, err)
	}

	// Verify all events written
	data, err := os.ReadFile(sinkPath)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		assert.Contains(t, string(data), fmt.Sprintf("rot-%d", i))
	}
}

func TestFileAuditSink_ClosedWrite(t *testing.T) {
	tmpDir := t.TempDir()
	sinkPath := filepath.Join(tmpDir, "rotation_audit.log")

	sink, err := NewFileAuditSink(sinkPath)
	require.NoError(t, err)

	err = sink.Close()
	require.NoError(t, err)

	// Writing after close should fail
	event := &RotationEvent{
		ID:       "rot-123",
		NewKeyID: "key-456",
	}
	err = sink.Write(event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestFileAuditSink_CreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "nested", "dir", "audit.log")

	sink, err := NewFileAuditSink(nestedPath)
	require.NoError(t, err)
	defer sink.Close()

	// Verify directory was created
	_, err = os.Stat(filepath.Dir(nestedPath))
	assert.NoError(t, err)
}

func TestMultiAuditSink(t *testing.T) {
	tmpDir := t.TempDir()

	sink1, err := NewFileAuditSink(filepath.Join(tmpDir, "audit1.log"))
	require.NoError(t, err)
	defer sink1.Close()

	sink2, err := NewFileAuditSink(filepath.Join(tmpDir, "audit2.log"))
	require.NoError(t, err)
	defer sink2.Close()

	multiSink := NewMultiAuditSink(sink1, sink2)
	defer multiSink.Close()

	event := &RotationEvent{
		ID:       "rot-multi",
		NewKeyID: "key-multi",
		Success:  true,
	}

	err = multiSink.Write(event)
	assert.NoError(t, err)

	// Verify both files have the event
	data1, _ := os.ReadFile(filepath.Join(tmpDir, "audit1.log"))
	data2, _ := os.ReadFile(filepath.Join(tmpDir, "audit2.log"))

	assert.Contains(t, string(data1), "rot-multi")
	assert.Contains(t, string(data2), "rot-multi")
}

func TestTenantAwareSink(t *testing.T) {
	tmpDir := t.TempDir()
	sinkPath := filepath.Join(tmpDir, "tenant_audit.log")

	fileSink, err := NewFileAuditSink(sinkPath)
	require.NoError(t, err)
	defer fileSink.Close()

	tenantSink := NewTenantAwareSink(fileSink, "tenant-abc")

	event := &RotationEvent{
		ID:       "rot-tenant",
		NewKeyID: "key-tenant",
		Success:  true,
	}

	err = tenantSink.Write(event)
	assert.NoError(t, err)

	// Verify tenant ID was added
	data, err := os.ReadFile(sinkPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "tenant-abc")
}

func TestHTTPAuditSink_Instantiation(t *testing.T) {
	// Just test creation, not actual HTTP call
	sink := NewHTTPAuditSink("https://example.com/audit")
	assert.NotNil(t, sink)

	err := sink.Close()
	assert.NoError(t, err)
}
