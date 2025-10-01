package handlers

// No imports needed as we don't use any in this file

// We use the events.EventHandler interface directly from the events package.
// The interface is defined as:
//
//	type EventHandler interface {
//		Handle(Event)
//	}
//
// All handlers in this package implement this interface.

// We use the events.MetricsCollector interface from the events package.
// The interface is defined as:
//
//	type MetricsCollector interface {
//		RecordAuthEvent(event Event)
//		RecordAuthzEvent(event Event)
//		RecordTokenEvent(event Event)
//	}
//
// The MetricsHandler in this package accepts implementations of this interface
// for collecting metrics from events.
