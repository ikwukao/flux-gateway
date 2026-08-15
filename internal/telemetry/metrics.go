package telemetry

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	RequestsTotal *prometheus.CounterVec

	RequestDuration *prometheus.HistogramVec

	ErrorsTotal *prometheus.CounterVec

	CircuitState prometheus.Gauge

	RateLimitRejected *prometheus.CounterVec
}

func NewMetrics(registerer prometheus.Registerer) *Metrics {
	metrics := &Metrics{
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "flux_gateway_requests_total",
				Help: "Total number of requests processed by the gateway.",
			},
			[]string{"method", "status"},
		),

		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "flux_gateway_request_duration_seconds",
				Help:    "Request duration in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method"},
		),

		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "flux_gateway_errors_total",
				Help: "Total number of gateway errors.",
			},
			[]string{"type"},
		),

		CircuitState: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "flux_gateway_circuit_state",
				Help: "Current circuit breaker state: 0=closed, 1=open, 2=half-open.",
			},
		),

		RateLimitRejected: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "flux_gateway_rate_limit_rejected_total",
				Help: "Total number of requests rejected by rate limiting.",
			},
			[]string{"key"},
		),
	}

	registerer.MustRegister(
		metrics.RequestsTotal,
		metrics.RequestDuration,
		metrics.ErrorsTotal,
		metrics.CircuitState,
		metrics.RateLimitRejected,
	)

	return metrics
}

func (m *Metrics) RecordRequest(
	method string,
	status int,
	duration time.Duration,
) {
	m.RequestsTotal.WithLabelValues(
		method,
		strconv.Itoa(status),
	).Inc()

	m.RequestDuration.WithLabelValues(method).Observe(
		duration.Seconds(),
	)
}

func (m *Metrics) RecordError(errorType string) {
	m.ErrorsTotal.WithLabelValues(errorType).Inc()
}

func (m *Metrics) SetCircuitState(state float64) {
	m.CircuitState.Set(state)
}

func (m *Metrics) RecordRateLimitRejection(key string) {
	m.RateLimitRejected.WithLabelValues(key).Inc()
}
