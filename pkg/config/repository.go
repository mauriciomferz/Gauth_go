package config

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides database operations for configuration management
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new configuration repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ============================================================================
// CONFIG VARIABLES
// ============================================================================

// ConfigVariableRecord represents a configuration variable in the database
type ConfigVariableRecord struct {
	ID            string
	TenantID      *string
	VariableKey   string
	VariableValue string
	VariableType  string
	Scope         string
	Description   *string
	IsSensitive   bool
	IsEncrypted   bool
	Category      *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	UpdatedBy     *string
}

// ListVariables retrieves all configuration variables for a tenant
func (r *Repository) ListVariables(ctx context.Context, tenantID string) ([]ConfigVariableRecord, error) {
	if r.db == nil {
		return []ConfigVariableRecord{}, nil
	}

	query := `
		SELECT id, tenant_id, variable_key, variable_value, variable_type, scope,
		       description, is_sensitive, is_encrypted, category,
		       created_at, updated_at, updated_by
		FROM config_variables
		WHERE tenant_id = $1 OR tenant_id IS NULL
		ORDER BY scope DESC, variable_key ASC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list config variables: %w", err)
	}
	defer rows.Close()

	var variables []ConfigVariableRecord
	for rows.Next() {
		var v ConfigVariableRecord
		err := rows.Scan(
			&v.ID, &v.TenantID, &v.VariableKey, &v.VariableValue, &v.VariableType, &v.Scope,
			&v.Description, &v.IsSensitive, &v.IsEncrypted, &v.Category,
			&v.CreatedAt, &v.UpdatedAt, &v.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan config variable: %w", err)
		}
		variables = append(variables, v)
	}

	return variables, rows.Err()
}

// CreateVariable creates a new configuration variable
func (r *Repository) CreateVariable(ctx context.Context, variable ConfigVariableRecord) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `
		INSERT INTO config_variables (
			tenant_id, variable_key, variable_value, variable_type, scope,
			description, is_sensitive, is_encrypted, category, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Exec(ctx, query,
		variable.TenantID, variable.VariableKey, variable.VariableValue,
		variable.VariableType, variable.Scope, variable.Description,
		variable.IsSensitive, variable.IsEncrypted, variable.Category,
		variable.UpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to create config variable: %w", err)
	}

	return nil
}

// UpdateVariable updates an existing configuration variable
func (r *Repository) UpdateVariable(ctx context.Context, tenantID, key string, variable ConfigVariableRecord) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `
		UPDATE config_variables
		SET variable_value = $1, variable_type = $2, description = $3,
		    is_sensitive = $4, updated_at = CURRENT_TIMESTAMP, updated_by = $5
		WHERE tenant_id = $6 AND variable_key = $7
	`

	result, err := r.db.Exec(ctx, query,
		variable.VariableValue, variable.VariableType, variable.Description,
		variable.IsSensitive, variable.UpdatedBy, tenantID, key,
	)
	if err != nil {
		return fmt.Errorf("failed to update config variable: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("config variable not found")
	}

	return nil
}

// DeleteVariable deletes a configuration variable
func (r *Repository) DeleteVariable(ctx context.Context, tenantID, key string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `DELETE FROM config_variables WHERE tenant_id = $1 AND variable_key = $2`

	result, err := r.db.Exec(ctx, query, tenantID, key)
	if err != nil {
		return fmt.Errorf("failed to delete config variable: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("config variable not found")
	}

	return nil
}

// ============================================================================
// CONFIG FILES
// ============================================================================

// ConfigFileRecord represents a configuration file in the database
type ConfigFileRecord struct {
	ID          string
	TenantID    *string
	FileName    string
	FileFormat  string
	FileContent string
	Description *string
	Checksum    *string
	SizeBytes   *int
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	UpdatedBy   *string
}

// GetConfigFile retrieves the latest version of a configuration file
func (r *Repository) GetConfigFile(ctx context.Context, tenantID, fileName, format string) (*ConfigFileRecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	query := `
		SELECT id, tenant_id, file_name, file_format, file_content,
		       description, checksum, size_bytes, version,
		       created_at, updated_at, updated_by
		FROM config_files
		WHERE tenant_id = $1 AND file_name = $2 AND file_format = $3
		ORDER BY version DESC
		LIMIT 1
	`

	var file ConfigFileRecord
	err := r.db.QueryRow(ctx, query, tenantID, fileName, format).Scan(
		&file.ID, &file.TenantID, &file.FileName, &file.FileFormat, &file.FileContent,
		&file.Description, &file.Checksum, &file.SizeBytes, &file.Version,
		&file.CreatedAt, &file.UpdatedAt, &file.UpdatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get config file: %w", err)
	}

	return &file, nil
}

// CreateConfigFile creates a new version of a configuration file
func (r *Repository) CreateConfigFile(ctx context.Context, file ConfigFileRecord) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	// Get the latest version for this file
	versionQuery := `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM config_files
		WHERE tenant_id = $1 AND file_name = $2
	`
	var nextVersion int
	err := r.db.QueryRow(ctx, versionQuery, file.TenantID, file.FileName).Scan(&nextVersion)
	if err != nil {
		return fmt.Errorf("failed to get next version: %w", err)
	}

	query := `
		INSERT INTO config_files (
			tenant_id, file_name, file_format, file_content, description,
			checksum, size_bytes, version, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.db.Exec(ctx, query,
		file.TenantID, file.FileName, file.FileFormat, file.FileContent,
		file.Description, file.Checksum, file.SizeBytes, nextVersion,
		file.UpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	return nil
}

// ListConfigVersions retrieves all versions of configuration files for a tenant
func (r *Repository) ListConfigVersions(ctx context.Context, tenantID string) ([]ConfigFileRecord, error) {
	if r.db == nil {
		return []ConfigFileRecord{}, nil
	}

	query := `
		SELECT id, tenant_id, file_name, file_format, file_content,
		       description, checksum, size_bytes, version,
		       created_at, updated_at, updated_by
		FROM config_files
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 20
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list config versions: %w", err)
	}
	defer rows.Close()

	var versions []ConfigFileRecord
	for rows.Next() {
		var v ConfigFileRecord
		err := rows.Scan(
			&v.ID, &v.TenantID, &v.FileName, &v.FileFormat, &v.FileContent,
			&v.Description, &v.Checksum, &v.SizeBytes, &v.Version,
			&v.CreatedAt, &v.UpdatedAt, &v.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan config version: %w", err)
		}
		versions = append(versions, v)
	}

	return versions, rows.Err()
}

