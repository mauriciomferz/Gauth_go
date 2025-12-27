package ledger

import (
	"context"
	"encoding/asn1"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/ledger/rfc3161"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRFC3161Provider_Integration(t *testing.T) {
	// 1. Setup Mock TSA Server
	mockTSA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		require.Equal(t, "application/timestamp-query", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req rfc3161.TimeStampReq
		_, err = asn1.Unmarshal(body, &req)
		require.NoError(t, err)

		// Create Mock Response with valid ASN.1 token
		dummyContent := struct{ ID string }{ID: "dummy-proof"}
		dummyToken, _ := asn1.Marshal(dummyContent)

		resp := rfc3161.TimeStampResp{
			Status:         rfc3161.PKIStatusInfo{Status: 0},
			TimeStampToken: asn1.RawValue{FullBytes: dummyToken},
		}

		respBytes, _ := asn1.Marshal(resp)
		w.Header().Set("Content-Type", "application/timestamp-reply")
		w.Write(respBytes)
	}))
	defer mockTSA.Close()

	// 2. Setup ExternalAuditLedger with RFC3161Provider
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "audit.db")
	receiptPath := filepath.Join(tmpDir, "receipts.json")

	provider := NewRFC3161Provider(mockTSA.URL)
	ledger, err := NewExternalAuditLedger(dbPath, provider, receiptPath, 100*time.Millisecond)
	require.NoError(t, err)
	defer ledger.Close()

	// 3. Append Entry
	entry := &Entry{
		ID:   "test-entry",
		TS:   time.Now().UTC(),
		Type: "test",
	}
	err = ledger.Append(context.Background(), entry)
	require.NoError(t, err)

	// 4. Force Anchor (since interval is async, force ensures immediate execution)
	err = ledger.ForceExternalAnchor()
	require.NoError(t, err)

	// 5. Verify Status
	status := ledger.ExternalAnchorStatus()
	assert.Equal(t, true, status["configured"])

	latest := status["latest_receipt"].(map[string]interface{})
	assert.Equal(t, "rfc3161", latest["provider"])
	assert.NotEmpty(t, latest["hash"])

	// 6. Verify Persistence
	// Check if receipts.json exists and has content
	receiptBytes, err := os.ReadFile(receiptPath)
	require.NoError(t, err)
	assert.NotEmpty(t, receiptBytes)
	assert.Contains(t, string(receiptBytes), "rfc3161")
}
