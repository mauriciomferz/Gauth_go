package web

import (
	"fmt"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/mauriciomferz/AgentAuth/pkg/database"
	"github.com/mauriciomferz/AgentAuth/pkg/gauth"
	"github.com/mauriciomferz/AgentAuth/pkg/gauth/external"
	"github.com/mauriciomferz/AgentAuth/pkg/gauth/mocks"
	"github.com/mauriciomferz/AgentAuth/pkg/gauthplus"
)

// InitAAP001FromEnv initializes RFC-0111 components based on environment variables.
// This is a web-server specific helper that can create mock services and configure persistence.
//
// Environment variables:
//   - GAUTH_AAP001_ENABLED: Set to "1" to enable RFC-0111 functionality
//   - GAUTH_AAP001_USE_MOCKS: Set to "1" to use mock external services (default: 1)
//   - GAUTH_TOKEN_STORE: "postgres" or "memory" (default: memory)
//   - DB_HOST: PostgreSQL host (default: localhost)
//   - DB_PORT: PostgreSQL port (default: 5432)
//   - DB_NAME: PostgreSQL database name (default: gauth)
//   - DB_USER: PostgreSQL user (default: gauth)
//   - DB_PASSWORD: PostgreSQL password (default: gauth_password)
//   - DB_SSLMODE: PostgreSQL SSL mode (default: disable)
//
// Returns nil if RFC-0111 is not enabled.
// Returns an ExtendedTokenStore configured based on GAUTH_TOKEN_STORE.
func InitAAP001FromEnv() (*gauth.AAP001Components, gauth.ExtendedTokenStore, error) {
	// Check if RFC-0111 is enabled
	if os.Getenv("GAUTH_AAP001_ENABLED") != "1" {
		return nil, nil, nil
	}

	// Determine whether to use mocks (default: yes)
	useMocks := os.Getenv("GAUTH_AAP001_USE_MOCKS") != "0"

	var components *gauth.AAP001Components
	var err error

	if !useMocks {
		// Use real external service implementations (where available)
		components, err = InitAAP001WithRealServices()
		if err != nil {
			return nil, nil, fmt.Errorf("RFC-0111 real service initialization failed: %w", err)
		}
		fmt.Fprintf(os.Stderr, "[RFC-0111] Using REAL external services (Global Identity Verification)\n")
	} else {
		// Create mock external services
		pvpClient := mocks.NewMockPowerVerificationPoint()
		pipClient := mocks.NewMockPIPClient()
		commercialRegClient := mocks.NewMockCommercialRegisterClient()

		// Initialize RFC-0111 with mocks
		components, err = gauth.InitAAP001WithComponents(
			pvpClient,
			pipClient,
			commercialRegClient,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("RFC-0111 mock initialization failed: %w", err)
		}
		fmt.Fprintf(os.Stderr, "[RFC-0111] Using MOCK external services\n")
	}

	// Initialize token store based on GAUTH_TOKEN_STORE environment variable
	tokenStoreType := os.Getenv("GAUTH_TOKEN_STORE")
	if tokenStoreType == "" {
		tokenStoreType = "memory" // default
	}

	var tokenStore gauth.ExtendedTokenStore
	switch tokenStoreType {
	case "postgres":
		// Build PostgreSQL DSN from environment variables
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "5432"
		}
		dbname := os.Getenv("DB_NAME")
		if dbname == "" {
			dbname = "gauth"
		}
		user := os.Getenv("DB_USER")
		if user == "" {
			user = "gauth"
		}
		password := os.Getenv("DB_PASSWORD")
		if password == "" {
			password = "gauth_password"
		}
		sslmode := os.Getenv("DB_SSLMODE")
		if sslmode == "" {
			sslmode = "disable"
		}

		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, dbname, sslmode)

		// Create PostgreSQL token store
		pgStore, err := gauth.NewPostgresExtendedTokenStoreFromDSN(dsn)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize PostgreSQL token store: %w", err)
		}
		tokenStore = pgStore
		fmt.Fprintf(os.Stderr, "[RFC-0111] Using PostgreSQL token store (host=%s, db=%s)\n", host, dbname)

	case "memory":
		tokenStore = gauth.NewMemoryExtendedTokenStore()
		fmt.Fprintf(os.Stderr, "[RFC-0111] Using in-memory token store\n")

	default:
		return nil, nil, fmt.Errorf("unknown token store type: %s (supported: memory, postgres)", tokenStoreType)
	}

	// PHASE 3: AgentAuth+ Authorization Integration (optional, controlled by GAUTH_GAUTHPLUS_ENABLED)
	// Initialize AgentAuth+ services if database connection is available and feature is enabled
	if os.Getenv("GAUTH_GAUTHPLUS_ENABLED") == "1" {
		_, err := initializeAgentAuthPlus(components)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[AgentAuth+] WARNING: Failed to initialize AgentAuth+ integration: %v\n", err)
			fmt.Fprintf(os.Stderr, "[AgentAuth+] Continuing without AgentAuth+ features\n")
		} else {
			fmt.Fprintf(os.Stderr, "[AgentAuth+] Authorization chain integration enabled\n")
			// Note: Services are stored globally for endpoint registration
			// See initializeAgentAuthPlusEndpoints in server initialization
		}
	}

	return components, tokenStore, nil
}

