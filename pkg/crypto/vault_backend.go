// Copyright 2025 Gimel Foundation
// SPDX-License-Identifier: Apache-2.0

package crypto

import (
	"context"
	"fmt"
	"time"

	vault "github.com/hashicorp/vault/api"
)

// VaultSDKClient implements VaultClient using the official Vault SDK.
type VaultSDKClient struct {
	client *vault.Client
}

// NewVaultSDKClient creates a new Vault client using the official SDK.
func NewVaultSDKClient(config VaultConfig) (*VaultSDKClient, error) {
	vaultConfig := vault.DefaultConfig()
	if config.Address != "" {
		vaultConfig.Address = config.Address
	}

	// Configure transport with timeout
	if vaultConfig.HttpClient != nil {
		vaultConfig.HttpClient.Timeout = 30 * time.Second
	}

	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	if config.Token != "" {
		client.SetToken(config.Token)
	}

	return &VaultSDKClient{
		client: client,
	}, nil
}

// Read reads a secret from Vault.
func (c *VaultSDKClient) Read(ctx context.Context, path string) (*VaultResponse, error) {
	// The SDK's logical read operations are context-aware if we use the RequestWithContext
	// but the standard Read doesn't take context. We should check if we can verify context.
	// For now, we use the standard Read.

	secret, err := c.client.Logical().Read(path)
	if err != nil {
		return nil, err
	}

	if secret == nil {
		return nil, nil // No data found
	}

	return &VaultResponse{
		Data: secret.Data,
	}, nil
}

// Write writes data to Vault.
func (c *VaultSDKClient) Write(ctx context.Context, path string, data map[string]interface{}) (*VaultResponse, error) {
	secret, err := c.client.Logical().Write(path, data)
	if err != nil {
		return nil, err
	}

	if secret == nil {
		return &VaultResponse{}, nil
	}

	return &VaultResponse{
		Data: secret.Data,
	}, nil
}

// Delete removes a secret from Vault.
func (c *VaultSDKClient) Delete(ctx context.Context, path string) error {
	_, err := c.client.Logical().Delete(path)
	return err
}

// Health checks Vault health.
func (c *VaultSDKClient) Health(ctx context.Context) error {
	health, err := c.client.Sys().Health()
	if err != nil {
		return err
	}

	if !health.Initialized {
		return fmt.Errorf("vault not initialized")
	}

	if health.Sealed {
		return fmt.Errorf("vault is sealed")
	}

	return nil
}

// RenewToken renews the client's token.
func (c *VaultSDKClient) RenewToken(ctx context.Context) error {
	_, err := c.client.Auth().Token().RenewSelf(0)
	return err
}
