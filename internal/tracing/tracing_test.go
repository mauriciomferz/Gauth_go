package tracing

import (
	"context"
	"testing"
	"time"
)

func TestStartSpanAndEnd(t *testing.T) {
	p, _ := NewTracerProvider(Config{ServiceName: "test"})
	ctx, span := p.StartSpan(context.Background(), "operation")
	if span == nil {
		t.Fatalf("expected span")
	}
	if span.Operation != "operation" {
		t.Fatalf("wrong operation")
	}
	if span.EndTime != (time.Time{}) {
		t.Fatalf("span should not be ended yet")
	}
	span.End()
	if span.EndTime == (time.Time{}) {
		t.Fatalf("span end not set")
	}
	_ = ctx
}

func TestAddEventAndStatus(t *testing.T) {
	p, _ := NewTracerProvider(Config{ServiceName: "test"})
	_, span := p.StartSpan(context.Background(), "op")
	AddEvent(span, "event1", Attribute{Key: "k", Value: "v"})
	span.SetStatus(StatusError, "failure")
	if span.Status != "failure" {
		t.Fatalf("expected status message recorded")
	}
	if len(span.Tags) == 0 {
		t.Fatalf("expected tags to contain event")
	}
}

func TestAttributeHelpers(t *testing.T) {
	a := AttributeTransactionType.String("txn")
	if a.Key != "transaction.type" || a.Value != "txn" {
		t.Fatalf("unexpected attribute %+v", a)
	}
}
