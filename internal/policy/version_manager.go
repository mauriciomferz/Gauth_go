package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/policy"
)

// SemanticVersion represents a semantic version (major.minor.patch).
type SemanticVersion struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

// String returns the string representation of the semantic version.
func (v SemanticVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare returns -1 if v < other, 0 if equal, 1 if v > other.
func (v SemanticVersion) Compare(other SemanticVersion) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// ParseSemanticVersion parses a semantic version string (e.g., "1.2.3").
func ParseSemanticVersion(s string) (SemanticVersion, error) {
	var v SemanticVersion
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)
	matches := re.FindStringSubmatch(s)
	if len(matches) != 4 {
		return v, fmt.Errorf("invalid semantic version: %s", s)
	}
	if _, err := fmt.Sscanf(matches[1], "%d", &v.Major); err != nil {
		return v, fmt.Errorf("invalid major version: %s", matches[1])
	}
	if _, err := fmt.Sscanf(matches[2], "%d", &v.Minor); err != nil {
		return v, fmt.Errorf("invalid minor version: %s", matches[2])
	}
	if _, err := fmt.Sscanf(matches[3], "%d", &v.Patch); err != nil {
		return v, fmt.Errorf("invalid patch version: %s", matches[3])
	}
	return v, nil
}

// PolicyVersionMetadata contains comprehensive metadata for a policy version.
type PolicyVersionMetadata struct {
	BundleVersion      int             `json:"bundle_version"`        // Integer version from bundle
	SemanticVersion    SemanticVersion `json:"semantic_version"`      // Semantic version
	Name               string          `json:"name"`                  // Human-readable version name
	Description        string          `json:"description"`           // Change description
	Author             string          `json:"author"`                // Author/creator
	EffectiveDate      time.Time       `json:"effective_date"`        // When version becomes active
	SunsetDate         *time.Time      `json:"sunset_date,omitempty"` // When version is deprecated
	Deprecated         bool            `json:"deprecated"`            // Deprecation flag
	DeprecationReason  string          `json:"deprecation_reason,omitempty"`
	BackwardCompatible bool            `json:"backward_compatible"` // Compatibility with previous version
	MigrationRequired  bool            `json:"migration_required"`  // Requires data migration
	MigrationScript    string          `json:"migration_script,omitempty"`
	Tags               []string        `json:"tags,omitempty"` // Version tags (e.g., "stable", "beta", "security-fix")
	ChangeLog          []ChangeEntry   `json:"changelog,omitempty"`
	RollbackAllowed    bool            `json:"rollback_allowed"`             // Can roll back to this version
	RequiredApprovals  []string        `json:"required_approvals,omitempty"` // Required approvals for activation
	ApprovalStatus     map[string]bool `json:"approval_status,omitempty"`    // Approval tracking
	CreatedAt          time.Time       `json:"created_at"`
	ActivatedAt        *time.Time      `json:"activated_at,omitempty"`
	Hash               string          `json:"hash"`          // Bundle hash
	PreviousHash       string          `json:"previous_hash"` // Previous bundle hash
	ValidationErrors   []string        `json:"validation_errors,omitempty"`
	ImpactAnalysis     *ImpactAnalysis `json:"impact_analysis,omitempty"`
}

