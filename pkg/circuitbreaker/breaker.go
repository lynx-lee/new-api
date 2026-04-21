// Package circuitbreaker provides a stateful circuit breaker implementation
// for protecting upstream AI API relay calls from cascading failures.
package circuitbreaker

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
)

// State represents the current state of a circuit breaker.
type State int

const (
	StateClosed   State = iota // Normal operation, requests flow through
	StateOpen                   // Circuit tripped, requests fail fast
	StateHalfOpen              // Probing state, limited requests allowed
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Counts tracks request statistics within a single interval.
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// BreakerStatus represents the current status of a circuit breaker (for external queries).
type BreakerStatus struct {
	Name            string  `json:"name"`
	State           string  `json:"state"`
	Requests        uint32  `json:"requests"`
	TotalSuccesses  uint32  `json:"total_successes"`
	TotalFailures   uint32  `json:"total_failures"`
	FailureRate     float64 `json:"failure_rate"`
}

// Option configures a CircuitBreaker.
type Option func(*CircuitBreaker)

// WithMaxRequests sets the maximum number of requests allowed in half-open state.
func WithMaxRequests(n uint32) Option {
	return func(cb *CircuitBreaker) { cb.maxRequests = n }
}

// WithInterval sets the cyclic reset period for the closed state's internal counter.
func WithInterval(d time.Duration) Option {
	return func(cb *CircuitBreaker) { cb.interval = d }
}

// WithTimeout sets how long to wait before transitioning from open to half-open.
func WithTimeout(d time.Duration) Option {
	return func(cb *CircuitBreaker) { cb.timeout = d }
}

// WithReadyToTrip sets the function that determines when to trip the breaker.
func WithReadyToTrip(f func(Counts) bool) Option {
	return func(cb *CircuitBreaker) { cb.readyToTrip = f }
}

// WithOnStateChange sets a callback invoked on every state transition.
func WithOnStateChange(f func(name string, from, to State)) Option {
	return func(cb *CircuitBreaker) { cb.onStateChange = f }
}

// defaultReadyToTrip trips when: consecutive failures >= threshold OR
// failure rate > 50% with at least minRequests samples.
func defaultReadyToTrip(threshold uint32) func(Counts) bool {
	minRequests := uint32(10)
	return func(c Counts) bool {
		if c.ConsecutiveFailures >= threshold {
			return true
		}
		if c.Requests >= minRequests && c.TotalFailures > 0 {
			failureRatio := float64(c.TotalFailures) / float64(c.Requests)
			return failureRatio >= common.CircuitBreakerErrorThreshold
		}
		return false
	}
}

// CircuitBreaker is a state machine that protects against cascading failures.
type CircuitBreaker struct {
	name         string
	maxRequests  uint32
	interval     time.Duration
	timeout      time.Duration
	readyToTrip func(Counts) bool
	onStateChange func(name string, from, to State)

	mu          sync.Mutex
	state       State
	generation  uint64
	counts      Counts
	expiry      time.Time // when the current counts window resets
}

// NewCircuitBreaker creates a new CircuitBreaker with given name and options.
func NewCircuitBreaker(name string, opts ...Option) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:         name,
		maxRequests:  uint32(common.CircuitBreakerHalfOpenMaxRequests),
		interval:     60 * time.Second,
		timeout:      time.Duration(common.CircuitBreakerTimeoutSeconds) * time.Second,
		readyToTrip:  defaultReadyToTrip(uint32(common.CircuitBreakerConsecutiveFailures)),
		state:       StateClosed,
		generation:  0,
		expiry:      time.Now().Add(60 * time.Second),
	}
	for _, opt := range opts {
		opt(cb)
	}
	return cb
}

// Name returns the name of this circuit breaker.
func (cb *CircuitBreaker) Name() string { return cb.name }

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	now := time.Now()
	return cb.currentState(now)
}

// Counts returns a snapshot of internal counts (thread-safe copy).
func (cb *CircuitBreaker) Counts() Counts {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.counts
}

// Execute runs the given request through the circuit breaker.
// If the circuit is open, it returns ErrOpen without calling req.
func (cb *CircuitBreaker) Execute(req func() error) error {
	generation, err := cb.beforeRequest()
	if err != nil {
		return err
	}

	err = req()

	cb.afterRequest(generation, err == nil)
	return err
}