// GetConfigVersion retrieves a specific version of a configuration file
func (r *Repository) GetConfigVersion(ctx context.Context, tenantID, versionID string) (*ConfigFileRecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	query := `
		SELECT id, tenant_id, file_name, file_format, file_content,
		       description, checksum, size_bytes, version,
		       created_at, updated_at, updated_by
		FROM config_files
		WHERE tenant_id = $1 AND id = $2
	`

	var file ConfigFileRecord
	err := r.db.QueryRow(ctx, query, tenantID, versionID).Scan(
		&file.ID, &file.TenantID, &file.FileName, &file.FileFormat, &file.FileContent,
		&file.Description, &file.Checksum, &file.SizeBytes, &file.Version,
		&file.CreatedAt, &file.UpdatedAt, &file.UpdatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get config version: %w", err)
	}

	return &file, nil
}

// ============================================================================
// SERVICE CONFIGS
// ============================================================================

// ServiceConfigRecord represents a service configuration in the database
type ServiceConfigRecord struct {
	ID              string
	TenantID        *string
	ServiceName     string
	ConfigVersion   string
	Status          string
	ConfigData      string // JSONB stored as string
	Environment     *string
	DeployedAt      *time.Time
	LastReloadAt    *time.Time
	RequiresRestart bool
	CreatedAt       time.Time
}

