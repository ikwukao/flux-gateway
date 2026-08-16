package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"flux-gateway/internal/config"
	"flux-gateway/internal/gateway"
	"flux-gateway/internal/limiter"
	"flux-gateway/internal/proxy"
	"flux-gateway/internal/resilience"
	"flux-gateway/internal/telemetry"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})

	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("redis shutdown error: %v", err)
		}
	}()

	metrics := telemetry.NewMetrics(
		prometheus.DefaultRegisterer,
	)

	rateLimiter := limiter.NewRedisLimiter(
		redisClient,
		cfg.RateLimitRequests,
		time.Duration(cfg.RateLimitWindowSec)*time.Second,
	)

	circuitBreaker := resilience.NewCircuitBreaker(
		cfg.CircuitFailureThreshold,
		time.Duration(cfg.CircuitOpenTimeoutSec)*time.Second,
	)

	proxyHandler, err := proxy.NewGatewayHandler(cfg.UpstreamURL)
	if err != nil {
		log.Fatalf("proxy initialization error: %v", err)
	}

	gatewayHandler := gateway.New(
		rateLimiter,
		circuitBreaker,
		proxyHandler,
		metrics,
		true,
	)

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.ServerPort),
		Handler:           gatewayHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	healthProber := resilience.NewHealthProber(
		cfg.UpstreamURL,
		time.Duration(cfg.HealthCheckIntervalSec)*time.Second,
		&http.Client{
			Timeout: 2 * time.Second,
		},
		circuitBreaker,
	)

	go healthProber.Start(ctx)

	go func() {
		log.Printf(
			"flux-gateway listening on :%d, upstream=%s",
			cfg.ServerPort,
			cfg.UpstreamURL,
		)

		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()

	log.Println("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	log.Println("flux-gateway stopped")
}
