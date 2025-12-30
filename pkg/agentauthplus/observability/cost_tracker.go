package observability

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ModelCostStats tracks usage and cost for a specific model on a specific day.
type ModelCostStats struct {
	InputTokens   int     `json:"input_tokens"`
	OutputTokens  int     `json:"output_tokens"`
	TotalTokens   int     `json:"total_tokens"`
	EstimatedCost float64 `json:"estimated_cost"`
}

// CostTracker persists daily AI model usage metrics to disk.
type CostTracker struct {
	mu       sync.RWMutex
	filePath string
	// dailyStats key format: "model_id:YYYY-MM-DD"
	dailyStats map[string]*ModelCostStats
	// pricing map: model_id -> {input_cost_per_1k, output_cost_per_1k}
	pricing map[string]ModelPricing
}

type ModelPricing struct {
	InputPer1k  float64
	OutputPer1k float64
}

// NewCostTracker creates a new tracker backed by the specified file.
func NewCostTracker(filePath string) *CostTracker {
	ct := &CostTracker{
		filePath:   filePath,
		dailyStats: make(map[string]*ModelCostStats),
		pricing: map[string]ModelPricing{
			// Example defaults (can be updated or loaded from config)
			"gpt-4":         {0.03, 0.06},
			"gpt-3.5-turbo": {0.0015, 0.002},
			"claude-v1":     {0.011, 0.032}, // fictitious
		},
	}
	return ct
}

// Load reads the persistent state from disk.
func (ct *CostTracker) Load() error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	data, err := os.ReadFile(ct.filePath)
	if os.IsNotExist(err) {
		return nil // New file
	}
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &ct.dailyStats)
}

// Save writes the current state to disk.
func (ct *CostTracker) Save() error {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	data, err := json.MarshalIndent(ct.dailyStats, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(ct.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(ct.filePath, data, 0644)
}

// TrackUsage records token usage for a model and updates estimated cost.
func (ct *CostTracker) TrackUsage(model string, input, output int) {
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("%s:%s", model, today)

	ct.mu.Lock()
	defer ct.mu.Unlock()

	stats, exists := ct.dailyStats[key]
	if !exists {
		stats = &ModelCostStats{}
		ct.dailyStats[key] = stats
	}

	stats.InputTokens += input
	stats.OutputTokens += output
	stats.TotalTokens += (input + output)

	// Calculate Cost
	if price, ok := ct.pricing[model]; ok {
		cost := (float64(input)/1000)*price.InputPer1k + (float64(output)/1000)*price.OutputPer1k
		stats.EstimatedCost += cost
	} else {
		// Default fallback or metrics only
	}
}

// GetDailyCost returns the estimated cost for a model for the current day.
func (ct *CostTracker) GetDailyCost(model string) float64 {
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("%s:%s", model, today)

	ct.mu.RLock()
	defer ct.mu.RUnlock()

	if stats, ok := ct.dailyStats[key]; ok {
		return stats.EstimatedCost
	}
	return 0.0
}

// GetDailyTokens returns total tokens for a model for the current day.
func (ct *CostTracker) GetDailyTokens(model string) int {
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("%s:%s", model, today)

	ct.mu.RLock()
	defer ct.mu.RUnlock()

	if stats, ok := ct.dailyStats[key]; ok {
		return stats.TotalTokens
	}
	return 0
}
