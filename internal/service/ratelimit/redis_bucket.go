package ratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisTokenBucket struct {
	client *redis.Client
}

func NewRedisTokenBucket(client *redis.Client) *RedisTokenBucket { return &RedisTokenBucket{client: client} }

// Allow returns true if a token was consumed, false otherwise. key typically per-tenant.
func (r *RedisTokenBucket) Allow(ctx context.Context, key string, ratePerMinute int) (bool, error) {
	// Simple fixed window using INCR + EXPIRE
	windowKey := key + ":" + time.Now().UTC().Format("200601021504")
	count, err := r.client.Incr(ctx, windowKey).Result()
	if err != nil { return false, err }
	if count == 1 {
		_ = r.client.Expire(ctx, windowKey, time.Minute)
	}
	return int(count) <= ratePerMinute, nil
}
