package audit

import (
	"bytes"
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// Test basic file operations
func TestExportService_FileOperations(t *testing.T) {
	tmpDir := t.TempDir()
	service := NewExportService(nil, tmpDir)
	
	// Test export directory creation
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("Export directory should exist")
	}
	
	if service.exportDir != tmpDir {
		t.Errorf("Expected exportDir=%s, got %s", tmpDir, service.exportDir)
	}
}

// Test all export formats
func TestExportFormats(t *testing.T) {
	tmpDir := t.TempDir()
	service := NewExportService(nil, tmpDir)
	
	// Create sample events
	events := []AuditEvent{
		{
			ID:           "evt-1",
			TenantID:     "test-tenant",
			Timestamp:    time.Now(),
			Category:     "access",
			Severity:     "medium",
			UserID:       "user-123",
			Action:       "poa.create",
			ResourceID:   "poa-456",
			ResourceType: "poa",
			Status:       "success",
			IPAddress:    "192.168.1.1",
		},
		{
			ID:           "evt-2",
			TenantID:     "test-tenant",
			Timestamp:    time.Now(),
			Category:     "security",
			Severity:     "high",
			UserID:       "user-456",
			Action:       "poa.revoke",
			ResourceID:   "poa-789",
			ResourceType: "poa",
			Status:       "success",
			IPAddress:    "192.168.1.2",
		},
	}
	
	// Test JSON export
	t.Run("JSON Export", func(t *testing.T) {
		var buf bytes.Buffer
		err := service.exportJSON(&buf, events)
		if err != nil {
			t.Fatalf("JSON export failed: %v", err)
		}
		
		// Verify JSON structure
		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}
		
		if _, ok := result["exported_at"]; !ok {
			t.Error("JSON should contain exported_at field")
		}
		if total, ok := result["total"].(float64); !ok || int(total) != 2 {
			t.Errorf("Expected total=2, got %v", result["total"])
		}
	})
	
	// Test CSV export
	t.Run("CSV Export", func(t *testing.T) {
		var buf bytes.Buffer
		err := service.exportCSV(&buf, events)
		if err != nil {
			t.Fatalf("CSV export failed: %v", err)
		}
		
		// Parse CSV
		reader := csv.NewReader(&buf)
		records, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("Failed to parse CSV: %v", err)
		}
		
		// Check header + 2 data rows
		if len(records) != 3 {
			t.Errorf("Expected 3 rows (header + 2 events), got %d", len(records))
		}
		
		// Verify header
		header := records[0]
		if header[0] != "ID" || header[2] != "TenantID" {
			t.Error("CSV header incorrect")
		}
	})
	
	// Test Syslog export
	t.Run("Syslog Export", func(t *testing.T) {
		var buf bytes.Buffer
		err := service.exportSyslog(&buf, events)
		if err != nil {
			t.Fatalf("Syslog export failed: %v", err)
		}
		
		content := buf.String()
		// Check for Syslog format markers
		if !strings.Contains(content, "gauth-audit") {
			t.Error("Syslog should contain gauth-audit hostname")
		}
		if !strings.Contains(content, "poa.create") {
			t.Error("Syslog should contain action")
		}
	})
	
	// Test CEF export
	t.Run("CEF Export", func(t *testing.T) {
		var buf bytes.Buffer
		err := service.exportCEF(&buf, events)
		if err != nil {
			t.Fatalf("CEF export failed: %v", err)
		}
		
		content := buf.String()
		// Check for CEF format markers
		if !strings.Contains(content, "CEF:0") {
			t.Error("CEF should start with CEF:0")
		}
		if !strings.Contains(content, "Gimel Foundation") {
			t.Error("CEF should contain vendor name")
		}
		if !strings.Contains(content, "GAuth") {
			t.Error("CEF should contain product name")
		}
	})
}

// Test compression and decompression
func TestCompressionFormats(t *testing.T) {
	tmpDir := t.TempDir()
	service := NewExportService(nil, tmpDir)
	
	events := []AuditEvent{
		{
			ID:           "evt-test",
			TenantID:     "test",
			Timestamp:    time.Now(),
			Category:     "test",
			Severity:     "low",
			UserID:       "user-1",
			Action:       "test.action",
			ResourceID:   "res-1",
			ResourceType: "test",
			Status:       "success",
			IPAddress:    "127.0.0.1",
		},
	}
	
	// Test uncompressed
	t.Run("Uncompressed", func(t *testing.T) {
		var buf bytes.Buffer
		err := service.exportJSON(&buf, events)
		if err != nil {
			t.Fatalf("Export failed: %v", err)
		}
		
		uncompressedSize := buf.Len()
		if uncompressedSize == 0 {
			t.Error("Uncompressed data should not be empty")
		}
		
		// Test compression
		var compressedBuf bytes.Buffer
		gzipWriter := gzip.NewWriter(&compressedBuf)
		_, err = gzipWriter.Write(buf.Bytes())
		if err != nil {
			t.Fatalf("Compression failed: %v", err)
		}
		gzipWriter.Close()
		
		compressedSize := compressedBuf.Len()
		if compressedSize >= uncompressedSize {
			t.Log("Warning: Compressed size not smaller (data too small to compress effectively)")
		}
		
		// Test decompression
		gzipReader, err := gzip.NewReader(&compressedBuf)
		if err != nil {
			t.Fatalf("Decompression failed: %v", err)
		}
		defer gzipReader.Close()
		
		decompressed, err := io.ReadAll(gzipReader)
		if err != nil {
			t.Fatalf("Failed to read decompressed data: %v", err)
		}
		
		if !bytes.Equal(buf.Bytes(), decompressed) {
			t.Error("Decompressed data does not match original")
		}
	})
}

