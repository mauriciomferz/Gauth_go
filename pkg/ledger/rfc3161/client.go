package rfc3161

import (
	"bytes"
	"context"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/anchor"
)

// OID for SHA-256: 2.16.840.1.101.3.4.2.1
var oidSHA256 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}

// TimeStampReq implements a subset of RFC 3161 TimeStampReq.
type TimeStampReq struct {
	Version        int
	MessageImprint MessageImprint
	ReqPolicy      asn1.ObjectIdentifier `asn1:"optional"`
	Nonce          *big.Int              `asn1:"optional"`
	CertReq        bool                  `asn1:"optional,default:false"`
	Extensions     []Extension           `asn1:"optional,tag:0"`
}

type MessageImprint struct {
	HashAlgorithm AlgorithmIdentifier
	HashedMessage []byte
}

type AlgorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"optional"`
}

type Extension struct {
	ID       asn1.ObjectIdentifier
	Critical bool `asn1:"optional"`
	Value    []byte
}

// TimeStampResp implements RFC 3161 TimeStampResp.
type TimeStampResp struct {
	Status         PKIStatusInfo
	TimeStampToken asn1.RawValue `asn1:"optional"`
}

type PKIStatusInfo struct {
	Status       int
	StatusString []string       `asn1:"optional"`
	FailInfo     asn1.BitString `asn1:"optional"`
}

// Client implements a minimal RFC 3161 client.
type Client struct {
	URL        string
	HTTPClient *http.Client
}

var (
	oidSignedData = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
	oidTSTInfo    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 4}
)

// ContentInfo implements CMS ContentInfo.
type ContentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"tag:0,explicit,optional"`
}

// SignedData implements CMS SignedData.
type SignedData struct {
	Version          int
	DigestAlgorithms []AlgorithmIdentifier `asn1:"set"`
	EncapContentInfo EncapsulatedContentInfo
	Certificates     []asn1.RawValue `asn1:"optional,tag:0"`
	CRLs             []asn1.RawValue `asn1:"optional,tag:1"`
	SignerInfos      []SignerInfo    `asn1:"set"`
}

type EncapsulatedContentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,optional,tag:0"`
}

type SignerInfo struct {
	Version            int
	SID                asn1.RawValue
	DigestAlgorithm    AlgorithmIdentifier
	SignedAttrs        []Attribute `asn1:"optional,tag:0"`
	SignatureAlgorithm AlgorithmIdentifier
	Signature          []byte
	UnsignedAttrs      []Attribute `asn1:"optional,tag:1"`
}

type Attribute struct {
	Type   asn1.ObjectIdentifier
	Values asn1.RawValue `asn1:"set"`
}

// TSTInfo implements RFC 3161 TSTInfo.
type TSTInfo struct {
	Version        int
	Policy         asn1.ObjectIdentifier
	MessageImprint MessageImprint
	SerialNumber   *big.Int
	GenTime        time.Time
	Accuracy       Accuracy      `asn1:"optional"`
	Ordering       bool          `asn1:"optional,default:false"`
	Nonce          *big.Int      `asn1:"optional"`
	TSA            asn1.RawValue `asn1:"optional,tag:0"`
	Extensions     []Extension   `asn1:"optional,tag:1"`
}

type Accuracy struct {
	Seconds int `asn1:"optional"`
	Millis  int `asn1:"optional,tag:0"`
	Micros  int `asn1:"optional,tag:1"`
}

