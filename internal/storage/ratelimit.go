package storage

// ratelimit.go — Fixed-window rate limiter backed by Redis.
//
// Per API key, per minute. Window resets on the first request each minute.
// Not a perfect sliding window, but simple and sufficient for v1.
//
// Redis key:  "rl:apikey:{keyID}"
// Strategy:  INCR + EXPIRE 60s on first increment.

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const rateLimitWindow = 60 * time.Second

type RateLimiter struct {
	rdb *redis.Client
}

func NewRateLimiter(redisURL string) (*RateLimiter, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("rate limiter: parse redis URL: %w", err)
	}
	rdb := redis.NewClient(opts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("rate limiter: redis ping: %w", err)
	}
	return &RateLimiter{rdb: rdb}, nil
}

// Allow returns (allowed, currentCount, error).
// limit is the maximum number of requests per minute for this key.
func (r *RateLimiter) Allow(ctx context.Context, keyID string, limit int) (bool, int64, error) {
	redisKey := "rl:apikey:" + keyID

	pipe := r.rdb.Pipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, rateLimitWindow)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, 0, fmt.Errorf("rate limiter: %w", err)
	}

	count := incr.Val()
	return count <= int64(limit), count, nil
}
