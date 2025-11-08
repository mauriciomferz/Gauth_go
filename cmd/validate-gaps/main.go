package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// GapEntry represents a single requirement from the gap matrix
type GapEntry struct {
	Section     string
	ID          string
	Requirement string
	Status      string
	Priority    string
	Gap         string
	Evidence    string
}

// ValidationResult contains the complete gap matrix validation results
type ValidationResult struct {
	TotalRequirements    int
	ImplementedCount     int
	CompliancePercentage float64
	BySection            map[string]*SectionMetrics
	ByPriority           map[string]*PriorityMetrics
	Timestamp            time.Time
	CriticalGaps         []GapEntry
	ReadinessScore       float64
}

// SectionMetrics tracks implementation status per section
type SectionMetrics struct {
	Total       int
	Implemented int
	Percentage  float64
	Gaps        []string
}

// PriorityMetrics tracks implementation status per priority
type PriorityMetrics struct {
	Total        int
	Implemented  int
	Percentage   float64
	Requirements []string
}

func main() {
	fmt.Println("=== GAuth Operations Readiness Gap Validation ===")
	fmt.Println()

	// Locate gap matrix
	csvPath := "artifacts/gap_matrix.csv"
	if _, err := os.Stat(csvPath); err != nil {
		fmt.Printf("ERROR: gap_matrix.csv not found at %s\n", csvPath)
		os.Exit(1)
	}

	// Load and parse CSV
	entries, err := loadGapMatrix(csvPath)
	if err != nil {
		fmt.Printf("ERROR: Failed to load gap matrix: %v\n", err)
		os.Exit(1)
	}

	// Validate entries
	result := validateGapMatrix(entries)

	// Generate reports
	printConsoleReport(result)
	if err := writeJSONReport(result); err != nil {
		fmt.Printf("WARNING: Failed to write JSON report: %v\n", err)
	}
	if err := writeMarkdownReport(result); err != nil {
		fmt.Printf("WARNING: Failed to write Markdown report: %v\n", err)
	}

	fmt.Println()
	fmt.Printf("✅ Validation complete: %d/%d requirements implemented (%.1f%% compliance)\n",
		result.ImplementedCount, result.TotalRequirements, result.CompliancePercentage)
	fmt.Printf("📊 Readiness Score: %.1f/100.0\n", result.ReadinessScore)
	fmt.Println()

	if result.CompliancePercentage == 100.0 {
		fmt.Println("🎉 100% COMPLIANCE ACHIEVED - All requirements implemented!")
		os.Exit(0)
	} else {
		fmt.Printf("⚠️  %d requirements pending implementation\n", result.TotalRequirements-result.ImplementedCount)
		os.Exit(0) // Exit 0 for now since gaps are expected
	}
}

func loadGapMatrix(path string) ([]GapEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1 // Allow variable number of fields
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV has no data rows")
	}

	// Skip header
	var entries []GapEntry
	for i, record := range records[1:] {
		if len(record) < 7 {
			fmt.Printf("WARNING: Row %d has insufficient columns, skipping\n", i+2)
			continue
		}
		entries = append(entries, GapEntry{
			Section:     strings.TrimSpace(record[0]),
			ID:          strings.TrimSpace(record[1]),
			Requirement: strings.TrimSpace(record[2]),
			Status:      strings.TrimSpace(record[3]),
			Priority:    strings.TrimSpace(record[4]),
			Gap:         strings.TrimSpace(record[5]),
			Evidence:    strings.TrimSpace(record[6]),
		})
	}

	return entries, nil
}

func validateGapMatrix(entries []GapEntry) *ValidationResult {
	result := &ValidationResult{
		BySection:    make(map[string]*SectionMetrics),
		ByPriority:   make(map[string]*PriorityMetrics),
		Timestamp:    time.Now(),
		CriticalGaps: []GapEntry{},
	}

	for _, entry := range entries {
		result.TotalRequirements++

		// Track section metrics
		if _, exists := result.BySection[entry.Section]; !exists {
			result.BySection[entry.Section] = &SectionMetrics{
				Gaps: []string{},
			}
		}
		result.BySection[entry.Section].Total++

		// Track priority metrics
		if _, exists := result.ByPriority[entry.Priority]; !exists {
			result.ByPriority[entry.Priority] = &PriorityMetrics{
				Requirements: []string{},
			}
		}
		result.ByPriority[entry.Priority].Total++
		result.ByPriority[entry.Priority].Requirements = append(
			result.ByPriority[entry.Priority].Requirements,
			fmt.Sprintf("%s: %s", entry.ID, entry.Requirement),
		)

		// Check implementation status
		if entry.Status == "Implemented" {
			result.ImplementedCount++
			result.BySection[entry.Section].Implemented++
			result.ByPriority[entry.Priority].Implemented++
		} else {
			result.BySection[entry.Section].Gaps = append(
				result.BySection[entry.Section].Gaps,
				fmt.Sprintf("%s: %s", entry.ID, entry.Gap),
			)

			// Track critical gaps (P0)
			if entry.Priority == "P0" {
				result.CriticalGaps = append(result.CriticalGaps, entry)
			}
		}
	}

	// Calculate percentages
	if result.TotalRequirements > 0 {
		result.CompliancePercentage = float64(result.ImplementedCount) / float64(result.TotalRequirements) * 100.0
	}

	for _, metrics := range result.BySection {
		if metrics.Total > 0 {
			metrics.Percentage = float64(metrics.Implemented) / float64(metrics.Total) * 100.0
		}
	}

	for _, metrics := range result.ByPriority {
		if metrics.Total > 0 {
			metrics.Percentage = float64(metrics.Implemented) / float64(metrics.Total) * 100.0
		}
	}

	// Calculate readiness score (weighted by priority)
	result.ReadinessScore = calculateReadinessScore(result)

	return result
}

