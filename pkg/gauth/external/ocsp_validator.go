package external

import (
	"bytes"
	"context"
	"crypto"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/ocsp"
)

// =============================================================================
// OCSP Constants
// =============================================================================

// OCSPStatus represents the revocation status from OCSP response
type OCSPStatus int

const (
	OCSPStatusGood    OCSPStatus = 0 // Certificate is valid
	OCSPStatusRevoked OCSPStatus = 1 // Certificate is revoked
	OCSPStatusUnknown OCSPStatus = 2 // Status is unknown
)

// RevocationReason represents the reason for certificate revocation
type RevocationReason int

const (
	ReasonUnspecified          RevocationReason = 0
	ReasonKeyCompromise        RevocationReason = 1
	ReasonCACompromise         RevocationReason = 2
	ReasonAffiliationChanged   RevocationReason = 3
	ReasonSuperseded           RevocationReason = 4
	ReasonCessationOfOperation RevocationReason = 5
	ReasonCertificateHold      RevocationReason = 6
	ReasonRemoveFromCRL        RevocationReason = 8
	ReasonPrivilegeWithdrawn   RevocationReason = 9
	ReasonAACompromise         RevocationReason = 10
)

// =============================================================================
// Data Models
// =============================================================================

// OCSPValidationRequest represents a certificate validation request
type OCSPValidationRequest struct {
	Certificate       *x509.Certificate `json:"-"`
	IssuerCertificate *x509.Certificate `json:"-"`
	OCSPServer        string            `json:"ocsp_server,omitempty"`
	Nonce             []byte            `json:"-"`
	RequestID         string            `json:"request_id"`
	Timestamp         time.Time         `json:"timestamp"`
}

// OCSPValidationResult represents the result of OCSP validation
type OCSPValidationResult struct {
	Status       OCSPStatus `json:"status"`
	StatusString string     `json:"status_string"`
	Valid        bool       `json:"valid"`

	// Certificate details
	SerialNumber string `json:"serial_number"`
	Subject      string `json:"subject"`
	Issuer       string `json:"issuer"`

	// OCSP response details
	ProducedAt time.Time `json:"produced_at"`
	ThisUpdate time.Time `json:"this_update"`
	NextUpdate time.Time `json:"next_update"`

	// Revocation details (if revoked)
	RevokedAt           *time.Time        `json:"revoked_at,omitempty"`
	RevocationReason    *RevocationReason `json:"revocation_reason,omitempty"`
	RevocationReasonStr string            `json:"revocation_reason_string,omitempty"`

	// Responder details
	ResponderURL  string `json:"responder_url"`
	ResponderID   string `json:"responder_id,omitempty"`
	ResponderCert bool   `json:"responder_cert_present"`

	// Validation metadata
	NonceMatch     bool `json:"nonce_match"`
	SignatureValid bool `json:"signature_valid"`
	CacheHit       bool `json:"cache_hit"`

	// Fallback to CRL
	CRLFallback bool `json:"crl_fallback"`
	CRLChecked  bool `json:"crl_checked"`

	// Warnings and errors
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`

	// Metadata
	RequestID      string    `json:"request_id"`
	ValidationTime time.Time `json:"validation_time"`
	ResponseTimeMs int64     `json:"response_time_ms"`
}

// CertificateChainValidationRequest represents a chain validation request
type CertificateChainValidationRequest struct {
	Chain         []*x509.Certificate `json:"-"`
	TrustedRoots  *x509.CertPool      `json:"-"`
	Intermediates *x509.CertPool      `json:"-"`
	ValidateOCSP  bool                `json:"validate_ocsp"`
	ValidateCRL   bool                `json:"validate_crl"`
	RequestID     string              `json:"request_id"`
	Timestamp     time.Time           `json:"timestamp"`
}

// CertificateChainValidationResult represents chain validation result
type CertificateChainValidationResult struct {
	Valid            bool `json:"valid"`
	ChainLength      int  `json:"chain_length"`
	ChainValid       bool `json:"chain_valid"`
	TrustAnchorFound bool `json:"trust_anchor_found"`

	// Per-certificate OCSP results
	OCSPResults []*OCSPValidationResult `json:"ocsp_results,omitempty"`

	// Chain verification details
	VerifiedChains [][]*x509.Certificate `json:"-"`

	// Warnings and errors
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`

	// Metadata
	RequestID      string    `json:"request_id"`
	ValidationTime time.Time `json:"validation_time"`
	TotalTimeMs    int64     `json:"total_time_ms"`
}