// ChangeEntry represents a single change in the changelog.
type ChangeEntry struct {
	Type        string    `json:"type"` // "added", "modified", "removed", "security", "bugfix"
	Description string    `json:"description"`
	PolicyID    string    `json:"policy_id,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// ImpactAnalysis summarizes the impact of a policy version.
type ImpactAnalysis struct {
	PoliciesAdded    int      `json:"policies_added"`
	PoliciesModified int      `json:"policies_modified"`
	PoliciesRemoved  int      `json:"policies_removed"`
	AffectedSubjects []string `json:"affected_subjects,omitempty"`
	AffectedActions  []string `json:"affected_actions,omitempty"`
	RiskLevel        string   `json:"risk_level"` // "low", "medium", "high"
	EstimatedImpact  string   `json:"estimated_impact"`
}

// PolicyVersionManager manages policy versions with metadata and rollback capabilities.
type PolicyVersionManager struct {
	mu               sync.RWMutex
	registry         *policy.Registry
	versionMetadata  map[int]*PolicyVersionMetadata // bundle_version -> metadata
	activeVersion    int
	auditCallback    func(event VersionAuditEvent)
	enableValidation bool
	store            PolicyVersionStore // Optional persistent storage
}

// VersionAuditEvent represents a version-related audit event.
type VersionAuditEvent struct {
	EventType     string                 `json:"event_type"` // "version_created", "version_activated", "version_deprecated", "rollback"
	Version       int                    `json:"version"`
	SemanticVer   string                 `json:"semantic_version,omitempty"`
	Actor         string                 `json:"actor,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Success       bool                   `json:"success"`
	Error         string                 `json:"error,omitempty"`
	ImpactSummary string                 `json:"impact_summary,omitempty"`
}

// NewPolicyVersionManager creates a new policy version manager.
func NewPolicyVersionManager(registry *policy.Registry) *PolicyVersionManager {
	return &PolicyVersionManager{
		registry:         registry,
		versionMetadata:  make(map[int]*PolicyVersionMetadata),
		enableValidation: true,
	}
}

// NewPolicyVersionManagerWithStore creates a new policy version manager with persistent storage.
func NewPolicyVersionManagerWithStore(registry *policy.Registry, store PolicyVersionStore) (*PolicyVersionManager, error) {
	m := &PolicyVersionManager{
		registry:         registry,
		versionMetadata:  make(map[int]*PolicyVersionMetadata),
		enableValidation: true,
		store:            store,
	}

	// Load existing versions from store
	if err := m.loadFromStore(); err != nil {
		return nil, fmt.Errorf("load from store: %w", err)
	}

	return m, nil
}

// loadFromStore loads all versions from persistent storage into memory.
func (m *PolicyVersionManager) loadFromStore() error {
	if m.store == nil {
		return nil
	}

	// Load all version numbers
	versions, err := m.store.ListVersions()
	if err != nil {
		return fmt.Errorf("list versions: %w", err)
	}

	// Load each version's metadata and bundle
	for _, version := range versions {
		bundle, metadata, err := m.store.LoadVersion(version)
		if err != nil {
			return fmt.Errorf("load version %d: %w", version, err)
		}

		// Store metadata in memory
		m.versionMetadata[version] = metadata

		// Register bundle with registry
		if _, err := m.registry.AddBundle(*bundle); err != nil {
			// Log error but continue (registry may already have bundle)
			// In production, use proper logging
		}
	}

	// Load active version
	activeVersion, err := m.store.LoadActiveVersion()
	if err == nil {
		m.activeVersion = activeVersion
	}

	return nil
}

