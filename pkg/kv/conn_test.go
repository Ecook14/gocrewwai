package kv

import (
	"testing"
)

func TestConfigFromEnv_Defaults(t *testing.T) {
	t.Setenv("REDIS_BACKEND", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("REDIS_DB", "")
	t.Setenv("REDIS_POOL_SIZE", "")
	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("defaults: %v", err)
	}
	if cfg.Backend != BackendRedis {
		t.Errorf("backend = %q", cfg.Backend)
	}
	if len(cfg.Addrs) != 1 || cfg.Addrs[0] != "localhost:6379" {
		t.Errorf("addrs = %v", cfg.Addrs)
	}
}

func TestConfigFromEnv_Dragonfly(t *testing.T) {
	t.Setenv("REDIS_BACKEND", "dragonfly")
	t.Setenv("REDIS_ADDR", "dragonfly:6379")
	t.Setenv("REDIS_PASSWORD", "pw")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("REDIS_POOL_SIZE", "25")
	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("dragonfly: %v", err)
	}
	if cfg.Backend != BackendDragonfly || cfg.Addrs[0] != "dragonfly:6379" ||
		cfg.Password != "pw" || cfg.DB != 2 || cfg.PoolSize != 25 {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestConfigFromEnv_Rejects(t *testing.T) {
	for _, env := range [][2]string{
		{"REDIS_BACKEND", "memcached"},
		{"REDIS_DB", "99"},
		{"REDIS_DB", "-1"},
		{"REDIS_POOL_SIZE", "0"},
		{"REDIS_POOL_SIZE", "abc"},
	} {
		t.Setenv(env[0], env[1])
		if _, err := ConfigFromEnv(); err == nil {
			t.Errorf("%s=%q accepted", env[0], env[1])
		}
		t.Setenv(env[0], "")
	}
}

func TestDial_NoAddrs(t *testing.T) {
	if _, err := Dial(t.Context(), Config{}); err == nil {
		t.Error("expected error for empty addrs")
	}
}

func TestDial_Unreachable(t *testing.T) {
	// 127.0.0.1:1 is (almost) certainly closed: fast refusal, no hang.
	cfg := Config{Backend: BackendDragonfly, Addrs: []string{"127.0.0.1:1"}}
	if _, err := Dial(t.Context(), cfg); err == nil {
		t.Error("expected dial failure")
	}
}