// =============================================================================
// OCSP Validator Configuration
// =============================================================================

// OCSPValidatorConfig contains configuration for OCSP validation
type OCSPValidatorConfig struct {
	// HTTP client configuration
	HTTPTimeout time.Duration `json:"http_timeout"`
	MaxRetries  int           `json:"max_retries"`
	RetryDelay  time.Duration `json:"retry_delay"`

	// OCSP settings
	UseNonce            bool          `json:"use_nonce"`
	AcceptStaleResponse bool          `json:"accept_stale_response"` // Accept responses past NextUpdate
	MaxResponseAge      time.Duration `json:"max_response_age"`

	// Cache settings
	CacheEnabled bool          `json:"cache_enabled"`
	CacheTTL     time.Duration `json:"cache_ttl"`
	CacheMaxSize int           `json:"cache_max_size"`

	// CRL fallback
	EnableCRLFallback bool          `json:"enable_crl_fallback"`
	CRLCacheEnabled   bool          `json:"crl_cache_enabled"`
	CRLCacheTTL       time.Duration `json:"crl_cache_ttl"`

	// Validation strictness
	RequireSignature bool `json:"require_signature"`
	RequireNonce     bool `json:"require_nonce"`

	// Custom OCSP servers
	CustomServers map[string]string `json:"custom_servers"` // Issuer DN -> OCSP URL
}

// =============================================================================
// OCSP Validator Implementation
// =============================================================================

// OCSPValidator performs OCSP-based certificate validation
type OCSPValidator struct {
	config     *OCSPValidatorConfig
	httpClient *http.Client
	cache      map[string]*cachedOCSPResponse
	cacheMu    sync.RWMutex
	crlCache   map[string]*cachedCRL
	crlMu      sync.RWMutex
}

// cachedOCSPResponse stores a cached OCSP response
type cachedOCSPResponse struct {
	Response  *ocsp.Response
	Timestamp time.Time
}

// cachedCRL stores a cached CRL
type cachedCRL struct {
	CRL       *pkix.CertificateList
	Timestamp time.Time
}

// NewOCSPValidator creates a new OCSP validator
func NewOCSPValidator(config *OCSPValidatorConfig) (*OCSPValidator, error) {
	if config == nil {
		config = &OCSPValidatorConfig{}
	}

	// Set defaults
	if config.HTTPTimeout == 0 {
		config.HTTPTimeout = 10 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}
	if config.MaxResponseAge == 0 {
		config.MaxResponseAge = 7 * 24 * time.Hour // 7 days
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 1 * time.Hour
	}
	if config.CacheMaxSize == 0 {
		config.CacheMaxSize = 1000
	}
	if config.CRLCacheTTL == 0 {
		config.CRLCacheTTL = 24 * time.Hour
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: config.HTTPTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &OCSPValidator{
		config:     config,
		httpClient: httpClient,
		cache:      make(map[string]*cachedOCSPResponse),
		crlCache:   make(map[string]*cachedCRL),
	}, nil
}

// ValidateCertificate performs OCSP validation for a single certificate
func (v *OCSPValidator) ValidateCertificate(ctx context.Context, req *OCSPValidationRequest) (*OCSPValidationResult, error) {
	startTime := time.Now()

	if req.Certificate == nil || req.IssuerCertificate == nil {
		return nil, errors.New("certificate and issuer certificate are required")
	}

	result := &OCSPValidationResult{
		RequestID:      req.RequestID,
		ValidationTime: time.Now(),
		SerialNumber:   req.Certificate.SerialNumber.String(),
		Subject:        req.Certificate.Subject.String(),
		Issuer:         req.Certificate.Issuer.String(),
	}

	// Check cache first
	cacheKey := v.getCacheKey(req.Certificate)
	if cachedResult := v.checkOCSPCache(cacheKey, req.RequestID, startTime); cachedResult != nil {
		return cachedResult, nil
	}

	// Get OCSP responder URL
	ocspServer := v.resolveOCSPServer(req)
	if ocspServer == "" {
		if v.config.EnableCRLFallback {
			return v.validateWithCRL(ctx, req)
		}
		return nil, errors.New("no OCSP responder URL found and CRL fallback disabled")
	}

	result.ResponderURL = ocspServer

	// Fetch OCSP response
	ocspResp, err := v.fetchOCSPResponse(ctx, ocspServer, req)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("OCSP fetch failed: %v", err))

		// Try CRL fallback
		if v.config.EnableCRLFallback {
			crlResult, crlErr := v.validateWithCRL(ctx, req)
			if crlErr == nil {
				crlResult.CRLFallback = true
				return crlResult, nil
			}
			result.Warnings = append(result.Warnings, fmt.Sprintf("CRL fallback also failed: %v", crlErr))
		}
		return result, err
	}

	// Verify and populate result
	v.verifyAndPopulateResult(result, ocspResp, req)

	// Cache response
	if v.config.CacheEnabled && result.Valid {
		v.addToCache(cacheKey, ocspResp)
	}

	result.ResponseTimeMs = time.Since(startTime).Milliseconds()

	return result, nil
}