// Test severity mapping functions
func TestSeverityMapping(t *testing.T) {
	tmpDir := t.TempDir()
	service := NewExportService(nil, tmpDir)
	
	tests := []struct {
		severity         string
		expectedPriority int
		expectedCEF      int
	}{
		{"critical", 18, 10},
		{"high", 19, 8},
		{"medium", 20, 5},
		{"low", 21, 3},
		{"info", 22, 1},
		{"unknown", 22, 1},
	}
	
	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			priority := service.severityToPriority(tt.severity)
			if priority != tt.expectedPriority {
				t.Errorf("Expected priority %d for %s, got %d", tt.expectedPriority, tt.severity, priority)
			}
			
			cef := service.severityToCEF(tt.severity)
			if cef != tt.expectedCEF {
				t.Errorf("Expected CEF severity %d for %s, got %d", tt.expectedCEF, tt.severity, cef)
			}
		})
	}
}

// Test CEF string escaping
func TestCEFEscaping(t *testing.T) {
	tmpDir := t.TempDir()
	service := NewExportService(nil, tmpDir)
	
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with=equals", "with\\=equals"},
		{"with\\backslash", "with\\\\backslash"},
		{"with\nnewline", "with\\nnewline"},
		{"with\rcarriage", "with\\rcarriage"},
		{"complex=value\\with\nspecial", "complex\\=value\\\\with\\nspecial"},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := service.escapeCEF(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Test job lifecycle
func TestExportJobLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	service := NewExportService(nil, tmpDir)
	
	// Test job creation
	job := &ExportJob{
		ID:         "test-job-123",
		TenantID:   "test-tenant",
		Format:     ExportFormatJSON,
		Compressed: false,
		Status:     ExportStatusPending,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	
	service.mu.Lock()
	service.jobs[job.ID] = job
	service.mu.Unlock()
	
	// Test retrieval
	t.Run("Get Job", func(t *testing.T) {
		retrieved, err := service.GetExportJob(job.ID)
		if err != nil {
			t.Fatalf("Failed to get job: %v", err)
		}
		if retrieved.ID != job.ID {
			t.Error("Retrieved job ID mismatch")
		}
	})
	
	// Test non-existent job
	t.Run("Get Non-Existent Job", func(t *testing.T) {
		_, err := service.GetExportJob("non-existent")
		if err == nil {
			t.Error("Should return error for non-existent job")
		}
	})
	
	// Test deletion
	t.Run("Delete Job", func(t *testing.T) {
		err := service.DeleteExportJob(job.ID)
		if err != nil {
			t.Fatalf("Failed to delete job: %v", err)
		}
		
		// Verify deletion
		_, err = service.GetExportJob(job.ID)
		if err == nil {
			t.Error("Job should be deleted")
		}
	})
}

// Test cleanup of expired jobs
func TestCleanupExpiredJobs(t *testing.T) {
	tmpDir := t.TempDir()
	service := NewExportService(nil, tmpDir)
	
	// Add expired job
	expiredJob := &ExportJob{
		ID:        "expired-job",
		TenantID:  "test",
		Status:    ExportStatusCompleted,
		CreatedAt: time.Now().Add(-25 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
	}
	
	// Add current job
	currentJob := &ExportJob{
		ID:        "current-job",
		TenantID:  "test",
		Status:    ExportStatusCompleted,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(23 * time.Hour),
	}
	
	service.mu.Lock()
	service.jobs[expiredJob.ID] = expiredJob
	service.jobs[currentJob.ID] = currentJob
	service.mu.Unlock()
	
	// Run cleanup
	service.CleanupExpiredJobs()
	
	// Verify expired job is removed
	service.mu.RLock()
	_, expiredExists := service.jobs[expiredJob.ID]
	_, currentExists := service.jobs[currentJob.ID]
	service.mu.RUnlock()
	
	if expiredExists {
		t.Error("Expired job should be removed")
	}
	if !currentExists {
		t.Error("Current job should still exist")
	}
}
