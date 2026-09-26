package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/joho/godotenv"
)

// Config represents the unified application configuration.
type Config struct {
	LLM           LLMConfig              `json:"llm"`
	Providers     map[string]Provider    `json:"providers"`
	Routing       RoutingConfig          `json:"routing"`
	Orchestration OrchestrationConfig    `json:"orchestration"`
	MCPServers    map[string]MCPServer   `json:"mcp_servers"`
	Tools         map[string]interface{} `json:"tools"`
	Memory        MemoryConfig           `json:"memory"`
	Models        map[string]ModelConfig `json:"models"`
	Persistence   PersistenceConfig      `json:"persistence"`
	Observability ObservabilityConfig    `json:"observability"`
	Security      SecurityConfig         `json:"security"`
	HITL          HITLConfig             `json:"hitl"`
}

type LLMConfig struct {
	DefaultModel    string        `json:"default_model"`
	FailoverModel   string        `json:"failover_model"`
	FailoverEnabled bool          `json:"failover_enabled"`
	MaxRetries      int           `json:"max_retries"`
	Timeout         time.Duration `json:"-"`
	TimeoutStr      string        `json:"timeout"`
	PricingTTL      time.Duration `json:"-"`
	PricingTTLStr   string        `json:"pricing_ttl"`
	MaxBudgetUSD    float64       `json:"max_budget_usd"`
}

type Provider struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

type ModelConfig struct {
	ProviderID      string  `json:"provider_id"`
	ModelID         string  `json:"model_id"`
	Name            string  `json:"name"`
	PromptPrice     float64 `json:"prompt_price_per_token"`
	CompletionPrice float64 `json:"completion_price_per_token"`
}

type RoutingConfig struct {
	Default        string `json:"default"`
	Vision         string `json:"vision"`
	RAG            string `json:"rag"`
	LongContext    string `json:"long_context"`
	CodeGeneration string `json:"code_generation"`
}

type OrchestrationConfig struct {
	SupervisorModel  string `json:"supervisor_model"`
	ResearcherModel  string `json:"researcher_model"`
	ArchitectModel   string `json:"architect_model"`
	ImplementerModel string `json:"implementer_model"`
}

type MCPServer struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type MemoryConfig struct {
	ChromaTimeout    time.Duration `json:"-"`
	ChromaTimeoutStr string        `json:"chroma_timeout"`
	EmbeddingModel   string        `json:"embedding_model"`
}

type PersistenceConfig struct {
	Sessions SessionConfig          `json:"sessions"`
	Cache    PersistenceCacheConfig `json:"cache"`
}

type SessionConfig struct {
	Driver                string        `json:"driver"`
	ConnectionString      string        `json:"connection_string"`
	CheckpointInterval    time.Duration `json:"-"`
	CheckpointIntervalStr string        `json:"checkpoint_interval"`
}

type PersistenceCacheConfig struct {
	Type  string      `json:"type"` // "file", "redis", "sql"
	Redis RedisConfig `json:"redis"`
}

type RedisConfig struct {
	Addr     string        `json:"addr"`
	Password string        `json:"password"`
	DB       int           `json:"db"`
	TTL      time.Duration `json:"-"`
	TTLStr   string        `json:"ttl"`
}

type ObservabilityConfig struct {
	Enabled     bool             `json:"enabled"`
	ServiceName string           `json:"service_name"`
	Prometheus  PrometheusConfig `json:"prometheus"`
	Splunk      SplunkConfig     `json:"splunk"`
	Tracing     TracingConfig    `json:"tracing"`
}

type PrometheusConfig struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
}

type SplunkConfig struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
	Token    string `json:"token"`
}

type TracingConfig struct {
	SamplingRate float64 `json:"sampling_rate"`
	Exporter     string  `json:"exporter"` // "otlp", "stdout"
}

type SecurityConfig struct {
	ShellAllowList []string `json:"shell_allow_list"`
	FileChroot     string   `json:"file_chroot"`
	NetworkEgress  string   `json:"network_egress"`
	MaxFileSizeMB  int      `json:"max_file_size_mb"`
}

type HITLConfig struct {
	Enabled        bool   `json:"enabled"`
	DefaultMode    string `json:"default_mode"` // "prompt", "registry"
	TimeoutSeconds int    `json:"timeout_seconds"`
}

var (
	instance *Config
	once     sync.Once
)

// parseDurationOrDefault parses a duration string, logging a warning and
// returning def when parsing fails instead of silently using zero values.
func parseDurationOrDefault(raw, field string, def time.Duration) time.Duration {
	if raw == "" {
		return def
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d
	} else {
		slog.Warn("config: invalid duration, using default",
			"field", field, "value", raw, "default", def.String(), "error", err)
		return def
	}
}