// CreateVersion creates a new policy version with metadata.
func (m *PolicyVersionManager) CreateVersion(ctx context.Context, bundle policy.Bundle, metadata PolicyVersionMetadata) (*PolicyVersionMetadata, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate bundle
	if err := policy.ValidateBundle(bundle); err != nil {
		m.audit(VersionAuditEvent{
			EventType: "version_created",
			Timestamp: time.Now(),
			Success:   false,
			Error:     fmt.Sprintf("bundle validation failed: %v", err),
		})
		return nil, fmt.Errorf("bundle validation failed: %w", err)
	}

	// Add bundle to registry
	storedBundle, err := m.registry.AddBundle(bundle)
	if err != nil {
		m.audit(VersionAuditEvent{
			EventType: "version_created",
			Timestamp: time.Now(),
			Success:   false,
			Error:     fmt.Sprintf("failed to add bundle: %v", err),
		})
		return nil, fmt.Errorf("failed to add bundle: %w", err)
	}

	// Complete metadata
	metadata.BundleVersion = storedBundle.Version
	metadata.Hash = storedBundle.Hash
	metadata.PreviousHash = storedBundle.PrevHash
	metadata.CreatedAt = storedBundle.Created
	if metadata.EffectiveDate.IsZero() {
		metadata.EffectiveDate = time.Now()
	}
	if metadata.SemanticVersion.Major == 0 && metadata.SemanticVersion.Minor == 0 && metadata.SemanticVersion.Patch == 0 {
		metadata.SemanticVersion = SemanticVersion{Major: 1, Minor: 0, Patch: storedBundle.Version - 1}
	}

	// Perform impact analysis if previous version exists
	if storedBundle.Version > 1 {
		analysis, err := m.analyzeImpact(storedBundle.Version-1, storedBundle.Version)
		if err == nil {
			metadata.ImpactAnalysis = analysis
		}
	}

	// Validate backward compatibility if required
	if m.enableValidation && storedBundle.Version > 1 {
		if err := m.validateBackwardCompatibility(storedBundle.Version-1, storedBundle.Version); err != nil {
			metadata.BackwardCompatible = false
			metadata.ValidationErrors = append(metadata.ValidationErrors, err.Error())
		} else {
			metadata.BackwardCompatible = true
		}
	}

	// Store metadata
	m.versionMetadata[storedBundle.Version] = &metadata

	// Persist to storage if available
	if m.store != nil {
		if err := m.store.SaveVersion(storedBundle.Version, storedBundle, &metadata); err != nil {
			// Log error but don't fail (in-memory state is consistent)
			// In production, use proper logging
		}
	}

	// Audit event
	m.audit(VersionAuditEvent{
		EventType:   "version_created",
		Version:     storedBundle.Version,
		SemanticVer: metadata.SemanticVersion.String(),
		Timestamp:   time.Now(),
		Success:     true,
		Metadata: map[string]interface{}{
			"name":                metadata.Name,
			"backward_compatible": metadata.BackwardCompatible,
			"hash":                metadata.Hash,
		},
	})

	// Persist audit event
	if m.store != nil {
		auditEvent := VersionAuditEvent{
			EventType:   "version_created",
			Version:     storedBundle.Version,
			SemanticVer: metadata.SemanticVersion.String(),
			Timestamp:   time.Now(),
			Success:     true,
			Metadata: map[string]interface{}{
				"name":                metadata.Name,
				"backward_compatible": metadata.BackwardCompatible,
				"hash":                metadata.Hash,
			},
		}
		if err := m.store.SaveAuditEvent(auditEvent); err != nil {
			// Log error but don't fail
		}
	}

	// Auto-activate if it's the first version
	if storedBundle.Version == 1 {
		m.activeVersion = 1
		now := time.Now()
		metadata.ActivatedAt = &now

		// Persist active version
		if m.store != nil {
			if err := m.store.SaveActiveVersion(1); err != nil {
				// Log error but don't fail
			}
		}
	}

	return &metadata, nil
}

// ActivateVersion activates a specific policy version.
func (m *PolicyVersionManager) ActivateVersion(ctx context.Context, version int, actor string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	metadata, exists := m.versionMetadata[version]
	if !exists {
		return fmt.Errorf("version %d not found", version)
	}

	// Check if deprecated
	if metadata.Deprecated {
		m.audit(VersionAuditEvent{
			EventType: "version_activated",
			Version:   version,
			Actor:     actor,
			Timestamp: time.Now(),
			Success:   false,
			Error:     "cannot activate deprecated version",
		})
		return fmt.Errorf("cannot activate deprecated version %d", version)
	}

	// Check effective date
	if time.Now().Before(metadata.EffectiveDate) {
		return fmt.Errorf("version %d effective date is in the future: %v", version, metadata.EffectiveDate)
	}

	// Check required approvals
	if len(metadata.RequiredApprovals) > 0 {
		for _, approval := range metadata.RequiredApprovals {
			if !metadata.ApprovalStatus[approval] {
				return fmt.Errorf("version %d missing required approval: %s", version, approval)
			}
		}
	}

	// Activate version
	previousVersion := m.activeVersion
	m.activeVersion = version
	now := time.Now()
	metadata.ActivatedAt = &now

	// Audit event
	m.audit(VersionAuditEvent{
		EventType:   "version_activated",
		Version:     version,
		SemanticVer: metadata.SemanticVersion.String(),
		Actor:       actor,
		Timestamp:   time.Now(),
		Success:     true,
		Metadata: map[string]interface{}{
			"previous_version": previousVersion,
			"name":             metadata.Name,
		},
	})

	return nil
}

