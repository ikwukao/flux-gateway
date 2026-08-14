package limiter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestLimiter(t *testing.T, limit int, window time.Duration) (
	*RedisLimiter,
	*miniredis.Miniredis,
) {
	t.Helper()

	server := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: server.Addr(),
	})

	limiter := NewRedisLimiter(
		client,
		limit,
		window,
	)

	t.Cleanup(func() {
		_ = client.Close()
	})

	return limiter, server
}

func TestRedisLimiterAllowsRequestsWithinLimit(t *testing.T) {
	limiter, _ := newTestLimiter(t, 3, time.Minute)

	ctx := context.Background()

	for i := 0; i < 3; i++ {
		allowed, err := limiter.Allow(ctx, "client-1")

		if err != nil {
			t.Fatalf("Allow() error = %v", err)
		}

		if !allowed {
			t.Fatalf("request %d was rejected, want allowed", i+1)
		}
	}
}

func TestRedisLimiterRejectsRequestsOverLimit(t *testing.T) {
	limiter, _ := newTestLimiter(t, 2, time.Minute)

	ctx := context.Background()

	for i := 0; i < 2; i++ {
		allowed, err := limiter.Allow(ctx, "client-1")

		if err != nil {
			t.Fatalf("Allow() error = %v", err)
		}

		if !allowed {
			t.Fatalf("request %d was rejected, want allowed", i+1)
		}
	}

	allowed, err := limiter.Allow(ctx, "client-1")

	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}

	if allowed {
		t.Fatal("request over limit was allowed")
	}
}

func TestRedisLimiterSeparatesKeys(t *testing.T) {
	limiter, _ := newTestLimiter(t, 1, time.Minute)

	ctx := context.Background()

	allowed, err := limiter.Allow(ctx, "client-1")
	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}

	if !allowed {
		t.Fatal("first client request was rejected")
	}

	allowed, err = limiter.Allow(ctx, "client-2")
	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}

	if !allowed {
		t.Fatal("second client request was rejected")
	}
}

func TestRedisLimiterWindowResets(t *testing.T) {
	limiter, server := newTestLimiter(t, 1, time.Second)

	ctx := context.Background()

	allowed, err := limiter.Allow(ctx, "client-1")
	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}

	if !allowed {
		t.Fatal("first request was rejected")
	}

	allowed, err = limiter.Allow(ctx, "client-1")
	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}

	if allowed {
		t.Fatal("request over limit was allowed")
	}

	server.FastForward(time.Second)

	allowed, err = limiter.Allow(ctx, "client-1")
	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}

	if !allowed {
		t.Fatal("request after window reset was rejected")
	}
}

func TestRedisLimiterReturnsErrorWhenRedisUnavailable(t *testing.T) {
	server := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: server.Addr(),
	})

	limiter := NewRedisLimiter(
		client,
		10,
		time.Minute,
	)

	server.Close()

	allowed, err := limiter.Allow(
		context.Background(),
		"client-1",
	)

	if !errors.Is(err, ErrRedisUnavailable) {
		t.Fatalf(
			"Allow() error = %v, want %v",
			err,
			ErrRedisUnavailable,
		)
	}

	if allowed {
		t.Fatal("request was allowed despite Redis being unavailable")
	}

	_ = client.Close()
}
