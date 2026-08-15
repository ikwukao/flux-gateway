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
	failOpen       bool
}

func New(
	rateLimiter *limiter.RedisLimiter,
	circuitBreaker *resilience.CircuitBreaker,
	proxyHandler *proxy.GatewayHandler,
	metrics *telemetry.Metrics,
	failOpen bool,
) *Gateway {
	return &Gateway{
		limiter:        rateLimiter,
		circuitBreaker: circuitBreaker,
		proxy:          proxyHandler,
		metrics:        metrics,
		failOpen:       failOpen,
	}
}

func (g *Gateway) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	start := time.Now()

	// 1. Rate limiting.
	if g.limiter != nil {
		allowed, err := g.limiter.Allow(
			r.Context(),
			r.RemoteAddr,
		)

		if err != nil {
			g.recordError("rate_limiter")

			if !g.failOpen {
				http.Error(
					w,
					"rate limiter unavailable",
					http.StatusServiceUnavailable,
				)
				return
			}
		} else if !allowed {
			g.recordRateLimitRejection(r.RemoteAddr)

			http.Error(
				w,
				"rate limit exceeded",
				http.StatusTooManyRequests,
			)
			return
		}
	}

	// 2. Circuit breaker.
	if g.circuitBreaker != nil {
		if !g.circuitBreaker.Allow() {
			g.recordError("circuit_breaker")

			http.Error(
				w,
				"upstream unavailable",
				http.StatusServiceUnavailable,
			)
			return
		}
	}

	// 3. Validate the upstream handler.
	if g.proxy == nil {
		g.recordError("proxy")

		http.Error(
			w,
			"gateway misconfigured",
			http.StatusInternalServerError,
		)
		return
	}

	// 4. Forward the request and capture the upstream response.
	recorder := &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	g.proxy.ServeHTTP(recorder, r)

	// 5. Update circuit-breaker state based on upstream outcome.
	if g.circuitBreaker != nil {
		if recorder.statusCode >= http.StatusInternalServerError {
			g.circuitBreaker.RecordFailure()
		} else {
			g.circuitBreaker.RecordSuccess()
		}
	}

	// 6. Record request metrics.
	g.recordRequest(
		r.Method,
		recorder.statusCode,
		time.Since(start),
	)
}

func (g *Gateway) recordError(component string) {
	if g.metrics != nil {
		g.metrics.RecordError(component)
	}
}

func (g *Gateway) recordRateLimitRejection(client string) {
	if g.metrics != nil {
		g.metrics.RecordRateLimitRejection(client)
	}
}

func (g *Gateway) recordRequest(
	method string,
	statusCode int,
	duration time.Duration,
) {
	if g.metrics != nil {
		g.metrics.RecordRequest(
			method,
			statusCode,
			duration,
		)
	}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}

	r.statusCode = statusCode
	r.wroteHeader = true

	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	return r.ResponseWriter.Write(body)
}
