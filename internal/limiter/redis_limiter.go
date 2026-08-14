package limiter

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrRedisUnavailable = errors.New("redis unavailable")
)

const rateLimitScript = `
local current = redis.call("INCR", KEYS[1])

if current == 1 then
	redis.call("EXPIRE", KEYS[1], ARGV[1])
end

if current > tonumber(ARGV[2]) then
	return 0
end

return 1
`

type RedisLimiter struct {
	client *redis.Client
	limit  int
	window time.Duration
}

func NewRedisLimiter(
	client *redis.Client,
	limit int,
	window time.Duration,
) *RedisLimiter {
	return &RedisLimiter{
		client: client,
		limit:  limit,
		window: window,
	}
}

func (l *RedisLimiter) Allow(
	ctx context.Context,
	key string,
) (bool, error) {
	if l.client == nil {
		return false, ErrRedisUnavailable
	}

	result, err := l.client.Eval(
		ctx,
		rateLimitScript,
		[]string{key},
		int(windowSeconds(l.window)),
		l.limit,
	).Int()

	if err != nil {
		return false, ErrRedisUnavailable
	}

	return result == 1, nil
}

func windowSeconds(window time.Duration) int64 {
	seconds := int64(window / time.Second)

	if seconds < 1 {
		return 1
	}

	return seconds
}