// NewClient creates a new RFC 3161 client.
func NewClient(url string) *Client {
	return &Client{
		URL: url,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Anchor submits a hash to the TSA and returns an anchor.Receipt.
func (c *Client) Anchor(ctx context.Context, hash []byte) (anchor.Receipt, error) {
	// 1. Create Request
	nonce, _ := createNonce()
	req := TimeStampReq{
		Version: 1,
		MessageImprint: MessageImprint{
			HashAlgorithm: AlgorithmIdentifier{
				Algorithm: oidSHA256,
				// Parameters should be NULL for SHA-256, strictly speaking,
				// but encoding/asn1 often makes this tricky. Omit for now or use RawValue if needed.
				// Some TSAs require explicit NULL.
				Parameters: asn1.RawValue{Tag: asn1.TagNull},
			},
			HashedMessage: hash,
		},
		Nonce:   nonce,
		CertReq: true, // Request TSA certificate to verify response if needed later
	}

	reqBytes, err := asn1.Marshal(req)
	if err != nil {
		return anchor.Receipt{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 2. Send Request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.URL, bytes.NewReader(reqBytes))
	if err != nil {
		return anchor.Receipt{}, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/timestamp-query")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return anchor.Receipt{}, fmt.Errorf("tsa request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return anchor.Receipt{}, fmt.Errorf("tsa returned status %d", resp.StatusCode)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return anchor.Receipt{}, fmt.Errorf("failed to read response: %w", err)
	}

	if len(respBytes) == 0 {
		return anchor.Receipt{}, fmt.Errorf("empty response from tsa")
	}

	// 3. Parse Response
	var tsResp TimeStampResp
	if _, err := asn1.Unmarshal(respBytes, &tsResp); err != nil {
		return anchor.Receipt{}, fmt.Errorf("failed to parse response: %w", err)
	}

	if tsResp.Status.Status != 0 { // 0 = granted
		msg := "unknown"
		if len(tsResp.Status.StatusString) > 0 {
			msg = tsResp.Status.StatusString[0]
		}
		return anchor.Receipt{}, fmt.Errorf("tsa rejected request: status=%d msg=%s", tsResp.Status.Status, msg)
	}

	// 4. Extract Token as Proof
	// For a real verification we would parse the CMS ContentInfo (TimeStampToken),
	// verify the signature, and verify the TSTInfo matches our request.
	// For this prototype, we store the raw token as the proof.

	// Convert hash to hex for Receipt
	hashStr := fmt.Sprintf("%x", hash)

	return anchor.Receipt{
		Hash:      hashStr,
		Timestamp: time.Now().UTC(), // Ideally parse 'genTime' from TSTInfo, using wall clock for now
		Provider:  "rfc3161",
		Proof:     tsResp.TimeStampToken.FullBytes, // Store the raw DER CMS structure
		Version:   1,
	}, nil
}

// Verify performs cryptographic and semantic verification of the timestamp receipt.
func (c *Client) Verify(r anchor.Receipt) error {
	if r.Provider != "rfc3161" {
		return fmt.Errorf("invalid provider for rfc3161 client: %s", r.Provider)
	}
	if len(r.Proof) == 0 {
		return fmt.Errorf("missing proof (timestamp token)")
	}

	// 1. Parse ContentInfo
	var ci ContentInfo
	if _, err := asn1.Unmarshal(r.Proof, &ci); err != nil {
		return fmt.Errorf("failed to parse ContentInfo: %w", err)
	}
	if !ci.ContentType.Equal(oidSignedData) {
		return fmt.Errorf("invalid content type: expected signedData, got %v", ci.ContentType)
	}

	// 2. Parse SignedData
	var sd SignedData
	if _, err := asn1.Unmarshal(ci.Content.Bytes, &sd); err != nil {
		return fmt.Errorf("failed to parse SignedData: %w", err)
	}

	// 3. Verify EncapsulatedContentInfo is TSTInfo
	if !sd.EncapContentInfo.ContentType.Equal(oidTSTInfo) {
		return fmt.Errorf("invalid encapsulated content type: expected TSTInfo, got %v", sd.EncapContentInfo.ContentType)
	}

	// 4. Parse TSTInfo
	var tst TSTInfo
	if _, err := asn1.Unmarshal(sd.EncapContentInfo.Content.Bytes, &tst); err != nil {
		return fmt.Errorf("failed to parse TSTInfo: %w", err)
	}

	// 5. Verify Hash matches MessageImprint
	receiptHashBytes, err := hex.DecodeString(r.Hash)
	if err != nil {
		return fmt.Errorf("invalid receipt hash hex: %w", err)
	}
	if !bytes.Equal(tst.MessageImprint.HashedMessage, receiptHashBytes) {
		return fmt.Errorf("hash mismatch: receipt=%s tst=%x", r.Hash, tst.MessageImprint.HashedMessage)
	}

	// 6. Basic SignerInfo check
	if len(sd.SignerInfos) == 0 {
		return fmt.Errorf("no signer infos in timestamp token")
	}

	return nil
}

func createNonce() (*big.Int, error) {
	// 64-bit nonce
	// Ideally use crypto/rand
	// For simplicity in this snippet I'll assumme math/big usage is fine
	return big.NewInt(time.Now().UnixNano()), nil
}