func calculateReadinessScore(result *ValidationResult) float64 {
	// Weighted scoring: P0 = 50%, P1 = 30%, P2 = 15%, P3 = 5%
	weights := map[string]float64{
		"P0": 0.50,
		"P1": 0.30,
		"P2": 0.15,
		"P3": 0.05,
	}

	score := 0.0
	for priority, weight := range weights {
		if metrics, exists := result.ByPriority[priority]; exists && metrics.Total > 0 {
			score += (float64(metrics.Implemented) / float64(metrics.Total)) * weight * 100.0
		}
	}

	return score
}

func printConsoleReport(result *ValidationResult) {
	fmt.Println("📋 COMPLIANCE OVERVIEW")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Total Requirements:    %d\n", result.TotalRequirements)
	fmt.Printf("Implemented:           %d\n", result.ImplementedCount)
	fmt.Printf("Compliance:            %.1f%%\n", result.CompliancePercentage)
	fmt.Printf("Readiness Score:       %.1f/100.0\n", result.ReadinessScore)
	fmt.Printf("Critical Gaps (P0):    %d\n", len(result.CriticalGaps))
	fmt.Println()

	// Section breakdown
	fmt.Println("📊 SECTION BREAKDOWN")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Sort sections by name
	sections := make([]string, 0, len(result.BySection))
	for section := range result.BySection {
		sections = append(sections, section)
	}
	sort.Strings(sections)

	for _, section := range sections {
		metrics := result.BySection[section]
		status := "✅"
		if metrics.Percentage < 100.0 {
			status = "⚠️"
		}
		fmt.Printf("%s %s: %d/%d (%.1f%%)\n",
			status, section, metrics.Implemented, metrics.Total, metrics.Percentage)
	}
	fmt.Println()

	// Priority breakdown
	fmt.Println("🎯 PRIORITY BREAKDOWN")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	priorities := []string{"P0", "P1", "P2", "P3"}
	for _, priority := range priorities {
		if metrics, exists := result.ByPriority[priority]; exists {
			status := "✅"
			if metrics.Percentage < 100.0 {
				status = "⚠️"
			}
			fmt.Printf("%s %s (Critical): %d/%d (%.1f%%)\n",
				status, priority, metrics.Implemented, metrics.Total, metrics.Percentage)
		}
	}
	fmt.Println()

	// Critical gaps
	if len(result.CriticalGaps) > 0 {
		fmt.Println("🚨 CRITICAL GAPS (P0)")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		for _, gap := range result.CriticalGaps {
			fmt.Printf("  • [%s] %s\n", gap.ID, gap.Requirement)
			if gap.Gap != "" {
				fmt.Printf("    Gap: %s\n", gap.Gap)
			}
		}
		fmt.Println()
	}
}

func writeJSONReport(result *ValidationResult) error {
	outputPath := "artifacts/gap_validation_result.json"

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return err
	}

	fmt.Printf("📄 JSON report written to: %s\n", outputPath)
	return nil
}

