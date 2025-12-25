// Package database provides query optimization and analysis tools
package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QueryAnalyzer analyzes query performance and provides optimization recommendations
type QueryAnalyzer struct {
	pool               *pgxpool.Pool
	slowQueryThreshold time.Duration
}

// QueryPlan represents an analyzed query execution plan
type QueryPlan struct {
	Query           string        `json:"query"`
	ExecutionTime   time.Duration `json:"execution_time"`
	PlanType        string        `json:"plan_type"`
	EstimatedRows   int64         `json:"estimated_rows"`
	ActualRows      int64         `json:"actual_rows"`
	IndexesUsed     []string      `json:"indexes_used"`
	Warnings        []string      `json:"warnings"`
	Recommendations []string      `json:"recommendations"`
	PlanJSON        string        `json:"plan_json,omitempty"`
}

// ExplainPlan represents the structure of EXPLAIN output
type ExplainPlan struct {
	Plan struct {
		NodeType            string  `json:"Node Type"`
		RelationName        string  `json:"Relation Name"`
		Strategy            string  `json:"Strategy"`
		TotalCost           float64 `json:"Total Cost"`
		PlanRows            int64   `json:"Plan Rows"`
		PlanWidth           int     `json:"Plan Width"`
		ActualTotalTime     float64 `json:"Actual Total Time"`
		ActualRows          int64   `json:"Actual Rows"`
		IndexName           string  `json:"Index Name"`
		SharedHitBlocks     int     `json:"Shared Hit Blocks"`
		SharedReadBlocks    int     `json:"Shared Read Blocks"`
		SharedDirtiedBlocks int     `json:"Shared Dirtied Blocks"`
		Plans               []struct {
			NodeType     string `json:"Node Type"`
			RelationName string `json:"Relation Name"`
			IndexName    string `json:"Index Name"`
		} `json:"Plans"`
	} `json:"Plan"`
	ExecutionTime float64 `json:"Execution Time"`
}

// NewQueryAnalyzer creates a new query analyzer
func NewQueryAnalyzer(pool *pgxpool.Pool) *QueryAnalyzer {
	return &QueryAnalyzer{
		pool:               pool,
		slowQueryThreshold: 100 * time.Millisecond,
	}
}

// AnalyzeQuery executes EXPLAIN ANALYZE and returns optimization recommendations
func (qa *QueryAnalyzer) AnalyzeQuery(ctx context.Context, query string, args ...interface{}) (*QueryPlan, error) {
	start := time.Now()

	// Execute EXPLAIN ANALYZE
	explainQuery := fmt.Sprintf("EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) %s", query)
	rows, err := qa.pool.Query(ctx, explainQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("explain query failed: %w", err)
	}
	defer rows.Close()

	var explainOutput []byte
	if rows.Next() {
		if err := rows.Scan(&explainOutput); err != nil {
			return nil, fmt.Errorf("scan explain output failed: %w", err)
		}
	}

	executionTime := time.Since(start)

	// Parse EXPLAIN output
	var explainPlans []ExplainPlan
	if err := json.Unmarshal(explainOutput, &explainPlans); err != nil {
		return nil, fmt.Errorf("parse explain output failed: %w", err)
	}

	if len(explainPlans) == 0 {
		return nil, fmt.Errorf("no explain plan returned")
	}

	plan := &QueryPlan{
		Query:           query,
		ExecutionTime:   executionTime,
		PlanType:        explainPlans[0].Plan.NodeType,
		EstimatedRows:   explainPlans[0].Plan.PlanRows,
		ActualRows:      explainPlans[0].Plan.ActualRows,
		IndexesUsed:     []string{},
		Warnings:        []string{},
		Recommendations: []string{},
		PlanJSON:        string(explainOutput),
	}

	// Extract indexes used
	if explainPlans[0].Plan.IndexName != "" {
		plan.IndexesUsed = append(plan.IndexesUsed, explainPlans[0].Plan.IndexName)
	}

	// Analyze plan and generate recommendations
	qa.analyzePlan(&explainPlans[0], plan)

	return plan, nil
}