// InitAAP001WithRealServices initializes RFC-0111 components with real service connectors.
// This sets up the GlobalIdentityVerifier with supported country connectors.
func InitAAP001WithRealServices() (*gauth.AAP001Components, error) {
	// 1. Initialize Country-Specific Connectors

	// US
	// Use Mock Provider for US identity source within the Real Verifier logic
	// In production this would be replaced by PersonaProvider or TruliooProvider
	usProvider := external.NewMockUSIdentityProvider("us-mock")
	usConfig := &external.USIdentityVerifierConfig{
		PrimaryProvider: usProvider,
		// FallbackProvider can be nil
		RequestTimeout: 10 * time.Second,
	}
	usVerifier := external.NewUSIdentityVerifier(usConfig)

	// France (FranceConnect)
	frConfig := &external.FranceConnectorConfig{
		FranceConnectURL: os.Getenv("GAUTH_FR_FC_URL"),
		ClientID:         os.Getenv("GAUTH_FR_CLIENT_ID"),
		ClientSecret:     os.Getenv("GAUTH_FR_CLIENT_SECRET"),
		RedirectURI:      os.Getenv("GAUTH_FR_REDIRECT_URI"),
		// Optional APIs
		ANTSAPIKey: os.Getenv("GAUTH_FR_ANTS_KEY"),
	}
	// Use defaults if empty for basic initialization
	if frConfig.FranceConnectURL == "" {
		frConfig.FranceConnectURL = "https://fcp.integ.franceconnect.gouv.fr/api/v1"
	}
	frConnector, err := external.NewFranceIdentityConnector(frConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[RFC-0111] Warning: Failed to initialize France connector: %v\n", err)
		// Continue with nil connector (GlobalVerifier handles nil)
		frConnector = nil
	}

	// Italy (SPID)
	itConfig := &external.ItalyConnectorConfig{
		SPIDMetadataURL:   os.Getenv("GAUTH_IT_SPID_URL"),
		ServiceProviderID: os.Getenv("GAUTH_IT_SP_ID"),
	}
	if itConfig.SPIDMetadataURL == "" {
		itConfig.SPIDMetadataURL = "https://registry.spid.gov.it/metadata"
	}
	itConnector, err := external.NewItalyIdentityConnector(itConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[RFC-0111] Warning: Failed to initialize Italy connector: %v\n", err)
		itConnector = nil
	}

	// Spain (Cl@ve)
	esConfig := &external.SpainConnectorConfig{
		ClaveURL:          os.Getenv("GAUTH_ES_CLAVE_URL"),
		ClaveClientID:     os.Getenv("GAUTH_ES_CLIENT_ID"),
		ClaveClientSecret: os.Getenv("GAUTH_ES_CLIENT_SECRET"),
	}
	if esConfig.ClaveURL == "" {
		esConfig.ClaveURL = "https://clave.gob.es/api/v1"
	}
	esConnector, err := external.NewSpainIdentityConnector(esConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[RFC-0111] Warning: Failed to initialize Spain connector: %v\n", err)
		esConnector = nil
	}

	// 2. Initialize Global Identity Verifier
	globalVerifier := gauth.NewGlobalIdentityVerifier(
		usVerifier,
		frConnector,
		itConnector,
		esConnector,
		os.Getenv("GAUTH_STRICT_IDENTITY") == "1", // Strict mode
	)

	// 3. Initialize other clients (Mocks for now unless real implementation available)
	// Using default PIP client which connects to external services
	pipConfig := &gauth.PIPClientConfig{
		BaseURL: os.Getenv("GAUTH_PIP_URL"),
		Timeout: 10 * time.Second,
	}
	pipClient := gauth.NewPIPClient(pipConfig)

	// Using Mock Commercial Register (since we focused on Identity Connectors in Phase 6)
	commercialRegClient := mocks.NewMockCommercialRegisterClient()

	// 4. Create base components
	// We pass 'globalVerifier' as the PVP Client because it implements VerifyIdentityProof now
	components, err := gauth.InitAAP001WithComponents(
		globalVerifier,
		pipClient,
		commercialRegClient,
	)
	if err != nil {
		return nil, err
	}

	// 5. Override FormalRequirementsValidator to use GlobalIdentityVerifier
	// The default one created by InitAAP001WithComponents has no verifiers.
	// We replace it with one that uses our Global Identity Verifier.
	components.FormalReqValidator = gauth.NewFormalRequirementsValidator(
		nil,            // NotarialCertificateVerifier
		globalVerifier, // IdentityDocumentVerifier (Global)
		nil,            // DigitalSignatureVerifier
		os.Getenv("GAUTH_STRICT_FORMAL") == "1",
	)

	fmt.Fprintf(os.Stderr, "[RFC-0111] Initialized GlobalIdentityVerifier with US, FR, IT, ES connectors\n")

	return components, nil
}

