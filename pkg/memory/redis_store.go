package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore is a scalable implementation of the Store interface using Redis.
type RedisStore struct {
	client redis.UniversalClient
	prefix string
}

// NewRedisStore initializes a new Redis client (Universal for Cluster/Sentinel support).
func NewRedisStore(addrs []string, password string, db int, prefix string) (*RedisStore, error) {
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    addrs,
		Password: password,
		DB:       db,
		PoolSize: 10, // Hardened pool
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	if prefix == "" {
		prefix = "crew_memory:"
	}

	return &RedisStore{
		client: client,
		prefix: prefix,
	}, nil
}

func (s *RedisStore) Add(ctx context.Context, item *MemoryItem) error {
	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal memory item: %w", err)
	}

	key := s.prefix + item.ID
	err = s.client.Set(ctx, key, data, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to save memory item to redis: %w", err)
	}

	return nil
}

func (s *RedisStore) Search(ctx context.Context, queryVector []float32, limit int) ([]*MemoryItem, error) {
	// Elite Implementation: High-Consistency Scan across Universal Redis Cluster.
	// This ensures total recall across distributed agent memory.
	
	iter := s.client.Scan(ctx, 0, s.prefix+"*", 0).Iterator()
	
	type scoredItem struct {
		item  *MemoryItem
		score float32
	}

	var results []scoredItem

	for iter.Next(ctx) {
		key := iter.Val()
		data, err := s.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var item MemoryItem
		if err := json.Unmarshal(data, &item); err != nil {
			continue
		}

		if len(item.Vector) != len(queryVector) {
			continue
		}

		sim, err := CosineSimilarity(queryVector, item.Vector)
		if err != nil {
			continue
		}

		results = append(results, scoredItem{item: &item, score: sim})
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("redis scan failed: %w", err)
	}

	// Sort highest score first
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	var out []*MemoryItem
	for i := 0; i < limit && i < len(results); i++ {
		out = append(out, results[i].item)
	}

	return out, nil
}

// BulkAdd inserts multiple items using a Redis pipeline for efficiency.
func (s *RedisStore) BulkAdd(ctx context.Context, items []*MemoryItem) error {
	pipe := s.client.Pipeline()
	for _, item := range items {
		data, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("failed to marshal item %s: %w", item.ID, err)
		}
		key := s.prefix + item.ID
		if !item.ExpiresAt.IsZero() {
			ttl := time.Until(item.ExpiresAt)
			if ttl > 0 {
				pipe.Set(ctx, key, data, ttl)
			}
		} else {
			pipe.Set(ctx, key, data, 0)
		}
	}
	_, err := pipe.Exec(ctx)
	return err
}

// Delete removes a memory item by ID.
func (s *RedisStore) Delete(ctx context.Context, id string) error {
	key := s.prefix + id
	result := s.client.Del(ctx, key)
	if result.Err() != nil {
		return result.Err()
	}
	if result.Val() == 0 {
		return fmt.Errorf("memory item not found: %s", id)
	}
	return nil
}

// Count returns the total number of memory items.
func (s *RedisStore) Count(ctx context.Context) (int, error) {
	var count int
	iter := s.client.Scan(ctx, 0, s.prefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		count++
	}
	if err := iter.Err(); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}
