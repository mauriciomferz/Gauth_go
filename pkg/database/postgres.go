package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds the database configuration
type Config struct {
	Host              string
	Port              int
	User              string
	Password          string
	Database          string
	SSLMode           string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// DB wraps the pgxpool connection pool
type DB struct {
	Pool *pgxpool.Pool
	cfg  *Config
}

// NewDB creates a new database connection pool
func NewDB(cfg *Config) (*DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database configuration cannot be nil")
	}

	// Set defaults
	if cfg.MaxConns == 0 {
		cfg.MaxConns = 25
	}
	if cfg.MinConns == 0 {
		cfg.MinConns = 5
	}
	if cfg.MaxConnLifetime == 0 {
		cfg.MaxConnLifetime = time.Hour
	}
	if cfg.MaxConnIdleTime == 0 {
		cfg.MaxConnIdleTime = 30 * time.Minute
	}
	if cfg.HealthCheckPeriod == 0 {
		cfg.HealthCheckPeriod = time.Minute
	}
	if cfg.SSLMode == "" {
		cfg.SSLMode = "prefer"
	}

	// Build connection string
	connString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	// Parse config
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod

	// Configure connection behavior
	poolConfig.ConnConfig.ConnectTimeout = 10 * time.Second
	poolConfig.ConnConfig.RuntimeParams = map[string]string{
		"application_name": "agentauth_admin_portal",
	}

	// Create connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Database connection pool established: %s@%s:%d/%s (max_conns=%d, min_conns=%d)",
		cfg.User, cfg.Host, cfg.Port, cfg.Database, cfg.MaxConns, cfg.MinConns)

	return &DB{
		Pool: pool,
		cfg:  cfg,
	}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
		log.Println("Database connection pool closed")
	}
}

// Ping checks if the database is reachable
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Stats returns connection pool statistics
func (db *DB) Stats() *pgxpool.Stat {
	return db.Pool.Stat()
}

// SetTenantContext sets the current tenant ID for Row-Level Security
func (db *DB) SetTenantContext(ctx context.Context, tenantID string) error {
	conn, err := db.Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	return nil
}

// BeginTx starts a new transaction
func (db *DB) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return db.Pool.Begin(ctx)
}

// BeginTxWithTenant starts a new transaction with tenant context
func (db *DB) BeginTxWithTenant(ctx context.Context, tenantID string) (pgx.Tx, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Set tenant context for this transaction
	_, err = tx.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("failed to set tenant context in transaction: %w", err)
	}

	return tx, nil
}

// HealthCheck performs a comprehensive health check
func (db *DB) HealthCheck(ctx context.Context) error {
	// Check connection
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Check pool stats
	stats := db.Stats()
	if stats.AcquireCount() > 0 && stats.IdleConns() == 0 && stats.TotalConns() >= db.cfg.MaxConns {
		return fmt.Errorf("connection pool exhausted: total=%d, max=%d", stats.TotalConns(), db.cfg.MaxConns)
	}

	// Execute a simple query
	var result int
	err := db.Pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("health check query failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected health check result: %d", result)
	}

	return nil
}

// ConnectionInfo returns formatted connection information
type ConnectionInfo struct {
	Host                 string
	Port                 int
	Database             string
	User                 string
	SSLMode              string
	MaxConns             int32
	MinConns             int32
	TotalConns           int32
	IdleConns            int32
	AcquiredConns        int32
	AcquireCount         int64
	EmptyAcquireCount    int64
	CanceledAcquireCount int64
}

// GetConnectionInfo returns current connection pool information
func (db *DB) GetConnectionInfo() ConnectionInfo {
	stats := db.Stats()
	return ConnectionInfo{
		Host:                 db.cfg.Host,
		Port:                 db.cfg.Port,
		Database:             db.cfg.Database,
		User:                 db.cfg.User,
		SSLMode:              db.cfg.SSLMode,
		MaxConns:             db.cfg.MaxConns,
		MinConns:             db.cfg.MinConns,
		TotalConns:           stats.TotalConns(),
		IdleConns:            stats.IdleConns(),
		AcquiredConns:        stats.AcquiredConns(),
		AcquireCount:         stats.AcquireCount(),
		EmptyAcquireCount:    stats.EmptyAcquireCount(),
		CanceledAcquireCount: stats.CanceledAcquireCount(),
	}
}

// WithTenantTx executes a function within a transaction with tenant context
func (db *DB) WithTenantTx(ctx context.Context, tenantID string, fn func(pgx.Tx) error) error {
	tx, err := db.BeginTxWithTenant(ctx, tenantID)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("transaction error: %w (rollback error: %v)", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
