//go:build go1.18

package gauth

import "testing"

// FuzzParseClaims ensures parser never panics on arbitrary input; errors are acceptable.
func FuzzParseClaims(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{"a":1}`),
		[]byte(`{"arr":[1,2,{"x":3}]}`),
		[]byte(`{"dup":1,"dup":2}`),
		[]byte(`[]`),
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_ = ParseClaims(data) // ignore result; panic is failure
	})
}