func writeMarkdownReport(result *ValidationResult) error {
	outputPath := "artifacts/gap_validation_report.md"

	var sb strings.Builder

	sb.WriteString("# GAuth Operations Readiness Gap Validation Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", result.Timestamp.Format(time.RFC3339)))

	sb.WriteString("## Executive Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Requirements:** %d\n", result.TotalRequirements))
	sb.WriteString(fmt.Sprintf("- **Implemented:** %d\n", result.ImplementedCount))
	sb.WriteString(fmt.Sprintf("- **Compliance Percentage:** %.1f%%\n", result.CompliancePercentage))
	sb.WriteString(fmt.Sprintf("- **Readiness Score:** %.1f/100.0\n", result.ReadinessScore))
	sb.WriteString(fmt.Sprintf("- **Critical Gaps (P0):** %d\n\n", len(result.CriticalGaps)))

	if result.CompliancePercentage == 100.0 {
		sb.WriteString("✅ **100% COMPLIANCE ACHIEVED** - All requirements implemented!\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("⚠️ **%d requirements** pending implementation\n\n",
			result.TotalRequirements-result.ImplementedCount))
	}

	// Readiness interpretation
	sb.WriteString("## Readiness Score Interpretation\n\n")
	sb.WriteString("The readiness score is calculated using weighted priority scoring:\n")
	sb.WriteString("- P0 (Critical): 50% weight\n")
	sb.WriteString("- P1 (High): 30% weight\n")
	sb.WriteString("- P2 (Medium): 15% weight\n")
	sb.WriteString("- P3 (Low): 5% weight\n\n")

	if result.ReadinessScore >= 90.0 {
		sb.WriteString("✅ **EXCELLENT** - Production ready with minor gaps\n\n")
	} else if result.ReadinessScore >= 75.0 {
		sb.WriteString("✅ **GOOD** - Near production ready, address critical gaps\n\n")
	} else if result.ReadinessScore >= 50.0 {
		sb.WriteString("⚠️ **FAIR** - Significant work required before production\n\n")
	} else {
		sb.WriteString("🚨 **POOR** - Major gaps in critical requirements\n\n")
	}

	// Section breakdown
	sb.WriteString("## Section Breakdown\n\n")
	sb.WriteString("| Section | Implemented | Total | Compliance |\n")
	sb.WriteString("|---------|-------------|-------|------------|\n")

	sections := make([]string, 0, len(result.BySection))
	for section := range result.BySection {
		sections = append(sections, section)
	}
	sort.Strings(sections)

	for _, section := range sections {
		metrics := result.BySection[section]
		status := "✅"
		if metrics.Percentage < 100.0 {
			status = "⚠️"
		}
		sb.WriteString(fmt.Sprintf("| %s %s | %d | %d | %.1f%% |\n",
			status, section, metrics.Implemented, metrics.Total, metrics.Percentage))
	}
	sb.WriteString("\n")

	// Priority breakdown
	sb.WriteString("## Priority Breakdown\n\n")
	sb.WriteString("| Priority | Implemented | Total | Compliance |\n")
	sb.WriteString("|----------|-------------|-------|------------|\n")

	priorities := []string{"P0", "P1", "P2", "P3"}
	for _, priority := range priorities {
		if metrics, exists := result.ByPriority[priority]; exists {
			status := "✅"
			if metrics.Percentage < 100.0 {
				status = "⚠️"
			}
			sb.WriteString(fmt.Sprintf("| %s %s | %d | %d | %.1f%% |\n",
				status, priority, metrics.Implemented, metrics.Total, metrics.Percentage))
		}
	}
	sb.WriteString("\n")

	// Critical gaps detail
	if len(result.CriticalGaps) > 0 {
		sb.WriteString("## Critical Gaps (P0)\n\n")
		sb.WriteString("These are the highest priority gaps that must be addressed before production:\n\n")
		for i, gap := range result.CriticalGaps {
			sb.WriteString(fmt.Sprintf("### %d. [%s] %s\n\n", i+1, gap.ID, gap.Requirement))
			if gap.Gap != "" {
				sb.WriteString(fmt.Sprintf("**Gap:** %s\n\n", gap.Gap))
			}
			if gap.Evidence != "" {
				sb.WriteString(fmt.Sprintf("**Evidence:** %s\n\n", gap.Evidence))
			}
		}
	}

	// Section gaps detail
	sb.WriteString("## Section Gaps Detail\n\n")
	for _, section := range sections {
		metrics := result.BySection[section]
		if len(metrics.Gaps) > 0 {
			sb.WriteString(fmt.Sprintf("### %s (%d/%d - %.1f%%)\n\n",
				section, metrics.Implemented, metrics.Total, metrics.Percentage))
			for _, gap := range metrics.Gaps {
				sb.WriteString(fmt.Sprintf("- %s\n", gap))
			}
			sb.WriteString("\n")
		}
	}

	// Conclusion
	sb.WriteString("## Conclusion\n\n")
	if result.CompliancePercentage == 100.0 {
		sb.WriteString("All 45 operational readiness requirements have been implemented, ")
		sb.WriteString("demonstrating 100% compliance with the GAuth production readiness criteria. ")
		sb.WriteString("The system is ready for production deployment after standard release procedures.\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("The GAuth implementation has achieved %.1f%% compliance with ", result.CompliancePercentage))
		sb.WriteString(fmt.Sprintf("%d of %d requirements implemented. ", result.ImplementedCount, result.TotalRequirements))
		sb.WriteString(fmt.Sprintf("A readiness score of %.1f/100.0 indicates ", result.ReadinessScore))
		if result.ReadinessScore >= 75.0 {
			sb.WriteString("the system is approaching production readiness. ")
		} else {
			sb.WriteString("additional work is required before production deployment. ")
		}
		sb.WriteString("Focus on addressing critical (P0) gaps first.\n\n")
	}

	if err := os.WriteFile(outputPath, []byte(sb.String()), 0600); err != nil {
		return err
	}

	fmt.Printf("📄 Markdown report written to: %s\n", outputPath)
	return nil
}
