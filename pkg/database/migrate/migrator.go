package migrate

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrator handles database migrations
type Migrator struct {
	db      *sql.DB
	migrate *migrate.Migrate
}

// NewMigrator creates a new migrator instance
func NewMigrator(databaseURL string) (*Migrator, error) {
	// Open database connection
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create driver instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create postgres driver: %w", err)
	}

	// Create source from embedded filesystem
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create source driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return &Migrator{
		db:      db,
		migrate: m,
	}, nil
}

// Close closes the migrator and database connection
func (m *Migrator) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// Up runs all available migrations
func (m *Migrator) Up() error {
	log.Println("Running migrations up...")
	if err := m.migrate.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("No migrations to run")
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Println("Migrations completed successfully")
	return nil
}

// Down rolls back all migrations
func (m *Migrator) Down() error {
	log.Println("Rolling back all migrations...")
	if err := m.migrate.Down(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("No migrations to roll back")
			return nil
		}
		return fmt.Errorf("failed to roll back migrations: %w", err)
	}
	log.Println("Rollback completed successfully")
	return nil
}

// Steps runs n migration steps
// n > 0 applies n up migrations
// n < 0 applies n down migrations
func (m *Migrator) Steps(n int) error {
	log.Printf("Running %d migration steps...\n", n)
	if err := m.migrate.Steps(n); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("No migrations to run")
			return nil
		}
		return fmt.Errorf("failed to run migration steps: %w", err)
	}
	log.Println("Migration steps completed successfully")
	return nil
}

// Force sets the migration version without running migrations
// Use with caution - typically for fixing dirty database state
func (m *Migrator) Force(version int) error {
	log.Printf("Forcing migration version to %d...\n", version)
	if err := m.migrate.Force(version); err != nil {
		return fmt.Errorf("failed to force migration version: %w", err)
	}
	log.Println("Migration version forced successfully")
	return nil
}

