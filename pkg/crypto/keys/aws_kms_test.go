package keys

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockKMSClient is a mock for the AWS SDK KMS client interface.
// Since the AWS SDK uses concrete structs, we might need an interface wrapper or mock the generated client if possible?
// AWS SDK v2 clients are structs but their methods take interfaces or options.
// Actually, `NewAWSKMSClient` uses `*kms.Client`. To mock it, we would need to interface-ify the SDK interaction
// OR use the SDK's interface if it existed (v2 usually doesn't provide one by default, requiring wrapper).

// To avoid over-engineering with wrappers for this task, we will test the logical components via a Mock KeyManager
// that behaves like the remote one, OR we can refactor AWSKMSClient to take an interface.
// Let's verify if `manager.go` uses `AWSKMSClient` struct directly or interface.
// It returns `KeyManager`.

// For unit testing `AWSKMSClient` logic specifically, we usually wrap the `Sign` and `GetPublicKey` calls.
// Given constraints, I will create a test that ensures `AWSKMSClient` satisfies `KMSClient` interface
// and relies on integration tests/manual check for real AWS connectivity,
// OR ideally, refactor `AWSKMSClient` to accept a `KMSAPI` interface.

// We use the KMSAPI interface defined in aws_kms.go.

// This test assumes `AWSKMSClient` struct has a field `api KMSAPI` instead of `client *kms.Client`.

type MockKMSAPI struct {
	mock.Mock
}

func (m *MockKMSAPI) Sign(ctx context.Context, params *kms.SignInput, optFns ...func(*kms.Options)) (*kms.SignOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*kms.SignOutput), args.Error(1)
}

func (m *MockKMSAPI) GetPublicKey(
	ctx context.Context,
	params *kms.GetPublicKeyInput,
	optFns ...func(*kms.Options),
) (*kms.GetPublicKeyOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*kms.GetPublicKeyOutput), args.Error(1)
}

func TestAWSKMSClient(t *testing.T) {
	mockAPI := new(MockKMSAPI)
	keyID := "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"

	// Create client with mocked API
	// Note: We need to enable this injection in the implementation
	client := &AWSKMSClient{
		api:   mockAPI,
		keyID: keyID,
	}

	// 1. Test PublicKey
	// Generate a real key to encode as DER for the mock
	realKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pubDer, err := x509.MarshalPKIXPublicKey(&realKey.PublicKey)
	require.NoError(t, err)

	mockAPI.On("GetPublicKey", mock.Anything, mock.MatchedBy(func(input *kms.GetPublicKeyInput) bool {
		return *input.KeyId == keyID
	})).Return(&kms.GetPublicKeyOutput{
		PublicKey: pubDer,
		KeyId:     &keyID,
	}, nil)

	pub, err := client.PublicKey(context.Background())
	require.NoError(t, err)
	require.Equal(t, &realKey.PublicKey, pub)

	// 2. Test SignDigest
	digest := make([]byte, 32)
	signature := []byte("signature")

	mockAPI.On("Sign", mock.Anything, mock.MatchedBy(func(input *kms.SignInput) bool {
		return *input.KeyId == keyID && string(input.Message) == string(digest) && input.MessageType == types.MessageTypeDigest
	})).Return(&kms.SignOutput{
		Signature: signature,
	}, nil)

	sig, err := client.SignDigest(context.Background(), digest)
	require.NoError(t, err)
	require.Equal(t, signature, sig)

	mockAPI.AssertExpectations(t)
}