// RollbackVersion rolls back to a previous policy version with safety checks.
func (m *PolicyVersionManager) RollbackVersion(ctx context.Context, targetVersion int, actor string, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	currentVersion := m.activeVersion
	if targetVersion == currentVersion {
		return fmt.Errorf("target version %d is already active", targetVersion)
	}

	metadata, exists := m.versionMetadata[targetVersion]
	if !exists {
		return fmt.Errorf("target version %d not found", targetVersion)
	}

	// Check if rollback is allowed
	if !metadata.RollbackAllowed {
		m.audit(VersionAuditEvent{
			EventType: "rollback",
			Version:   targetVersion,
			Actor:     actor,
			Timestamp: time.Now(),
			Success:   false,
			Error:     "rollback not allowed for this version",
		})
		return fmt.Errorf("rollback not allowed for version %d", targetVersion)
	}

	// Check if deprecated
	if metadata.Deprecated {
		return fmt.Errorf("cannot rollback to deprecated version %d", targetVersion)
	}

	// Perform rollback safety validation
	if m.enableValidation {
		if err := m.validateRollbackSafety(currentVersion, targetVersion); err != nil {
			m.audit(VersionAuditEvent{
				EventType: "rollback",
				Version:   targetVersion,
				Actor:     actor,
				Timestamp: time.Now(),
				Success:   false,
				Error:     fmt.Sprintf("rollback safety check failed: %v", err),
			})
			return fmt.Errorf("rollback safety check failed: %w", err)
		}
	}

	// Execute rollback in registry
	if err := m.registry.Rollback(targetVersion); err != nil {
		m.audit(VersionAuditEvent{
			EventType: "rollback",
			Version:   targetVersion,
			Actor:     actor,
			Timestamp: time.Now(),
			Success:   false,
			Error:     fmt.Sprintf("registry rollback failed: %v", err),
		})
		return fmt.Errorf("registry rollback failed: %w", err)
	}

	// Update active version
	m.activeVersion = targetVersion

	// Persist active version
	if m.store != nil {
		if err := m.store.SaveActiveVersion(targetVersion); err != nil {
			// Log error but don't fail (in-memory state is consistent)
		}
	}

	// Audit event
	auditEvent := VersionAuditEvent{
		EventType:   "rollback",
		Version:     targetVersion,
		SemanticVer: metadata.SemanticVersion.String(),
		Actor:       actor,
		Timestamp:   time.Now(),
		Success:     true,
		Metadata: map[string]interface{}{
			"previous_version": currentVersion,
			"reason":           reason,
			"name":             metadata.Name,
		},
		ImpactSummary: fmt.Sprintf("Rolled back from v%d to v%d", currentVersion, targetVersion),
	}

	m.audit(auditEvent)

	// Persist audit event
	if m.store != nil {
		if err := m.store.SaveAuditEvent(auditEvent); err != nil {
			// Log error but don't fail
		}
	}

	return nil
}

// DeprecateVersion marks a version as deprecated.
func (m *PolicyVersionManager) DeprecateVersion(ctx context.Context, version int, reason string, sunsetDate *time.Time, actor string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	metadata, exists := m.versionMetadata[version]
	if !exists {
		return fmt.Errorf("version %d not found", version)
	}

	if metadata.Deprecated {
		return fmt.Errorf("version %d is already deprecated", version)
	}

	// Cannot deprecate active version
	if version == m.activeVersion {
		return fmt.Errorf("cannot deprecate active version %d", version)
	}

	metadata.Deprecated = true
	metadata.DeprecationReason = reason
	metadata.SunsetDate = sunsetDate

	m.audit(VersionAuditEvent{
		EventType:   "version_deprecated",
		Version:     version,
		SemanticVer: metadata.SemanticVersion.String(),
		Actor:       actor,
		Timestamp:   time.Now(),
		Success:     true,
		Metadata: map[string]interface{}{
			"reason":      reason,
			"sunset_date": sunsetDate,
		},
	})

	return nil
}

