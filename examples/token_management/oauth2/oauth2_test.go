package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuth2AuthorizationCodeFlow(t *testing.T) {
	ctx := context.Background()
	flow := NewOAuth2Flow()

	clientID := "example-client"
	userID := "user-123"
	scopes := []string{"read", "write"}

	accessTokenID, refreshTokenID, err := flow.AuthorizationCodeFlow(ctx, clientID, userID, scopes)
	require.NoError(t, err)
	assert.NotEmpty(t, accessTokenID)
	assert.NotEmpty(t, refreshTokenID)

	// Validate access token
	accessTokenObj, err := flow.store.Get(ctx, accessTokenID)
	require.NoError(t, err)
	err = flow.Validator.Validate(ctx, accessTokenObj)
	assert.NoError(t, err)
	assert.Equal(t, userID, accessTokenObj.Subject)
	assert.ElementsMatch(t, scopes, accessTokenObj.Scopes)

	// Validate refresh token
	refreshTokenObj, err := flow.store.Get(ctx, refreshTokenID)
	require.NoError(t, err)
	err = flow.Validator.Validate(ctx, refreshTokenObj)
	assert.NoError(t, err)
	assert.Equal(t, userID, refreshTokenObj.Subject)
	assert.Equal(t, []string{"refresh"}, refreshTokenObj.Scopes)
}

func TestOAuth2RefreshTokenFlow(t *testing.T) {
	ctx := context.Background()
	flow := NewOAuth2Flow()

	clientID := "example-client"
	userID := "user-456"
	scopes := []string{"read"}

	_, refreshTokenID, err := flow.AuthorizationCodeFlow(ctx, clientID, userID, scopes)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshTokenID)

	// Simulate waiting for access token expiry
	time.Sleep(10 * time.Millisecond)

	newAccessTokenID, err := flow.RefreshTokenFlow(ctx, refreshTokenID)
	assert.NoError(t, err)
	assert.NotEmpty(t, newAccessTokenID)

	newAccessTokenObj, err := flow.store.Get(ctx, newAccessTokenID)
	require.NoError(t, err)
	err = flow.Validator.Validate(ctx, newAccessTokenObj)
	assert.NoError(t, err)
	assert.Equal(t, userID, newAccessTokenObj.Subject)
	assert.ElementsMatch(t, scopes, newAccessTokenObj.Scopes)
}
