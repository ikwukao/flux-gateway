package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	ServerPort              int
	UpstreamURL             string
	RedisURL                string
	RateLimitRequests       int
	RateLimitWindowSec      int
	CircuitFailureThreshold int
	CircuitOpenTimeoutSec   int
	HealthCheckIntervalSec  int
}

func Load() (Config, error) {
	cfg := Config{
		ServerPort:              getEnvInt("GATEWAY_PORT", 8080),
		UpstreamURL:             os.Getenv("UPSTREAM_URL"),
		RedisURL:                getEnv("REDIS_URL", "redis://localhost:6379"),
		RateLimitRequests:       getEnvInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindowSec:      getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60),
		CircuitFailureThreshold: getEnvInt("CIRCUIT_FAILURE_THRESHOLD", 5),
		CircuitOpenTimeoutSec:   getEnvInt("CIRCUIT_OPEN_TIMEOUT_SECONDS", 30),
		HealthCheckIntervalSec:  getEnvInt("HEALTH_CHECK_INTERVAL_SECONDS", 5),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.ServerPort < 1 || c.ServerPort > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535")
	}

	if c.UpstreamURL == "" {
		return fmt.Errorf("upstream URL must not be empty")
	}

	if c.RateLimitRequests < 1 {
		return fmt.Errorf("rate limit requests must be greater than zero")
	}

	if c.RateLimitWindowSec < 1 {
		return fmt.Errorf("rate limit window must be greater than zero")
	}

	if c.CircuitFailureThreshold < 1 {
		return fmt.Errorf("circuit failure threshold must be greater than zero")
	}

	if c.CircuitOpenTimeoutSec < 1 {
		return fmt.Errorf("circuit open timeout must be greater than zero")
	}

	if c.HealthCheckIntervalSec < 1 {
		return fmt.Errorf("health check interval must be greater than zero")
	}

	return nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
