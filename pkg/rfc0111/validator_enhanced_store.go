package rfc0111

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// BoltDailyLimitStore implements DailyLimitStore using BoltDB persistence
type BoltDailyLimitStore struct {
	dbPath      string
	data        map[string]map[string]float64 // delegationID -> date -> amount
	mu          sync.RWMutex
	persistence *dailyLimitPersistence
}

type dailyLimitPersistence struct {
	Version   string                        `json:"version"`
	Timestamp time.Time                     `json:"timestamp"`
	Data      map[string]map[string]float64 `json:"data"`
	Metadata  map[string]interface{}        `json:"metadata"`
}

// NewBoltDailyLimitStore creates a new BoltDB-backed daily limit store
func NewBoltDailyLimitStore(dbPath string) (*BoltDailyLimitStore, error) {
	store := &BoltDailyLimitStore{
		dbPath: dbPath,
		data:   make(map[string]map[string]float64),
		persistence: &dailyLimitPersistence{
			Version:  "1.0",
			Data:     make(map[string]map[string]float64),
			Metadata: make(map[string]interface{}),
		},
	}

	// Ensure directory exists with restricted permissions (0750 instead of 0755)
	if err := os.MkdirAll(filepath.Dir(dbPath), 0750); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	} // Load existing data
	if err := store.loadFromDisk(); err != nil {
		// If file doesn't exist, that's okay - we'll create it on first save
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load daily limits: %w", err)
		}
	}

	return store, nil
}

// GetDailyUsage retrieves the current usage for a delegation on a specific date
func (s *BoltDailyLimitStore) GetDailyUsage(delegationID, date string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if delegationData, exists := s.data[delegationID]; exists {
		if usage, exists := delegationData[date]; exists {
			return usage, nil
		}
	}

	return 0.0, nil // No usage recorded yet
}

// IncrementDailyUsage adds to the daily usage for a delegation
func (s *BoltDailyLimitStore) IncrementDailyUsage(delegationID, date string, amount float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data[delegationID] == nil {
		s.data[delegationID] = make(map[string]float64)
	}

	s.data[delegationID][date] += amount

	// Persist to disk
	return s.saveToDisk()
}

// ResetDailyUsage resets the usage for a delegation on a specific date
func (s *BoltDailyLimitStore) ResetDailyUsage(delegationID, date string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if delegationData, exists := s.data[delegationID]; exists {
		delete(delegationData, date)
		if len(delegationData) == 0 {
			delete(s.data, delegationID)
		}
	}

	// Persist to disk
	return s.saveToDisk()
}

// ExportDailyLimits exports all daily limit data for analysis
func (s *BoltDailyLimitStore) ExportDailyLimits(ctx context.Context) (map[string]map[string]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Deep copy the data
	result := make(map[string]map[string]float64)
	for delegationID, dateMap := range s.data {
		result[delegationID] = make(map[string]float64)
		for date, amount := range dateMap {
			result[delegationID][date] = amount
		}
	}

	return result, nil
}

// CleanupOldData removes data older than specified retention period
func (s *BoltDailyLimitStore) CleanupOldData(retentionDays int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoffDate := time.Now().AddDate(0, 0, -retentionDays).Format("2006-01-02")

	for delegationID, dateMap := range s.data {
		for date := range dateMap {
			if date < cutoffDate {
				delete(dateMap, date)
			}
		}
		if len(dateMap) == 0 {
			delete(s.data, delegationID)
		}
	}

	return s.saveToDisk()
}

// loadFromDisk loads daily limit data from persistent storage
func (s *BoltDailyLimitStore) loadFromDisk() error {
	file, err := os.ReadFile(s.dbPath)
	if err != nil {
		return err
	}

	var persistence dailyLimitPersistence
	if err := json.Unmarshal(file, &persistence); err != nil {
		return fmt.Errorf("failed to unmarshal daily limits: %w", err)
	}

	s.persistence = &persistence
	s.data = persistence.Data
	if s.data == nil {
		s.data = make(map[string]map[string]float64)
	}

	return nil
}

