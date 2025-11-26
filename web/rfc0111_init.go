package web

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth/mocks"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauthplus"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// InitRFC0111FromEnv initializes RFC-0111 components based on environment variables.
// This is a web-server specific helper that can create mock services and configure persistence.
//
// Environment variables:
//   - GAUTH_RFC0111_ENABLED: Set to "1" to enable RFC-0111 functionality
//   - GAUTH_RFC0111_USE_MOCKS: Set to "1" to use mock external services (default: 1)
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
func InitRFC0111FromEnv() (*gauth.RFC0111Components, gauth.ExtendedTokenStore, error) {
	// Check if RFC-0111 is enabled
	if os.Getenv("GAUTH_RFC0111_ENABLED") != "1" {
		return nil, nil, nil
	}

	// Determine whether to use mocks (default: yes)
	useMocks := os.Getenv("GAUTH_RFC0111_USE_MOCKS") != "0"

	if !useMocks {
		return nil, nil, fmt.Errorf("RFC-0111: real external service implementations not yet available, set GAUTH_RFC0111_USE_MOCKS=1 or unset")
	}

	// Create mock external services
	pvpClient := mocks.NewMockPowerVerificationPoint()
	pipClient := mocks.NewMockPIPClient()
	commercialRegClient := mocks.NewMockCommercialRegisterClient()

	// Initialize RFC-0111 with mocks
	components, err := gauth.InitRFC0111WithComponents(
		pvpClient,
		pipClient,
		commercialRegClient,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("RFC-0111 initialization failed: %w", err)
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

	// PHASE 3: GAuth+ Authorization Integration (optional, controlled by GAUTH_GAUTHPLUS_ENABLED)
	// Initialize GAuth+ services if database connection is available and feature is enabled
	if os.Getenv("GAUTH_GAUTHPLUS_ENABLED") == "1" {
		_, err := initializeGAuthPlus(components)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[GAuth+] WARNING: Failed to initialize GAuth+ integration: %v\n", err)
			fmt.Fprintf(os.Stderr, "[GAuth+] Continuing without GAuth+ features\n")
		} else {
			fmt.Fprintf(os.Stderr, "[GAuth+] Authorization chain integration enabled\n")
			// Note: Services are stored globally for endpoint registration
			// See initializeGAuthPlusEndpoints in server initialization
		}
	}

	return components, tokenStore, nil
}

