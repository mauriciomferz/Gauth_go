package poa

import (
	"bytes"
	"encoding/json"
	"errors"
	"sync"

	"github.com/ugorji/go/codec"
)

// internal representation used for CBOR/JSON canonical serialization ordering.
type poaCBOR struct {
    ID         string                 `json:"id" cbor:"id"`
    Parties    []string               `json:"parties,omitempty" cbor:"parties,omitempty"`
    Scope      string                 `json:"scope,omitempty" cbor:"scope,omitempty"`
    Conditions map[string]interface{} `json:"conditions,omitempty" cbor:"conditions,omitempty"`
}

var (
    cborHandleOnce sync.Once
    cborHandle     *codec.CborHandle
)

func getCBORHandle() *codec.CborHandle {
    cborHandleOnce.Do(func() {
        h := new(codec.CborHandle)
        // Canonical ensures deterministic key ordering & shortest integer forms.
        h.Canonical = true
        // Prefer indefinite length disabled for deterministic size.
        h.TimeRFC3339 = true
        cborHandle = h
    })
    return cborHandle
}

// EncodeCBOR encodes a PowerOfAttorney into deterministic CBOR bytes.
func EncodeCBOR(poA *PowerOfAttorney) ([]byte, error) {
    if poA == nil {
        return nil, errors.New("poa: cannot encode nil PowerOfAttorney")
    }
    rep := poaCBOR{
        ID:         poA.ID,
        Parties:    poA.Parties,
        Scope:      poA.Scope,
        Conditions: poA.Conditions,
    }
    var buf bytes.Buffer
    enc := codec.NewEncoder(&buf, getCBORHandle())
    if err := enc.Encode(rep); err != nil {
        return nil, err
    }
    return buf.Bytes(), nil
}

// DecodeCBOR decodes CBOR bytes into a PowerOfAttorney and populates RawJSON with canonical JSON.
func DecodeCBOR(data []byte) (*PowerOfAttorney, error) {
    if len(data) == 0 {
        return nil, errors.New("poa: cannot decode empty CBOR payload")
    }
    var rep poaCBOR
    dec := codec.NewDecoderBytes(data, getCBORHandle())
    if err := dec.Decode(&rep); err != nil {
        return nil, err
    }
    // Canonical JSON: struct field order ensures stable key order.
    rawJSON, err := json.Marshal(rep)
    if err != nil {
        return nil, err
    }
    return &PowerOfAttorney{
        ID:         rep.ID,
        Parties:    rep.Parties,
        Scope:      rep.Scope,
        Conditions: rep.Conditions,
        RawJSON:    rawJSON,
    }, nil
}

// Implement the CBORCodec interface using EncodeCBOR / DecodeCBOR.
type CanonicalCBORCodec struct{}

func (c *CanonicalCBORCodec) Encode(poA *PowerOfAttorney) ([]byte, error) { return EncodeCBOR(poA) }
func (c *CanonicalCBORCodec) Decode(data []byte) (*PowerOfAttorney, error) { return DecodeCBOR(data) }
