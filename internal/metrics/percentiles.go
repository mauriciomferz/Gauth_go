package metrics

import "sort"

// ApproximateQuantiles returns requested quantiles (0..1) from a slice of samples (nanoseconds) using sorting.
// For small reservoirs this is acceptable; future improvement may use streaming algorithms.
func ApproximateQuantiles(samples []uint64, qs []float64) map[float64]uint64 {
	res := make(map[float64]uint64, len(qs))
	if len(samples) == 0 { return res }
	cp := make([]uint64, len(samples))
	copy(cp, samples)
	// Sort ascending
	SortUint64(cp)
	for _, q := range qs {
		if q < 0 { q = 0 } else if q > 1 { q = 1 }
		idx := int(float64(len(cp)-1) * q)
		res[q] = cp[idx]
	}
	return res
}

// SortUint64 is a tiny wrapper so we can swap in radix sort later.
func SortUint64(v []uint64) { sort.Slice(v, func(i, j int) bool { return v[i] < v[j] }) }
