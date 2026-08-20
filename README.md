# Flux-Gateway

A production-oriented HTTP gateway written in Go, designed to demonstrate resilience, rate limiting, reverse proxying, health probing, observability, and containerized deployment.

## Motivation

Flux-Gateway was built to explore the engineering challenges involved in placing a resilient gateway in front of upstream services. The project focuses on failure handling and operational visibility rather than simply forwarding HTTP requests.

It demonstrates:

* **Redis-backed rate limiting**
* **Circuit breaker protection**
* **Upstream health probing**
* **Fail-open behavior** when Redis is unavailable
* **Reverse proxying**
* **Prometheus metrics**
* **Graceful shutdown**
* **Docker-based deployment**
* **Race-tested concurrent Go code**

---

## Architecture

```text
                    ┌─────────────────────┐
                    │       Client        │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │    Flux-Gateway     │
                    │                     │
                    │  HTTP Server        │
                    │  Rate Limiter       │
                    │  Circuit Breaker    │
                    │  Reverse Proxy      │
                    │  Telemetry          │
                    └──────┬───────┬──────┘
                           │       │
                 ┌─────────┘       └─────────┐
                 ▼                           ▼
        ┌─────────────────┐        ┌─────────────────┐
        │      Redis      │        │ Upstream Service│
        │                 │        │                 │
        │ Rate limiting   │        │ HTTP service    │
        └─────────────────┘        └─────────────────┘
```

### Request Flow

A normal request passes through the gateway in this order:

1. Rate limiting
2. Circuit breaker
3. Request validation
4. Reverse proxy
5. Circuit-breaker outcome recording
6. Prometheus request metrics

* **Fail-Open Behavior:** If Redis becomes unavailable, the gateway can fail open and continue serving traffic rather than making Redis availability a single point of failure.
* **Circuit Breaker Restrictions:** If the circuit breaker is open, requests are rejected before reaching the upstream service.

---

## Features

### Reverse Proxy

Flux-Gateway uses Go's `httputil.ReverseProxy` to forward requests to the configured upstream service while preserving upstream responses and headers.

### Redis Rate Limiting

Requests are rate limited using Redis and an atomic Lua script. The limiter provides:

* Per-key request tracking
* Fixed-window expiration
* Atomic counter updates
* Configurable request limits
* Configurable windows

### Circuit Breaker

The circuit breaker implements three states:

```text
Closed ──failure threshold──> Open
  ▲                            │
  │                            │ timeout
  │                            ▼
  └────── successful probe ─ Half-Open
```

* Only one probe is allowed while the circuit is half-open.
* Repeated upstream failures reopen the circuit.

### Health Probing

A background health prober periodically checks the upstream service. Healthy responses close the circuit, while connection failures and non-2xx responses contribute to circuit failure handling.

### Fail-Open Redis Policy

Redis is used for rate limiting, but Redis availability does not have to become a complete gateway dependency. When configured for fail-open behavior, Redis failures are recorded and requests continue to the upstream service.

### Prometheus Metrics

The gateway exposes Prometheus metrics at `/metrics`. Available metrics include:

* `flux_gateway_requests_total`
* `flux_gateway_request_duration_seconds`
* `flux_gateway_errors_total`
* `flux_gateway_circuit_state`
* `flux_gateway_rate_limit_rejected_total`

---

## Operational Endpoints

### Health

`GET /health`

**Returns:**

```json
{"status":"ok"}
```

### Readiness

`GET /ready`

**Returns:**

```json
{"status":"ready"}
```

### Metrics

`GET /metrics`

**Returns:** Prometheus-formatted metrics.

### Gateway

All other requests are forwarded to the configured upstream service.

---

## Configuration

Flux-Gateway is configured through environment variables.

| Variable                         | Default                     | Description                           |
| :---                             | :---                        | :---                                  |
| `GATEWAY_PORT`                   | `8080`                      | HTTP server port                      |
| `UPSTREAM_URL`                   | *Required*                  | Upstream service URL                  |
| `REDIS_URL`                      | `redis://localhost:6379`    | Redis connection address              |
| `RATE_LIMIT_REQUESTS`            | `100`                       | Requests allowed per window           |
| `RATE_LIMIT_WINDOW_SECONDS`      | `60`                        | Rate-limit window                     |
| `CIRCUIT_FAILURE_THRESHOLD`      | `5`                         | Failures before opening circuit       |
| `CIRCUIT_OPEN_TIMEOUT_SECONDS`   | `30`                        | Time before half-open transition      |
| `HEALTH_CHECK_INTERVAL_SECONDS`  | `5`                         | Upstream health-check interval        |

---

## Quick Start

### Requirements

* Go 1.26+
* Redis
* Docker and Docker Compose (recommended)

### Run with Docker Compose

```bash
docker compose -f deployments/docker-compose.yml up --build
```

The gateway will be available at: `http://localhost:8080`

**Test the gateway:**

```bash
curl http://localhost:8080/
# Expected response: hello from flux upstream
```

**Check health:**

```bash
curl http://localhost:8080/health
```

**Check readiness:**

```bash
curl http://localhost:8080/ready
```

**View metrics:**

```bash
curl http://localhost:8080/metrics
```

### Run Locally

Start Redis and configure the gateway:

```bash
export GATEWAY_PORT=8080
export UPSTREAM_URL=http://localhost:9000
export REDIS_URL=redis://localhost:6379
```

Then run the application:

```bash
go run ./cmd/gateway
```

### Testing

**Run the complete test suite:**

```bash
go test ./...
```

**Run the race detector:**

```bash
go test -race ./...
```

**Run static analysis:**

```bash
go vet ./...
```

**Build the application:**

```bash
go build ./cmd/gateway
```

---

## Project Structure

```text
flux-gateway/
├── cmd/
│   └── gateway/
│       └── main.go
├── deployments/
│   ├── Dockerfile
│   └── docker-compose.yml
├── internal/
│   ├── config/
│   ├── gateway/
│   ├── limiter/
│   ├── proxy/
│   ├── resilience/
│   └── telemetry/
├── .dockerignore
├── .gitignore
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## Engineering Decisions

* **Why Go?** Go provides a strong standard library for HTTP services, lightweight concurrency, explicit error handling, and excellent tooling for building network services.
* **Why Redis?** Rate limiting requires shared state when multiple gateway instances are running. Redis provides a centralized, low-latency store suitable for this purpose.
* **Why a Circuit Breaker?** Without a circuit breaker, a failing upstream can cause the gateway to continue sending requests into an unhealthy dependency. The circuit breaker provides fast failure and gives the upstream time to recover.
* **Why Health Probing?** Health probing provides an independent signal about upstream availability and allows the circuit breaker to react to service recovery.
* **Why Prometheus?** Prometheus provides a standard way to expose application-level operational metrics and integrates naturally with containerized infrastructure.

### Resilience Model

**Redis failure chain:**

```text
Redis failure ──> Rate limiter error ──> Fail-open policy ──> Request continues
```

**Upstream failure chain:**

```text
Upstream failure
     │
     ▼
Circuit failure recorded
     │
     ▼
Failure threshold reached
     │
     ▼
Circuit OPEN ──> Requests fail fast
     │
     ▼
Timeout expires
     │
     ▼
HALF-OPEN probe
     │
     ├── success ──> CLOSED
     │
     └── failure ──> OPEN
```

---

## Contributing

Contributions are welcome. Before submitting changes, please run:

```bash
gofmt -w .
go test ./...
go test -race ./...
go vet ./...
```

Please keep changes focused and include tests for new behavior.

## License

This project is licensed under the MIT License. See `LICENSE`.
