package circuitbreaker

import (
	"sync"

	"github.com/QuantumNous/new-api/common"
)

// Manager manages a collection of circuit breakers.
type Manager struct {
	breakers sync.Map // map[string]*CircuitBreaker
}

var globalManager *Manager
var managerOnce sync.Once

// InitManager initializes the global circuit breaker manager.
func InitManager() {
	managerOnce.Do(func() {
		globalManager = &Manager{}
		if common.CircuitBreakerEnabled {
			common.SysLog("circuit breaker enabled")
		} else {
			common.SysLog("circuit breaker disabled (no-op mode)")
		}
	})
}

// GetManager returns the global circuit breaker manager instance.
func GetManager() *Manager {
	return globalManager
}

// GetOrCreateBreaker returns an existing breaker or creates a new one.
func (m *Manager) GetOrCreateBreaker(name string, opts ...Option) *CircuitBreaker {
	if !common.CircuitBreakerEnabled {
		return noopBreaker(name)
	}
	if v, ok := m.breakers.Load(name); ok {
		return v.(*CircuitBreaker)
	}
	cb := NewCircuitBreaker(name, opts...)
	actual, _ := m.breakers.LoadOrStore(name, cb)
	return actual.(*CircuitBreaker)
}

// GetBreaker returns an existing breaker by name, nil if not found.
func (m *Manager) GetBreaker(name string) *CircuitBreaker {
	if v, ok := m.breakers.Load(name); ok {
		return v.(*CircuitBreaker)
	}
	return nil
}

// GetAllStatus returns status snapshots for all active breakers.
func (m *Manager) GetAllStatus() map[string]BreakerStatus {
	result := make(map[string]BreakerStatus)
	m.breakers.Range(func(key, value any) bool {
		cb := value.(*CircuitBreaker)
		result[key.(string)] = cb.Status()
		return true
	})
	return result
}

// LoadOrStore atomically loads or stores a breaker.
func (m *Manager) LoadOrStore(key string, value *CircuitBreaker) (*CircuitBreaker, bool) {
	actual, loaded := m.breakers.LoadOrStore(key, value)
	return actual.(*CircuitBreaker), loaded
}

// noopBreaker returns a no-op circuit breaker that always allows requests through.
func noopBreaker(name string) *CircuitBreaker {
	return NewCircuitBreaker(
		name,
		WithReadyToTrip(func(Counts) bool { return false }),
	)
}
