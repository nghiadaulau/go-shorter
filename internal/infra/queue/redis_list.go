package queue

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisListQueue struct {
	client *redis.Client
	key    string
}

func NewRedisListQueue(client *redis.Client, key string) *RedisListQueue {
	return &RedisListQueue{client: client, key: key}
}

func (q *RedisListQueue) Enqueue(ctx context.Context, payload string) error {
	return q.client.LPush(ctx, q.key, payload).Err()
}

func (q *RedisListQueue) Dequeue(ctx context.Context, block time.Duration) (string, error) {
	res, err := q.client.BRPop(ctx, block, q.key).Result()
	if err != nil {
		return "", err
	}
	if len(res) == 2 {
		return res[1], nil
	}
	return "", nil
}
