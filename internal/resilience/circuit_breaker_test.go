package resilience

import (
	"testing"
	"time"
)

func TestCircuitBreakerStartsClosed(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Second)

	if got := cb.State(); got != StateClosed {
		t.Fatalf("State() = %v, want %v", got, StateClosed)
	}

	if !cb.Allow() {
		t.Fatal("Allow() = false, want true")
	}
}

func TestCircuitBreakerOpensAfterFailureThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Second)

	cb.RecordFailure()
	cb.RecordFailure()

	if got := cb.State(); got != StateClosed {
		t.Fatalf("State() = %v, want %v", got, StateClosed)
	}

	cb.RecordFailure()

	if got := cb.State(); got != StateOpen {
		t.Fatalf("State() = %v, want %v", got, StateOpen)
	}

	if cb.Allow() {
		t.Fatal("Allow() = true, want false while circuit is open")
	}
}

func TestCircuitBreakerTransitionsToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 20*time.Millisecond)

	cb.RecordFailure()

	if got := cb.State(); got != StateOpen {
		t.Fatalf("State() = %v, want %v", got, StateOpen)
	}

	time.Sleep(30 * time.Millisecond)

	if !cb.Allow() {
		t.Fatal("Allow() = false, want true after open timeout")
	}

	if got := cb.State(); got != StateHalfOpen {
		t.Fatalf("State() = %v, want %v", got, StateHalfOpen)
	}
}

func TestCircuitBreakerSuccessClosesCircuit(t *testing.T) {
	cb := NewCircuitBreaker(1, time.Second)

	cb.RecordFailure()

	if got := cb.State(); got != StateOpen {
		t.Fatalf("State() = %v, want %v", got, StateOpen)
	}

	cb.RecordSuccess()

	if got := cb.State(); got != StateClosed {
		t.Fatalf("State() = %v, want %v", got, StateClosed)
	}

	if !cb.Allow() {
		t.Fatal("Allow() = false, want true after successful recovery")
	}
}

func TestCircuitBreakerHalfOpenFailureReopensCircuit(t *testing.T) {
	cb := NewCircuitBreaker(1, 20*time.Millisecond)

	cb.RecordFailure()

	time.Sleep(30 * time.Millisecond)

	if !cb.Allow() {
		t.Fatal("Allow() = false, want true in half-open state")
	}

	if got := cb.State(); got != StateHalfOpen {
		t.Fatalf("State() = %v, want %v", got, StateHalfOpen)
	}

	cb.RecordFailure()

	if got := cb.State(); got != StateOpen {
		t.Fatalf("State() = %v, want %v", got, StateOpen)
	}

	if cb.Allow() {
		t.Fatal("Allow() = true, want false after half-open failure")
	}
}

func TestCircuitBreakerAllowsOnlyOneHalfOpenProbe(t *testing.T) {
	cb := NewCircuitBreaker(1, 20*time.Millisecond)

	cb.RecordFailure()

	time.Sleep(30 * time.Millisecond)

	if !cb.Allow() {
		t.Fatal("first half-open request was rejected")
	}

	if cb.Allow() {
		t.Fatal("second half-open request was allowed")
	}

	cb.RecordSuccess()

	if !cb.Allow() {
		t.Fatal("request after successful recovery was rejected")
	}
}
