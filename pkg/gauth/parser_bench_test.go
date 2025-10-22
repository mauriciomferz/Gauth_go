package gauth

import (
	"encoding/json"
	"testing"
)

var benchJSON = []byte(`{"sub":"alice","exp":123456,"scope":["read","write"],"meta":{"tier":"gold","flags":[1,2,3]}}`)

func BenchmarkParseClaims(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := ParseClaims(benchJSON); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlibUnmarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var m map[string]any
		if err := json.Unmarshal(benchJSON, &m); err != nil {
			b.Fatal(err)
		}
	}
}
