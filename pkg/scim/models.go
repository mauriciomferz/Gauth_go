package scim

import "time"

// User represents a SCIM User Resource
// https://datatracker.ietf.org/doc/html/rfc7643#section-4.1
type User struct {
	Schemas    []string `json:"schemas"`
	ID         string   `json:"id"`
	ExternalID string   `json:"externalId,omitempty"`
	UserName   string   `json:"userName"`
	Name       Name     `json:"name"`
	Emails     []Email  `json:"emails"`
	Active     bool     `json:"active"`
	Meta       Meta     `json:"meta"`
}

type Name struct {
	Formatted  string `json:"formatted,omitempty"`
	FamilyName string `json:"familyName,omitempty"`
	GivenName  string `json:"givenName,omitempty"`
}

type Email struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

type Meta struct {
	ResourceType string    `json:"resourceType"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"lastModified"`
	Location     string    `json:"location"`
	Version      string    `json:"version"` // ETag
}

// ListResponse represents a SCIM List Response
type ListResponse struct {
	Schemas      []string      `json:"schemas"`
	TotalResults int           `json:"totalResults"`
	ItemsPerPage int           `json:"itemsPerPage"`
	StartIndex   int           `json:"startIndex"`
	Resources    []interface{} `json:"Resources"`
}

// ErrorResponse represents a SCIM Error
type ErrorResponse struct {
	Schemas  []string `json:"schemas"`
	Status   string   `json:"status"`
	ScimType string   `json:"scimType,omitempty"`
	Detail   string   `json:"detail"`
}

const (
	UserSchema  = "urn:ietf:params:scim:schemas:core:2.0:User"
	ListSchema  = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	ErrorSchema = "urn:ietf:params:scim:api:messages:2.0:Error"
)

// SCIMClient represents a client authorized to use the SCIM API
type SCIMClient struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenantId"`
	ClientName  string    `json:"clientName"`
	TokenID     string    `json:"tokenId"`     // Associated Access Token ID
	SCIMBaseURL string    `json:"scimBaseUrl"` // Optional
	IsActive    bool      `json:"isActive"`
	Permissions []string  `json:"permissions"` // e.g. ["read", "write"]
	CreatedAt   time.Time `json:"createdAt"`
	LastUsedAt  time.Time `json:"lastUsedAt"`
}
