package notary

import (
	"errors"
	"time"
)

// RFC3161Provider is a scaffold for a future RFC3161 Time-Stamp Authority integration.
// It implements the existing Notarizer interface (hash string -> Receipt) but returns
// an ErrNotImplemented to indicate placeholder status.
type RFC3161Provider struct {
	EndpointURL  string
	ProviderName string
}

var ErrRFC3161NotImplemented = errors.New("rfc3161 provider not implemented")

// Notarize builds a minimal Receipt referencing the provider then returns ErrRFC3161NotImplemented.
// Future implementation steps:
// 1. Construct TimeStampReq ASN.1 with hashed message (SHA256).
// 2. POST to TSA endpoint (DER payload) with content-type application/timestamp-query.
// 3. Parse TimeStampResp DER, extract genTime, serialNumber, tsa certificate.
// 4. Validate PKI chain & signature; embed selected fields in Receipt.
func (p *RFC3161Provider) Notarize(hash string) (Receipt, error) {
	if hash == "" {
		return Receipt{}, errors.New("hash required")
	}
	start := time.Now()
	r := Receipt{
		Hash:           hash,
		Timestamp:      time.Now().UTC().Format(time.RFC3339Nano),
		Provider:       p.ProviderName,
		Version:        1,
		Success:        false,
		LatencySeconds: time.Since(start).Seconds(),
	}
	return r, ErrRFC3161NotImplemented
}
