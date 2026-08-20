# Flux-Gateway Performance Benchmarks

This document records observed runtime characteristics of `flux-gateway` in the local Docker Compose environment.

These measurements are intended as reproducible baseline observations, not production capacity guarantees.

## Test Environment

| Component | Configuration |
| --- | --- |
| CPU | Intel Core i5-3427U @ 1.80 GHz |
| CPU Threads | 2 |
| Memory | 4.6 GiB |
| OS | Ubuntu 24.04.4 LTS |
| Architecture | linux/amd64 |
| Go | 1.26.4 |
| Docker | 29.7.2 |
| Docker Compose | 5.3.1 |
| Redis | 8.10.0 |
| Gateway Image | 9.89 MB |

## Deployment

The benchmark environment consists of three Docker services:

```text
                    ┌────────────────────┐
                    │    Flux-Gateway    │
                    │      :8080         │
                    └─────────┬──────────┘
                              │
                     ┌────────┴────────┐
                     │                 │
                     ▼                 ▼
              ┌──────────────┐  ┌──────────────┐
              │    Redis     │  │   Upstream   │
              │    :6379     │  │    :9000     │
              └──────────────┘  └──────────────┘
```

### Gateway Configuration

- Rate limit: `100 requests / 60 seconds`
- Circuit-breaker failure threshold: `5`
- Circuit-breaker open timeout: `30 seconds`
- Health-check interval: `5 seconds`
- Gateway port: `8080`

## Baseline Request

A request was sent through the complete Dockerized gateway stack:

```bash
curl -s -o /dev/null -w \
'HTTP=%{http_code}\nTTFB=%{time_starttransfer}s\nTOTAL=%{time_total}s\n' \
http://localhost:8080/
```

Observed result:

```text
HTTP=200
TTFB=0.203131s
TOTAL=0.203237s
```

This confirms successful end-to-end routing through:

```text
Client → Flux-Gateway → Redis Rate Limiter → Upstream
```

The observed approximately `203 ms` request time is specific to the local development environment and should not be interpreted as a production latency target.

## Runtime Resource Usage

At the time of measurement:

| Container | CPU | Memory | PIDs |
| --- | ---: | ---: | ---: |
| `flux-gateway` | 0.04% | 12.08 MiB | 9 |
| Redis | 0.39% | 12.93 MiB | 6 |
| Upstream | 0.00% | 4.05 MiB | 9 |

The gateway container was using approximately **12 MiB of memory** during the observed workload.

## Gateway Metrics

The Prometheus metrics endpoint reported:

```text
flux_gateway_circuit_state 0
flux_gateway_errors_total{type="rate_limiter"} 11
flux_gateway_requests_total{method="GET",status="200"} 11
flux_gateway_request_duration_seconds_count{method="GET"} 11
flux_gateway_request_duration_seconds_sum{method="GET"} 9.392719578999998
```

The observed sample contained:

- **11 successful GET requests**
- **0 circuit-breaker activation**
- **11 recorded request durations**
- **9.3927 seconds cumulative recorded request duration**

The arithmetic mean of the recorded request durations was approximately:

```text
9.3927 / 11 ≈ 0.854 seconds
```

This is an observed sample average, not a latency SLA or production performance guarantee.

## Current Benchmark Scope

The current measurements establish a reproducible baseline for:

- End-to-end gateway routing
- Redis-backed rate limiting
- Prometheus instrumentation
- Container resource consumption
- Health and readiness endpoints
- Dockerized service operation

### Not Yet Measured

The project does not currently claim:

- Maximum requests per second
- Sustained throughput capacity
- p95 latency
- p99 latency
- Maximum concurrent connections
- Circuit-breaker transition latency under controlled load
- Long-duration memory stability under high concurrency

Those figures require a dedicated load-testing run using a tool such as `k6`, `wrk`, or `hey`.

## Reproducing the Measurements

Start the complete environment:

```bash
docker compose -f deployments/docker-compose.yml up -d
```

Verify the services:

```bash
docker compose -f deployments/docker-compose.yml ps
```

Test the gateway:

```bash
curl -i http://localhost:8080/
```

Test health and readiness:

```bash
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready
```

Measure request timing:

```bash
curl -s -o /dev/null -w \
'HTTP=%{http_code}\nTTFB=%{time_starttransfer}s\nTOTAL=%{time_total}s\n' \
http://localhost:8080/
```

Inspect gateway metrics:

```bash
curl -s http://localhost:8080/metrics | grep '^flux_gateway_'
```

Inspect container resource usage:

```bash
docker stats --no-stream
```

## Future Benchmarking

A dedicated load-testing suite can extend these measurements to evaluate:

1. Requests per second at increasing concurrency.
2. p50, p95, and p99 latency.
3. Gateway throughput with Redis available.
4. Rate-limit rejection behavior.
5. Circuit-breaker opening and recovery behavior.
6. Resource consumption during sustained load.
7. Behavior during upstream failure.
8. Behavior when Redis becomes unavailable.

Until those tests are performed, this document intentionally reports **observed measurements rather than unsupported capacity claims**.
