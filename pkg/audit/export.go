package audit

import (
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ExportFormat represents the export file format
type ExportFormat string

const (
	ExportFormatJSON   ExportFormat = "json"
	ExportFormatCSV    ExportFormat = "csv"
	ExportFormatSyslog ExportFormat = "syslog"
	ExportFormatCEF    ExportFormat = "cef"
)

// ExportStatus represents the status of an export job
type ExportStatus string

const (
	ExportStatusPending    ExportStatus = "pending"
	ExportStatusProcessing ExportStatus = "processing"
	ExportStatusCompleted  ExportStatus = "completed"
	ExportStatusFailed     ExportStatus = "failed"
)

// ExportJob represents an async export job
type ExportJob struct {
	ID          string       `json:"id"`
	TenantID    string       `json:"tenantId"`
	Format      ExportFormat `json:"format"`
	Compressed  bool         `json:"compressed"`
	Status      ExportStatus `json:"status"`
	TotalEvents int          `json:"totalEvents"`
	FilePath    string       `json:"filePath"`
	FileSize    int64        `json:"fileSize"`
	Error       string       `json:"error,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
	CompletedAt *time.Time   `json:"completedAt,omitempty"`
	ExpiresAt   time.Time    `json:"expiresAt"`
}

// ExportFilter defines filtering criteria for exports
type ExportFilter struct {
	StartDate    time.Time
	EndDate      time.Time
	Actor        string
	Action       string
	Category     string
	Severity     string
	ResourceType string
	Result       string
	Limit        int
	Offset       int
}

// ExportService manages audit log export operations
type ExportService struct {
	repo      *Repository
	exportDir string
	jobs      map[string]*ExportJob
	mu        sync.RWMutex
}

// NewExportService creates a new export service
func NewExportService(repo *Repository, exportDir string) *ExportService {
	if exportDir == "" {
		exportDir = "/tmp/agentauth-audit-exports"
	}

	// Create export directory if it doesn't exist
	// #nosec G301
	if err := os.MkdirAll(exportDir, 0750); err != nil {
		// Log error but continue - exports will fail later
	}

	return &ExportService{
		repo:      repo,
		exportDir: exportDir,
		jobs:      make(map[string]*ExportJob),
	}
}

// CreateExportJob creates a new export job
func (s *ExportService) CreateExportJob(ctx context.Context, tenantID string, format ExportFormat, filter ExportFilter, compress bool) (*ExportJob, error) {
	job := &ExportJob{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		Format:     format,
		Compressed: compress,
		Status:     ExportStatusPending,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(24 * time.Hour), // Exports expire after 24 hours
	}

	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()

	// Start async export
	go s.processExport(ctx, job, filter)

	return job, nil
}

// GetExportJob retrieves an export job by ID
func (s *ExportService) GetExportJob(jobID string) (*ExportJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("export job not found")
	}

	return job, nil
}

// DeleteExportJob deletes an export job and its file
func (s *ExportService) DeleteExportJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("export job not found")
	}

	// Delete file if it exists
	if job.FilePath != "" {
		_ = os.Remove(job.FilePath)
	}

	delete(s.jobs, jobID)
	return nil
}

// CleanupExpiredJobs removes expired export jobs and their files
func (s *ExportService) CleanupExpiredJobs() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for jobID, job := range s.jobs {
		if now.After(job.ExpiresAt) {
			if job.FilePath != "" {
				_ = os.Remove(job.FilePath)
			}
			delete(s.jobs, jobID)
		}
	}
}

// processExport processes an export job asynchronously
func (s *ExportService) processExport(ctx context.Context, job *ExportJob, filter ExportFilter) {
	s.updateJobStatus(job.ID, ExportStatusProcessing, "")

	// Convert ExportFilter to EventFilters
	repoFilter := EventFilters{
		TenantID:     job.TenantID,
		Category:     filter.Category,
		Severity:     filter.Severity,
		UserID:       filter.Actor,
		Action:       filter.Action,
		Status:       filter.Result,
		ResourceType: filter.ResourceType,
		StartTime:    &filter.StartDate,
		EndTime:      &filter.EndDate,
		Limit:        filter.Limit,
		Offset:       filter.Offset,
	}

	// Query audit events
	events, _, err := s.repo.ListEvents(ctx, repoFilter)
	if err != nil {
		s.updateJobStatus(job.ID, ExportStatusFailed, fmt.Sprintf("failed to query events: %v", err))
		return
	}

	job.TotalEvents = len(events)

	// Generate filename
	ext := string(job.Format)
	if job.Compressed {
		ext += ".gz"
	}
	filename := fmt.Sprintf("audit-export-%s-%s.%s", job.ID, time.Now().Format("20060102-150405"), ext)
	filePath := filepath.Join(s.exportDir, filename)

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		s.updateJobStatus(job.ID, ExportStatusFailed, fmt.Sprintf("failed to create file: %v", err))
		return
	}
	defer file.Close()

	var writer io.Writer = file
	var gzipWriter *gzip.Writer

	// Add gzip compression if requested
	if job.Compressed {
		gzipWriter = gzip.NewWriter(file)
		writer = gzipWriter
		defer gzipWriter.Close()
	}

	// Export based on format
	switch job.Format {
	case ExportFormatJSON:
		err = s.exportJSON(writer, events)
	case ExportFormatCSV:
		err = s.exportCSV(writer, events)
	case ExportFormatSyslog:
		err = s.exportSyslog(writer, events)
	case ExportFormatCEF:
		err = s.exportCEF(writer, events)
	default:
		err = fmt.Errorf("unsupported format: %s", job.Format)
	}

	if err != nil {
		s.updateJobStatus(job.ID, ExportStatusFailed, fmt.Sprintf("export failed: %v", err))
		_ = os.Remove(filePath)
		return
	}

	// Flush gzip writer
	if gzipWriter != nil {
		if err := gzipWriter.Close(); err != nil {
			s.updateJobStatus(job.ID, ExportStatusFailed, fmt.Sprintf("failed to finalize compression: %v", err))
			_ = os.Remove(filePath)
			return
		}
	}

	// Get file size
	info, err := file.Stat()
	if err != nil {
		s.updateJobStatus(job.ID, ExportStatusFailed, fmt.Sprintf("failed to stat file: %v", err))
		return
	}

	// Update job
	s.mu.Lock()
	job.FilePath = filePath
	job.FileSize = info.Size()
	job.Status = ExportStatusCompleted
	now := time.Now()
	job.CompletedAt = &now
	s.mu.Unlock()
}

// updateJobStatus updates the status of an export job
func (s *ExportService) updateJobStatus(jobID string, status ExportStatus, errorMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job, exists := s.jobs[jobID]; exists {
		job.Status = status
		job.Error = errorMsg
		if status == ExportStatusCompleted || status == ExportStatusFailed {
			now := time.Now()
			job.CompletedAt = &now
		}
	}
}

// exportJSON exports events as JSON
func (s *ExportService) exportJSON(w io.Writer, events []AuditEvent) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	return encoder.Encode(map[string]interface{}{
		"exported_at": time.Now().Format(time.RFC3339),
		"total":       len(events),
		"events":      events,
	})
}

// exportCSV exports events as CSV
func (s *ExportService) exportCSV(w io.Writer, events []AuditEvent) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{
		"ID", "Timestamp", "TenantID", "UserID", "Action", "ResourceID",
		"ResourceType", "Status", "Category", "Severity", "IPAddress", "UserAgent",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write events
	for _, event := range events {
		userAgent := ""
		if event.UserAgent != nil {
			userAgent = *event.UserAgent
		}

		record := []string{
			event.ID,
			event.Timestamp.Format(time.RFC3339),
			event.TenantID,
			event.UserID,
			event.Action,
			event.ResourceID,
			event.ResourceType,
			event.Status,
			event.Category,
			event.Severity,
			event.IPAddress,
			userAgent,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// exportSyslog exports events in Syslog format (RFC 5424)
func (s *ExportService) exportSyslog(w io.Writer, events []AuditEvent) error {
	for _, event := range events {
		// Map severity to syslog priority
		priority := s.severityToPriority(event.Severity)

		// Format: <priority>version timestamp hostname app-name procid msgid structured-data message
		line := fmt.Sprintf("<%d>1 %s agentauth-audit - - - [tenant=\"%s\" user=\"%s\" action=\"%s\" resource=\"%s\" status=\"%s\"] %s\n",
			priority,
			event.Timestamp.Format(time.RFC3339),
			event.TenantID,
			event.UserID,
			event.Action,
			event.ResourceID,
			event.Status,
			event.Action,
		)

		if _, err := w.Write([]byte(line)); err != nil {
			return err
		}
	}

	return nil
}

// exportCEF exports events in Common Event Format
func (s *ExportService) exportCEF(w io.Writer, events []AuditEvent) error {
	for _, event := range events {
		// CEF Format: CEF:Version|Device Vendor|Device Product|Device Version|Signature ID|Name|Severity|Extension
		line := fmt.Sprintf("CEF:0|AgentAuth Community|AgentAuth|1.0|%s|%s|%d|rt=%d tenantId=%s suser=%s act=%s src=%s outcome=%s cat=%s\n",
			event.Category,
			event.Action,
			s.severityToCEF(event.Severity),
			event.Timestamp.Unix()*1000, // milliseconds
			event.TenantID,
			s.escapeCEF(event.UserID),
			s.escapeCEF(event.Action),
			event.IPAddress,
			event.Status,
			event.Category,
		)

		if _, err := w.Write([]byte(line)); err != nil {
			return err
		}
	}

	return nil
}

// severityToPriority maps audit severity to syslog priority
func (s *ExportService) severityToPriority(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 18 // facility 2 (mail), severity 2 (critical)
	case "high":
		return 19 // facility 2, severity 3 (error)
	case "medium":
		return 20 // facility 2, severity 4 (warning)
	case "low":
		return 21 // facility 2, severity 5 (notice)
	default:
		return 22 // facility 2, severity 6 (informational)
	}
}

// severityToCEF maps audit severity to CEF severity (0-10)
func (s *ExportService) severityToCEF(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 10
	case "high":
		return 8
	case "medium":
		return 5
	case "low":
		return 3
	default:
		return 1
	}
}

// escapeCEF escapes special characters in CEF fields
func (s *ExportService) escapeCEF(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "=", "\\=")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, "\r", "\\r")
	return value
}
