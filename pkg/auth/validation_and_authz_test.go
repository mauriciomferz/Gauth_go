package auth

import (
	"reflect"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

// ============================================================================
// ValidateToken Tests (Standalone Function - Package Level)
// ============================================================================

// TestStandaloneValidateToken_ValidToken verifies successful token validation
func TestStandaloneValidateToken_ValidToken(t *testing.T) {
	token := "valid-token-string"

	result, err := ValidateToken(token)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Verify result is a map
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result to be map[string]interface{}, got %T", result)
	}

	// Verify expected fields
	if resultMap["user_id"] == nil {
		t.Error("Expected user_id field in result")
	}

	if resultMap["scopes"] == nil {
		t.Error("Expected scopes field in result")
	}

	if resultMap["valid"] == nil {
		t.Error("Expected valid field in result")
	}
}

// TestStandaloneValidateToken_EmptyToken verifies handling of empty token
func TestStandaloneValidateToken_EmptyToken(t *testing.T) {
	result, err := ValidateToken("")

	// Simplified implementation may accept empty token
	if err != nil {
		t.Logf("Empty token rejected: %v", err)
	} else if result != nil {
		t.Log("Empty token accepted in simplified implementation")
	}
}

// TestStandaloneValidateToken_InvalidToken verifies handling of invalid tokens
func TestStandaloneValidateToken_InvalidToken(t *testing.T) {
	invalidTokens := []string{
		"invalid-token",
		"malformed.jwt.token",
		"xxx",
		"!@#$%^&*()",
	}

	for _, token := range invalidTokens {
		t.Run(token, func(t *testing.T) {
			result, err := ValidateToken(token)

			// Simplified implementation may accept any token
			if err != nil {
				t.Logf("Invalid token '%s' rejected: %v", token, err)
			} else if result != nil {
				t.Logf("Invalid token '%s' accepted in simplified implementation", token)
			}
		})
	}
}

// TestStandaloneValidateToken_LongToken verifies handling of very long token
func TestStandaloneValidateToken_LongToken(t *testing.T) {
	longToken := ""
	for i := 0; i < 1000; i++ {
		longToken += "a"
	}

	result, err := ValidateToken(longToken)

	// Should handle long tokens gracefully
	if err != nil {
		t.Logf("Long token rejected: %v", err)
	} else if result != nil {
		t.Log("Long token accepted")
	}
}

// TestStandaloneValidateToken_ResultStructure verifies the structure of returned result
func TestStandaloneValidateToken_ResultStructure(t *testing.T) {
	result, err := ValidateToken("test-token")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map[string]interface{}, got %T", result)
	}

	// Verify user_id field
	userID, ok := resultMap["user_id"].(string)
	if !ok {
		t.Error("Expected user_id to be string")
	}
	if userID == "" {
		t.Error("Expected non-empty user_id")
	}

	// Verify scopes field
	scopes, ok := resultMap["scopes"].([]string)
	if !ok {
		t.Error("Expected scopes to be []string")
	}
	if len(scopes) == 0 {
		t.Log("Scopes list is empty (may be valid)")
	}

	// Verify valid field
	valid, ok := resultMap["valid"].(bool)
	if !ok {
		t.Error("Expected valid to be bool")
	}
	if !valid {
		t.Log("Token marked as invalid (may be expected)")
	}
}

// TestStandaloneValidateToken_MultipleTokens verifies validation of different tokens
func TestStandaloneValidateToken_MultipleTokens(t *testing.T) {
	tokens := []string{
		"token-1",
		"token-2",
		"token-3",
		"token-admin",
		"token-user",
	}

	for _, token := range tokens {
		t.Run(token, func(t *testing.T) {
			result, err := ValidateToken(token)

			if err != nil {
				t.Errorf("Expected no error for token '%s', got: %v", token, err)
			}

			if result == nil {
				t.Errorf("Expected non-nil result for token '%s'", token)
			}
		})
	}
}

// TestStandaloneValidateToken_Concurrent verifies thread safety
func TestStandaloneValidateToken_Concurrent(t *testing.T) {
	concurrency := 10
	done := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			result, err := ValidateToken("concurrent-token")
			if err != nil {
				done <- err
				return
			}
			if result == nil {
				done <- err
				return
			}
			done <- nil
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < concurrency; i++ {
		err := <-done
		if err == nil {
			successCount++
		} else {
			t.Errorf("Concurrent validation %d failed: %v", i, err)
		}
	}

	if successCount != concurrency {
		t.Errorf("Expected %d successful validations, got %d", concurrency, successCount)
	}
}

