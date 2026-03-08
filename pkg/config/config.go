package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FrameworkConfig holds framework-wide defaults and feature flags.
type FrameworkConfig struct {
	TelemetryEnabled bool
	DefaultTimeout   int
	LoggingLevel     string // "info", "debug", "warn", "error"
}

// Global defaults
var DefaultConfig = FrameworkConfig{
	TelemetryEnabled: false,
	DefaultTimeout:   30,
	LoggingLevel:     "info",
}

// AgentOverrides specific overrides for an individual agent
type AgentOverrides struct {
	AllowDelegation bool
	MemoryEnabled   bool
	SelfHealing     bool
	MaxIterations   int
}

// CrewConfig specific overrides for a crew
type CrewConfig struct {
	Verbose        bool
	ProcessTimeout int
	MemoryBackend  string // "sqlite", "redis", "chroma"
}

// ---------------------------------------------------------------------------
// Expanded Configuration — Environment-Aware, Validatable
// ---------------------------------------------------------------------------

// Config holds all Crew-GO runtime settings with env override support.
type Config struct {
	mu sync.RWMutex

	// LLM Settings
	DefaultModel    string        `json:"default_model"`
	DefaultProvider string        `json:"default_provider"`
	RequestTimeout  time.Duration `json:"request_timeout"`
	MaxRetries      int           `json:"max_retries"`

	// Rate Limiting
	RateLimitRPM int `json:"rate_limit_rpm"` // Requests per minute
	RateLimitTPM int `json:"rate_limit_tpm"` // Tokens per minute

	// Memory
	MemoryBackend string        `json:"memory_backend"` // "sqlite", "redis", "chroma", "qdrant", "pinecone", "weaviate", "memory"
	MemoryDBPath  string        `json:"memory_db_path"`
	MemoryTTL     time.Duration `json:"memory_ttl"`

	// Server
	ServerAddr     string `json:"server_addr"`
	MetricsEnabled bool   `json:"metrics_enabled"`

	// Logging
	LogLevel     string `json:"log_level"`  // "debug", "info", "warn", "error"
	LogFormat    string `json:"log_format"` // "json", "text"
	AuditLogPath string `json:"audit_log_path"`

	// Security
	APIKeyEnvVar string `json:"api_key_env_var"`

	// Execution
	MaxConcurrency int           `json:"max_concurrency"`
	TaskTimeout    time.Duration `json:"task_timeout"`
	Verbose        bool          `json:"verbose"`
}

// NewDefaultConfig returns production-ready defaults.
func NewDefaultConfig() *Config {
	return &Config{
		DefaultModel:    "gpt-4o",
		DefaultProvider: "openai",
		RequestTimeout:  60 * time.Second,
		MaxRetries:      3,
		RateLimitRPM:    60,
		RateLimitTPM:    100000,
		MemoryBackend:   "sqlite",
		MemoryDBPath:    "./crew_memory.db",
		MemoryTTL:       24 * time.Hour,
		ServerAddr:      ":9090",
		MetricsEnabled:  true,
		LogLevel:        "info",
		LogFormat:       "json",
		MaxConcurrency:  10,
		TaskTimeout:     5 * time.Minute,
		Verbose:         false,
	}
}

// LoadFromFile reads config from a JSON file, then applies env overrides.
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := NewDefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.applyEnvOverrides()
	return cfg, nil
}

// LoadFromEnv creates config purely from environment variables over defaults.
func LoadFromEnv() *Config {
	cfg := NewDefaultConfig()
	cfg.applyEnvOverrides()
	return cfg
}

// applyEnvOverrides applies CREW_GO_* environment variable overrides.
func (c *Config) applyEnvOverrides() {
	envStr := map[string]*string{
		"CREW_GO_DEFAULT_MODEL":    &c.DefaultModel,
		"CREW_GO_DEFAULT_PROVIDER": &c.DefaultProvider,
		"CREW_GO_MEMORY_BACKEND":   &c.MemoryBackend,
		"CREW_GO_MEMORY_DB_PATH":   &c.MemoryDBPath,
		"CREW_GO_SERVER_ADDR":      &c.ServerAddr,
		"CREW_GO_LOG_LEVEL":        &c.LogLevel,
		"CREW_GO_LOG_FORMAT":       &c.LogFormat,
		"CREW_GO_AUDIT_LOG_PATH":   &c.AuditLogPath,
		"CREW_GO_API_KEY_ENV_VAR":  &c.APIKeyEnvVar,
	}
	for env, target := range envStr {
		if val := os.Getenv(env); val != "" {
			*target = val
		}
	}

	envInt := map[string]*int{
		"CREW_GO_RATE_LIMIT_RPM":  &c.RateLimitRPM,
		"CREW_GO_RATE_LIMIT_TPM":  &c.RateLimitTPM,
		"CREW_GO_MAX_RETRIES":     &c.MaxRetries,
		"CREW_GO_MAX_CONCURRENCY": &c.MaxConcurrency,
	}
	for env, target := range envInt {
		if val := os.Getenv(env); val != "" {
			if n, err := strconv.Atoi(val); err == nil {
				*target = n
			}
		}
	}

	envDur := map[string]*time.Duration{
		"CREW_GO_REQUEST_TIMEOUT": &c.RequestTimeout,
		"CREW_GO_TASK_TIMEOUT":    &c.TaskTimeout,
		"CREW_GO_MEMORY_TTL":      &c.MemoryTTL,
	}
	for env, target := range envDur {
		if val := os.Getenv(env); val != "" {
			if d, err := time.ParseDuration(val); err == nil {
				*target = d
			}
		}
	}

	envBool := map[string]*bool{
		"CREW_GO_METRICS_ENABLED": &c.MetricsEnabled,
		"CREW_GO_VERBOSE":         &c.Verbose,
	}
	for env, target := range envBool {
		if val := os.Getenv(env); val != "" {
			*target = strings.EqualFold(val, "true") || val == "1"
		}
	}
}

// Get returns a thread-safe snapshot.
func (c *Config) Get() Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return *c
}

// SaveToFile persists the config as formatted JSON.
func (c *Config) SaveToFile(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Validate checks for common configuration errors.
func (c *Config) Validate() error {
	if c.MaxConcurrency < 1 {
		return fmt.Errorf("max_concurrency must be >= 1")
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be >= 0")
	}
	if c.RateLimitRPM < 1 {
		return fmt.Errorf("rate_limit_rpm must be >= 1")
	}
	validBackends := map[string]bool{
		"sqlite": true, "redis": true, "chroma": true,
		"qdrant": true, "pinecone": true, "weaviate": true, "memory": true,
	}
	if !validBackends[c.MemoryBackend] {
		return fmt.Errorf("unsupported memory_backend: %s", c.MemoryBackend)
	}
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.LogLevel] {
		return fmt.Errorf("unsupported log_level: %s", c.LogLevel)
	}
	return nil
}
