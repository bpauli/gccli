package garminapi

import (
	"testing"
	"time"
)

func TestCircuitBreaker_StartsClosedAndAllows(t *testing.T) {
	cb := NewCircuitBreaker(5, 30*time.Second)

	if err := cb.Allow(); err != nil {
		t.Fatalf("Allow() on closed circuit: %v", err)
	}
	if s := cb.State(); s != "closed" {
		t.Errorf("State() = %q, want closed", s)
	}
	if cb.Failures() != 0 {
		t.Errorf("Failures() = %d, want 0", cb.Failures())
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)

	// Record failures below threshold.
	cb.RecordFailure()
	cb.RecordFailure()
	if err := cb.Allow(); err != nil {
		t.Fatalf("Allow() with 2 failures (threshold 3): %v", err)
	}
	if cb.Failures() != 2 {
		t.Errorf("Failures() = %d, want 2", cb.Failures())
	}

	// Third failure opens the circuit.
	cb.RecordFailure()
	if s := cb.State(); s != "open" {
		t.Errorf("State() = %q, want open", s)
	}
	if cb.Failures() != 3 {
		t.Errorf("Failures() = %d, want 3", cb.Failures())
	}

	err := cb.Allow()
	if err != ErrCircuitOpen {
		t.Errorf("Allow() on open circuit: got %v, want ErrCircuitOpen", err)
	}
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	now := time.Now()
	cb := NewCircuitBreaker(2, 10*time.Second)
	cb.nowFn = func() time.Time { return now }

	// Open the circuit.
	cb.RecordFailure()
	cb.RecordFailure()
	if s := cb.State(); s != "open" {
		t.Fatalf("State() = %q, want open", s)
	}

	// Still open before timeout.
	cb.nowFn = func() time.Time { return now.Add(5 * time.Second) }
	if err := cb.Allow(); err != ErrCircuitOpen {
		t.Errorf("Allow() before timeout: got %v, want ErrCircuitOpen", err)
	}

	// Advance past timeout → half-open.
	cb.nowFn = func() time.Time { return now.Add(11 * time.Second) }
	if s := cb.State(); s != "half-open" {
		t.Errorf("State() after timeout = %q, want half-open", s)
	}

	// First Allow should succeed (half-open probe).
	if err := cb.Allow(); err != nil {
		t.Fatalf("Allow() in half-open: %v", err)
	}

	// Second Allow should block (already in half-open, probe in progress).
	if err := cb.Allow(); err != ErrCircuitOpen {
		t.Errorf("second Allow() in half-open: got %v, want ErrCircuitOpen", err)
	}
}

func TestCircuitBreaker_SuccessResetsFromHalfOpen(t *testing.T) {
	now := time.Now()
	cb := NewCircuitBreaker(2, 10*time.Second)
	cb.nowFn = func() time.Time { return now }

	// Open the circuit.
	cb.RecordFailure()
	cb.RecordFailure()

	// Advance past timeout → half-open.
	cb.nowFn = func() time.Time { return now.Add(11 * time.Second) }
	if err := cb.Allow(); err != nil {
		t.Fatalf("Allow() in half-open: %v", err)
	}

	// Probe succeeds → close circuit.
	cb.RecordSuccess()
	if s := cb.State(); s != "closed" {
		t.Errorf("State() after success = %q, want closed", s)
	}
	if cb.Failures() != 0 {
		t.Errorf("Failures() = %d, want 0", cb.Failures())
	}

	// Should allow requests again.
	if err := cb.Allow(); err != nil {
		t.Errorf("Allow() after reset: %v", err)
	}
}

func TestCircuitBreaker_FailureInHalfOpenReopens(t *testing.T) {
	now := time.Now()
	cb := NewCircuitBreaker(2, 10*time.Second)
	cb.nowFn = func() time.Time { return now }

	// Open the circuit.
	cb.RecordFailure()
	cb.RecordFailure()

	// Advance past timeout → half-open.
	now = now.Add(11 * time.Second)
	cb.nowFn = func() time.Time { return now }
	if err := cb.Allow(); err != nil {
		t.Fatalf("Allow() in half-open: %v", err)
	}

	// Probe fails → reopen.
	cb.RecordFailure()
	// failures=3 which is >= threshold(2), so circuit is open again.
	if s := cb.State(); s != "open" {
		t.Errorf("State() after half-open failure = %q, want open", s)
	}
}

func TestCircuitBreaker_SuccessResetsDuringClosed(t *testing.T) {
	cb := NewCircuitBreaker(5, 30*time.Second)

	// Record some failures then a success.
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.Failures() != 3 {
		t.Errorf("Failures() = %d, want 3", cb.Failures())
	}

	cb.RecordSuccess()
	if cb.Failures() != 0 {
		t.Errorf("Failures() after success = %d, want 0", cb.Failures())
	}
	if s := cb.State(); s != "closed" {
		t.Errorf("State() = %q, want closed", s)
	}
}

func TestCircuitBreaker_DefaultValues(t *testing.T) {
	// Zero values should get defaults.
	cb := NewCircuitBreaker(0, 0)

	// Should need 5 failures to trip (default threshold).
	for i := 0; i < 4; i++ {
		cb.RecordFailure()
	}
	if err := cb.Allow(); err != nil {
		t.Errorf("Allow() with 4 failures (default threshold 5): %v", err)
	}

	cb.RecordFailure()
	if err := cb.Allow(); err != ErrCircuitOpen {
		t.Errorf("Allow() after 5 failures: got %v, want ErrCircuitOpen", err)
	}
}

func TestCircuitBreaker_ErrCircuitOpenMessage(t *testing.T) {
	if ErrCircuitOpen.Error() != "circuit breaker is open" {
		t.Errorf("ErrCircuitOpen = %q", ErrCircuitOpen.Error())
	}
}