// Get returns the global configuration singleton.
// It panics on first load if the config file cannot be read or parsed —
// this is intentional for fail-fast startup: a running process with broken
// config is unsafe. Use TryGet or LoadConfigFile for non-panicking variants.
func Get() *Config {
	once.Do(func() {
		instance = loadConfig()
	})
	return instance
}

// TryGet returns the global configuration singleton without panicking.
// It returns an error if the config cannot be loaded, allowing callers
// (servers, CLIs) to log and exit gracefully instead of crashing.
func TryGet() (*Config, error) {
	var loadErr error
	once.Do(func() {
		var err error
		instance, err = loadConfigE()
		if err != nil {
			loadErr = err
		}
	})
	if loadErr != nil {
		return nil, loadErr
	}
	return instance, nil
}

// LoadConfigFile reads and parses a config file without panicking.
// It returns an error if the file cannot be read or parsed, allowing
// callers to handle configuration failures gracefully.
func LoadConfigFile(path string) (*Config, error) {
	if path == "" {
		path = "config.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read configuration file: %w", err)
	}
	expanded := os.ExpandEnv(string(data))
	cfg := &Config{
		Tools:      make(map[string]interface{}),
		Models:     make(map[string]ModelConfig),
		MCPServers: make(map[string]MCPServer),
		Providers:  make(map[string]Provider),
	}
	if err := json.Unmarshal([]byte(expanded), cfg); err != nil {
		slog.Warn("config: failed to parse configuration file", "path", path, "error", err)
		return nil, fmt.Errorf("failed to parse configuration file: %w", err)
	}
	// Double check API keys if expansion failed
	for k, p := range cfg.Providers {
		if p.APIKey == "" || p.APIKey == "${"+strings.ToUpper(k)+"_API_KEY}" {
			envKey := strings.ToUpper(k) + "_API_KEY"
			if val := os.Getenv(envKey); val != "" {
				p.APIKey = val
				cfg.Providers[k] = p
			}
		}
	}
	// Parse durations with safe defaults; warnings emitted on invalid values.
	cfg.LLM.Timeout = parseDurationOrDefault(cfg.LLM.TimeoutStr, "llm.timeout", 30*time.Second)
	cfg.LLM.PricingTTL = parseDurationOrDefault(cfg.LLM.PricingTTLStr, "llm.pricing_ttl", 1*time.Hour)
	cfg.Memory.ChromaTimeout = parseDurationOrDefault(cfg.Memory.ChromaTimeoutStr, "memory.chroma_timeout", 10*time.Second)
	cfg.Persistence.Sessions.CheckpointInterval = parseDurationOrDefault(cfg.Persistence.Sessions.CheckpointIntervalStr, "persistence.sessions.checkpoint_interval", 30*time.Second)
	cfg.Persistence.Cache.Redis.TTL = parseDurationOrDefault(cfg.Persistence.Cache.Redis.TTLStr, "persistence.cache.redis.ttl", 24*time.Hour)
	llm.SetGlobalBudget(cfg.LLM.MaxBudgetUSD)
	for name, model := range cfg.Models {
		if model.PromptPrice > 0 || model.CompletionPrice > 0 {
			llm.SetModelPricing(name, llm.ModelPricing{
				PromptPricePerToken:     model.PromptPrice,
				CompletionPricePerToken: model.CompletionPrice,
			})
		}
	}
	return cfg, nil
}

func loadConfig() *Config {
	cfg, err := loadConfigE()
	if err != nil {
		// Fail fast at startup but don't leak internal paths to end users;
		// full path is logged at Warn level inside LoadConfigFile.
		panic(fmt.Sprintf("config: %v", err))
	}
	return cfg
}

// loadConfigE is the error-returning core used by TryGet and loadConfig.
func loadConfigE() (*Config, error) {
	_ = godotenv.Load()
	path := os.Getenv("CREW_CONFIG_PATH")
	if path == "" {
		path = "config.json"
	}
	cfg, err := LoadConfigFile(path)
	if err != nil {
		return nil, err
	}
	instance = cfg
	return cfg, nil
}

// GetToolParam returns a tool-specific configuration parameter.
func (c *Config) GetToolParam(tool, key string) string {
	if params, ok := c.Tools[tool].(map[string]interface{}); ok {
		if val, ok := params[key].(string); ok {
			return val
		}
	}
	return ""
}

// resetGetForTest resets the singleton for testing purposes.
// This is only used in tests to avoid singleton caching issues.
func resetGetForTest() {
	instance = nil
	once = sync.Once{}
}