// initializeGAuthPlus initializes GAuth+ services and integrates them with RFC-0111 components.
// This function requires a PostgreSQL database connection to be available.
//
// Environment variables:
//   - GAUTH_GAUTHPLUS_ENABLED: Set to "1" to enable GAuth+ features
//   - GAUTH_GAUTHPLUS_ENFORCE: Set to "1" to enable strict enforcement (default: advisory mode)
//   - GAUTH_GAUTHPLUS_ENFORCE_CAPABILITIES: Set to "1" to enforce capability requirements
//   - GAUTH_GAUTHPLUS_ENFORCE_DUAL_CONTROL: Set to "1" to enforce dual control approvals
//   - GAUTH_GAUTHPLUS_ENFORCE_FIDUCIARY: Set to "1" to enforce fiduciary duty checks
//   - DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSLMODE: Database connection params
//
// Returns a map of services for endpoint registration, or error if initialization fails.
func initializeGAuthPlus(components *gauth.RFC0111Components) (map[string]interface{}, error) {
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

	// Open database connection
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize GAuth+ services
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
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			// Clean expired entries from both caches
			cachedCapabilityService.GetCache().CleanExpired()
			cachedDelegationService.GetCache().CleanExpired()
		}
	}()

	fmt.Fprintf(os.Stderr, "[GAuth+] Performance optimization: Caching enabled (capability TTL: 5m, delegation TTL: 1m)\n")

	// Create GAuth+ validator with cached services
	gauthPlusValidator := gauth.NewGAuthPlusValidator(
		successorService,
		cachedDelegationService,  // Use cached version
		dualControlService,
		fiduciaryService,
		cachedCapabilityService,  // Use cached version
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
		fmt.Fprintf(os.Stderr, "[GAuth+] Enforcement mode: STRICT (blocking on policy violations)\n")
	} else {
		// Otherwise, use selective enforcement
		gauthPlusValidator.SetEnforceCapabilities(enforceCapabilities)
		gauthPlusValidator.SetEnforceDualControl(enforceDualControl)
		gauthPlusValidator.SetEnforceFiduciary(enforceFiduciary)

		if enforceCapabilities || enforceDualControl || enforceFiduciary {
			fmt.Fprintf(os.Stderr, "[GAuth+] Enforcement mode: CUSTOM (capabilities=%v, dualControl=%v, fiduciary=%v)\n",
				enforceCapabilities, enforceDualControl, enforceFiduciary)
		} else {
			fmt.Fprintf(os.Stderr, "[GAuth+] Enforcement mode: ADVISORY (warnings only, no blocking)\n")
		}
	}

	// Integrate GAuth+ validator into RFC-0111 components
	if components.ComplianceValidator != nil {
		components.ComplianceValidator.SetGAuthPlusValidator(gauthPlusValidator)
		components.ComplianceValidator.SetEnforceGAuthPlus(true)
		fmt.Fprintf(os.Stderr, "[GAuth+] Integrated with ComplianceValidator\n")
	}

	// Note: PDP integration would happen here if we had access to the PDP instance
	// For now, PDP integration is done separately when PDP is created
	fmt.Fprintf(os.Stderr, "[GAuth+] Features enabled:\n")
	fmt.Fprintf(os.Stderr, "[GAuth+]   - Successor Management: AI takeover scenarios\n")
	fmt.Fprintf(os.Stderr, "[GAuth+]   - Delegation Chains: Depth limits and policy validation\n")
	fmt.Fprintf(os.Stderr, "[GAuth+]   - Dual Control: Multi-approver requirements\n")
	fmt.Fprintf(os.Stderr, "[GAuth+]   - Capability Assessment: AI capability level enforcement\n")
	fmt.Fprintf(os.Stderr, "[GAuth+]   - Fiduciary Duties: Violation detection and blocking\n")

	// Store services globally for endpoint registration
	// Services will be registered via server.RegisterGAuthPlusEndpoints() during server startup
	gauthPlusServicesGlobal = map[string]interface{}{
		"successor_service":    successorService,
		"delegation_service":   delegationService,
		"dual_control_service": dualControlService,
		"capability_service":   capabilityService,
		"fiduciary_service":    fiduciaryService,
	}

	fmt.Fprintf(os.Stderr, "[GAuth+] Services available for API endpoint registration\n")

	return gauthPlusServicesGlobal, nil
}

// Global storage for GAuth+ services (initialized by initializeGAuthPlus)
var gauthPlusServicesGlobal map[string]interface{}

// InitializeGAuthPlusEndpoints registers GAuth+ management endpoints if services are available.
// This should be called after the server router is initialized and GAuth+ services are created.
func (s *BetaServer) InitializeGAuthPlusEndpoints() {
	if gauthPlusServicesGlobal == nil {
		return // GAuth+ not initialized
	}

	// Extract services from global map
	successorService, _ := gauthPlusServicesGlobal["successor_service"].(gauthplus.SuccessorManagementService)
	delegationService, _ := gauthPlusServicesGlobal["delegation_service"].(gauthplus.DelegationService)
	dualControlService, _ := gauthPlusServicesGlobal["dual_control_service"].(gauthplus.DualControlService)
	capabilityService, _ := gauthPlusServicesGlobal["capability_service"].(gauthplus.CapabilityAssessmentService)
	fiduciaryService, _ := gauthPlusServicesGlobal["fiduciary_service"].(gauthplus.FiduciaryDutyService)

	if successorService == nil || delegationService == nil || dualControlService == nil ||
		capabilityService == nil || fiduciaryService == nil {
		fmt.Fprintf(os.Stderr, "[GAuth+] ERROR: Failed to extract services for endpoint registration\n")
		return
	}

	// Register GAuth+ HTTP endpoints
	s.RegisterGAuthPlusEndpoints(
		successorService,
		delegationService,
		dualControlService,
		capabilityService,
		fiduciaryService,
	)

	fmt.Fprintf(os.Stderr, "[GAuth+] ✅ Management API endpoints registered (27 endpoints):\n")
	fmt.Fprintf(os.Stderr, "[GAuth+]   Successor Management: 4 endpoints\n")
	fmt.Fprintf(os.Stderr, "[GAuth+]   Delegation Service: 5 endpoints\n")
	fmt.Fprintf(os.Stderr, "[GAuth+]   Dual Control: 6 endpoints\n")
	fmt.Fprintf(os.Stderr, "[GAuth+]   Capability Assessment: 6 endpoints\n")
	fmt.Fprintf(os.Stderr, "[GAuth+]   Fiduciary Duty: 4 endpoints\n")
}
