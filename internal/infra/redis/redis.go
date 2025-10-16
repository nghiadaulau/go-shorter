package redis

import (
	"time"

	"github.com/redis/go-redis/v9"
)

func NewClient(addr string, db int, pool int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         addr,
		DB:           db,
		PoolSize:     pool,
		DialTimeout:  200 * time.Millisecond,
		ReadTimeout:  150 * time.Millisecond,
		WriteTimeout: 150 * time.Millisecond,
	})
}
