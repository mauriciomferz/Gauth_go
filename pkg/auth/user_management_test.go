package auth

import (
	"context"
	"testing"
	"time"
)

// TestUser_Structure tests User struct field validation
func TestUser_Structure(t *testing.T) {
	user := &User{
		ID:       "test-id",
		Username: "testuser",
		Password: "password123",
		Roles:    []string{"admin", "user"},
		Active:   true,
	}

	if user.ID == "" {
		t.Error("Expected non-empty ID")
	}
	if user.Username == "" {
		t.Error("Expected non-empty Username")
	}
	if user.Password == "" {
		t.Error("Expected non-empty Password")
	}
	if len(user.Roles) == 0 {
		t.Error("Expected non-empty Roles")
	}
	if !user.Active {
		t.Error("Expected Active to be true")
	}
}

// TestUser_EmptyFields tests User with empty fields
func TestUser_EmptyFields(t *testing.T) {
	user := &User{
		ID:       "",
		Username: "",
		Password: "",
		Roles:    []string{},
		Active:   false,
	}

	// Empty user should be creatable (validation happens elsewhere)
	if user == nil {
		t.Error("User struct should be creatable with empty fields")
	}

	if user.ID != "" {
		t.Error("Expected empty ID")
	}
	if len(user.Roles) != 0 {
		t.Error("Expected empty Roles slice")
	}
}

// TestUser_MultipleRoles tests User with multiple roles
func TestUser_MultipleRoles(t *testing.T) {
	roles := []string{"admin", "user", "moderator", "editor", "viewer"}
	user := &User{
		ID:       "multi-role-user",
		Username: "multirole",
		Password: "password",
		Roles:    roles,
		Active:   true,
	}

	if len(user.Roles) != len(roles) {
		t.Errorf("Expected %d roles, got %d", len(roles), len(user.Roles))
	}

	for i, role := range roles {
		if user.Roles[i] != role {
			t.Errorf("Role %d: expected '%s', got '%s'", i, role, user.Roles[i])
		}
	}
}

// TestUser_InactiveUser tests inactive user
func TestUser_InactiveUser(t *testing.T) {
	user := &User{
		ID:       "inactive-user",
		Username: "inactive",
		Password: "password",
		Roles:    []string{"user"},
		Active:   false,
	}

	if user.Active {
		t.Error("Expected Active to be false")
	}
}

// TestCredentials_Structure tests Credentials struct
func TestCredentials_Structure(t *testing.T) {
	creds := &Credentials{
		Username:     "testuser",
		Password:     "password123",
		ClientID:     "client-123",
		ClientSecret: "secret-456",
		GrantType:    "password",
		Scope:        "read write",
	}

	if creds.Username == "" {
		t.Error("Expected non-empty Username")
	}
	if creds.Password == "" {
		t.Error("Expected non-empty Password")
	}
	if creds.ClientID == "" {
		t.Error("Expected non-empty ClientID")
	}
	if creds.ClientSecret == "" {
		t.Error("Expected non-empty ClientSecret")
	}
	if creds.GrantType == "" {
		t.Error("Expected non-empty GrantType")
	}
	if creds.Scope == "" {
		t.Error("Expected non-empty Scope")
	}
}

// TestCredentials_MinimalFields tests Credentials with only required fields
func TestCredentials_MinimalFields(t *testing.T) {
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
	}

	if creds.Username == "" {
		t.Error("Expected non-empty Username")
	}
	if creds.Password == "" {
		t.Error("Expected non-empty Password")
	}
	if creds.ClientID != "" {
		t.Log("ClientID is empty (optional field)")
	}
	if creds.ClientSecret != "" {
		t.Log("ClientSecret is empty (optional field)")
	}
}

// TestAuthResult_Structure tests AuthResult struct
func TestAuthResult_Structure(t *testing.T) {
	now := time.Now()
	result := &AuthResult{
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		Scope:        "read write",
		Subject:      "user-id-789",
		IssuedAt:     now,
	}

	if result.AccessToken == "" {
		t.Error("Expected non-empty AccessToken")
	}
	if result.RefreshToken == "" {
		t.Error("Expected non-empty RefreshToken")
	}
	if result.TokenType != "Bearer" {
		t.Errorf("Expected TokenType 'Bearer', got '%s'", result.TokenType)
	}
	if result.ExpiresIn <= 0 {
		t.Error("Expected positive ExpiresIn")
	}
	if result.Subject == "" {
		t.Error("Expected non-empty Subject")
	}
	if result.IssuedAt.IsZero() {
		t.Error("Expected non-zero IssuedAt")
	}
}