// saveToDisk persists daily limit data to disk
func (s *BoltDailyLimitStore) saveToDisk() error {
	s.persistence.Data = s.data
	s.persistence.Timestamp = time.Now()

	// Safely increment save_count
	if count, exists := s.persistence.Metadata["save_count"]; exists {
		if countInt, ok := count.(int); ok {
			s.persistence.Metadata["save_count"] = countInt + 1
		} else {
			s.persistence.Metadata["save_count"] = 1
		}
	} else {
		s.persistence.Metadata["save_count"] = 1
	}

	data, err := json.MarshalIndent(s.persistence, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal daily limits: %w", err)
	}

	return os.WriteFile(s.dbPath, data, 0600)
}

// SimpleConditionalEngine implements basic conditional expression evaluation
type SimpleConditionalEngine struct{}

// NewSimpleConditionalEngine creates a new simple conditional engine
func NewSimpleConditionalEngine() *SimpleConditionalEngine {
	return &SimpleConditionalEngine{}
}

// EvaluateCondition evaluates a conditional expression against provided context
func (e *SimpleConditionalEngine) EvaluateCondition(condition string, context map[string]interface{}) (bool, error) {
	// Simple implementation for basic conditions
	// Format: "field operator value" or "field operator value AND field2 operator2 value2"

	// Split by AND/OR operators
	parts := strings.Split(condition, " AND ")
	if len(parts) == 1 {
		parts = strings.Split(condition, " OR ")
		if len(parts) > 1 {
			// OR logic
			for _, part := range parts {
				result, err := e.evaluateSingleCondition(strings.TrimSpace(part), context)
				if err != nil {
					return false, err
				}
				if result {
					return true, nil // Any true condition satisfies OR
				}
			}
			return false, nil
		}
	} else {
		// AND logic
		for _, part := range parts {
			result, err := e.evaluateSingleCondition(strings.TrimSpace(part), context)
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil // Any false condition fails AND
			}
		}
		return true, nil
	}

	// Single condition
	return e.evaluateSingleCondition(condition, context)
}

