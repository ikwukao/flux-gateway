# Flux-Gateway

**A production-oriented HTTP gateway and reverse proxy built in Go, designed to keep services fast, observable, and resilient when traffic or dependencies fail.**

[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-Containerized-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![Redis](https://img.shields.io/badge/Redis-Rate_Limiting-DC382D?logo=redis&logoColor=white)](https://redis.io/)
[![Prometheus](https://img.shields.io/badge/Prometheus-Metrics-E6522C?logo=prometheus&logoColor=white)](https://prometheus.io/)
[![GitHub Actions](https://img.shields.io/badge/GitHub_Actions-CI-2088FF?logo=githubactions&logoColor=white)](https://github.com/features/actions)
[![Make](https://img.shields.io/badge/Make-Build_Automation-427819?logo=gnu&logoColor=white)](https://www.gnu.org/software/make/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

> A resilient gateway that controls traffic, protects unhealthy services, and exposes the information operators need to understand what's happening.

Instead of simply forwarding every request, it decides:

* Is this request allowed through?
* Is the upstream service healthy?
* Should traffic be rejected because the upstream is failing?
* Should the gateway continue operating if Redis becomes unavailable?
* What is happening to the system right now?

The result is a small but realistic demonstration of the engineering patterns used to build **reliable backend and distributed systems**.

---

## Why Flux-Gateway Exists

Imagine an online service suddenly receiving thousands of requests.

Without a gateway, every request may go directly to the application. A sudden traffic spike, abusive client, or failing dependency can overwhelm the application and potentially trigger a chain reaction across other services.

Flux-Gateway provides a protective layer in front of that application.

Think of it as a **smart traffic cop**:

```text
                         INTERNET
                            │
                            │ Requests
                            ▼
                 ┌─────────────────────┐
                 │    FLUX-GATEWAY     │
                 │                     │
                 │  "Should this       │
                 │   request continue?"│
                 └──────────┬──────────┘
                            │
              ┌─────────────┼─────────────┐
              │             │             │
              ▼             ▼             ▼
          Rate Limit   Circuit Breaker   Proxy
              │             │             │
              └─────────────┼─────────────┘
                            │
                            ▼
                    UPSTREAM SERVICE
```

The gateway can slow down excessive traffic, stop sending requests to an unhealthy service, and provide operational information about what is happening.

---

## What It Demonstrates

Flux-Gateway brings several important backend and distributed-systems concepts together in one project:

* **HTTP reverse proxying**
* **Redis-backed rate limiting**
* **Atomic Redis operations using Lua**
* **Circuit breaker pattern**
* **Upstream health probing**
* **Fail-open behavior**
* **Prometheus observability**
* **Graceful shutdown**
* **Concurrent, race-tested Go code**
* **Docker-based deployment**
* **Automated development workflows with Make**
* **Unit and integration-style testing**

The project is intentionally focused on **failure handling and operational reliability**, rather than simply forwarding HTTP requests.

---

## How Flux-Gateway Works

A normal request follows this general path:

```text
Client
  │
  ▼
┌──────────────────┐
│   Flux-Gateway   │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│   Rate Limiter   │
└────────┬─────────┘
         │ allowed
         ▼
┌──────────────────┐
│ Circuit Breaker  │
└────────┬─────────┘
         │ permitted
         ▼
┌──────────────────┐
│  Reverse Proxy   │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Upstream Service │
└──────────────────┘
```

At the same time, the gateway records operational information through its telemetry layer.

Redis provides shared state for rate limiting, while a background health prober monitors the upstream service.

---

## Resilience in Action

The most important part of Flux-Gateway is what happens when something **goes wrong**.

## 1. Excessive Traffic

Suppose a client starts sending requests too quickly.

The rate limiter tracks requests using Redis.

When the configured limit is exceeded:

```text
Client
  │
  │ Too many requests
  ▼
Flux-Gateway
  │
  ▼
Rate Limiter
  │
  ├── Allowed ──────► Upstream
  │
  └── Limit reached ─► Request rejected
```

This prevents a single client from consuming an unreasonable amount of gateway capacity.

The request limit and time window are configurable.

---

## 2. Upstream Service Failure

Now imagine the application behind the gateway becomes unhealthy.

Without protection, the gateway could continue sending requests to the broken service.

Flux-Gateway uses a **circuit breaker** to prevent this.

```text
                 ┌──────────────┐
                 │    CLOSED    │
                 │ Normal state │
                 └──────┬───────┘
                        │
                  failures reach
                   threshold
                        │
                        ▼
                 ┌──────────────┐
                 │     OPEN     │
                 │ Fail fast    │
                 └──────┬───────┘
                        │
                   timeout expires
                        │
                        ▼
                 ┌──────────────┐
                 │  HALF-OPEN   │
                 │ Test recovery│
                 └──────┬───────┘
                        │
                 ┌──────┴──────┐
                 │             │
              success        failure
                 │             │
                 ▼             ▼
              CLOSED          OPEN
```

When the circuit is open, requests fail quickly instead of repeatedly reaching the unhealthy upstream.

This gives the upstream service time to recover and prevents unnecessary resource consumption.

---

## 3. Redis Failure

Redis is important for rate limiting, but it does not have to become a single point of failure for the entire gateway.

When fail-open behavior is enabled:

```text
Redis unavailable
       │
       ▼
Rate limiter error
       │
       ▼
Fail-open policy
       │
       ▼
Request continues
       │
       ▼
Upstream service
```

This is an intentional resilience trade-off:

> A temporary failure of the rate-limiting dependency does not automatically take the entire gateway offline.

---

## 4. Health Probing

Flux-Gateway periodically checks the upstream service in the background.

The health prober helps the system determine whether the upstream is recovering or continuing to fail.

Healthy responses indicate recovery, while connection failures and unsuccessful HTTP responses contribute to circuit-breaker failure handling.

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
                         │   HTTP Server       │
                         │   Rate Limiter      │
                         │   Circuit Breaker   │
                         │   Health Prober     │
                         │   Reverse Proxy     │
                         │   Telemetry         │
                         └──────┬───────┬──────┘
                                │       │
                         ┌──────┘       └──────────┐
                         ▼                         ▼
                ┌─────────────────┐       ┌─────────────────┐
                │      Redis      │       │ Upstream Service│
                │                 │       │                 │
                │ Rate-limit      │       │ Application/API │
                │ state           │       │                 │
                └─────────────────┘       └─────────────────┘
```

### Main Components

| Component           | Responsibility                                         |
| ------------------- | ------------------------------------------------------ |
| **Gateway**         | Coordinates request handling                           |
| **Rate Limiter**    | Controls request volume using Redis                    |
| **Circuit Breaker** | Prevents traffic from reaching a failing upstream      |
| **Health Prober**   | Periodically checks upstream availability              |
| **Reverse Proxy**   | Forwards permitted requests to the upstream            |
| **Telemetry**       | Exposes Prometheus metrics                             |
| **Config**          | Loads gateway configuration from environment variables |

---

## Quick Start

You can run the complete system with Docker Compose without installing Redis or manually configuring the individual services.

## Requirements

For the recommended setup, install:

* [Docker](https://www.docker.com/)
* Git
* `make`

For local development without Docker, Go and Redis are also required.

---

## 1. Clone the Project

```bash
git clone https://github.com/ikwukao/flux-gateway.git
cd flux-gateway
```

## 2. Start Flux-Gateway

The easiest way to start the complete environment is:

```bash
make up
```

This starts the Docker Compose environment defined in:

```text
deployments/docker-compose.yml
```

To see the gateway logs:

```bash
make logs
```

---

## 3. Try the Gateway

Once the services are running, open:

```text
http://localhost:8080
```

Or use:

```bash
curl http://localhost:8080/
```

Requests reaching the gateway are forwarded to the configured upstream service.

---

## Health and Readiness

Flux-Gateway provides dedicated endpoints for checking its operational state.

### Health

```http
GET /health
```

Example:

```bash
make health
```

A successful response looks like:

```json
{"status":"ok"}
```

### Readiness

```http
GET /ready
```

Example:

```bash
make ready
```

A successful response looks like:

```json
{"status":"ready"}
```

These endpoints are useful for monitoring systems, container orchestration, and operational checks.

---

## Metrics

Flux-Gateway exposes Prometheus-compatible metrics through:

```http
GET /metrics
```

Run:

```bash
make metrics
```

The metrics include information such as:

* Total requests
* Request duration
* Gateway errors
* Circuit-breaker state
* Rate-limit rejections

Important metrics include:

```text
flux_gateway_requests_total
flux_gateway_request_duration_seconds
flux_gateway_errors_total
flux_gateway_circuit_state
flux_gateway_rate_limit_rejected_total
```

This makes the gateway observable rather than treating it as a black box.

---

## Configuration

Flux-Gateway is configured through environment variables.

| Variable                        | Default                  | Description                               |
| ------------------------------- | ------------------------ | ----------------------------------------- |
| `GATEWAY_PORT`                  | `8080`                   | HTTP server port                          |
| `UPSTREAM_URL`                  | Required                 | URL of the upstream service               |
| `REDIS_URL`                     | `redis://localhost:6379` | Redis connection address                  |
| `RATE_LIMIT_REQUESTS`           | `100`                    | Requests allowed per window               |
| `RATE_LIMIT_WINDOW_SECONDS`     | `60`                     | Rate-limit window                         |
| `CIRCUIT_FAILURE_THRESHOLD`     | `5`                      | Failures required to open the circuit     |
| `CIRCUIT_OPEN_TIMEOUT_SECONDS`  | `30`                     | Time before the circuit attempts recovery |
| `HEALTH_CHECK_INTERVAL_SECONDS` | `5`                      | Upstream health-check interval            |

---

## Running Locally

Docker Compose is the easiest way to run the complete system, but the gateway can also be run directly with Go.

Start Redis and an upstream HTTP service, then configure the gateway:

```bash
export GATEWAY_PORT=8080
export UPSTREAM_URL=http://localhost:9000
export REDIS_URL=redis://localhost:6379
```

Then start the gateway:

```bash
go run ./cmd/gateway
```

---

## Makefile

The project includes a Makefile to provide a consistent development and operational workflow.

Run:

```bash
make help
```

to see all available commands.

### Development

Format the Go source:

```bash
make fmt
```

Run unit tests:

```bash
make test
```

Run tests with Go's race detector:

```bash
make test-race
```

Run static analysis:

```bash
make vet
```

Build the gateway:

```bash
make build
```

Run the complete local quality check:

```bash
make check
```

`make check` runs:

```text
formatting
   ↓
tests
   ↓
race detection
   ↓
go vet
   ↓
build
```

This provides a single command for validating the project before committing changes.

---

## Docker Operations

Build the Docker environment:

```bash
make docker-build
```

Start the stack:

```bash
make up
```

Stop the stack:

```bash
make down
```

Restart the stack:

```bash
make restart
```

Follow gateway logs:

```bash
make logs
```

---

## Operational Checks

Check gateway health:

```bash
make health
```

Check readiness:

```bash
make ready
```

View Prometheus metrics:

```bash
make metrics
```

---

## Cleanup

Remove the locally built gateway binary and coverage artifacts:

```bash
make clean
```

---

## Testing and Code Quality

Flux-Gateway is designed to be tested as a concurrent network service.

Run the standard test suite:

```bash
make test
```

Run tests with race detection:

```bash
make test-race
```

Run static analysis:

```bash
make vet
```

Or run the complete validation workflow:

```bash
make check
```

The project uses Go's testing and race-detection tooling to verify both functional behavior and concurrency safety.

---

## Project Structure

```text
flux-gateway/
├── cmd/
│   └── gateway/
│       └── main.go
│
├── deployments/
│   ├── Dockerfile
│   ├── docker-compose.yml
│   └── prometheus.yml
│
├── internal/
│   ├── config/
│   ├── gateway/
│   ├── limiter/
│   ├── proxy/
│   ├── resilience/
│   └── telemetry/
│
├── .dockerignore
├── .gitignore
├── go.mod
├── go.sum
├── LICENSE
├── Makefile
└── README.md
```

### What These Directories Mean

**`cmd/gateway`**

The application's entry point. It starts and configures the gateway.

**`internal/config`**

Loads and validates configuration.

**`internal/gateway`**

Coordinates the gateway's HTTP request lifecycle.

**`internal/limiter`**

Implements Redis-backed request limiting.

**`internal/proxy`**

Handles forwarding requests to the upstream service.

**`internal/resilience`**

Contains the circuit breaker and health-probing logic.

**`internal/telemetry`**

Defines Prometheus metrics used to observe the gateway.

**`deployments`**

Contains the Docker and Docker Compose configuration needed to run the system.

---

## Engineering Decisions

## Why Go?

Go is well suited to network services because it provides:

* A strong standard HTTP library
* Lightweight concurrency
* Explicit error handling
* Fast compilation
* Built-in testing and race detection
* Simple deployment as a compiled binary

For a gateway sitting directly on the network boundary, these characteristics make Go a practical choice.

## Why Redis?

A gateway may eventually run as multiple instances.

Rate-limit state therefore cannot always live only inside one gateway process.

Redis provides centralized shared state with low-latency access, allowing multiple gateway instances to participate in the same rate-limiting strategy.

## Why a Circuit Breaker?

An unhealthy upstream should not continuously consume gateway resources.

The circuit breaker allows Flux-Gateway to detect repeated failures, stop sending traffic to the failing dependency, and periodically test whether the dependency has recovered.

## Why Health Probing?

The circuit breaker primarily observes request failures, while the health prober provides an independent, continuous signal about upstream availability.

Together they improve the gateway's ability to recognize both failure and recovery.

## Why Prometheus?

Operational systems need visibility.

Prometheus provides a widely adopted format for exposing application metrics and integrates naturally with containerized infrastructure and monitoring platforms.

---

## Resilience Model

Flux-Gateway is designed around two important failure scenarios.

### Redis Failure

```text
Redis unavailable
       │
       ▼
Rate limiter error
       │
       ▼
Fail-open policy
       │
       ▼
Request continues
       │
       ▼
Upstream service
```

The gateway can therefore remain available even when its rate-limiting dependency temporarily fails.

### Upstream Failure

```text
Upstream failure
       │
       ▼
Failure recorded
       │
       ▼
Failure threshold reached
       │
       ▼
Circuit OPEN
       │
       ▼
Requests fail fast
       │
       ▼
Timeout expires
       │
       ▼
HALF-OPEN probe
       │
       ├──────── success ────────► CLOSED
       │
       └──────── failure ────────► OPEN
```

This prevents an unhealthy dependency from becoming a continuous source of wasted requests and cascading failures.

---

## What This Project Demonstrates

Flux-Gateway is intentionally more than a basic reverse proxy.

It demonstrates how several backend engineering concepts work together:

```text
                  ┌─────────────────────┐
                  │       Clients       │
                  └──────────┬──────────┘
                             │
                             ▼
                  ┌─────────────────────┐
                  │    Flux-Gateway     │
                  └──────────┬──────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
         Rate Limiting   Resilience    Observability
              │              │              │
              ▼              ▼              ▼
           Redis       Circuit Breaker   Prometheus
                             │
                             ▼
                     Upstream Service
```

The project therefore provides practical experience with:

=> **Backend Engineering**

* HTTP services
* Reverse proxies
* Request processing
* Configuration management
* Concurrent Go programming

=> **Distributed Systems**

* Shared state
* Dependency failures
* Failure isolation
* Service health
* Recovery strategies

=> **Reliability Engineering**

* Rate limiting
* Circuit breakers
* Health checks
* Fail-open policies
* Graceful shutdown

=> **Operations**

* Docker
* Docker Compose
* Prometheus metrics
* Health/readiness endpoints
* Automated Makefile workflows

---

## Contributing

Contributions are welcome.

Before submitting a change, run:

```bash
make check
```

Please keep changes focused and include tests for new behavior.

For larger changes, explain the problem being solved and the reasoning behind the proposed implementation.

---

## License

Flux-Gateway is licensed under the **MIT License**.

See [`LICENSE`](LICENSE) for the complete license text.