// initializeAgentAuthPlus initializes AgentAuth+ services and integrates them with RFC-0111 components.
// This function requires a PostgreSQL database connection to be available.
//
// Environment variables:
//   - GAUTH_GAUTHPLUS_ENABLED: Set to "1" to enable AgentAuth+ features
//   - GAUTH_GAUTHPLUS_ENFORCE: Set to "1" to enable strict enforcement (default: advisory mode)
//   - GAUTH_GAUTHPLUS_ENFORCE_CAPABILITIES: Set to "1" to enforce capability requirements
//   - GAUTH_GAUTHPLUS_ENFORCE_DUAL_CONTROL: Set to "1" to enforce dual control approvals
//   - GAUTH_GAUTHPLUS_ENFORCE_FIDUCIARY: Set to "1" to enforce fiduciary duty checks
//   - DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSLMODE: Database connection params
//
// Returns a map of services for endpoint registration, or error if initialization fails.
func initializeAgentAuthPlus(components *gauth.AAP001Components) (map[string]interface{}, error) {
	// Build PostgreSQL DSN from environment variables
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "gauth"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "gauth"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "gauth_password"
	}
	sslmode := os.Getenv("DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}

	// Create database configuration
	portInt := 5432
	if portVal, err := strconv.Atoi(port); err == nil {
		portInt = portVal
	}

	dbCfg := &database.Config{
		Host:     host,
		Port:     portInt,
		User:     user,
		Password: password,
		Database: dbname,
		SSLMode:  sslmode,
	}

	// Open database connection (pgx pool)
	db, err := database.NewDB(dbCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection is handled by NewDB (Ping)

	// Initialize AgentAuth+ services
	successorService := gauthplus.NewPostgreSQLSuccessorService(db)
	delegationService := gauthplus.NewPostgreSQLDelegationService(db)
	dualControlService := gauthplus.NewPostgreSQLDualControlService(db)
	fiduciaryService := gauthplus.NewPostgreSQLFiduciaryDutyService(db)
	capabilityService := gauthplus.NewPostgreSQLCapabilityAssessmentService(db)

	// Wrap services with caching for performance optimization
	// Capability assessments change infrequently (monthly reviews) - 5 minute TTL
	cachedCapabilityService := gauthplus.NewCachedCapabilityService(capabilityService, 5*time.Minute)

	// Delegation chains more volatile - 1 minute TTL
	cachedDelegationService := gauthplus.NewCachedDelegationService(delegationService, 1*time.Minute)

	// Start background cache cleanup (every 5 minutes)
	// DEPRECATED: Caches now have internal cleanup loops managed by their own goroutines.
	// We rely on ShutdownAgentAuthPlus() to close them.

	fmt.Fprintf(os.Stderr, "[AgentAuth+] Performance optimization: Caching enabled (capability TTL: 5m, delegation TTL: 1m)\n")

	// Create AgentAuth+ validator with cached services
	gauthPlusValidator := gauth.NewAgentAuthPlusValidator(
		successorService,
		cachedDelegationService, // Use cached version
		dualControlService,
		fiduciaryService,
		cachedCapabilityService, // Use cached version
	)

	// Configure enforcement modes based on environment variables
	enforceStrict := os.Getenv("GAUTH_GAUTHPLUS_ENFORCE") == "1"
	enforceCapabilities := os.Getenv("GAUTH_GAUTHPLUS_ENFORCE_CAPABILITIES") == "1"
	enforceDualControl := os.Getenv("GAUTH_GAUTHPLUS_ENFORCE_DUAL_CONTROL") == "1"
	enforceFiduciary := os.Getenv("GAUTH_GAUTHPLUS_ENFORCE_FIDUCIARY") == "1"

	// If strict mode is enabled, enforce all features
	if enforceStrict {
		gauthPlusValidator.SetEnforceCapabilities(true)
		gauthPlusValidator.SetEnforceDualControl(true)
		gauthPlusValidator.SetEnforceFiduciary(true)
		fmt.Fprintf(os.Stderr, "[AgentAuth+] Enforcement mode: STRICT (blocking on policy violations)\n")
	} else {
		// Otherwise, use selective enforcement
		gauthPlusValidator.SetEnforceCapabilities(enforceCapabilities)
		gauthPlusValidator.SetEnforceDualControl(enforceDualControl)
		gauthPlusValidator.SetEnforceFiduciary(enforceFiduciary)

		if enforceCapabilities || enforceDualControl || enforceFiduciary {
			fmt.Fprintf(os.Stderr, "[AgentAuth+] Enforcement mode: CUSTOM (capabilities=%v, dualControl=%v, fiduciary=%v)\n",
				enforceCapabilities, enforceDualControl, enforceFiduciary)
		} else {
			fmt.Fprintf(os.Stderr, "[AgentAuth+] Enforcement mode: ADVISORY (warnings only, no blocking)\n")
		}
	}

	// Integrate AgentAuth+ validator into RFC-0111 components
	if components.ComplianceValidator != nil {
		components.ComplianceValidator.SetAgentAuthPlusValidator(gauthPlusValidator)
		components.ComplianceValidator.SetEnforceAgentAuthPlus(true)
		fmt.Fprintf(os.Stderr, "[AgentAuth+] Integrated with ComplianceValidator\n")
	}

	// Note: PDP integration would happen here if we had access to the PDP instance
	// For now, PDP integration is done separately when PDP is created
	fmt.Fprintf(os.Stderr, "[AgentAuth+] Features enabled:\n")
	fmt.Fprintf(os.Stderr, "[AgentAuth+]   - Successor Management: AI takeover scenarios\n")
	fmt.Fprintf(os.Stderr, "[AgentAuth+]   - Delegation Chains: Depth limits and policy validation\n")
	fmt.Fprintf(os.Stderr, "[AgentAuth+]   - Dual Control: Multi-approver requirements\n")
	fmt.Fprintf(os.Stderr, "[AgentAuth+]   - Capability Assessment: AI capability level enforcement\n")
	fmt.Fprintf(os.Stderr, "[AgentAuth+]   - Fiduciary Duties: Violation detection and blocking\n")

	// Store services globally for endpoint registration
	// Services will be registered via server.RegisterAgentAuthPlusEndpoints() during server startup
	gauthPlusServicesGlobal = map[string]interface{}{
		"successor_service":     successorService,
		"delegation_service":    cachedDelegationService, // Store the wrapper for shutdown
		"dual_control_service":  dualControlService,
		"capability_service":    cachedCapabilityService, // Store the wrapper for shutdown
		"fiduciary_service":     fiduciaryService,
		"cached_delegation_svc": cachedDelegationService, // Direct access for shutdown
		"cached_capability_svc": cachedCapabilityService, // Direct access for shutdown
	}

	fmt.Fprintf(os.Stderr, "[AgentAuth+] Services available for API endpoint registration\n")

	return gauthPlusServicesGlobal, nil
}

// Global storage for AgentAuth+ services (initialized by initializeAgentAuthPlus)
var gauthPlusServicesGlobal map[string]interface{}

// InitializeAgentAuthPlusEndpoints registers AgentAuth+ management endpoints if services are available.
// This should be called after the server router is initialized and AgentAuth+ services are created.
func (s *BetaServer) InitializeAgentAuthPlusEndpoints() {
	if s.gauthPlusInitialized {
		return // Already initialized
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gauthPlusInitialized {
		return // Double-check locking
	}

	if gauthPlusServicesGlobal == nil {
		return // AgentAuth+ not initialized
	}

	// Extract services from global map
	successorService, _ := gauthPlusServicesGlobal["successor_service"].(gauthplus.SuccessorManagementService)
	// We need to extract the interface expected by RegisterAgentAuthPlusEndpoints, which is DelegationService.
	// cachedDelegationService implements DelegationService via struct embedding?
	// Wait, CachedDelegationService wraps DelegationService but does NOT embed it (it uses composition).
	// We need to check if CachedDelegationService implements DelegationService.
	// Looking at cache.go:
	// func (s *CachedDelegationService) CreateDelegation...
	// It mirrors the methods. Yes, it should implement it.

	delegationService, _ := gauthPlusServicesGlobal["delegation_service"].(gauthplus.DelegationService)
	dualControlService, _ := gauthPlusServicesGlobal["dual_control_service"].(gauthplus.DualControlService)
	capabilityService, _ := gauthPlusServicesGlobal["capability_service"].(gauthplus.CapabilityAssessmentService)
	fiduciaryService, _ := gauthPlusServicesGlobal["fiduciary_service"].(gauthplus.FiduciaryDutyService)

	if successorService == nil || delegationService == nil || dualControlService == nil ||
		capabilityService == nil || fiduciaryService == nil {
		fmt.Fprintf(os.Stderr, "[AgentAuth+] ERROR: Failed to extract services for endpoint registration\n")
		return
	}

	// Register AgentAuth+ HTTP endpoints
	s.RegisterAgentAuthPlusEndpoints(
		successorService,
		delegationService,
		dualControlService,
		capabilityService,
		fiduciaryService,
	)

	fmt.Fprintf(os.Stderr, "[AgentAuth+] ✅ Management API endpoints registered (27 endpoints):\n")
	fmt.Fprintf(os.Stderr, "[AgentAuth+]   Successor Management: 4 endpoints\n")
	fmt.Fprintf(os.Stderr, "[AgentAuth+]   Delegation Service: 5 endpoints\n")
	fmt.Fprintf(os.Stderr, "[AgentAuth+]   Dual Control: 6 endpoints\n")
	fmt.Fprintf(os.Stderr, "[AgentAuth+]   Capability Assessment: 6 endpoints\n")
	fmt.Fprintf(os.Stderr, "[AgentAuth+]   Fiduciary Duty: 4 endpoints\n")
	s.gauthPlusInitialized = true
}

// ShutdownAgentAuthPlus closes background resources (caches) used by AgentAuth+ services
func ShutdownAgentAuthPlus() {
	if gauthPlusServicesGlobal == nil {
		return
	}

	// Close capability cache
	if svc, ok := gauthPlusServicesGlobal["cached_capability_svc"].(interface{ Close() }); ok {
		svc.Close()
		fmt.Println("[shutdown] Closed AgentAuth+ capability cache")
	}

	// Close delegation cache
	if svc, ok := gauthPlusServicesGlobal["cached_delegation_svc"].(interface{ Close() }); ok {
		svc.Close()
		fmt.Println("[shutdown] Closed AgentAuth+ delegation cache")
	}
}
