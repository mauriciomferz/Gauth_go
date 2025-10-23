package metrics

import (
	"testing"
	"time"
)

func TestMemoryMetrics(t *testing.T) {
	m := NewMemory()
	// Initially zero snapshot (extended fields included)
	d, vc, tot, mn, mx, avg, p50, p90, p99, si, sif, sv, svf, rif, spkm, aa, af, rse, rslc, rslt, rslm, rh, rm := m.Snapshot()
	if d != 0 || vc != 0 || tot != 0 || mn != 0 || mx != 0 || avg != 0 || p50 != 0 || p90 != 0 || p99 != 0 || si != 0 || sif != 0 || sv != 0 || svf != 0 || rif != 0 || spkm != 0 || aa != 0 || af != 0 || rse != 0 || rslc != 0 || rslt != 0 || rslm != 0 || rh != 0 || rm != 0 {
		t.Fatalf("expected zeroed snapshot got d=%d vc=%d tot=%d mn=%d mx=%d avg=%v p50=%v p90=%v p99=%v si=%d sif=%d sv=%d svf=%d rif=%d spkm=%d aa=%d af=%d rse=%d rslc=%d rslt=%d rslm=%d rh=%d rm=%d", d, vc, tot, mn, mx, avg, p50, p90, p99, si, sif, sv, svf, rif, spkm, aa, af, rse, rslc, rslt, rslm, rh, rm)
	}
	// Increment delegations
	m.IncDelegationsCreated()
	m.IncDelegationsCreated()
	// Observe two latencies: 10ms and 30ms
	m.ObserveValidationLatency(10 * time.Millisecond)
	m.ObserveValidationLatency(30 * time.Millisecond)
	d, vc, tot, mn, mx, avg, p50, p90, p99, _, sif, sv, svf, rif, spkm, aa, af, rse, rslc, rslt, rslm, rh, rm = m.Snapshot()
	if d != 2 {
		t.Errorf("expected delegations=2 got %d", d)
	}
	if vc != 2 {
		t.Errorf("expected validations=2 got %d", vc)
	}
	// total should be ~40ms in ns
	if tot < uint64(40*time.Millisecond.Nanoseconds())*95/100 || tot > uint64(40*time.Millisecond.Nanoseconds())*105/100 { // allow 5% wiggle though deterministic
		t.Errorf("unexpected total latency ns got %d", tot)
	}
	if mx < uint64(30*time.Millisecond.Nanoseconds()) {
		t.Errorf("expected max >=30ms got %d", mx)
	}
	if mn > uint64(10*time.Millisecond.Nanoseconds()) {
		t.Errorf("expected min <=10ms got %d", mn)
	}
	// average ~20ms
	if avg < 18*time.Millisecond || avg > 22*time.Millisecond {
		t.Errorf("expected avg ~20ms got %v", avg)
	}
	// percentiles should be within bounds: p50 between 10 and 30ms, p90 >= 30ms since only two samples
	if p50 < 10*time.Millisecond || p50 > 30*time.Millisecond {
		t.Errorf("p50 out of range got %v", p50)
	}
	if p90 < 30*time.Millisecond {
		t.Errorf("p90 expected >=30ms got %v", p90)
	}
	if p99 < 30*time.Millisecond { // with only two samples, p99==max
		t.Errorf("p99 expected >=30ms got %v", p99)
	}
	// Replay store stats still zero
	if rse != 0 || rslc != 0 || rslt != 0 || rslm != 0 || rh != 0 || rm != 0 {
		t.Errorf("expected replay store metrics still zero got rse=%d rslc=%d rslt=%d rslm=%d rh=%d rm=%d", rse, rslc, rslt, rslm, rh, rm)
	}
}