// GetVersionMetadata returns metadata for a specific version.
func (m *PolicyVersionManager) GetVersionMetadata(version int) (*PolicyVersionMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metadata, exists := m.versionMetadata[version]
	if !exists {
		return nil, fmt.Errorf("version %d not found", version)
	}

	return metadata, nil
}

// ListVersions returns all version metadata.
func (m *PolicyVersionManager) ListVersions() []*PolicyVersionMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	versions := make([]*PolicyVersionMetadata, 0, len(m.versionMetadata))
	for _, metadata := range m.versionMetadata {
		versions = append(versions, metadata)
	}

	return versions
}

// GetActiveVersion returns the currently active version number.
func (m *PolicyVersionManager) GetActiveVersion() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeVersion
}

// CompareVersions compares two policy versions and returns the difference.
func (m *PolicyVersionManager) CompareVersions(fromVersion, toVersion int) (*policy.PolicyDiff, error) {
	diff, err := m.registry.Diff(fromVersion, toVersion)
	if err != nil {
		return nil, err
	}
	return &diff, nil
}

// SetAuditCallback sets the audit callback function.
func (m *PolicyVersionManager) SetAuditCallback(callback func(event VersionAuditEvent)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.auditCallback = callback
}

// SetValidationEnabled enables or disables validation checks.
func (m *PolicyVersionManager) SetValidationEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enableValidation = enabled
}

// analyzeImpact analyzes the impact of upgrading from one version to another.
func (m *PolicyVersionManager) analyzeImpact(fromVersion, toVersion int) (*ImpactAnalysis, error) {
	diff, err := m.registry.Diff(fromVersion, toVersion)
	if err != nil {
		return nil, err
	}

	analysis := &ImpactAnalysis{
		PoliciesAdded:    len(diff.Added),
		PoliciesModified: len(diff.Changed),
		PoliciesRemoved:  len(diff.Removed),
		AffectedSubjects: []string{},
		AffectedActions:  []string{},
	}

	// Collect affected subjects and actions
	subjectMap := make(map[string]bool)
	actionMap := make(map[string]bool)

	for _, p := range diff.Added {
		for _, s := range p.Subjects {
			subjectMap[s] = true
		}
		for _, r := range p.Rules {
			for _, a := range r.Actions {
				actionMap[a] = true
			}
		}
	}

	for _, c := range diff.Changed {
		for _, s := range c.To.Subjects {
			subjectMap[s] = true
		}
		for _, r := range c.To.Rules {
			for _, a := range r.Actions {
				actionMap[a] = true
			}
		}
	}

	for _, p := range diff.Removed {
		for _, s := range p.Subjects {
			subjectMap[s] = true
		}
		for _, r := range p.Rules {
			for _, a := range r.Actions {
				actionMap[a] = true
			}
		}
	}

	for s := range subjectMap {
		analysis.AffectedSubjects = append(analysis.AffectedSubjects, s)
	}
	for a := range actionMap {
		analysis.AffectedActions = append(analysis.AffectedActions, a)
	}

	// Determine risk level
	totalChanges := analysis.PoliciesAdded + analysis.PoliciesModified + analysis.PoliciesRemoved
	if totalChanges == 0 {
		analysis.RiskLevel = "none"
		analysis.EstimatedImpact = "No changes detected"
	} else if totalChanges <= 2 && analysis.PoliciesRemoved == 0 {
		analysis.RiskLevel = "low"
		analysis.EstimatedImpact = fmt.Sprintf("Minor changes: %d policies affected", totalChanges)
	} else if totalChanges <= 5 || analysis.PoliciesRemoved <= 1 {
		analysis.RiskLevel = "medium"
		analysis.EstimatedImpact = fmt.Sprintf("Moderate changes: %d policies affected, %d removed", totalChanges, analysis.PoliciesRemoved)
	} else {
		analysis.RiskLevel = "high"
		analysis.EstimatedImpact = fmt.Sprintf("Major changes: %d policies affected, %d removed", totalChanges, analysis.PoliciesRemoved)
	}

	return analysis, nil
}

