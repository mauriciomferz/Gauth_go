package monitoring

import (
	"context"
	"testing"
)

func TestMultiSignatureAdoptionGauge(t *testing.T) {
	mc := NewMetricsCollector()
	ctx := context.Background()
	mc.MultiSignatureAdoptionGauge(ctx, 0.75)
	metrics := mc.GetAllMetrics()
	if mv, ok := metrics["multi_signature_adoption"]; !ok || mv.Value != 0.75 {
		t.Fatalf("expected multi_signature_adoption=0.75, got %v", mv.Value)
	}
}

func TestVerificationCounters(t *testing.T) {
	mc := NewMetricsCollector()
	ctx := context.Background()
	mc.VerificationSuccessCounter(ctx)
	mc.VerificationSuccessCounter(ctx)
	mc.VerificationFailureCounter(ctx)
	metrics := mc.GetAllMetrics()
	if mv, ok := metrics["verification_success_total"]; !ok || mv.Value != 2 {
		t.Fatalf("expected verification_success_total=2, got %v", mv.Value)
	}
	if mv, ok := metrics["verification_failure_total"]; !ok || mv.Value != 1 {
		t.Fatalf("expected verification_failure_total=1, got %v", mv.Value)
	}
}

func TestRecordWithTrace(t *testing.T) {
	mc := NewMetricsCollector()
	ctx := context.Background()
	mc.RecordWithTrace(ctx, "custom_metric", 42)
	metrics := mc.GetAllMetrics()
	if mv, ok := metrics["custom_metric"]; !ok || mv.Value != 42 {
		t.Fatalf("expected custom_metric=42, got %v", mv.Value)
	}
}
