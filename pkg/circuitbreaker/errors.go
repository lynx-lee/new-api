package circuitbreaker

import "errors"

var (
	// ErrCircuitOpen is returned when the circuit breaker is open.
	ErrCircuitOpen = errors.New("circuit breaker is open")
	// ErrTooManyRequests is returned when half-open max requests exceeded.
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)