// ListServiceConfigs retrieves all service configurations
func (r *Repository) ListServiceConfigs(ctx context.Context, tenantID string) ([]ServiceConfigRecord, error) {
	if r.db == nil {
		return []ServiceConfigRecord{}, nil
	}

	query := `
		SELECT DISTINCT ON (service_name)
		       id, tenant_id, service_name, config_version, status,
		       config_data::text, environment, deployed_at, last_reload_at,
		       requires_restart, created_at
		FROM service_configs
		WHERE tenant_id = $1
		ORDER BY service_name, created_at DESC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list service configs: %w", err)
	}
	defer rows.Close()

	var configs []ServiceConfigRecord
	for rows.Next() {
		var c ServiceConfigRecord
		err := rows.Scan(
			&c.ID, &c.TenantID, &c.ServiceName, &c.ConfigVersion, &c.Status,
			&c.ConfigData, &c.Environment, &c.DeployedAt, &c.LastReloadAt,
			&c.RequiresRestart, &c.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan service config: %w", err)
		}
		configs = append(configs, c)
	}

	return configs, rows.Err()
}

// UpdateServiceReload updates the last reload timestamp for a service
func (r *Repository) UpdateServiceReload(ctx context.Context, tenantID, serviceName string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `
		UPDATE service_configs
		SET last_reload_at = CURRENT_TIMESTAMP, status = 'active'
		WHERE tenant_id = $1 AND service_name = $2
		  AND id IN (
		      SELECT id FROM service_configs
		      WHERE tenant_id = $1 AND service_name = $2
		      ORDER BY created_at DESC
		      LIMIT 1
		  )
	`

	_, err := r.db.Exec(ctx, query, tenantID, serviceName)
	if err != nil {
		return fmt.Errorf("failed to update service reload: %w", err)
	}

	return nil
}

// ============================================================================
// TENANT CONFIG OVERRIDES
// ============================================================================

// TenantOverrideRecord represents a tenant-specific configuration override
type TenantOverrideRecord struct {
	ID            string
	TenantID      string
	ConfigKey     string
	OverrideValue string
	OverrideType  string
	Enabled       bool
	Priority      int
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CreatedBy     *string
}

// ListTenantOverrides retrieves all tenant configuration overrides
func (r *Repository) ListTenantOverrides(ctx context.Context, tenantID string) ([]TenantOverrideRecord, error) {
	if r.db == nil {
		return []TenantOverrideRecord{}, nil
	}

	query := `
		SELECT id, tenant_id, config_key, override_value, override_type,
		       enabled, priority, created_at, updated_at, created_by
		FROM tenant_config_overrides
		WHERE tenant_id = $1
		ORDER BY priority DESC, config_key ASC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant overrides: %w", err)
	}
	defer rows.Close()

	var overrides []TenantOverrideRecord
	for rows.Next() {
		var o TenantOverrideRecord
		err := rows.Scan(
			&o.ID, &o.TenantID, &o.ConfigKey, &o.OverrideValue, &o.OverrideType,
			&o.Enabled, &o.Priority, &o.CreatedAt, &o.UpdatedAt, &o.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant override: %w", err)
		}
		overrides = append(overrides, o)
	}

	return overrides, rows.Err()
}