// analyzePlan analyzes the explain plan and generates recommendations
func (qa *QueryAnalyzer) analyzePlan(explainPlan *ExplainPlan, plan *QueryPlan) {
	// Check for sequential scans
	if strings.Contains(explainPlan.Plan.NodeType, "Seq Scan") {
		plan.Warnings = append(plan.Warnings, "Sequential scan detected - may benefit from an index")
		plan.Recommendations = append(plan.Recommendations,
			fmt.Sprintf("Consider adding an index on table '%s'", explainPlan.Plan.RelationName))
	}

	// Check for high estimated vs actual rows (outdated statistics)
	if plan.EstimatedRows > 0 && plan.ActualRows > 0 {
		ratio := float64(plan.EstimatedRows) / float64(plan.ActualRows)
		if ratio > 10 || ratio < 0.1 {
			plan.Warnings = append(plan.Warnings,
				fmt.Sprintf("Statistics may be outdated (estimated: %d, actual: %d)",
					plan.EstimatedRows, plan.ActualRows))
			plan.Recommendations = append(plan.Recommendations,
				"Run ANALYZE on affected tables to update statistics")
		}
	}

	// Check for nested loops with large datasets
	if strings.Contains(explainPlan.Plan.NodeType, "Nested Loop") && plan.ActualRows > 1000 {
		plan.Warnings = append(plan.Warnings,
			"Nested loop on large dataset detected")
		plan.Recommendations = append(plan.Recommendations,
			"Consider optimizing JOIN strategy or adding indexes")
	}

	// Check for high buffer reads (disk I/O)
	if explainPlan.Plan.SharedReadBlocks > 1000 {
		plan.Warnings = append(plan.Warnings,
			fmt.Sprintf("High disk I/O detected (%d blocks read)",
				explainPlan.Plan.SharedReadBlocks))
		plan.Recommendations = append(plan.Recommendations,
			"Consider increasing shared_buffers or adding indexes to reduce disk reads")
	}

	// Check execution time
	if plan.ExecutionTime > qa.slowQueryThreshold {
		plan.Warnings = append(plan.Warnings,
			fmt.Sprintf("Slow query detected (%v > %v)",
				plan.ExecutionTime, qa.slowQueryThreshold))
	}

	// Check for missing index on WHERE clause
	queryLower := strings.ToLower(plan.Query)
	if strings.Contains(queryLower, "where") &&
		strings.Contains(explainPlan.Plan.NodeType, "Seq Scan") {
		plan.Recommendations = append(plan.Recommendations,
			"Query contains WHERE clause but uses sequential scan - add index on filter columns")
	}

	// Check for cartesian product (missing JOIN condition)
	if strings.Contains(queryLower, "join") &&
		explainPlan.Plan.ActualRows > plan.EstimatedRows*10 {
		plan.Warnings = append(plan.Warnings,
			"Possible cartesian product detected - check JOIN conditions")
		plan.Recommendations = append(plan.Recommendations,
			"Verify all JOIN clauses have proper ON conditions")
	}
}

// GetSlowQueries retrieves slow queries from pg_stat_statements
func (qa *QueryAnalyzer) GetSlowQueries(ctx context.Context, limit int) ([]SlowQuery, error) {
	query := `
		SELECT 
			query,
			calls,
			total_exec_time,
			mean_exec_time,
			max_exec_time,
			stddev_exec_time,
			rows
		FROM pg_stat_statements
		WHERE mean_exec_time > $1
		ORDER BY mean_exec_time DESC
		LIMIT $2
	`

	rows, err := qa.pool.Query(ctx, query, qa.slowQueryThreshold.Milliseconds(), limit)
	if err != nil {
		return nil, fmt.Errorf("query slow queries failed: %w", err)
	}
	defer rows.Close()

	var slowQueries []SlowQuery
	for rows.Next() {
		var sq SlowQuery
		if err := rows.Scan(
			&sq.Query,
			&sq.Calls,
			&sq.TotalExecTime,
			&sq.MeanExecTime,
			&sq.MaxExecTime,
			&sq.StddevExecTime,
			&sq.Rows,
		); err != nil {
			return nil, fmt.Errorf("scan slow query failed: %w", err)
		}
		slowQueries = append(slowQueries, sq)
	}

	return slowQueries, nil
}

// SlowQuery represents a slow query from pg_stat_statements
type SlowQuery struct {
	Query          string  `json:"query"`
	Calls          int64   `json:"calls"`
	TotalExecTime  float64 `json:"total_exec_time_ms"`
	MeanExecTime   float64 `json:"mean_exec_time_ms"`
	MaxExecTime    float64 `json:"max_exec_time_ms"`
	StddevExecTime float64 `json:"stddev_exec_time_ms"`
	Rows           int64   `json:"rows"`
}

// BatchInserter provides efficient batch insert operations
type BatchInserter struct {
	pool      *pgxpool.Pool
	batchSize int
}

// NewBatchInserter creates a new batch inserter
func NewBatchInserter(pool *pgxpool.Pool) *BatchInserter {
	return &BatchInserter{
		pool:      pool,
		batchSize: 1000,
	}
}