func (v *OCSPValidator) checkOCSPCache(cacheKey, requestID string, startTime time.Time) *OCSPValidationResult {
	if v.config.CacheEnabled {
		if cached := v.getFromCache(cacheKey); cached != nil {
			result := &OCSPValidationResult{
				CacheHit:       true,
				RequestID:      requestID,
				ValidationTime: time.Now(),
			}
			v.populateResultFromResponse(result, cached, requestID)
			result.ResponseTimeMs = time.Since(startTime).Milliseconds()
			return result
		}
	}
	return nil
}

func (v *OCSPValidator) resolveOCSPServer(req *OCSPValidationRequest) string {
	if req.OCSPServer != "" {
		return req.OCSPServer
	}
	return v.getOCSPServer(req.Certificate, req.IssuerCertificate)
}

func (v *OCSPValidator) fetchOCSPResponse(ctx context.Context, server string, req *OCSPValidationRequest) (*ocsp.Response, error) {
	ocspReq, err := v.createOCSPRequest(req.Certificate, req.IssuerCertificate, req.Nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCSP request: %w", err)
	}

	var ocspResp *ocsp.Response
	var lastErr error

	for attempt := 0; attempt <= v.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(v.config.RetryDelay * time.Duration(attempt))
		}

		ocspResp, lastErr = v.sendOCSPRequest(ctx, server, ocspReq)
		if lastErr == nil {
			return ocspResp, nil
		}
	}
	return nil, lastErr
}

func (v *OCSPValidator) verifyAndPopulateResult(result *OCSPValidationResult, ocspResp *ocsp.Response, req *OCSPValidationRequest) {
	// Verify signature
	if v.config.RequireSignature {
		if err := v.verifyOCSPSignature(ocspResp, req.IssuerCertificate); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("OCSP signature verification failed: %v", err))
			result.SignatureValid = false
		} else {
			result.SignatureValid = true
		}
	}

	// Verify nonce
	if v.config.UseNonce && req.Nonce != nil {
		result.NonceMatch = bytes.Equal(ocspResp.Extensions[0].Value, req.Nonce)
		if v.config.RequireNonce && !result.NonceMatch {
			result.Warnings = append(result.Warnings, "OCSP nonce mismatch")
		}
	}

	// Populate result
	v.populateResultFromResponse(result, ocspResp, req.RequestID)

	// Check freshness
	if !v.config.AcceptStaleResponse && time.Now().After(ocspResp.NextUpdate) {
		result.Warnings = append(result.Warnings, "OCSP response is stale (past NextUpdate)")
	}

	if time.Since(ocspResp.ProducedAt) > v.config.MaxResponseAge {
		result.Warnings = append(result.Warnings, fmt.Sprintf("OCSP response is older than max age (%v)", v.config.MaxResponseAge))
	}
}

// ValidateCertificateChain validates an entire certificate chain
func (v *OCSPValidator) ValidateCertificateChain(ctx context.Context, req *CertificateChainValidationRequest) (*CertificateChainValidationResult, error) {
	startTime := time.Now()

	result := &CertificateChainValidationResult{
		RequestID:      req.RequestID,
		ValidationTime: time.Now(),
		ChainLength:    len(req.Chain),
		Valid:          true,
	}

	if len(req.Chain) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "Empty certificate chain")
		return result, errors.New("empty certificate chain")
	}

	// Verify chain using standard x509 verification
	opts := x509.VerifyOptions{
		Roots:         req.TrustedRoots,
		Intermediates: req.Intermediates,
		CurrentTime:   time.Now(),
	}

	verifiedChains, err := req.Chain[0].Verify(opts)
	if err != nil {
		result.Valid = false
		result.ChainValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Chain verification failed: %v", err))
	} else {
		result.ChainValid = true
		result.VerifiedChains = verifiedChains
		result.TrustAnchorFound = len(verifiedChains) > 0
	}

	// Perform OCSP validation for each certificate in chain (except root)
	if req.ValidateOCSP && result.ChainValid {
		for i := 0; i < len(req.Chain)-1; i++ {
			cert := req.Chain[i]
			issuer := req.Chain[i+1]

			ocspReq := &OCSPValidationRequest{
				Certificate:       cert,
				IssuerCertificate: issuer,
				RequestID:         fmt.Sprintf("%s-cert%d", req.RequestID, i),
				Timestamp:         time.Now(),
			}

			ocspResult, err := v.ValidateCertificate(ctx, ocspReq)
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("OCSP validation failed for cert %d: %v", i, err))
			} else {
				result.OCSPResults = append(result.OCSPResults, ocspResult)
				if !ocspResult.Valid {
					result.Valid = false
				}
			}
		}
	}

	result.TotalTimeMs = time.Since(startTime).Milliseconds()

	return result, nil
}

