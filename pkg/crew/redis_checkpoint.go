package crew

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCheckpointStore persists checkpoints to Redis with TTL support.
type RedisCheckpointStore struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
	mu     sync.Mutex
}

// RedisCheckpointConfig holds Redis connection settings for checkpoint storage.
type RedisCheckpointConfig struct {
	Addr     string
	Password string
	DB       int
	Prefix   string // key prefix, default "crew_checkpoint:"
	TTL      string // e.g. "24h", "72h"
}

// NewRedisCheckpointStore creates a Redis-backed checkpoint store.
func NewRedisCheckpointStore(cfg RedisCheckpointConfig) (*RedisCheckpointStore, error) {
	if cfg.Prefix == "" {
		cfg.Prefix = "crew_checkpoint:"
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis for checkpoints: %w", err)
	}

	ttl := 72 * time.Hour
	if cfg.TTL != "" {
		if d, err := time.ParseDuration(cfg.TTL); err == nil {
			ttl = d
		} else {
			slog.Warn("invalid Redis checkpoint TTL, using default 72h", "error", err)
		}
	}

	return &RedisCheckpointStore{
		client: client,
		prefix: cfg.Prefix,
		ttl:    ttl,
	}, nil
}

func (s *RedisCheckpointStore) key(crewID string, ts int64) string {
	return s.prefix + crewID + ":" + fmt.Sprintf("%d", ts)
}

func (s *RedisCheckpointStore) latestKey(crewID string) string {
	return s.prefix + crewID + ":latest"
}

// Save writes a checkpoint to Redis.
func (s *RedisCheckpointStore) Save(ctx context.Context, cp *Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp.Timestamp = time.Now()
	cp.Version++

	data, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	// Save versioned checkpoint
	key := s.key(cp.CrewID, cp.Timestamp.UnixMilli())
	if err := s.client.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to save checkpoint to redis: %w", err)
	}

	// Update latest pointer
	latest := s.latestKey(cp.CrewID)
	if err := s.client.Set(ctx, latest, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to save latest checkpoint to redis: %w", err)
	}

	return nil
}

// LoadLatest reads the most recent checkpoint for a crew.
func (s *RedisCheckpointStore) LoadLatest(ctx context.Context, crewID string) (*Checkpoint, error) {
	data, err := s.client.Get(ctx, s.latestKey(crewID)).Bytes()
	if err == redis.Nil {
		return nil, nil // no prior state
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load latest checkpoint from redis: %w", err)
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}
	return &cp, nil
}

// LoadByID reads a specific checkpoint by timestamp.
func (s *RedisCheckpointStore) LoadByID(ctx context.Context, crewID string, timestamp int64) (*Checkpoint, error) {
	data, err := s.client.Get(ctx, s.key(crewID, timestamp)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint from redis: %w", err)
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}
	return &cp, nil
}

// ListCheckpoints returns all checkpoints for a crew, newest first.
func (s *RedisCheckpointStore) ListCheckpoints(ctx context.Context, crewID string) ([]*Checkpoint, error) {
	var checkpoints []*Checkpoint

	iter := s.client.Scan(ctx, 0, s.prefix+crewID+":", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		// Skip the "latest" pointer
		if key == s.latestKey(crewID) {
			continue
		}
		data, err := s.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var cp Checkpoint
		if err := json.Unmarshal(data, &cp); err != nil {
			continue
		}
		checkpoints = append(checkpoints, &cp)
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("redis scan for checkpoints failed: %w", err)
	}

	// Sort newest first
	for i := 0; i < len(checkpoints); i++ {
		for j := i + 1; j < len(checkpoints); j++ {
			if checkpoints[j].Timestamp.After(checkpoints[i].Timestamp) {
				checkpoints[i], checkpoints[j] = checkpoints[j], checkpoints[i]
			}
		}
	}

	return checkpoints, nil
}

// Delete removes a specific checkpoint.
func (s *RedisCheckpointStore) Delete(ctx context.Context, crewID string, timestamp int64) error {
	key := s.key(crewID, timestamp)
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete checkpoint from redis: %w", err)
	}
	return nil
}

// Close shuts down the Redis connection.
func (s *RedisCheckpointStore) Close() error {
	return s.client.Close()
}