// TestTokenClaims_Structure tests TokenClaims struct
func TestTokenClaims_Structure(t *testing.T) {
	now := time.Now()
	claims := &TokenClaims{
		Subject:   "user-123",
		Issuer:    "https://auth.example.com",
		Audience:  "https://api.example.com",
		ExpiresAt: now.Add(1 * time.Hour),
		IssuedAt:  now,
		NotBefore: now,
		Scope:     "read write",
		KeyID:     "key-1",
	}

	if claims.Subject == "" {
		t.Error("Expected non-empty Subject")
	}
	if claims.Issuer == "" {
		t.Error("Expected non-empty Issuer")
	}
	if claims.Audience == "" {
		t.Error("Expected non-empty Audience")
	}
	if claims.ExpiresAt.IsZero() {
		t.Error("Expected non-zero ExpiresAt")
	}
	if claims.IssuedAt.IsZero() {
		t.Error("Expected non-zero IssuedAt")
	}
	if claims.NotBefore.IsZero() {
		t.Error("Expected non-zero NotBefore")
	}
}

// TestSimpleAuthenticator_UserLookup tests user lookup functionality
func TestSimpleAuthenticator_UserLookup(t *testing.T) {
	auth := NewAuthenticator(nil)

	testCases := []struct {
		name     string
		username string
		exists   bool
	}{
		{"AdminUser", "admin", true},
		{"RegularUser", "user", true},
		{"NonexistentUser", "nonexistent", false},
		{"EmptyUsername", "", false},
		{"CaseSensitive", "Admin", false}, // Should be case-sensitive
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			user, exists := auth.users[tc.username]

			if exists != tc.exists {
				t.Errorf("Expected exists=%v for username '%s', got %v", tc.exists, tc.username, exists)
			}

			if tc.exists && user == nil {
				t.Error("Expected non-nil user for existing username")
			}
			if !tc.exists && user != nil {
				t.Error("Expected nil user for non-existing username")
			}
		})
	}
}

// TestSimpleAuthenticator_AdminUser tests admin user properties
func TestSimpleAuthenticator_AdminUser(t *testing.T) {
	auth := NewAuthenticator(nil)

	adminUser, exists := auth.users["admin"]
	if !exists {
		t.Fatal("Admin user should exist in default users")
	}

	if adminUser.ID != "admin-id" {
		t.Errorf("Expected admin ID 'admin-id', got '%s'", adminUser.ID)
	}
	if adminUser.Username != "admin" {
		t.Errorf("Expected username 'admin', got '%s'", adminUser.Username)
	}
	if !adminUser.Active {
		t.Error("Admin user should be active")
	}
	if len(adminUser.Roles) == 0 {
		t.Error("Admin user should have roles")
	}
	
	// Check for admin role
	hasAdminRole := false
	for _, role := range adminUser.Roles {
		if role == "admin" {
			hasAdminRole = true
			break
		}
	}
	if !hasAdminRole {
		t.Error("Admin user should have 'admin' role")
	}
}

// TestSimpleAuthenticator_RegularUser tests regular user properties
func TestSimpleAuthenticator_RegularUser(t *testing.T) {
	auth := NewAuthenticator(nil)

	regularUser, exists := auth.users["user"]
	if !exists {
		t.Fatal("Regular user should exist in default users")
	}

	if regularUser.ID != "user-id" {
		t.Errorf("Expected user ID 'user-id', got '%s'", regularUser.ID)
	}
	if regularUser.Username != "user" {
		t.Errorf("Expected username 'user', got '%s'", regularUser.Username)
	}
	if !regularUser.Active {
		t.Error("Regular user should be active")
	}
	if len(regularUser.Roles) == 0 {
		t.Error("Regular user should have roles")
	}

	// Check for user role
	hasUserRole := false
	for _, role := range regularUser.Roles {
		if role == "user" {
			hasUserRole = true
			break
		}
	}
	if !hasUserRole {
		t.Error("Regular user should have 'user' role")
	}
}

// TestSimpleAuthenticator_Authenticate_InactiveUser tests inactive user authentication
func TestSimpleAuthenticator_Authenticate_InactiveUser(t *testing.T) {
	auth := NewAuthenticator(nil)
	
	// Temporarily mark user as inactive
	if user, exists := auth.users["user"]; exists {
		originalActive := user.Active
		user.Active = false
		defer func() { user.Active = originalActive }()

		ctx := context.Background()
		creds := &Credentials{
			Username: "user",
			Password: "user",
		}

		result, err := auth.Authenticate(ctx, creds)

		if err == nil {
			t.Error("Expected error for inactive user authentication")
		}
		if result != nil {
			t.Error("Expected nil result for inactive user")
		}
	}
}

// TestSimpleAuthenticator_Authenticate_WrongPassword tests wrong password
func TestSimpleAuthenticator_Authenticate_WrongPassword(t *testing.T) {
	auth := NewAuthenticator(nil)
	ctx := context.Background()

	creds := &Credentials{
		Username: "admin",
		Password: "wrong-password",
	}

	result, err := auth.Authenticate(ctx, creds)

	if err == nil {
		t.Error("Expected error for wrong password")
	}
	if result != nil {
		t.Error("Expected nil result for wrong password")
	}
}

// TestSimpleAuthenticator_UserCount tests number of default users
func TestSimpleAuthenticator_UserCount(t *testing.T) {
	auth := NewAuthenticator(nil)

	expectedCount := 2 // admin and user
	actualCount := len(auth.users)

	if actualCount != expectedCount {
		t.Errorf("Expected %d default users, got %d", expectedCount, actualCount)
	}
}
