// Package kv is the shared Redis-protocol connection layer for the engine.
//
// All Redis-protocol backends (Redis, Dragonfly, Valkey) speak the same wire
// protocol, so one helper dials all of them via go-redis. REDIS_BACKEND
// selects the backend for logging/metrics labels only — never for code
// forks. Supported: "redis" (default), "dragonfly", "valkey".
package kv

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Backends known to the engine. Unknown values are rejected (fail closed)
// so a typo can never silently target the wrong store.
const (
	BackendRedis     = "redis"
	BackendDragonfly = "dragonfly"
	BackendValkey    = "valkey"
	defaultPoolSize  = 10
	defaultDial      = 5 * time.Second
)

// Config describes a Redis-protocol endpoint.
type Config struct {
	// Backend is one of redis|dragonfly|valkey. Informational: the wire
	// protocol is identical; the value is logged and exposed for metrics.
	Backend     string
	Addrs       []string
	Password    string
	DB          int
	PoolSize    int
	DialTimeout time.Duration
}

// envInt parses a positive integer env var with a bound check, returning
// fallback if unset/empty. Returns an error only on malformed values.
func envInt(name string, fallback, min, max int) (int, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("kv: invalid %s %q", name, v)
	}
	return n, nil
}

// ConfigFromEnv builds a Config from REDIS_BACKEND / REDIS_ADDR /
// REDIS_PASSWORD / REDIS_DB / REDIS_POOL_SIZE. Defaults target a local
// single instance (localhost:6379) on either backend.
func ConfigFromEnv() (Config, error) {
	backend := strings.ToLower(strings.TrimSpace(os.Getenv("REDIS_BACKEND")))
	if backend == "" {
		backend = BackendRedis
	}
	switch backend {
	case BackendRedis, BackendDragonfly, BackendValkey:
	default:
		return Config{}, fmt.Errorf("kv: unsupported REDIS_BACKEND %q (want redis|dragonfly|valkey)", backend)
	}
	var addrs []string
	if v := strings.TrimSpace(os.Getenv("REDIS_ADDR")); v != "" {
		for _, a := range strings.Split(v, ",") {
			if a = strings.TrimSpace(a); a != "" {
				addrs = append(addrs, a)
			}
		}
	} else {
		addrs = []string{"localhost:6379"}
	}
	db, err := envInt("REDIS_DB", 0, 0, 15)
	if err != nil {
		return Config{}, err
	}
	pool, err := envInt("REDIS_POOL_SIZE", defaultPoolSize, 1, 500)
	if err != nil {
		return Config{}, err
	}
	return Config{
		Backend:     backend,
		Addrs:       addrs,
		Password:    os.Getenv("REDIS_PASSWORD"),
		DB:          db,
		PoolSize:    pool,
		DialTimeout: defaultDial,
	}, nil
}

// Dial connects, Pings with a timeout, and logs the selected backend.
// Callers keep their own key prefixes/TTLs; this helper owns only transport.
func Dial(ctx context.Context, cfg Config) (redis.UniversalClient, error) {
	if len(cfg.Addrs) == 0 {
		return nil, fmt.Errorf("kv: no addresses configured")
	}
	pool := cfg.PoolSize
	if pool <= 0 {
		pool = defaultPoolSize
	}
	timeout := cfg.DialTimeout
	if timeout <= 0 {
		timeout = defaultDial
	}
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    cfg.Addrs,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: pool,
	})
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("kv: ping %s at %s failed: %w", cfg.Backend, strings.Join(cfg.Addrs, ","), err)
	}
	slog.Info("kv: connected", slog.String("backend", cfg.Backend), slog.String("addrs", strings.Join(cfg.Addrs, ",")))
	return client, nil
}
