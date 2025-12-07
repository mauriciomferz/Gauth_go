package modellimits

import "testing"

// FuzzModelLimitsParse exercises ParseModelLimitsJSON with arbitrary inputs ensuring no panics.
// Run manually: go test -fuzz=FuzzModelLimitsParse -run=^$
func FuzzModelLimitsParse(f *testing.F) {
	f.Add([]byte(`{"model_limits":{"m":{"max_input_tokens":10}}}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 32*1024 { // cap size to avoid excessive allocations
			return
		}
		_, _ = ParseModelLimitsJSON(data)
	})
}
