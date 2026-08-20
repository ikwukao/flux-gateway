.PHONY: help fmt test test-race vet build check \
	docker-build up down restart logs health ready metrics \
	clean

# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------

APP_NAME := flux-gateway
COMPOSE := docker compose -f deployments/docker-compose.yml

# -----------------------------------------------------------------------------
# Help
# -----------------------------------------------------------------------------

help:
	@echo "Flux-Gateway development commands:"
	@echo ""
	@echo "  make fmt           Format Go source files"
	@echo "  make test          Run unit tests"
	@echo "  make test-race     Run tests with race detection"
	@echo "  make vet           Run go vet"
	@echo "  make build         Build the gateway binary"
	@echo "  make check         Run formatting, tests, race tests, vet, and build"
	@echo ""
	@echo "  make docker-build  Build Docker image"
	@echo "  make up            Start the Docker Compose stack"
	@echo "  make down          Stop the Docker Compose stack"
	@echo "  make restart       Restart the Docker Compose stack"
	@echo "  make logs          Follow gateway logs"
	@echo ""
	@echo "  make health        Check gateway health"
	@echo "  make ready         Check gateway readiness"
	@echo "  make metrics       Display gateway Prometheus metrics"
	@echo ""
	@echo "  make clean         Remove local build artifacts"

# -----------------------------------------------------------------------------
# Development
# -----------------------------------------------------------------------------

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(APP_NAME) ./cmd/gateway

check: fmt test test-race vet build

# -----------------------------------------------------------------------------
# Docker
# -----------------------------------------------------------------------------

docker-build:
	$(COMPOSE) build

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

restart:
	$(COMPOSE) down
	$(COMPOSE) up -d

logs:
	$(COMPOSE) logs -f gateway

# -----------------------------------------------------------------------------
# Operations
# -----------------------------------------------------------------------------

health:
	curl -fsS http://localhost:8080/health
	@echo

ready:
	curl -fsS http://localhost:8080/ready
	@echo

metrics:
	curl -fsS http://localhost:8080/metrics | grep '^flux_gateway_'

# -----------------------------------------------------------------------------
# Cleanup
# -----------------------------------------------------------------------------

clean:
	rm -f $(APP_NAME)
	rm -f coverage.out