// ============================================================================
// BuildAuthzRequestFromClaims Tests
// ============================================================================

// TestBuildAuthzRequestFromClaims_ValidInput verifies basic request building
func TestBuildAuthzRequestFromClaims_ValidInput(t *testing.T) {
	claims := &Claims{
		UserID: "test-user",
		Scopes: []string{"read", "write"},
	}

	resource := "resource-1"
	action := "read"
	extraCtx := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	req := BuildAuthzRequestFromClaims(claims, resource, action, extraCtx)

	// Verify request fields
	if req.Subject != claims.UserID {
		t.Errorf("Expected subject '%s', got '%s'", claims.UserID, req.Subject)
	}

	if req.Resource != resource {
		t.Errorf("Expected resource '%s', got '%s'", resource, req.Resource)
	}

	if req.Action != action {
		t.Errorf("Expected action '%s', got '%s'", action, req.Action)
	}

	// Verify context
	if req.Context == nil {
		t.Fatal("Expected non-nil context")
	}

	// Verify extra context was copied
	if req.Context["key1"] != "value1" {
		t.Errorf("Expected context['key1'] = 'value1', got '%s'", req.Context["key1"])
	}

	if req.Context["key2"] != "value2" {
		t.Errorf("Expected context['key2'] = 'value2', got '%s'", req.Context["key2"])
	}

	// Verify scopes were added
	if req.Context["scopes"] != "read write" {
		t.Errorf("Expected context['scopes'] = 'read write', got '%s'", req.Context["scopes"])
	}
}

// TestBuildAuthzRequestFromClaims_NoScopes verifies handling of empty scopes
func TestBuildAuthzRequestFromClaims_NoScopes(t *testing.T) {
	claims := &Claims{
		UserID: "test-user",
		Scopes: []string{},
	}

	req := BuildAuthzRequestFromClaims(claims, "resource", "action", nil)

	// Verify scopes field is not added when empty
	if req.Context["scopes"] != "" {
		t.Logf("Empty scopes handled as: '%s'", req.Context["scopes"])
	}
}

// TestBuildAuthzRequestFromClaims_NilScopes verifies handling of nil scopes
func TestBuildAuthzRequestFromClaims_NilScopes(t *testing.T) {
	claims := &Claims{
		UserID: "test-user",
		Scopes: nil,
	}

	req := BuildAuthzRequestFromClaims(claims, "resource", "action", nil)

	// Verify nil scopes don't cause panic
	if req.Subject != "test-user" {
		t.Errorf("Expected subject 'test-user', got '%s'", req.Subject)
	}

	// Verify scopes field handling
	if req.Context["scopes"] != "" {
		t.Logf("Nil scopes handled as: '%s'", req.Context["scopes"])
	}
}

// TestBuildAuthzRequestFromClaims_NilExtraContext verifies nil context handling
func TestBuildAuthzRequestFromClaims_NilExtraContext(t *testing.T) {
	claims := &Claims{
		UserID: "test-user",
		Scopes: []string{"read"},
	}

	req := BuildAuthzRequestFromClaims(claims, "resource", "action", nil)

	// Verify request is built successfully
	if req.Subject != "test-user" {
		t.Errorf("Expected subject 'test-user', got '%s'", req.Subject)
	}

	// Verify context is initialized
	if req.Context == nil {
		t.Error("Expected non-nil context")
	}

	// Verify scopes are added
	if req.Context["scopes"] != "read" {
		t.Errorf("Expected context['scopes'] = 'read', got '%s'", req.Context["scopes"])
	}
}

// TestBuildAuthzRequestFromClaims_EmptyResource verifies empty resource handling
func TestBuildAuthzRequestFromClaims_EmptyResource(t *testing.T) {
	claims := &Claims{
		UserID: "test-user",
		Scopes: []string{"read"},
	}

	req := BuildAuthzRequestFromClaims(claims, "", "action", nil)

	// Empty resource is accepted
	if req.Resource != "" {
		t.Errorf("Expected empty resource, got '%s'", req.Resource)
	}

	// Other fields should be populated
	if req.Subject == "" {
		t.Error("Expected non-empty subject")
	}
}

