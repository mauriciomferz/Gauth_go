package keys

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

// AWSKMSClient implements the KMSClient interface using AWS KMS.
type AWSKMSClient struct {
	api       KMSAPI
	keyID     string
	prevKeyID string // RR-005
	pubKey    crypto.PublicKey
	mu        sync.RWMutex
}

// KMSAPI interface defines the subset of KMS methods we use, allowing mocking.
type KMSAPI interface {
	Sign(ctx context.Context, params *kms.SignInput, optFns ...func(*kms.Options)) (*kms.SignOutput, error)
	GetPublicKey(ctx context.Context, params *kms.GetPublicKeyInput, optFns ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error)
}

// NewAWSKMSClient creates a new AWSKMSClient.
// It loads the default AWS config from the environment/profile.
func NewAWSKMSClient(ctx context.Context, keyID string, region string) (*AWSKMSClient, error) {
	if keyID == "" {
		return nil, errors.New("keyID is required")
	}

	opts := []func(*config.LoadOptions) error{}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	client := kms.NewFromConfig(cfg)

	// Determine Previous Key ID
	prevID := os.Getenv("GAUTH_KMS_PREVIOUS_KEY_ID")

	kmsClient := &AWSKMSClient{
		api:       client,
		keyID:     keyID,
		prevKeyID: prevID,
	}

	// Pre-fetch public key to validate access and cache it
	// (PublicKey does not change for a given key version/ID usually)
	if _, err := kmsClient.PublicKey(ctx); err != nil {
		return nil, fmt.Errorf("failed to fetch public key for %s: %w", keyID, err)
	}

	return kmsClient, nil
}

// SignDigest signs the given digest using AWS KMS.
// It assumes the key is an RSA signing key (RSASSA_PKCS1_V1_5_SHA_256).
// Future improvements could make the algorithm configurable.
func (c *AWSKMSClient) SignDigest(ctx context.Context, digest []byte) ([]byte, error) {
	input := &kms.SignInput{
		KeyId:            aws.String(c.keyID),
		Message:          digest,
		MessageType:      types.MessageTypeDigest,
		SigningAlgorithm: types.SigningAlgorithmSpecRsassaPkcs1V15Sha256,
	}

	resp, err := c.api.Sign(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("kms sign failed: %w", err)
	}

	return resp.Signature, nil
}

// PublicKey returns the public key associated with the remote private key.
func (c *AWSKMSClient) PublicKey(ctx context.Context) (crypto.PublicKey, error) {
	c.mu.RLock()
	if c.pubKey != nil {
		c.mu.RUnlock()
		return c.pubKey, nil
	}
	c.mu.RUnlock()

	return c.fetchPublicKey(ctx, c.keyID)
}

// LookupPublicKey validates and returns key if it matches active or previous ID.
func (c *AWSKMSClient) LookupPublicKey(ctx context.Context, kid string) (crypto.PublicKey, error) {
	c.mu.RLock()
	// Check Active
	if kid == c.keyID && c.pubKey != nil {
		defer c.mu.RUnlock()
		return c.pubKey, nil
	}
	// Check Previous
	if c.prevKeyID != "" && kid == c.prevKeyID {
		c.mu.RUnlock()
		return c.fetchPublicKey(ctx, c.prevKeyID)
	}
	c.mu.RUnlock()

	// If missing, try standard fetch if it matches active ID but wasn't cached?
	if kid == c.keyID {
		return c.PublicKey(ctx)
	}

	return nil, fmt.Errorf("key not found: %s", kid)
}

func (c *AWSKMSClient) fetchPublicKey(ctx context.Context, targetKeyID string) (crypto.PublicKey, error) {
	input := &kms.GetPublicKeyInput{
		KeyId: aws.String(targetKeyID),
	}

	resp, err := c.api.GetPublicKey(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("kms get public key failed: %w", err)
	}

	// Try PEM decode first
	block, _ := pem.Decode(resp.PublicKey)
	if block != nil {
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err == nil {
			if targetKeyID == c.keyID {
				c.mu.Lock()
				c.pubKey = pub
				c.mu.Unlock()
			}
			return pub, nil
		}
	}

	// Try direct DER parse
	pub, err := x509.ParsePKIXPublicKey(resp.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key der: %w", err)
	}

	if targetKeyID == c.keyID {
		c.mu.Lock()
		c.pubKey = pub
		c.mu.Unlock()
	}

	return pub, nil
}

// KeyID returns the identifier of the key.
func (c *AWSKMSClient) KeyID() string {
	return c.keyID
}
