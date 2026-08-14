package resilience

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthProberRecordsHealthyUpstream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cb := NewCircuitBreaker(1, time.Second)

	hp := NewHealthProber(
		server.URL,
		time.Second,
		server.Client(),
		cb,
	)

	hp.check(context.Background())

	if got := cb.State(); got != StateClosed {
		t.Fatalf("State() = %v, want %v", got, StateClosed)
	}
}

func TestHealthProberOpensCircuitForFailedUpstream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cb := NewCircuitBreaker(1, time.Second)

	hp := NewHealthProber(
		server.URL,
		time.Second,
		server.Client(),
		cb,
	)

	hp.check(context.Background())

	if got := cb.State(); got != StateOpen {
		t.Fatalf("State() = %v, want %v", got, StateOpen)
	}
}

func TestHealthProberHandlesUnavailableUpstream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusOK)
	}))
	serverURL := server.URL
	server.Close()

	cb := NewCircuitBreaker(1, time.Second)

	hp := NewHealthProber(
		serverURL,
		time.Second,
		nil,
		cb,
	)

	hp.check(context.Background())

	if got := cb.State(); got != StateOpen {
		t.Fatalf("State() = %v, want %v", got, StateOpen)
	}
}

func TestHealthProberStopsWithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cb := NewCircuitBreaker(1, time.Second)

	hp := NewHealthProber(
		server.URL,
		10*time.Millisecond,
		server.Client(),
		cb,
	)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		hp.Start(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("health prober did not stop after context cancellation")
	}
}