// validateBackwardCompatibility checks if a new version is backward compatible.
func (m *PolicyVersionManager) validateBackwardCompatibility(fromVersion, toVersion int) error {
	diff, err := m.registry.Diff(fromVersion, toVersion)
	if err != nil {
		return err
	}

	// Backward incompatible if policies are removed
	if len(diff.Removed) > 0 {
		return fmt.Errorf("backward incompatible: %d policies removed", len(diff.Removed))
	}

	// Check if any policy subjects were removed (tightening access)
	for _, c := range diff.Changed {
		fromSubjects := make(map[string]bool)
		for _, s := range c.From.Subjects {
			fromSubjects[s] = true
		}
		for _, s := range c.From.Subjects {
			found := false
			for _, ts := range c.To.Subjects {
				if ts == s {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("backward incompatible: policy %s removed subject %s", c.ID, s)
			}
		}
	}

	return nil
}

// validateRollbackSafety checks if rollback is safe.
func (m *PolicyVersionManager) validateRollbackSafety(currentVersion, targetVersion int) error {
	currentMeta, exists := m.versionMetadata[currentVersion]
	if !exists {
		return fmt.Errorf("current version %d metadata not found", currentVersion)
	}

	targetMeta, exists := m.versionMetadata[targetVersion]
	if !exists {
		return fmt.Errorf("target version %d metadata not found", targetVersion)
	}

	// Check if current version requires migration
	if currentMeta.MigrationRequired {
		return fmt.Errorf("current version requires migration, rollback may cause data inconsistency")
	}

	// Check semantic version compatibility
	if currentMeta.SemanticVersion.Major > targetMeta.SemanticVersion.Major {
		return fmt.Errorf("cannot rollback across major version boundary (%s -> %s)",
			currentMeta.SemanticVersion.String(), targetMeta.SemanticVersion.String())
	}

	return nil
}

// audit sends an audit event.
func (m *PolicyVersionManager) audit(event VersionAuditEvent) {
	// For deterministic tests, execute callback synchronously.
	if m.auditCallback != nil {
		m.auditCallback(event)
	}
}

// ExportMetadata exports all version metadata as JSON.
func (m *PolicyVersionManager) ExportMetadata() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	export := struct {
		ActiveVersion   int                            `json:"active_version"`
		TotalVersions   int                            `json:"total_versions"`
		VersionMetadata map[int]*PolicyVersionMetadata `json:"version_metadata"`
		ChainHashes     []string                       `json:"chain_hashes"`
		ExportedAt      time.Time                      `json:"exported_at"`
	}{
		ActiveVersion:   m.activeVersion,
		TotalVersions:   len(m.versionMetadata),
		VersionMetadata: m.versionMetadata,
		ChainHashes:     m.registry.ChainHashes(),
		ExportedAt:      time.Now(),
	}

	return json.MarshalIndent(export, "", "  ")
}

// ApproveVersion records an approval for a version.
func (m *PolicyVersionManager) ApproveVersion(version int, approver string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	metadata, exists := m.versionMetadata[version]
	if !exists {
		return fmt.Errorf("version %d not found", version)
	}

	if metadata.ApprovalStatus == nil {
		metadata.ApprovalStatus = make(map[string]bool)
	}

	metadata.ApprovalStatus[approver] = true

	m.audit(VersionAuditEvent{
		EventType:   "version_approved",
		Version:     version,
		SemanticVer: metadata.SemanticVersion.String(),
		Actor:       approver,
		Timestamp:   time.Now(),
		Success:     true,
	})

	return nil
}