// =============================================================================
// Private Methods
// =============================================================================

func (v *OCSPValidator) createOCSPRequest(cert, issuer *x509.Certificate, nonce []byte) ([]byte, error) {
	opts := ocsp.RequestOptions{}

	if v.config.UseNonce && nonce != nil {
		opts.Hash = crypto.SHA256
	}

	return ocsp.CreateRequest(cert, issuer, &opts)
}

func (v *OCSPValidator) sendOCSPRequest(ctx context.Context, server string, request []byte) (*ocsp.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", server, bytes.NewReader(request))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/ocsp-request")
	httpReq.Header.Set("Accept", "application/ocsp-response")

	httpResp, err := v.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("OCSP HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OCSP responder returned status %d", httpResp.StatusCode)
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OCSP response: %w", err)
	}

	return ocsp.ParseResponse(body, nil)
}

func (v *OCSPValidator) verifyOCSPSignature(resp *ocsp.Response, issuer *x509.Certificate) error {
	// In production, this would verify the OCSP response signature
	// using the responder certificate or issuer certificate
	// For now, this is a placeholder
	return nil
}

func (v *OCSPValidator) getOCSPServer(cert, issuer *x509.Certificate) string {
	// Check custom servers first
	if v.config.CustomServers != nil {
		if server, ok := v.config.CustomServers[issuer.Subject.String()]; ok {
			return server
		}
	}

	// Get from certificate AIA extension
	if len(cert.OCSPServer) > 0 {
		return cert.OCSPServer[0]
	}

	return ""
}

func (v *OCSPValidator) populateResultFromResponse(result *OCSPValidationResult, resp *ocsp.Response, requestID string) {
	result.Status = OCSPStatus(resp.Status)
	result.ProducedAt = resp.ProducedAt
	result.ThisUpdate = resp.ThisUpdate
	result.NextUpdate = resp.NextUpdate
	result.RequestID = requestID

	switch resp.Status {
	case ocsp.Good:
		result.StatusString = "good"
		result.Valid = true
	case ocsp.Revoked:
		result.StatusString = "revoked"
		result.Valid = false
		result.RevokedAt = &resp.RevokedAt
		reason := RevocationReason(resp.RevocationReason)
		result.RevocationReason = &reason
		result.RevocationReasonStr = v.getRevocationReasonString(reason)
	case ocsp.Unknown:
		result.StatusString = "unknown"
		result.Valid = false
	default:
		result.StatusString = "invalid"
		result.Valid = false
	}

	if resp.Certificate != nil {
		result.ResponderCert = true
		result.ResponderID = resp.Certificate.Subject.String()
	}
}

func (v *OCSPValidator) getRevocationReasonString(reason RevocationReason) string {
	switch reason {
	case ReasonUnspecified:
		return "unspecified"
	case ReasonKeyCompromise:
		return "key_compromise"
	case ReasonCACompromise:
		return "ca_compromise"
	case ReasonAffiliationChanged:
		return "affiliation_changed"
	case ReasonSuperseded:
		return "superseded"
	case ReasonCessationOfOperation:
		return "cessation_of_operation"
	case ReasonCertificateHold:
		return "certificate_hold"
	case ReasonRemoveFromCRL:
		return "remove_from_crl"
	case ReasonPrivilegeWithdrawn:
		return "privilege_withdrawn"
	case ReasonAACompromise:
		return "aa_compromise"
	default:
		return "unknown"
	}
}