// Version returns the current migration version
func (m *Migrator) Version() (uint, bool, error) {
	version, dirty, err := m.migrate.Version()
	if err != nil {
		if err == migrate.ErrNilVersion {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}
	return version, dirty, nil
}

// Status returns the current migration status
func (m *Migrator) Status() (string, error) {
	version, dirty, err := m.Version()
	if err != nil {
		return "", err
	}
	if version == 0 {
		return "No migrations applied", nil
	}
	status := fmt.Sprintf("Current version: %d", version)
	if dirty {
		status += " (dirty - migration failed, use Force to fix)"
	}
	return status, nil
}

// ============================================================================
// SEED DATA FUNCTIONS
// ============================================================================

// SeedData populates the database with initial/demo data
func (m *Migrator) SeedData(ctx context.Context) error {
	log.Println("Seeding initial data...")

	// Start transaction
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // Ignore rollback error; will be committed on success

	// Seed subscribers
	if err := m.seedSubscribers(ctx, tx); err != nil {
		return fmt.Errorf("failed to seed subscribers: %w", err)
	}

	// Seed event types
	if err := m.seedEventTypes(ctx, tx); err != nil {
		return fmt.Errorf("failed to seed event types: %w", err)
	}

	// Seed PoA templates
	if err := m.seedPoATemplates(ctx, tx); err != nil {
		return fmt.Errorf("failed to seed PoA templates: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Println("Seed data inserted successfully")
	return nil
}

func (m *Migrator) seedSubscribers(ctx context.Context, tx *sql.Tx) error {
	query := `
		INSERT INTO subscribers (
			tenant_name, tenant_id, status, tier, 
			oidc_provider, oidc_issuer, oidc_client_id,
			policy_template, legal_framework,
			notification_channels, notification_email,
			contact_email, contact_name, domain
		) VALUES
		('Acme Corporation', 'acme-corp', 'active', 'enterprise',
			'okta', 'https://acme.okta.com', 'acme-client-id',
			'enterprise_standard', 'gdpr',
			ARRAY['email', 'webhook'], 'admin@acme.com',
			'admin@acme.com', 'John Admin', 'acme.com'),
		('TechStart Inc', 'techstart', 'active', 'premium',
			'azure', 'https://login.microsoftonline.com/techstart', 'techstart-client-id',
			'startup_flexible', 'ccpa',
			ARRAY['email', 'slack'], 'admin@techstart.io',
			'admin@techstart.io', 'Jane Startup', 'techstart.io'),
		('Global Services Ltd', 'global-services', 'active', 'standard',
			'google', 'https://accounts.google.com', 'global-client-id',
			'standard_template', 'soc2',
			ARRAY['email'], 'admin@globalservices.com',
			'admin@globalservices.com', 'Bob Manager', 'globalservices.com'),
		('Beta Testing Co', 'beta-test', 'pending', 'free',
			'auth0', 'https://beta.auth0.com', 'beta-client-id',
			'trial_template', 'basic',
			ARRAY['email'], 'admin@betatest.com',
			'admin@betatest.com', 'Alice Tester', 'betatest.com')
		ON CONFLICT (tenant_id) DO NOTHING
	`
	_, err := tx.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to seed subscribers: %w", err)
	}
	log.Println("✓ Seeded 4 sample subscribers")
	return nil
}

func (m *Migrator) seedEventTypes(ctx context.Context, tx *sql.Tx) error {
	query := `
		INSERT INTO event_types (
			event_type, category, description, severity, is_system_event, retention_days
		) VALUES
		('user.login', 'authentication', 'User successfully logged in', 'info', true, 90),
		('user.logout', 'authentication', 'User logged out', 'info', true, 90),
		('user.login.failed', 'authentication', 'Failed login attempt', 'medium', true, 180),
		('token.issued', 'token', 'New token issued', 'info', true, 90),
		('token.revoked', 'token', 'Token revoked', 'high', true, 365),
		('token.expired', 'token', 'Token expired', 'low', true, 30),
		('policy.created', 'authorization', 'New policy created', 'medium', true, 365),
		('policy.updated', 'authorization', 'Policy updated', 'medium', true, 365),
		('policy.deleted', 'authorization', 'Policy deleted', 'high', true, 365),
		('poa.created', 'power_of_attorney', 'New PoA created', 'medium', true, 365),
		('poa.revoked', 'power_of_attorney', 'PoA revoked', 'high', true, 365),
		('config.changed', 'configuration', 'Configuration changed', 'high', true, 365)
		ON CONFLICT (event_type) DO NOTHING
	`
	_, err := tx.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to seed event types: %w", err)
	}
	log.Println("✓ Seeded 12 system event types")
	return nil
}

func (m *Migrator) seedPoATemplates(ctx context.Context, tx *sql.Tx) error {
	query := `
		INSERT INTO poa_templates (
			template_name, description, scope_type, 
			default_actions, default_duration_days, is_system_template
		) VALUES
		('Full Access Template', 'Complete access to all resources', 'full',
			ARRAY['read', 'write', 'delete', 'manage'], 365, true),
		('Financial Operations', 'Access to financial and payment operations', 'financial',
			ARRAY['view_finances', 'approve_payments', 'manage_budgets'], 180, true),
		('Administrative Template', 'Standard administrative access', 'administrative',
			ARRAY['manage_users', 'view_reports', 'configure_settings'], 90, true)
		ON CONFLICT DO NOTHING
	`
	_, err := tx.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to seed PoA templates: %w", err)
	}
	log.Println("✓ Seeded 3 PoA templates")
	return nil
}

// ClearData removes all data from the database (for testing)
func (m *Migrator) ClearData(ctx context.Context) error {
	log.Println("WARNING: Clearing all data from database...")

	tables := []string{
		"merkle_tree_nodes", "merkle_proofs", "revocations", "append_only_log",
		"config_variables", "config_files", "service_configs", "tenant_config_overrides", "feature_flags",
		"audit_events", "compliance_reports", "event_correlation_patterns", "siem_integrations",
		"circuit_breakers", "rate_limiters", "retry_policies", "bulkheads",
		"event_types", "events", "event_handlers",
		"poa_records", "poa_templates",
		"policies", "policy_attributes", "authorization_logs",
		"subscribers", "tokens", "token_blacklist",
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // Ignore rollback error; will be committed on success

	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Println("All data cleared from database")
	return nil
}
