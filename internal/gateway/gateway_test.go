package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"flux-gateway/internal/proxy"
	"flux-gateway/internal/resilience"
	"flux-gateway/internal/telemetry"

	"github.com/prometheus/client_golang/prometheus"
)

func newTestMetrics() *telemetry.Metrics {
	return telemetry.NewMetrics(prometheus.NewRegistry())
}

func TestGatewayForwardsHealthyRequest(t *testing.T) {
	upstream := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello"))
		}),
	)
	defer upstream.Close()

	proxyHandler, err := proxy.NewGatewayHandler(upstream.URL)
	if err != nil {
		t.Fatalf("NewGatewayHandler() error = %v", err)
	}

	cb := resilience.NewCircuitBreaker(
		3,
		time.Second,
	)

	gateway := New(
		nil,
		cb,
		proxyHandler,
		newTestMetrics(),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"http://gateway.local/",
		nil,
	)

	recorder := httptest.NewRecorder()

	gateway.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusOK,
		)
	}

	if body := recorder.Body.String(); body != "hello" {
		t.Fatalf(
			"body = %q, want %q",
			body,
			"hello",
		)
	}
}

func TestGatewayRejectsOpenCircuit(t *testing.T) {
	upstream := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			t.Fatal("upstream should not have been called")
		}),
	)
	defer upstream.Close()

	proxyHandler, err := proxy.NewGatewayHandler(upstream.URL)
	if err != nil {
		t.Fatalf("NewGatewayHandler() error = %v", err)
	}

	cb := resilience.NewCircuitBreaker(
		1,
		time.Minute,
	)

	cb.RecordFailure()

	gateway := New(
		nil,
		cb,
		proxyHandler,
		newTestMetrics(),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"http://gateway.local/",
		nil,
	)

	recorder := httptest.NewRecorder()

	gateway.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusServiceUnavailable,
		)
	}
}
