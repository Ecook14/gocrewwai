package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache implement the Cache interface using a Redis backend.
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisCache creates a new Redis-backed cache.
func NewRedisCache(addr, password string, db int, ttl time.Duration) (*RedisCache, error) {
	if addr == "" {
		return nil, fmt.Errorf("redis address is required")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
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
		fmt.Printf("Redis Get error: %v\n", err)
		return "", false
	}
	return val, true
}

func (c *RedisCache) Set(key, value string) error {
	ctx := context.Background()
	return c.client.Set(ctx, key, value, c.ttl).Err()
}