// evaluateSingleCondition evaluates a single conditional expression
func (e *SimpleConditionalEngine) evaluateSingleCondition(condition string, context map[string]interface{}) (bool, error) {
	// Parse condition: "field operator value"
	parts := strings.Fields(condition)
	if len(parts) != 3 {
		return false, fmt.Errorf("invalid condition format: expected 'field operator value'")
	}

	field, operator, expectedValue := parts[0], parts[1], parts[2]

	actualValue, exists := context[field]
	if !exists {
		return false, fmt.Errorf("field %s not found in context", field)
	}

	switch operator {
	case "==", "=":
		return fmt.Sprintf("%v", actualValue) == expectedValue, nil
	case "!=", "<>":
		return fmt.Sprintf("%v", actualValue) != expectedValue, nil
	case ">":
		return e.compareNumbers(actualValue, expectedValue, ">")
	case ">=":
		return e.compareNumbers(actualValue, expectedValue, ">=")
	case "<":
		return e.compareNumbers(actualValue, expectedValue, "<")
	case "<=":
		return e.compareNumbers(actualValue, expectedValue, "<=")
	case "contains":
		actualStr := fmt.Sprintf("%v", actualValue)
		return strings.Contains(actualStr, expectedValue), nil
	case "starts_with":
		actualStr := fmt.Sprintf("%v", actualValue)
		return strings.HasPrefix(actualStr, expectedValue), nil
	case "ends_with":
		actualStr := fmt.Sprintf("%v", actualValue)
		return strings.HasSuffix(actualStr, expectedValue), nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

// compareNumbers compares numeric values
func (e *SimpleConditionalEngine) compareNumbers(actual interface{}, expected string, operator string) (bool, error) {
	actualFloat, err := e.toFloat64(actual)
	if err != nil {
		return false, fmt.Errorf("actual value is not numeric: %v", actual)
	}

	expectedFloat, err := strconv.ParseFloat(expected, 64)
	if err != nil {
		return false, fmt.Errorf("expected value is not numeric: %s", expected)
	}

	switch operator {
	case ">":
		return actualFloat > expectedFloat, nil
	case ">=":
		return actualFloat >= expectedFloat, nil
	case "<":
		return actualFloat < expectedFloat, nil
	case "<=":
		return actualFloat <= expectedFloat, nil
	default:
		return false, fmt.Errorf("invalid numeric operator: %s", operator)
	}
}

// toFloat64 converts various numeric types to float64
func (e *SimpleConditionalEngine) toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// ValidateConditionSyntax validates the syntax of a conditional expression
func (e *SimpleConditionalEngine) ValidateConditionSyntax(condition string) error {
	if strings.TrimSpace(condition) == "" {
		return fmt.Errorf("empty condition")
	}

	// Check for supported operators
	supportedOperators := []string{"==", "=", "!=", "<>", ">", ">=", "<", "<=", "contains", "starts_with", "ends_with"}
	hasOperator := false

	for _, op := range supportedOperators {
		if strings.Contains(condition, " "+op+" ") {
			hasOperator = true
			break
		}
	}

	if !hasOperator {
		return fmt.Errorf("no supported operator found in condition")
	}

	// Basic structure validation for AND/OR
	if strings.Contains(condition, " AND ") && strings.Contains(condition, " OR ") {
		return fmt.Errorf("cannot mix AND and OR operators in same condition")
	}

	// Split by logical operators and validate each part
	var parts []string
	switch {
	case strings.Contains(condition, " AND "):
		parts = strings.Split(condition, " AND ")
	case strings.Contains(condition, " OR "):
		parts = strings.Split(condition, " OR ")
	default:
		parts = []string{condition}
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		fields := strings.Fields(part)
		if len(fields) != 3 {
			return fmt.Errorf("invalid condition part: '%s' (expected 'field operator value')", part)
		}
	}

	return nil
}

// InMemoryValidationMetrics implements ValidationMetricsRecorder for testing
type InMemoryValidationMetrics struct {
	successCounts    map[string]int
	failureCounts    map[string]int
	warningCounts    map[string]int
	dailyLimitChecks []DailyLimitCheckRecord
	mu               sync.RWMutex
}

type DailyLimitCheckRecord struct {
	DelegationID string    `json:"delegation_id"`
	Used         float64   `json:"used"`
	Limit        float64   `json:"limit"`
	Exceeded     bool      `json:"exceeded"`
	Timestamp    time.Time `json:"timestamp"`
}

// NewInMemoryValidationMetrics creates a new in-memory metrics recorder
func NewInMemoryValidationMetrics() *InMemoryValidationMetrics {
	return &InMemoryValidationMetrics{
		successCounts:    make(map[string]int),
		failureCounts:    make(map[string]int),
		warningCounts:    make(map[string]int),
		dailyLimitChecks: make([]DailyLimitCheckRecord, 0),
	}
}

// RecordValidationSuccess records a successful validation
func (m *InMemoryValidationMetrics) RecordValidationSuccess(validatorType, scope string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", validatorType, scope)
	m.successCounts[key]++
}

// RecordValidationFailure records a validation failure
func (m *InMemoryValidationMetrics) RecordValidationFailure(validatorType, scope, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", validatorType, scope, reason)
	m.failureCounts[key]++
}

// RecordWarning records a validation warning
func (m *InMemoryValidationMetrics) RecordWarning(category, severity string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", category, severity)
	m.warningCounts[key]++
}

// RecordDailyLimitCheck records a daily limit check event
func (m *InMemoryValidationMetrics) RecordDailyLimitCheck(delegationID string, used, limit float64, exceeded bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.dailyLimitChecks = append(m.dailyLimitChecks, DailyLimitCheckRecord{
		DelegationID: delegationID,
		Used:         used,
		Limit:        limit,
		Exceeded:     exceeded,
		Timestamp:    time.Now(),
	})
}

// GetMetricsSummary returns a summary of recorded metrics
func (m *InMemoryValidationMetrics) GetMetricsSummary() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Calculate total validations by summing all success and failure counts
	totalValidations := 0
	for _, count := range m.successCounts {
		totalValidations += count
	}
	for _, count := range m.failureCounts {
		totalValidations += count
	}

	return map[string]interface{}{
		"success_counts":     m.successCounts,
		"failure_counts":     m.failureCounts,
		"warning_counts":     m.warningCounts,
		"daily_limit_checks": len(m.dailyLimitChecks),
		"total_validations":  totalValidations,
	}
}
