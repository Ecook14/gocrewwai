package llm

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/kv"
	"github.com/redis/go-redis/v9"
)

// RedisCache implement the Cache interface using a Redis-protocol backend
// (Redis, Dragonfly, or Valkey).
type RedisCache struct {
	client redis.UniversalClient
	ttl    time.Duration
}

// NewRedisCache creates a new Redis-backed cache.
func NewRedisCache(addr, password string, db int, ttl time.Duration) (*RedisCache, error) {
	if addr == "" {
		return nil, fmt.Errorf("redis address is required")
	}

	client, err := kv.Dial(context.Background(), kv.Config{
		Addrs:    []string{addr},
		Password: password,
		DB:       db,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	if ttl == 0 {
		ttl = 72 * time.Hour
	}

	return &RedisCache{
		client: client,
		ttl:    ttl,
	}, nil
}

func (c *RedisCache) Get(key string) (string, bool) {
	ctx := context.Background()
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false
	} else if err != nil {
		slog.Warn("llm cache: redis get failed", slog.String("error", err.Error()))
		return "", false
	}
	return val, true
}

func (c *RedisCache) Set(key, value string) error {
	ctx := context.Background()
	return c.client.Set(ctx, key, value, c.ttl).Err()
}