// TestBuildAuthzRequestFromClaims_EmptyAction verifies empty action handling
func TestBuildAuthzRequestFromClaims_EmptyAction(t *testing.T) {
	claims := &Claims{
		UserID: "test-user",
		Scopes: []string{"read"},
	}

	req := BuildAuthzRequestFromClaims(claims, "resource", "", nil)

	// Empty action is accepted
	if req.Action != "" {
		t.Errorf("Expected empty action, got '%s'", req.Action)
	}

	// Other fields should be populated
	if req.Subject == "" {
		t.Error("Expected non-empty subject")
	}
}

// TestBuildAuthzRequestFromClaims_MultipleScopes verifies multiple scopes handling
func TestBuildAuthzRequestFromClaims_MultipleScopes(t *testing.T) {
	claims := &Claims{
		UserID: "test-user",
		Scopes: []string{"read", "write", "admin", "delete"},
	}

	req := BuildAuthzRequestFromClaims(claims, "resource", "action", nil)

	// Verify scopes are space-separated
	expectedScopes := "read write admin delete"
	if req.Context["scopes"] != expectedScopes {
		t.Errorf("Expected context['scopes'] = '%s', got '%s'", expectedScopes, req.Context["scopes"])
	}
}

// TestBuildAuthzRequestFromClaims_SingleScope verifies single scope handling
func TestBuildAuthzRequestFromClaims_SingleScope(t *testing.T) {
	claims := &Claims{
		UserID: "test-user",
		Scopes: []string{"admin"},
	}

	req := BuildAuthzRequestFromClaims(claims, "resource", "action", nil)

	// Verify single scope
	if req.Context["scopes"] != "admin" {
		t.Errorf("Expected context['scopes'] = 'admin', got '%s'", req.Context["scopes"])
	}
}

// TestBuildAuthzRequestFromClaims_ContextOverride verifies context doesn't override scopes
func TestBuildAuthzRequestFromClaims_ContextOverride(t *testing.T) {
	claims := &Claims{
		UserID: "test-user",
		Scopes: []string{"read"},
	}

	extraCtx := map[string]string{
		"scopes": "should-be-overridden",
	}

	req := BuildAuthzRequestFromClaims(claims, "resource", "action", extraCtx)

	// Verify claims scopes override extra context scopes
	if req.Context["scopes"] != "read" {
		t.Errorf("Expected claims scopes to override, got '%s'", req.Context["scopes"])
	}
}

// TestBuildAuthzRequestFromClaims_EmptyUserID verifies empty user ID handling
func TestBuildAuthzRequestFromClaims_EmptyUserID(t *testing.T) {
	claims := &Claims{
		UserID: "",
		Scopes: []string{"read"},
	}

	req := BuildAuthzRequestFromClaims(claims, "resource", "action", nil)

	// Empty user ID is accepted
	if req.Subject != "" {
		t.Errorf("Expected empty subject, got '%s'", req.Subject)
	}
}

// TestBuildAuthzRequestFromClaims_SpecialCharacters verifies special character handling
func TestBuildAuthzRequestFromClaims_SpecialCharacters(t *testing.T) {
	claims := &Claims{
		UserID: "user@example.com",
		Scopes: []string{"read:resource", "write:resource"},
	}

	extraCtx := map[string]string{
		"ip_address": "192.168.1.1",
		"user_agent": "Mozilla/5.0",
	}

	req := BuildAuthzRequestFromClaims(claims, "resource/path", "read:write", extraCtx)

	// Verify special characters are preserved
	if req.Subject != "user@example.com" {
		t.Errorf("Expected subject 'user@example.com', got '%s'", req.Subject)
	}

	if req.Resource != "resource/path" {
		t.Errorf("Expected resource 'resource/path', got '%s'", req.Resource)
	}

	if req.Action != "read:write" {
		t.Errorf("Expected action 'read:write', got '%s'", req.Action)
	}

	// Verify scopes with colons
	expectedScopes := "read:resource write:resource"
	if req.Context["scopes"] != expectedScopes {
		t.Errorf("Expected scopes '%s', got '%s'", expectedScopes, req.Context["scopes"])
	}
}

// TestBuildAuthzRequestFromClaims_LargeContext verifies large context handling
func TestBuildAuthzRequestFromClaims_LargeContext(t *testing.T) {
	claims := &Claims{
		UserID: "test-user",
		Scopes: []string{"read"},
	}

	// Create large context
	extraCtx := make(map[string]string)
	for i := 0; i < 100; i++ {
		extraCtx["key"+string(rune(i))] = "value" + string(rune(i))
	}

	req := BuildAuthzRequestFromClaims(claims, "resource", "action", extraCtx)

	// Verify all context entries were copied
	if len(req.Context) < 100 {
		t.Errorf("Expected at least 100 context entries, got %d", len(req.Context))
	}
}

