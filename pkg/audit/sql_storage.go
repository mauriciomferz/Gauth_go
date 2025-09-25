package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

// SQLStorage implements the Storage interface using PostgreSQL
type SQLStorage struct {
	db *sql.DB
}

// SQLConfig holds configuration for SQL storage
type SQLConfig struct {
	// Driver name (e.g., "postgres")
	Driver string

	// DSN for database connection
	DSN string

	// MaxOpenConns sets the maximum number of open connections
	MaxOpenConns int

	// MaxIdleConns sets the maximum number of idle connections
	MaxIdleConns int

	// ConnMaxLifetime sets the maximum amount of time a connection may be reused
	ConnMaxLifetime time.Duration
}

const createTableSQL = `
CREATE TABLE IF NOT EXISTS audit_entries (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    action TEXT NOT NULL,
    result TEXT NOT NULL,
    level TEXT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    chain_id TEXT,
    prev_hash TEXT,
    actor_id TEXT,
    actor_type TEXT,
    actor_name TEXT,
    session_id TEXT,
    client_ip TEXT,
    client_info TEXT,
    target_id TEXT,
    target_type TEXT,
    target_name TEXT,
    target_changes JSONB,
    location TEXT,
    trace_id TEXT,
    tags TEXT[],
    metadata JSONB,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_type ON audit_entries(type);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_entries(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_actor_id ON audit_entries(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_chain_id ON audit_entries(chain_id);
CREATE INDEX IF NOT EXISTS idx_audit_target_id ON audit_entries(target_id);
CREATE INDEX IF NOT EXISTS idx_audit_tags ON audit_entries USING gin(tags);
`

// NewSQLStorage creates a new SQL-backed storage
func NewSQLStorage(config SQLConfig) (*SQLStorage, error) {
	db, err := sql.Open(config.Driver, config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to ping database: %w, and failed to close db: %w", err, closeErr)
		}
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Create table and indices
	if _, err := db.Exec(createTableSQL); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to create table: %w, and failed to close db: %w", err, closeErr)
		}
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &SQLStorage{db: db}, nil
}

// Store implements the Storage interface
func (s *SQLStorage) Store(ctx context.Context, entry *Entry) error {
	query := `
		INSERT INTO audit_entries (
			id, type, action, result, level, timestamp, chain_id, prev_hash,
			actor_id, actor_type, actor_name, session_id, client_ip, client_info,
			target_id, target_type, target_name, target_changes,
			location, trace_id, tags, metadata, error
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21, $22, $23
		)`

	targetChanges, err := json.Marshal(entry.TargetChanges)
	if err != nil {
		return fmt.Errorf("failed to marshal target changes: %w", err)
	}

	metadata, err := json.Marshal(entry.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query,
		entry.ID, entry.Type, entry.Action, entry.Result, entry.Level,
		entry.Timestamp, entry.ChainID, entry.PrevHash,
		entry.ActorID, entry.ActorType, entry.ActorName, entry.SessionID,
		entry.ClientIP, entry.ClientInfo,
		entry.TargetID, entry.TargetType, entry.TargetName, targetChanges,
		entry.Location, entry.TraceID, pq.Array(entry.Tags), metadata,
		entry.Error,
	)

	if err != nil {
		return fmt.Errorf("failed to insert entry: %w", err)
	}

	return nil
}

// Search implements the Storage interface
func (s *SQLStorage) Search(ctx context.Context, filter *Filter) ([]*Entry, error) {
	var conditions []string
	var args []interface{}
	argCount := 1

	if len(filter.Types) > 0 {
		placeholders := make([]string, len(filter.Types))
		for i, t := range filter.Types {
			placeholders[i] = fmt.Sprintf("$%d", argCount)
			args = append(args, t)
			argCount++
		}
		conditions = append(conditions, fmt.Sprintf("type = ANY(ARRAY[%s])", strings.Join(placeholders, ",")))
	}

	if len(filter.ActorIDs) > 0 {
		placeholders := make([]string, len(filter.ActorIDs))
		for i, id := range filter.ActorIDs {
			placeholders[i] = fmt.Sprintf("$%d", argCount)
			args = append(args, id)
			argCount++
		}
		conditions = append(conditions, fmt.Sprintf("actor_id = ANY(ARRAY[%s])", strings.Join(placeholders, ",")))
	}

	if filter.TimeRange != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp BETWEEN $%d AND $%d", argCount, argCount+1))
		args = append(args, filter.TimeRange.Start, filter.TimeRange.End)
		argCount += 2
	}

	if filter.ChainID != "" {
		conditions = append(conditions, fmt.Sprintf("chain_id = $%d", argCount))
		args = append(args, filter.ChainID)
		argCount++
	}

	query := "SELECT * FROM audit_entries"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}
	defer rows.Close()

	var entries []*Entry
	for rows.Next() {
		var entry Entry
		var targetChanges, metadata []byte
		var tags []string

		err := rows.Scan(
			&entry.ID, &entry.Type, &entry.Action, &entry.Result, &entry.Level,
			&entry.Timestamp, &entry.ChainID, &entry.PrevHash,
			&entry.ActorID, &entry.ActorType, &entry.ActorName, &entry.SessionID,
			&entry.ClientIP, &entry.ClientInfo,
			&entry.TargetID, &entry.TargetType, &entry.TargetName, &targetChanges,
			&entry.Location, &entry.TraceID, pq.Array(&tags), &metadata,
			&entry.Error,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}

		if err := json.Unmarshal(targetChanges, &entry.TargetChanges); err != nil {
			return nil, fmt.Errorf("failed to unmarshal target changes: %w", err)
		}

		if err := json.Unmarshal(metadata, &entry.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		entry.Tags = tags
		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return entries, nil
}

// GetByID implements the Storage interface
func (s *SQLStorage) GetByID(ctx context.Context, id string) (*Entry, error) {
	entries, err := s.Search(ctx, &Filter{
		Metadata: []MetadataFilter{{
			Key:      "id",
			Value:    id,
			Operator: "eq",
		}},
		Limit: 1,
	})
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("entry not found")
	}
	return entries[0], nil
}

// GetChain implements the Storage interface
func (s *SQLStorage) GetChain(ctx context.Context, chainID string) ([]*Entry, error) {
	return s.Search(ctx, &Filter{
		ChainID: chainID,
	})
}

// Cleanup implements the Storage interface
func (s *SQLStorage) Cleanup(ctx context.Context, before time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM audit_entries WHERE timestamp < $1",
		before,
	)
	if err != nil {
		return fmt.Errorf("failed to cleanup entries: %w", err)
	}
	return nil
}

// Close implements io.Closer
func (s *SQLStorage) Close() error {
	return s.db.Close()
}
