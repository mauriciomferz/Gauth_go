package events

import "sync"

type EventBus struct {
	handlers []EventHandler
	mu       sync.RWMutex
}

func (bus *EventBus) Subscribe(handler EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.handlers = append(bus.handlers, handler)
}

func (bus *EventBus) Publish(event Event) {
	bus.mu.RLock()
	handlers := make([]EventHandler, len(bus.handlers))
	copy(handlers, bus.handlers)
	bus.mu.RUnlock()

	for _, handler := range handlers {
		handler(event)
	}
}
