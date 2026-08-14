package resilience

import (
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mu sync.RWMutex

	state State

	failureThreshold int
	openTimeout      time.Duration

	failures int
	openedAt time.Time
}

func NewCircuitBreaker(
	failureThreshold int,
	openTimeout time.Duration,
) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		openTimeout:      openTimeout,
	}
}

func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return cb.state
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		if time.Since(cb.openedAt) >= cb.openTimeout {
			cb.state = StateHalfOpen
			return true
		}

		return false

	case StateHalfOpen:
		return true

	default:
		return false
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.openedAt = time.Time{}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.state = StateOpen
		cb.openedAt = time.Now()
		return
	}

	if cb.state != StateClosed {
		return
	}

	cb.failures++

	if cb.failures >= cb.failureThreshold {
		cb.state = StateOpen
		cb.openedAt = time.Now()
	}
}
