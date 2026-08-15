package gateway

import (
	"net/http"
	"time"

	"flux-gateway/internal/limiter"
	"flux-gateway/internal/proxy"
	"flux-gateway/internal/resilience"
	"flux-gateway/internal/telemetry"
)

type Gateway struct {
	limiter        *limiter.RedisLimiter
	circuitBreaker *resilience.CircuitBreaker
	proxy          *proxy.GatewayHandler
	metrics        *telemetry.Metrics
}

func New(
	limiter *limiter.RedisLimiter,
	circuitBreaker *resilience.CircuitBreaker,
	proxyHandler *proxy.GatewayHandler,
	metrics *telemetry.Metrics,
) *Gateway {
	return &Gateway{
		limiter:        limiter,
		circuitBreaker: circuitBreaker,
		proxy:          proxyHandler,
		metrics:        metrics,
	}
}

func (g *Gateway) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	start := time.Now()

	if g.limiter != nil {
		allowed, err := g.limiter.Allow(
			r.Context(),
			r.RemoteAddr,
		)

		if err != nil {
			g.metrics.RecordError("rate_limiter")
			http.Error(
				w,
				"rate limiter unavailable",
				http.StatusServiceUnavailable,
			)
			return
		}

		if !allowed {
			g.metrics.RecordRateLimitRejection(r.RemoteAddr)
			http.Error(
				w,
				"rate limit exceeded",
				http.StatusTooManyRequests,
			)
			return
		}
	}

	if g.circuitBreaker != nil {
		if !g.circuitBreaker.Allow() {
			g.metrics.RecordError("circuit_breaker")
			http.Error(
				w,
				"upstream unavailable",
				http.StatusServiceUnavailable,
			)
			return
		}
	}

	recorder := &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	g.proxy.ServeHTTP(recorder, r)

	if g.circuitBreaker != nil {
		if recorder.statusCode >= http.StatusInternalServerError {
			g.circuitBreaker.RecordFailure()
		} else {
			g.circuitBreaker.RecordSuccess()
		}
	}

	g.metrics.RecordRequest(
		r.Method,
		recorder.statusCode,
		time.Since(start),
	)
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	return r.ResponseWriter.Write(body)
}
