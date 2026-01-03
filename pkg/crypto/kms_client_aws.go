// Copyright 2025 AgentAuth Contributors
// SPDX-License-Identifier: Apache-2.0

package crypto

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

// AWSKMSClient implements KMSClient using AWS KMS.
type AWSKMSClient struct {
	client *kms.Client
}

// NewAWSKMSClient creates a new AWS KMS client.
func NewAWSKMSClient(ctx context.Context, region string) (*AWSKMSClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := kms.NewFromConfig(cfg)
	return &AWSKMSClient{client: client}, nil
}

func (c *AWSKMSClient) CreateKey(ctx context.Context, params *CreateKeyInput) (*CreateKeyOutput, error) {
	input := &kms.CreateKeyInput{
		Description: aws.String(params.Description),
		KeyUsage:    types.KeyUsageType(params.KeyUsage),
		// #nosec G115
		KeySpec: types.KeySpec(params.KeySpec),
	}

	if len(params.Tags) > 0 {
		var tags []types.Tag
		for k, v := range params.Tags {
			tags = append(tags, types.Tag{
				TagKey:   aws.String(k),
				TagValue: aws.String(v),
			})
		}
		input.Tags = tags
	}

	output, err := c.client.CreateKey(ctx, input)
	if err != nil {
		return nil, err
	}

	return &CreateKeyOutput{
		KeyID: *output.KeyMetadata.KeyId,
		Arn:   *output.KeyMetadata.Arn,
	}, nil
}

func (c *AWSKMSClient) DescribeKey(ctx context.Context, keyID string) (*DescribeKeyOutput, error) {
	output, err := c.client.DescribeKey(ctx, &kms.DescribeKeyInput{
		KeyId: aws.String(keyID),
	})
	if err != nil {
		return nil, err
	}

	return &DescribeKeyOutput{
		KeyID:       *output.KeyMetadata.KeyId,
		Arn:         *output.KeyMetadata.Arn,
		KeyUsage:    string(output.KeyMetadata.KeyUsage),
		KeySpec:     string(output.KeyMetadata.KeySpec),
		KeyState:    string(output.KeyMetadata.KeyState),
		Description: aws.ToString(output.KeyMetadata.Description),
		CreatedAt:   *output.KeyMetadata.CreationDate,
	}, nil
}

func (c *AWSKMSClient) ListKeys(ctx context.Context, params *ListKeysInput) (*ListKeysOutput, error) {
	input := &kms.ListKeysInput{
		// #nosec G115
		Limit: aws.Int32(int32(params.Limit)),
	}
	if params.Marker != "" {
		input.Marker = aws.String(params.Marker)
	}

	output, err := c.client.ListKeys(ctx, input)
	if err != nil {
		return nil, err
	}

	var keys []KeyListEntry
	for _, k := range output.Keys {
		keys = append(keys, KeyListEntry{
			KeyID: *k.KeyId,
			Arn:   *k.KeyArn,
		})
	}

	return &ListKeysOutput{
		Keys:       keys,
		NextMarker: aws.ToString(output.NextMarker),
		Truncated:  output.Truncated,
	}, nil
}

func (c *AWSKMSClient) ScheduleKeyDeletion(ctx context.Context, keyID string, pendingWindowInDays int) error {
	_, err := c.client.ScheduleKeyDeletion(ctx, &kms.ScheduleKeyDeletionInput{
		KeyId: aws.String(keyID),
		// #nosec G115
		PendingWindowInDays: aws.Int32(int32(pendingWindowInDays)),
	})
	return err
}

func (c *AWSKMSClient) Encrypt(ctx context.Context, keyID string, plaintext []byte) ([]byte, error) {
	output, err := c.client.Encrypt(ctx, &kms.EncryptInput{
		KeyId:     aws.String(keyID),
		Plaintext: plaintext,
	})
	if err != nil {
		return nil, err
	}

	return output.CiphertextBlob, nil
}

func (c *AWSKMSClient) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	output, err := c.client.Decrypt(ctx, &kms.DecryptInput{
		CiphertextBlob: ciphertext,
	})
	if err != nil {
		return nil, err
	}

	return output.Plaintext, nil
}

func (c *AWSKMSClient) TagResource(ctx context.Context, keyID string, tags map[string]string) error {
	var awsTags []types.Tag
	for k, v := range tags {
		awsTags = append(awsTags, types.Tag{
			TagKey:   aws.String(k),
			TagValue: aws.String(v),
		})
	}

	_, err := c.client.TagResource(ctx, &kms.TagResourceInput{
		KeyId: aws.String(keyID),
		Tags:  awsTags,
	})
	return err
}

func (c *AWSKMSClient) ListResourceTags(ctx context.Context, keyID string) (map[string]string, error) {
	// Note: AWS KMS ListResourceTags likely has pagination, for simplicity we ignore it here
	// assuming we don't have >50 tags per key (limit is usually 50).
	output, err := c.client.ListResourceTags(ctx, &kms.ListResourceTagsInput{
		KeyId: aws.String(keyID),
	})
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, t := range output.Tags {
		result[*t.TagKey] = *t.TagValue
	}

	return result, nil
}
