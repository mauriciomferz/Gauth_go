package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileStorage implements the Storage interface using JSON files
type FileStorage struct {
	directory string
	file      *os.File
	writer    *bufio.Writer
	mu        sync.Mutex
}

// FileConfig holds configuration for file storage
type FileConfig struct {
	// Directory where audit logs are stored
	Directory string

	// FilePattern for log file names (default: audit-2006-01-02.log)
	FilePattern string

	// RotateInterval for log files (default: 24h)
	RotateInterval time.Duration

	// MaxFileSize in bytes before rotation (default: 100MB)
	MaxFileSize int64
}

// NewFileStorage creates a new file-based storage
func NewFileStorage(config FileConfig) (*FileStorage, error) {
	if config.Directory == "" {
		return nil, fmt.Errorf("directory is required")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(config.Directory, 0750); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	fs := &FileStorage{
		directory: config.Directory,
	}

	if err := fs.rotate(); err != nil {
		return nil, err
	}

	return fs, nil
}

// Store implements the Storage interface
func (fs *FileStorage) Store(ctx context.Context, entry *Entry) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	if _, err := fs.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write entry: %w", err)
	}
	if _, err := fs.writer.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return fs.writer.Flush()
}

// Search implements the Storage interface
func (fs *FileStorage) Search(ctx context.Context, filter *Filter) ([]*Entry, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	var results []*Entry

	// List all log files
	files, err := filepath.Glob(filepath.Join(fs.directory, "audit-*.log"))
	if err != nil {
		return nil, fmt.Errorf("failed to list log files: %w", err)
	}

	// Process each file
	for _, file := range files {
		if err := fs.searchFile(ctx, file, filter, &results); err != nil {
			return nil, err
		}
		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}

	return results, nil
}

// GetByID implements the Storage interface
func (fs *FileStorage) GetByID(ctx context.Context, id string) (*Entry, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(fs.directory, "audit-*.log"))
	if err != nil {
		return nil, fmt.Errorf("failed to list log files: %w", err)
	}

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var entry Entry
			if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
				continue
			}
			if entry.ID == id {
				f.Close()
				return &entry, nil
			}
		}
		f.Close()
	}
	return nil, fmt.Errorf("entry not found")
}

// GetChain implements the Storage interface
func (fs *FileStorage) GetChain(ctx context.Context, chainID string) ([]*Entry, error) {
	return fs.Search(ctx, &Filter{
		ChainID: chainID,
	})
}

// Cleanup implements the Storage interface
func (fs *FileStorage) Cleanup(ctx context.Context, before time.Time) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(fs.directory, "audit-*.log"))
	if err != nil {
		return fmt.Errorf("failed to list log files: %w", err)
	}

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}
		defer f.Close()

		tmpFile := file + ".tmp"
		out, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
		if err != nil {
			f.Close()
			continue
		}

		scanner := bufio.NewScanner(f)
		writer := bufio.NewWriter(out)
		kept := 0
		for scanner.Scan() {
			var entry Entry
			if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
				continue
			}
			if entry.Timestamp.After(before) {
				data, err := json.Marshal(entry)
				if err != nil {
					continue
				}
				writer.Write(data)
				writer.WriteString("\n")
				kept++
			}
		}
		writer.Flush()
		f.Close()
		out.Close()

		if kept == 0 {
			os.Remove(file)
			os.Remove(tmpFile)
		} else {
			os.Rename(tmpFile, file)
		}
	}

	return nil
}

// Close implements io.Closer
func (fs *FileStorage) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.writer != nil {
		if err := fs.writer.Flush(); err != nil {
			return err
		}
	}
	if fs.file != nil {
		return fs.file.Close()
	}
	return nil
}

// Helper methods

func (fs *FileStorage) rotate() error {
	if fs.file != nil {
		if err := fs.Close(); err != nil {
			return err
		}
	}

	filename := filepath.Join(fs.directory, fmt.Sprintf("audit-%s.log", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	fs.file = file
	fs.writer = bufio.NewWriter(file)
	return nil
}

func (fs *FileStorage) searchFile(ctx context.Context, filename string, filter *Filter, results *[]*Entry) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip invalid entries
		}

		if fs.matchesFilter(&entry, filter) {
			*results = append(*results, &entry)
			if filter.Limit > 0 && len(*results) >= filter.Limit {
				break
			}
		}
	}

	return scanner.Err()
}

func (fs *FileStorage) matchesFilter(entry *Entry, filter *Filter) bool {
	// Check actor IDs
	if len(filter.ActorIDs) > 0 {
		actorMatch := false
		for _, id := range filter.ActorIDs {
			if entry.ActorID == id {
				actorMatch = true
				break
			}
		}
		if !actorMatch {
			return false
		}
	}
	if filter == nil {
		return true
	}

	// Check type
	if len(filter.Types) > 0 {
		typeMatch := false
		for _, t := range filter.Types {
			if entry.Type == t {
				typeMatch = true
				break
			}
		}
		if !typeMatch {
			return false
		}
	}

	// Check action
	if len(filter.Actions) > 0 {
		actionMatch := false
		for _, a := range filter.Actions {
			if entry.Action == a {
				actionMatch = true
				break
			}
		}
		if !actionMatch {
			return false
		}
	}

	// Check result
	if len(filter.Results) > 0 {
		resultMatch := false
		for _, r := range filter.Results {
			if entry.Result == r {
				resultMatch = true
				break
			}
		}
		if !resultMatch {
			return false
		}
	}

	// Check time range
	if filter.TimeRange != nil {
		if entry.Timestamp.Before(filter.TimeRange.Start) ||
			entry.Timestamp.After(filter.TimeRange.End) {
			return false
		}
	}

	// Check chain ID
	if filter.ChainID != "" && entry.ChainID != filter.ChainID {
		return false
	}

	// Check tags
	if len(filter.Tags) > 0 {
		tagMatch := false
		for _, wantTag := range filter.Tags {
			for _, tag := range entry.Tags {
				if tag == wantTag {
					tagMatch = true
					break
				}
			}
			if tagMatch {
				break
			}
		}
		if !tagMatch {
			return false
		}
	}

	// Check metadata
	for _, mf := range filter.Metadata {
		value, exists := entry.Metadata[mf.Key]
		if !exists {
			return false
		}
		switch mf.Operator {
		case "eq":
			if value != mf.Value {
				return false
			}
		case "ne":
			if value == mf.Value {
				return false
			}
		case "contains":
			if !contains(value, mf.Value) {
				return false
			}
		}
	}

	return true
}

func contains(s, substr string) bool {
	return s != "" && substr != "" && s != substr && len(s) > len(substr) && s[len(s)-len(substr):] == substr
}
