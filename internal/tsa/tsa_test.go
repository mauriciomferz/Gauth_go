package tsa

import (
	"context"
	"testing"
)

func TestMemoryClient_Anchor(t *testing.T) {
	client := NewMemoryClient("test-tsa")
	
	tests := []struct {
		name    string
		req     *TimestampRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &TimestampRequest{
				Digest:    []byte("test-digest-hash"),
				DigestAlg: "sha256",
			},
			wantErr: false,
		},
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "empty digest",
			req: &TimestampRequest{
				Digest:    []byte{},
				DigestAlg: "sha256",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Anchor(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Anchor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && resp == nil {
				t.Error("Anchor() expected response, got nil")
			}
		})
	}
}

func TestMemoryClient_Name(t *testing.T) {
	tests := []struct {
		name     string
		clientID string
		want     string
	}{
		{
			name:     "custom id",
			clientID: "custom-tsa",
			want:     "custom-tsa",
		},
		{
			name:     "default id",
			clientID: "",
			want:     "memory-tsa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewMemoryClient(tt.clientID)
			if got := client.Name(); got != tt.want {
				t.Errorf("Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemoryVerifier_Verify(t *testing.T) {
	verifier := NewMemoryVerifier("test-anchor")
	client := NewMemoryClient("test-tsa")

	digest := []byte("test-digest-value")
	req := &TimestampRequest{
		Digest:    digest,
		DigestAlg: "sha256",
	}

	resp, err := client.Anchor(context.Background(), req)
	if err != nil {
		t.Fatalf("Failed to create test response: %v", err)
	}

	tests := []struct {
		name    string
		req     *TimestampRequest
		resp    *TimestampResponse
		wantErr bool
	}{
		{
			name:    "valid verification",
			req:     req,
			resp:    resp,
			wantErr: false,
		},
		{
			name:    "nil request",
			req:     nil,
			resp:    resp,
			wantErr: true,
		},
		{
			name:    "nil response",
			req:     req,
			resp:    nil,
			wantErr: true,
		},
		{
			name: "digest mismatch",
			req:  req,
			resp: &TimestampResponse{
				ReceiptBytes: []byte("different-digest"),
				IssuedAt:     resp.IssuedAt,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifier.Verify(context.Background(), tt.req, tt.resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("Verify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemoryVerifier_VerifyDetailed(t *testing.T) {
	verifier := NewMemoryVerifier("test-anchor")
	client := NewMemoryClient("test-tsa")

	digest := []byte("test-digest-value")
	req := &TimestampRequest{
		Digest:    digest,
		DigestAlg: "sha256",
	}

	resp, err := client.Anchor(context.Background(), req)
	if err != nil {
		t.Fatalf("Failed to create test response: %v", err)
	}

	result, err := verifier.VerifyDetailed(context.Background(), req, resp)
	if err != nil {
		t.Errorf("VerifyDetailed() unexpected error = %v", err)
	}
	if result == nil {
		t.Fatal("VerifyDetailed() returned nil result")
	}
	if !result.Valid {
		t.Errorf("VerifyDetailed() result.Valid = false, want true, reason: %s", result.Reason)
	}
}

func TestMemoryVerifier_TrustAnchorID(t *testing.T) {
	tests := []struct {
		name     string
		anchorID string
		want     string
	}{
		{
			name:     "custom anchor id",
			anchorID: "custom-anchor",
			want:     "custom-anchor",
		},
		{
			name:     "default anchor id",
			anchorID: "",
			want:     "memory-anchor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := NewMemoryVerifier(tt.anchorID)
			if got := verifier.TrustAnchorID(); got != tt.want {
				t.Errorf("TrustAnchorID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrors(t *testing.T) {
	if ErrInvalidRequest == nil {
		t.Error("ErrInvalidRequest should not be nil")
	}
	if ErrVerificationFailed == nil {
		t.Error("ErrVerificationFailed should not be nil")
	}
}