// CreateTenantOverride creates a new tenant configuration override
func (r *Repository) CreateTenantOverride(ctx context.Context, override TenantOverrideRecord) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `
		INSERT INTO tenant_config_overrides (
			tenant_id, config_key, override_value, override_type, created_by
		) VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Exec(ctx, query,
		override.TenantID, override.ConfigKey, override.OverrideValue,
		override.OverrideType, override.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to create tenant override: %w", err)
	}

	return nil
}

// ToggleTenantOverride enables or disables a tenant override
func (r *Repository) ToggleTenantOverride(ctx context.Context, tenantID, overrideID string, enabled bool) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `
		UPDATE tenant_config_overrides
		SET enabled = $1, updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id = $2 AND id = $3
	`

	result, err := r.db.Exec(ctx, query, enabled, tenantID, overrideID)
	if err != nil {
		return fmt.Errorf("failed to toggle tenant override: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("tenant override not found")
	}

	return nil
}

// DeleteTenantOverride deletes a tenant configuration override
func (r *Repository) DeleteTenantOverride(ctx context.Context, tenantID, overrideID string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `DELETE FROM tenant_config_overrides WHERE tenant_id = $1 AND id = $2`

	result, err := r.db.Exec(ctx, query, tenantID, overrideID)
	if err != nil {
		return fmt.Errorf("failed to delete tenant override: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("tenant override not found")
	}

	return nil
}

// ============================================================================
// FEATURE FLAGS
// ============================================================================

// FeatureFlagRecord represents a feature flag in the database
type FeatureFlagRecord struct {
	ID                string
	TenantID          *string
	FlagKey           string
	FlagName          string
	Description       *string
	Enabled           bool
	RolloutPercentage int
	TargetingRules    *string // JSONB stored as string
	Category          *string
	Tags              []string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	UpdatedBy         *string
}

// ListFeatureFlags retrieves all feature flags for a tenant
func (r *Repository) ListFeatureFlags(ctx context.Context, tenantID string) ([]FeatureFlagRecord, error) {
	if r.db == nil {
		return []FeatureFlagRecord{}, nil
	}

	query := `
		SELECT id, tenant_id, flag_key, flag_name, description, enabled,
		       rollout_percentage, targeting_rules::text, category, tags,
		       created_at, updated_at, updated_by
		FROM feature_flags
		WHERE tenant_id = $1 OR tenant_id IS NULL
		ORDER BY flag_name ASC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list feature flags: %w", err)
	}
	defer rows.Close()

	var flags []FeatureFlagRecord
	for rows.Next() {
		var f FeatureFlagRecord
		err := rows.Scan(
			&f.ID, &f.TenantID, &f.FlagKey, &f.FlagName, &f.Description, &f.Enabled,
			&f.RolloutPercentage, &f.TargetingRules, &f.Category, &f.Tags,
			&f.CreatedAt, &f.UpdatedAt, &f.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feature flag: %w", err)
		}
		flags = append(flags, f)
	}

	return flags, rows.Err()
}

// CreateFeatureFlag creates a new feature flag
func (r *Repository) CreateFeatureFlag(ctx context.Context, flag FeatureFlagRecord) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `
		INSERT INTO feature_flags (
			tenant_id, flag_key, flag_name, description, enabled,
			rollout_percentage, targeting_rules, category, tags, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10)
	`

	_, err := r.db.Exec(ctx, query,
		flag.TenantID, flag.FlagKey, flag.FlagName, flag.Description, flag.Enabled,
		flag.RolloutPercentage, flag.TargetingRules, flag.Category, flag.Tags,
		flag.UpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to create feature flag: %w", err)
	}

	return nil
}

// ToggleFeatureFlag enables or disables a feature flag
func (r *Repository) ToggleFeatureFlag(ctx context.Context, tenantID, flagID string, enabled bool) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `
		UPDATE feature_flags
		SET enabled = $1, updated_at = CURRENT_TIMESTAMP
		WHERE (tenant_id = $2 OR tenant_id IS NULL) AND id = $3
	`

	result, err := r.db.Exec(ctx, query, enabled, tenantID, flagID)
	if err != nil {
		return fmt.Errorf("failed to toggle feature flag: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("feature flag not found")
	}

	return nil
}

// DeleteFeatureFlag deletes a feature flag
func (r *Repository) DeleteFeatureFlag(ctx context.Context, tenantID, flagID string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `DELETE FROM feature_flags WHERE (tenant_id = $1 OR tenant_id IS NULL) AND id = $2`

	result, err := r.db.Exec(ctx, query, tenantID, flagID)
	if err != nil {
		return fmt.Errorf("failed to delete feature flag: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("feature flag not found")
	}

	return nil
}