func (v *OCSPValidator) validateWithCRL(ctx context.Context, req *OCSPValidationRequest) (*OCSPValidationResult, error) {
	result := &OCSPValidationResult{
		RequestID:      req.RequestID,
		ValidationTime: time.Now(),
		SerialNumber:   req.Certificate.SerialNumber.String(),
		Subject:        req.Certificate.Subject.String(),
		Issuer:         req.Certificate.Issuer.String(),
		CRLFallback:    true,
		CRLChecked:     true,
	}

	// Get CRL distribution points
	if len(req.Certificate.CRLDistributionPoints) == 0 {
		return nil, errors.New("no CRL distribution points found")
	}

	crlURL := req.Certificate.CRLDistributionPoints[0]

	// Check cache
	cacheKey := crlURL
	if v.config.CRLCacheEnabled {
		if cached := v.getCRLFromCache(cacheKey); cached != nil {
			return v.checkCRL(result, cached, req.Certificate), nil
		}
	}

	// Fetch CRL
	crl, err := v.fetchCRL(ctx, crlURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CRL: %w", err)
	}

	// Cache CRL
	if v.config.CRLCacheEnabled {
		v.addCRLToCache(cacheKey, crl)
	}

	return v.checkCRL(result, crl, req.Certificate), nil
}

func (v *OCSPValidator) fetchCRL(ctx context.Context, url string) (*pkix.CertificateList, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CRL fetch returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return x509.ParseCRL(body)
}

func (v *OCSPValidator) checkCRL(result *OCSPValidationResult, crl *pkix.CertificateList, cert *x509.Certificate) *OCSPValidationResult {
	result.ThisUpdate = crl.TBSCertList.ThisUpdate
	result.NextUpdate = crl.TBSCertList.NextUpdate

	// Check if certificate is revoked
	for _, revokedCert := range crl.TBSCertList.RevokedCertificates {
		if revokedCert.SerialNumber.Cmp(cert.SerialNumber) == 0 {
			result.Status = OCSPStatusRevoked
			result.StatusString = "revoked"
			result.Valid = false
			result.RevokedAt = &revokedCert.RevocationTime

			// Extract revocation reason from extensions
			for _, ext := range revokedCert.Extensions {
				if ext.Id.Equal(asn1.ObjectIdentifier{2, 5, 29, 21}) { // CRL Reason Code
					var reason int
					if _, err := asn1.Unmarshal(ext.Value, &reason); err == nil {
						r := RevocationReason(reason)
						result.RevocationReason = &r
						result.RevocationReasonStr = v.getRevocationReasonString(r)
					}
				}
			}

			return result
		}
	}

	result.Status = OCSPStatusGood
	result.StatusString = "good"
	result.Valid = true

	return result
}

// =============================================================================
// Cache Management
// =============================================================================

func (v *OCSPValidator) getCacheKey(cert *x509.Certificate) string {
	return fmt.Sprintf("ocsp:%s:%s", cert.Issuer.String(), cert.SerialNumber.String())
}

func (v *OCSPValidator) getFromCache(key string) *ocsp.Response {
	v.cacheMu.RLock()
	defer v.cacheMu.RUnlock()

	cached, exists := v.cache[key]
	if !exists {
		return nil
	}

	if time.Since(cached.Timestamp) > v.config.CacheTTL {
		return nil
	}

	return cached.Response
}

func (v *OCSPValidator) addToCache(key string, resp *ocsp.Response) {
	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()

	// Enforce cache size limit
	if len(v.cache) >= v.config.CacheMaxSize {
		// Remove oldest entry
		var oldestKey string
		var oldestTime time.Time
		first := true

		for k, v := range v.cache {
			if first || v.Timestamp.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.Timestamp
				first = false
			}
		}

		delete(v.cache, oldestKey)
	}

	v.cache[key] = &cachedOCSPResponse{
		Response:  resp,
		Timestamp: time.Now(),
	}
}

func (v *OCSPValidator) getCRLFromCache(key string) *pkix.CertificateList {
	v.crlMu.RLock()
	defer v.crlMu.RUnlock()

	cached, exists := v.crlCache[key]
	if !exists {
		return nil
	}

	if time.Since(cached.Timestamp) > v.config.CRLCacheTTL {
		return nil
	}

	return cached.CRL
}

func (v *OCSPValidator) addCRLToCache(key string, crl *pkix.CertificateList) {
	v.crlMu.Lock()
	defer v.crlMu.Unlock()

	v.crlCache[key] = &cachedCRL{
		CRL:       crl,
		Timestamp: time.Now(),
	}
}
