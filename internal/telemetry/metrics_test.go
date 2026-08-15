package telemetry

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsRecordRequest(t *testing.T) {
	registerer := prometheus.NewRegistry()
	metrics := NewMetrics(registerer)

	metrics.RecordRequest(
		"GET",
		200,
		100*time.Millisecond,
	)

	count := testutil.ToFloat64(
		metrics.RequestsTotal.WithLabelValues("GET", "200"),
	)

	if count != 1 {
		t.Fatalf("request count = %v, want 1", count)
	}
}

func TestMetricsRecordError(t *testing.T) {
	registerer := prometheus.NewRegistry()
	metrics := NewMetrics(registerer)

	metrics.RecordError("upstream")

	count := testutil.ToFloat64(
		metrics.ErrorsTotal.WithLabelValues("upstream"),
	)

	if count != 1 {
		t.Fatalf("error count = %v, want 1", count)
	}
}

func TestMetricsSetCircuitState(t *testing.T) {
	registerer := prometheus.NewRegistry()
	metrics := NewMetrics(registerer)

	metrics.SetCircuitState(1)

	value := testutil.ToFloat64(metrics.CircuitState)

	if value != 1 {
		t.Fatalf("circuit state = %v, want 1", value)
	}
}

func TestMetricsRecordRateLimitRejection(t *testing.T) {
	registerer := prometheus.NewRegistry()
	metrics := NewMetrics(registerer)

	metrics.RecordRateLimitRejection("client-1")

	count := testutil.ToFloat64(
		metrics.RateLimitRejected.WithLabelValues("client-1"),
	)

	if count != 1 {
		t.Fatalf("rate limit rejection count = %v, want 1", count)
	}
}
