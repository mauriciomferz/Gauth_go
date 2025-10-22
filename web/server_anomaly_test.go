package web

import (
	"testing"
)

// TestSemanticAnomalySpike validates that updateSemanticAnomalies produces a positive score after a rate spike.
func TestSemanticAnomalySpike(t *testing.T) {
	s := &BetaServer{}
	s.initSemanticAnomaly()
	// Simulate stable low rates then a spike for amount_limit_exceeded.
	rates := []map[string]float64{
		{"amount_limit_exceeded": 1},
		{"amount_limit_exceeded": 1},
		{"amount_limit_exceeded": 1},
		{"amount_limit_exceeded": 1},
		{"amount_limit_exceeded": 1},
		{"amount_limit_exceeded": 10}, // spike
	}
	for _, r := range rates {
		s.updateSemanticAnomalies(r)
	}
	score := s.currentSemanticScores()["amount_limit_exceeded"]
	if score <= 0 {
		// With variance > 0 and spike, expect positive z-score.
		// Allow small >0 threshold to avoid floating precision issues if implementation changes.
		// If variance still zero (e.g., logic regression) produce detailed state dump.
		if ewma, ok := s.semanticEWMA["amount_limit_exceeded"]; ok {
			t.Fatalf("expected positive anomaly score after spike; got %f (mean=%f count=%d M2=%f)", score, ewma.Mean, ewma.Count, ewma.M2)
		}
		f := s.semanticEWMA["amount_limit_exceeded"]
		_ = f // ensure referenced
		if score == 0 {
			// Provide hint for potential update logic regression.
			t.Fatalf("anomaly score remained zero after spike; check variance calculation and sample count")
		}
	}
}