// BatchInsertPoAs inserts multiple PoAs in a single batch operation
func (bi *BatchInserter) BatchInsertPoAs(ctx context.Context, poas []interface{}) error {
	if len(poas) == 0 {
		return nil
	}

	batch := &pgx.Batch{}

	for range poas {
		// Type assert to your PoA struct
		// This is a placeholder - adjust based on your actual PoA struct
		batch.Queue(
			`INSERT INTO poas (id, user_id, external_id, status, poa_type, metadata, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT (id) DO NOTHING`,
			nil, nil, nil, nil, nil, nil, time.Now(), // Replace with actual poa fields
		)
	}

	results := bi.pool.SendBatch(ctx, batch)
	defer results.Close()

	for i := 0; i < len(poas); i++ {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("batch insert failed at index %d: %w", i, err)
		}
	}

	return nil
}

// OptimizedPoAQueries provides optimized PoA query methods
type OptimizedPoAQueries struct {
	pool *pgxpool.Pool
}

// NewOptimizedPoAQueries creates optimized PoA query handler
func NewOptimizedPoAQueries(pool *pgxpool.Pool) *OptimizedPoAQueries {
	return &OptimizedPoAQueries{pool: pool}
}

// ListPoAsWithUsers retrieves PoAs with user details in a single query (eliminates N+1)
func (opq *OptimizedPoAQueries) ListPoAsWithUsers(ctx context.Context, userID string, limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			p.id as poa_id,
			p.external_id,
			p.status,
			p.poa_type,
			p.metadata,
			p.created_at as poa_created_at,
			u.id as user_id,
			u.name as user_name,
			u.email as user_email,
			u.role as user_role
		FROM poas p
		INNER JOIN users u ON p.user_id = u.id
		WHERE p.user_id = $1 
		  AND p.deleted_at IS NULL
		ORDER BY p.created_at DESC
		LIMIT $2
	`

	rows, err := opq.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list poas with users failed: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		result := make(map[string]interface{})
		var (
			poaID        string
			externalID   *string
			status       string
			poaType      string
			metadata     []byte
			poaCreatedAt time.Time
			userID       string
			userName     string
			userEmail    string
			userRole     string
		)

		if err := rows.Scan(
			&poaID, &externalID, &status, &poaType, &metadata, &poaCreatedAt,
			&userID, &userName, &userEmail, &userRole,
		); err != nil {
			return nil, fmt.Errorf("scan row failed: %w", err)
		}

		result["poa"] = map[string]interface{}{
			"id":          poaID,
			"external_id": externalID,
			"status":      status,
			"type":        poaType,
			"metadata":    metadata,
			"created_at":  poaCreatedAt,
		}

		result["user"] = map[string]interface{}{
			"id":    userID,
			"name":  userName,
			"email": userEmail,
			"role":  userRole,
		}

		results = append(results, result)
	}

	return results, nil
}

// GetPoACountsByStatus retrieves PoA counts grouped by status (optimized with GROUP BY)
func (opq *OptimizedPoAQueries) GetPoACountsByStatus(ctx context.Context, userID string) (map[string]int64, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM poas
		WHERE user_id = $1 AND deleted_at IS NULL
		GROUP BY status
	`

	rows, err := opq.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get poa counts by status failed: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan row failed: %w", err)
		}
		counts[status] = count
	}

	return counts, nil
}

// IndexUsageStats represents index usage statistics
type IndexUsageStats struct {
	SchemaName string `json:"schema_name"`
	TableName  string `json:"table_name"`
	IndexName  string `json:"index_name"`
	IndexScans int64  `json:"index_scans"`
	TuplesRead int64  `json:"tuples_read"`
	IndexSize  string `json:"index_size"`
}

// GetIndexUsageStats retrieves index usage statistics
func (qa *QueryAnalyzer) GetIndexUsageStats(ctx context.Context) ([]IndexUsageStats, error) {
	query := `
		SELECT 
			schemaname,
			tablename,
			indexname,
			idx_scan,
			idx_tup_read,
			pg_size_pretty(pg_relation_size(indexrelid)) as index_size
		FROM pg_stat_user_indexes
		ORDER BY idx_scan ASC
		LIMIT 50
	`

	rows, err := qa.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get index usage stats failed: %w", err)
	}
	defer rows.Close()

	var stats []IndexUsageStats
	for rows.Next() {
		var s IndexUsageStats
		if err := rows.Scan(
			&s.SchemaName,
			&s.TableName,
			&s.IndexName,
			&s.IndexScans,
			&s.TuplesRead,
			&s.IndexSize,
		); err != nil {
			return nil, fmt.Errorf("scan index stats failed: %w", err)
		}
		stats = append(stats, s)
	}

	return stats, nil
}
