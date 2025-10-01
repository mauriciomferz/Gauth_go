package events

// MetricsCollector interface for event metrics
type MetricsCollector interface {
	RecordAuthEvent(event Event)
	RecordAuthzEvent(event Event)
	RecordTokenEvent(event Event)
}