// ExecuteResult is like Execute but allows returning a value alongside the error.
func (cb *CircuitBreaker) ExecuteResult(req func() (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	result, err := req()

	cb.afterRequest(generation, err == nil)
	return result, err
}

// beforeRequest checks if the request should be allowed through.
// Returns generation number or an error if the circuit is open.
func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	if !common.CircuitBreakerEnabled {
		return 0, nil
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state := cb.currentState(now)

	switch state {
	case StateOpen:
		if now.After(cb.expiry) {
			cb.setState(StateHalfOpen, now)
			return cb.generation, nil
		}
		return 0, fmt.Errorf("%w: %s", ErrCircuitOpen, cb.name)

	case StateHalfOpen:
		if atomic.LoadUint32(&cb.counts.Requests) < cb.maxRequests {
			atomic.AddUint32(&cb.counts.Requests, 1)
			return cb.generation, nil
		}
		return 0, fmt.Errorf("%w: %s", ErrTooManyRequests, cb.name)

	default: // Closed
		return cb.generation, nil
	}
}

// afterRequest records success/failure and potentially transitions state.
func (cb *CircuitBreaker) afterRequest(generation uint64, success bool) {
	if !common.CircuitBreakerEnabled {
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Ignore stale generations (expired window)
	if generation != cb.generation {
		return
	}

	now := time.Now()

	if success {
		cb.onSuccess(now)
	} else {
		cb.onFailure(now)
	}
}

// currentState returns the current state, handling expiry transitions.
func (cb *CircuitBreaker) currentState(now time.Time) State {
	switch cb.state {
	case StateClosed:
		if !now.Before(cb.expiry) {
			cb.resetCounts(now)
		}
		return StateClosed

	case StateOpen:
		if now.After(cb.expiry) {
			return cb.toHalfOpen(now)
		}
		return StateOpen

	default: // HalfOpen
		return StateHalfOpen
	}
}

func (cb *CircuitBreaker) onSuccess(now time.Time) {
	atomic.AddUint32(&cb.counts.TotalSuccesses, 1)
	atomic.AddUint32(&cb.counts.ConsecutiveSuccesses, 1)
	atomic.StoreUint32(&cb.counts.ConsecutiveFailures, 0)

	if cb.state == StateHalfOpen && cb.readyToTrip(cb.counts) == false {
		cb.setState(StateClosed, now)
	}
}

func (cb *CircuitBreaker) onFailure(now time.Time) {
	atomic.AddUint32(&cb.counts.TotalFailures, 1)
	atomic.AddUint32(&cb.counts.ConsecutiveFailures, 1)
	atomic.StoreUint32(&cb.counts.ConsecutiveSuccesses, 0)

	if cb.readyToTrip(cb.counts) {
		cb.setState(StateOpen, now)
	}
}

func (cb *CircuitBreaker) setState(to State, now time.Time) {
	if cb.state == to {
		return
	}
	from := cb.state
	logger.LogWarn(nil, fmt.Sprintf("circuit breaker '%s' state change: %s -> %s", cb.name, from.String(), to.String()))
	cb.state = to

	switch to {
	case StateClosed:
		cb.resetCounts(now)
	case StateOpen:
		cb.generation++
		cb.expiry = now.Add(cb.timeout)
	case StateHalfOpen:
		cb.generation++
		cb.resetCounts(now)
	}

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, from, to)
	}
}

func (cb *CircuitBreaker) toHalfOpen(now time.Time) State {
	cb.setState(StateHalfOpen, now)
	return StateHalfOpen
}

func (cb *CircuitBreaker) resetCounts(now time.Time) {
	cb.counts = Counts{}
	cb.expiry = now.Add(cb.interval)
}

// Status returns a thread-safe status snapshot.
func (cb *CircuitBreaker) Status() BreakerStatus {
	c := cb.Counts()
	s := cb.State()
	var rate float64
	if c.Requests > 0 {
		rate = math.Round(float64(c.TotalFailures)/float64(c.Requests)*10000) / 100
	}
	return BreakerStatus{
		Name:           cb.name,
		State:          s.String(),
		Requests:       c.Requests,
		TotalSuccesses: c.TotalSuccesses,
		TotalFailures:  c.TotalFailures,
		FailureRate:    rate,
	}
}