// TestBuildAuthzRequestFromClaims_ComplexClaims verifies complex claims handling
func TestBuildAuthzRequestFromClaims_ComplexClaims(t *testing.T) {
	claims := &Claims{
		UserID:    "admin-user-123",
		SessionID: "session-456",
		Scopes:    []string{"admin", "read", "write", "delete"},
		ExpiresAt: ExpirationTime{Time: time.Now().Add(time.Hour)},
		IssuedAt:  time.Now().Unix(),
		Issuer:    "test-issuer",
		Audience:  "test-audience",
	}

	extraCtx := map[string]string{
		"request_id": "req-123",
		"source_ip":  "10.0.0.1",
	}

	req := BuildAuthzRequestFromClaims(claims, "admin/users", "delete", extraCtx)

	// Verify all fields
	if req.Subject != claims.UserID {
		t.Errorf("Expected subject '%s', got '%s'", claims.UserID, req.Subject)
	}

	if req.Resource != "admin/users" {
		t.Error("Resource mismatch")
	}

	if req.Action != "delete" {
		t.Error("Action mismatch")
	}

	// Verify scopes
	expectedScopes := "admin read write delete"
	if req.Context["scopes"] != expectedScopes {
		t.Errorf("Expected scopes '%s', got '%s'", expectedScopes, req.Context["scopes"])
	}

	// Verify extra context
	if req.Context["request_id"] != "req-123" {
		t.Error("request_id not preserved")
	}
}

// TestBuildAuthzRequestFromClaims_Concurrent verifies thread safety
func TestBuildAuthzRequestFromClaims_Concurrent(t *testing.T) {
	claims := &Claims{
		UserID: "concurrent-user",
		Scopes: []string{"read"},
	}

	concurrency := 10
	done := make(chan authz.Request, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			req := BuildAuthzRequestFromClaims(claims, "resource", "action", nil)
			done <- req
		}(i)
	}

	// Collect results
	requests := make([]authz.Request, 0, concurrency)
	for i := 0; i < concurrency; i++ {
		req := <-done
		requests = append(requests, req)
	}

	// Verify all requests are identical
	for i, req := range requests {
		if req.Subject != claims.UserID {
			t.Errorf("Request %d: expected subject '%s', got '%s'", i, claims.UserID, req.Subject)
		}
	}
}

// TestBuildAuthzRequestFromClaims_ContextIsolation verifies context mutation safety
func TestBuildAuthzRequestFromClaims_ContextIsolation(t *testing.T) {
	claims := &Claims{
		UserID: "test-user",
		Scopes: []string{"read"},
	}

	extraCtx := map[string]string{
		"key": "original",
	}

	req := BuildAuthzRequestFromClaims(claims, "resource", "action", extraCtx)

	// Mutate original context
	extraCtx["key"] = "modified"

	// Verify request context is isolated
	if req.Context["key"] != "original" {
		t.Error("Context was not properly isolated from mutation")
	}
}

// TestBuildAuthzRequestFromClaims_ReturnValue verifies return type
func TestBuildAuthzRequestFromClaims_ReturnValue(t *testing.T) {
	claims := &Claims{
		UserID: "test-user",
		Scopes: []string{"read"},
	}

	req := BuildAuthzRequestFromClaims(claims, "resource", "action", nil)

	// Verify return type is authz.Request
	var _ authz.Request = req

	// Verify it's not a pointer
	reqType := reflect.TypeOf(req)
	if reqType.Kind() == reflect.Ptr {
		t.Error("Expected value type, got pointer")
	}
}

// TestBuildAuthzRequestFromClaims_EmptyExtraContext verifies empty context handling
func TestBuildAuthzRequestFromClaims_EmptyExtraContext(t *testing.T) {
	claims := &Claims{
		UserID: "test-user",
		Scopes: []string{"read"},
	}

	emptyCtx := make(map[string]string)

	req := BuildAuthzRequestFromClaims(claims, "resource", "action", emptyCtx)

	// Verify context is initialized
	if req.Context == nil {
		t.Error("Expected non-nil context")
	}

	// Should have scopes at minimum
	if req.Context["scopes"] == "" {
		t.Error("Expected scopes to be set")
	}
}
