package mesh

// Package mesh provides the core service mesh implementation.

import (
	"sync"
)

// Microservice represents an individual service in the mesh, equipped with resilience patterns and health monitoring.
type Microservice struct {
	Type         ServiceType
	Name         string
	Dependencies []ServiceType
	Health       *HealthMetrics
	loadFactor   float64
	mu           sync.RWMutex
}

// NewMicroservice creates a new Microservice instance.
func NewMicroservice(sType ServiceType, name string, deps []ServiceType) *Microservice {
	return &Microservice{
		Type:         sType,
		Name:         name,
		Dependencies: deps,
		Health:       &HealthMetrics{},
		loadFactor:   1.0,
	}
}

// SetLoadFactor sets the load factor for the microservice.
func (m *Microservice) SetLoadFactor(factor float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadFactor = factor
}

// GetLoadFactor returns the current load factor for the microservice.
func (m *Microservice) GetLoadFactor() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.loadFactor
}
