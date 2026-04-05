package llm

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CacheConfig represents the settings for the LLM cache.
type CacheConfig struct {
	Type  string
	Dir   string
	Redis struct {
		Addr     string
		Password string
		DB       int
		TTL      time.Duration
	}
}

// Cache interface for persisting LLM responses and tool results.
type Cache interface {
	Get(key string) (string, bool)
	Set(key, value string) error
}

// FileCache implements the Cache interface using the local filesystem.
type FileCache struct {
	Dir string
	mu  sync.RWMutex
}

func NewFileCache(dir string) *FileCache {
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".gocrew", "cache")
	}
	_ = os.MkdirAll(dir, 0755)
	return &FileCache{Dir: dir}
}

func (c *FileCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
	path := filepath.Join(c.Dir, hash)

	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
}

func (c *FileCache) Set(key, value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
	path := filepath.Join(c.Dir, hash)

	return os.WriteFile(path, []byte(value), 0644)
}

// NewCache returns the configured cache implementation.
func NewCache(cfg CacheConfig) Cache {
	switch strings.ToLower(cfg.Type) {
	case "redis":
		if c, err := NewRedisCache(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, cfg.Redis.TTL); err == nil {
			return c
		}
		fmt.Printf("Warning: Failed to init Redis cache, falling back to FileCache\n")
		return NewFileCache(cfg.Dir)
	default:
		return NewFileCache(cfg.Dir)
	}
}

// GenerateCacheKey creates a stable key from the prompt and options.
func GenerateCacheKey(model, prompt string, options interface{}) string {
	optsJson, _ := json.Marshal(options)
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s|%s|%s", model, prompt, string(optsJson))))
	return fmt.Sprintf("llm:cache:%x", hasher.Sum(nil))
}
