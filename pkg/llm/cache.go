package llm

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

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
		dir = filepath.Join(home, ".crewai-go", "cache")
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

// GenerateCacheKey creates a stable key from the prompt and options.
func GenerateCacheKey(model, prompt string, options map[string]interface{}) string {
	optsJson, _ := json.Marshal(options)
	return fmt.Sprintf("model:%s|prompt:%s|opts:%s", model, prompt, string(optsJson))
}
