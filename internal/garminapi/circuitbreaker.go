package garminapi

import (
	"errors"
	"sync"
	"time"
)

// CircuitBreaker implements a simple circuit breaker pattern.
// It tracks consecutive failures and opens the circuit when a threshold is reached,
// preventing further requests until a reset timeout elapses.
type CircuitBreaker struct {
	mu sync.Mutex

	// threshold is the number of consecutive failures before the circuit opens.
	threshold int

	// resetTimeout is how long the circuit stays open before transitioning to half-open.
	resetTimeout time.Duration

	// state tracking
	failures  int
	state     cbState
	openSince time.Time

	// nowFn allows tests to control time.
	nowFn func() time.Time
}

type cbState int

const (
	cbClosed   cbState = iota // Normal operation, requests allowed.
	cbOpen                    // Circuit tripped, requests blocked.
	cbHalfOpen                // Tentatively allowing a single request.
)

// ErrCircuitOpen is returned when the circuit breaker is open and not accepting requests.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// NewCircuitBreaker creates a circuit breaker with the given threshold and reset timeout.
// Default: threshold=5, resetTimeout=30s.
func NewCircuitBreaker(threshold int, resetTimeout time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 5
	}
	if resetTimeout <= 0 {
		resetTimeout = 30 * time.Second
	}
	return &CircuitBreaker{
		threshold:    threshold,
		resetTimeout: resetTimeout,
		state:        cbClosed,
	}
}

// Allow checks if a request is permitted. Returns an error if the circuit is open.
// If the circuit is half-open, it allows one probe request.
func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case cbClosed:
		return nil
	case cbOpen:
		if cb.now().Sub(cb.openSince) >= cb.resetTimeout {
			cb.state = cbHalfOpen
			return nil
		}
		return ErrCircuitOpen
	case cbHalfOpen:
		// Already in half-open, only one probe allowed at a time.
		return ErrCircuitOpen
	}
	return nil
}

// RecordSuccess records a successful request, resetting the failure count
// and closing the circuit if it was half-open.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = cbClosed
}

// RecordFailure records a failed request. If the failure threshold is reached,
// the circuit opens.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	if cb.failures >= cb.threshold {
		cb.state = cbOpen
		cb.openSince = cb.now()
	}
}

// State returns the current circuit breaker state as a string (for diagnostics).
func (cb *CircuitBreaker) State() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case cbOpen:
		if cb.now().Sub(cb.openSince) >= cb.resetTimeout {
			return "half-open"
		}
		return "open"
	case cbHalfOpen:
		return "half-open"
	default:
		return "closed"
	}
}

// Failures returns the current consecutive failure count.
func (cb *CircuitBreaker) Failures() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failures
}

func (cb *CircuitBreaker) now() time.Time {
	if cb.nowFn != nil {
		return cb.nowFn()
	}
	return time.Now()
}
