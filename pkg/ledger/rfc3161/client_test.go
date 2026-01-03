package rfc3161

import (
	"context"
	"encoding/asn1"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Anchor(t *testing.T) {
	// 1. Setup Mock TSA Server
	mockTSA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Request
		require.Equal(t, "POST", r.Method)
		require.Equal(t, "application/timestamp-query", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NotEmpty(t, body)

		var req TimeStampReq
		_, err = asn1.Unmarshal(body, &req)
		require.NoError(t, err)

		// Verify the hash in the request matches what we sent (sha256 of "test")
		// "test" sha256: 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
		expectedHash, _ := hex.DecodeString("9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08")
		assert.Equal(t, expectedHash, req.MessageImprint.HashedMessage)

		// Create Mock Response
		// We'll create a dummy token. In real life this is a CMS ContentInfo.
		// We need valid ASN.1 bytes for Unmarshal to work.
		dummyContent := struct {
			ID string
		}{
			ID: "dummy-token",
		}
		dummyToken, srcErr := asn1.Marshal(dummyContent)
		require.NoError(t, srcErr)

		resp := TimeStampResp{
			Status: PKIStatusInfo{
				Status: 0, // Granted
			},
			TimeStampToken: asn1.RawValue{
				FullBytes: dummyToken, // Use valid ASN.1 bytes
			},
		}

		respBytes, err := asn1.Marshal(resp)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/timestamp-reply")
		if _, err := w.Write(respBytes); err != nil {
			t.Logf("failed to write response: %v", err)
		}
	}))
	defer mockTSA.Close()

	// 2. Create Client
	client := NewClient(mockTSA.URL)

	// 3. Test Anchor
	hash, _ := hex.DecodeString("9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08")
	receipt, err := client.Anchor(context.Background(), hash)

	require.NoError(t, err)
	assert.Equal(t, "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", receipt.Hash)
	assert.Equal(t, "rfc3161", receipt.Provider)
	assert.Equal(t, 1, receipt.Version)
	assert.NotEmpty(t, receipt.Proof)
	assert.WithinDuration(t, time.Now().UTC(), receipt.Timestamp, 2*time.Second)
}

func TestClient_Anchor_Error(t *testing.T) {
	mockTSA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockTSA.Close()

	client := NewClient(mockTSA.URL)
	_, err := client.Anchor(context.Background(), []byte("hash"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tsa returned status 500")
}

func TestClient_Anchor_Rejection(t *testing.T) {
	mockTSA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := TimeStampResp{
			Status: PKIStatusInfo{
				Status:       2, // Rejection
				StatusString: []string{"Bad Data"},
			},
		}
		respBytes, _ := asn1.Marshal(resp)
		if _, err := w.Write(respBytes); err != nil {
			t.Logf("failed to write response: %v", err)
		}
	}))
	defer mockTSA.Close()

	client := NewClient(mockTSA.URL)
	_, err := client.Anchor(context.Background(), []byte("hash"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tsa rejected request")
	assert.Contains(t, err.Error(), "Bad Data")
}
