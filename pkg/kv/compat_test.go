package kv_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/kv"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/memory"
)

// Compat-matrix harness: exercises every Redis-protocol consumer (memory,
// checkpoints, LLM cache) against a live backend. Backend-agnostic by
// design — point KV_TEST_ADDR at Dragonfly, Redis, or Valkey:
//
//	REDIS_BACKEND=dragonfly KV_TEST_ADDR=localhost:6379 go test ./pkg/kv/ -run TestCompat -v
//
// Skips when nothing listens (unit suites stay hermetic; CI provides the
// service container). Asserts only commands both engines guarantee:
// SET/GET/DEL/SCAN + TTL expiry.
func compatAddr(t *testing.T) string {
	t.Helper()
	addr := os.Getenv("KV_TEST_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	cfg, err := kv.ConfigFromEnv()
	if err != nil {
		t.Fatalf("env: %v", err)
	}
	cfg.Addrs = []string{addr}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := kv.Dial(ctx, cfg)
	if err != nil {
		t.Skipf("no live backend at %s (%s): %v", addr, cfg.Backend, err)
	}
	client.Close()
	t.Logf("compat matrix vs backend=%s addr=%s", cfg.Backend, addr)
	return addr
}

func TestCompat_MemoryStore(t *testing.T) {
	addr := compatAddr(t)
	ctx := context.Background()
	st, err := memory.NewRedisStore([]string{addr}, "", 0, "compat-mem:")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	item := &memory.MemoryItem{ID: "compat-1", Text: "hello"}
	if err := st.Add(ctx, item); err != nil {
		t.Fatalf("add: %v", err)
	}
	n, err := st.Count(ctx)
	if err != nil || n < 1 {
		t.Fatalf("count = %d, %v", n, err)
	}
	if err := st.Delete(ctx, "compat-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := st.Reset(ctx); err != nil {
		t.Fatalf("reset: %v", err)
	}
}

func TestCompat_CheckpointStore(t *testing.T) {
	addr := compatAddr(t)
	ctx := context.Background()
	st, err := crew.NewRedisCheckpointStore(crew.RedisCheckpointConfig{
		Addr: addr, Prefix: "compat-cp:", TTL: "1h",
	})
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	cp := &crew.Checkpoint{CrewID: "compat-crew"}
	if err := st.Save(ctx, cp); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := st.LoadLatest(ctx, "compat-crew")
	if err != nil || got == nil || got.CrewID != "compat-crew" {
		t.Fatalf("load = %+v, %v", got, err)
	}
	// Regression: ListCheckpoints SCANs with a glob pattern. A pattern ending
	// in ":" (no "*") matches only an exact key of that name, so this returned
	// 0 on every Redis-protocol backend while Save/LoadLatest kept working.
	list, err := st.ListCheckpoints(ctx, "compat-crew")
	if err != nil || len(list) == 0 {
		t.Fatalf("list = %d, %v", len(list), err)
	}
	// The "latest" pointer is not a checkpoint entry and must be excluded.
	for _, c := range list {
		if c == nil || c.CrewID != "compat-crew" {
			t.Fatalf("unexpected checkpoint in list: %+v", c)
		}
	}
	// Listing a crew with no checkpoints returns empty, not an error.
	empty, err := st.ListCheckpoints(ctx, "compat-crew-absent")
	if err != nil {
		t.Fatalf("list absent crew: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected no checkpoints for absent crew, got %d", len(empty))
	}
}

func TestCompat_LLMCache(t *testing.T) {
	addr := compatAddr(t)
	c, err := llm.NewRedisCache(addr, "", 0, time.Minute)
	if err != nil {
		t.Fatalf("cache: %v", err)
	}
	if err := c.Set("compat-k", "compat-v"); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, ok := c.Get("compat-k")
	if !ok || got != "compat-v" {
		t.Fatalf("get = %q, %v", got, ok)
	}
}
