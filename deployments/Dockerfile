FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /flux-gateway \
    ./cmd/gateway

FROM alpine:3.22

RUN adduser -D -H -s /sbin/nologin gateway

COPY --from=builder /flux-gateway /usr/local/bin/flux-gateway

USER gateway

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/flux-gateway"]
