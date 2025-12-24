package scim

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Service handles SCIM logic and mapping
type Service struct {
	repo *Repository
}

// NewService creates a new SCIM service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateUser handles SCIM user creation
func (s *Service) CreateUser(ctx context.Context, tenantID string, req *User) (*User, error) {
	// Validation (basic)
	if req.UserName == "" {
		return nil, fmt.Errorf("userName is required")
	}

	// Map to DB
	u := &UserDB{
		ID:         uuid.New().String(), // Ignore ID if sent, generate new
		TenantID:   tenantID,
		ExternalID: req.ExternalID,
		UserName:   req.UserName,
		GivenName:  req.Name.GivenName,
		FamilyName: req.Name.FamilyName,
		Active:     req.Active,
	}
	if len(req.Emails) > 0 {
		u.Email = req.Emails[0].Value
	}

	if err := s.repo.CreateUser(ctx, u); err != nil {
		return nil, err
	}

	return s.mapToSCIM(u), nil
}

// GetUser retrieves a SCIM user
func (s *Service) GetUser(ctx context.Context, tenantID, userID string) (*User, error) {
	u, err := s.repo.GetUser(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	return s.mapToSCIM(u), nil
}

// ListUsers retrieves multiple SCIM users
func (s *Service) ListUsers(ctx context.Context, tenantID string, limit, offset int) (*ListResponse, error) {
	users, total, err := s.repo.ListUsers(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}

	resources := make([]interface{}, len(users))
	for i, u := range users {
		resources[i] = s.mapToSCIM(&u)
	}

	// SCIM is 1-indexed for startIndex usually, but we accept 0 or 1
	start := offset + 1

	return &ListResponse{
		Schemas:      []string{ListSchema},
		TotalResults: total,
		ItemsPerPage: len(users),
		StartIndex:   start,
		Resources:    resources,
	}, nil
}

// mapToSCIM helper
func (s *Service) mapToSCIM(u *UserDB) *User {
	emails := []Email{}
	if u.Email != "" {
		emails = append(emails, Email{Value: u.Email, Primary: true, Type: "work"})
	}

	return &User{
		Schemas:    []string{UserSchema},
		ID:         u.ID,
		ExternalID: u.ExternalID,
		UserName:   u.UserName,
		Name: Name{
			GivenName:  u.GivenName,
			FamilyName: u.FamilyName,
			Formatted:  fmt.Sprintf("%s %s", u.GivenName, u.FamilyName),
		},
		Emails: emails,
		Active: u.Active,
		Meta: Meta{
			ResourceType: "User",
			Created:      u.CreatedAt,
			LastModified: u.UpdatedAt,
			Version:      fmt.Sprintf("W/\"%d\"", u.UpdatedAt.Unix()),
			Location:     "/api/scim/v2/Users/" + u.ID,
		},
	}
}

// ---------------------------------------------------------------------
// SCIM Client Service Methods
// ---------------------------------------------------------------------

// CreateClient creates a new SCIM client and generates a token
func (s *Service) CreateClient(ctx context.Context, tenantID, clientName string) (*SCIMClient, error) {
	// Generate new token
	token := uuid.New().String()

	c := &SCIMClient{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		ClientName:  clientName,
		TokenID:     token,
		IsActive:    true,
		Permissions: []string{"read", "write"}, // Default permissions
	}

	if err := s.repo.CreateClient(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// ListClients returns all clients
func (s *Service) ListClients(ctx context.Context, tenantID string) ([]SCIMClient, error) {
	return s.repo.ListClients(ctx, tenantID)
}

// DeleteClient deletes a client
func (s *Service) DeleteClient(ctx context.Context, tenantID, id string) error {
	return s.repo.DeleteClient(ctx, tenantID, id)
}
