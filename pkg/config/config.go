package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/joho/godotenv"
)

// Config represents the unified application configuration.
type Config struct {
	LLM           LLMConfig               `json:"llm"`
	Providers     map[string]Provider     `json:"providers"`
	Routing       RoutingConfig           `json:"routing"`
	Orchestration OrchestrationConfig     `json:"orchestration"`
	MCPServers    map[string]MCPServer    `json:"mcp_servers"`
	Tools         map[string]interface{}  `json:"tools"`
	Memory        MemoryConfig            `json:"memory"`
	Models        map[string]ModelConfig  `json:"models"`
	Persistence   PersistenceConfig       `json:"persistence"`
	Observability ObservabilityConfig     `json:"observability"`
	Security      SecurityConfig          `json:"security"`
	HITL          HITLConfig              `json:"hitl"`
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
	RAG           string `json:"rag"`
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
	Sessions SessionConfig        `json:"sessions"`
	Cache    PersistenceCacheConfig `json:"cache"`
}

type SessionConfig struct {
	Driver             string        `json:"driver"`
	ConnectionString   string        `json:"connection_string"`
	CheckpointInterval time.Duration `json:"-"`
	CheckpointIntervalStr string    `json:"checkpoint_interval"`
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

// Get returns the global configuration singleton.
func Get() *Config {
	once.Do(func() {
		instance = loadConfig()
	})
	return instance
}

func loadConfig() *Config {
	_ = godotenv.Load()

	path := os.Getenv("CREW_CONFIG_PATH")
	if path == "" {
		path = "config.json"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Warning: Failed to read config file at %s: %v\nUsing defaults.\n", path, err)
		return &Config{}
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	cfg := &Config{
		Tools:      make(map[string]interface{}),
		Models:     make(map[string]ModelConfig),
		MCPServers: make(map[string]MCPServer),
		Providers:  make(map[string]Provider),
	}
	if err := json.Unmarshal([]byte(expanded), cfg); err != nil {
		fmt.Printf("Warning: Failed to parse config JSON: %v\n", err)
		_ = json.Unmarshal(data, cfg)
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

	// Parse durations
	if cfg.LLM.TimeoutStr != "" {
		cfg.LLM.Timeout, _ = time.ParseDuration(cfg.LLM.TimeoutStr)
	}
	if cfg.LLM.PricingTTLStr != "" {
		cfg.LLM.PricingTTL, _ = time.ParseDuration(cfg.LLM.PricingTTLStr)
	}
	if cfg.Memory.ChromaTimeoutStr != "" {
		cfg.Memory.ChromaTimeout, _ = time.ParseDuration(cfg.Memory.ChromaTimeoutStr)
	}
	if cfg.Persistence.Sessions.CheckpointIntervalStr != "" {
		cfg.Persistence.Sessions.CheckpointInterval, _ = time.ParseDuration(cfg.Persistence.Sessions.CheckpointIntervalStr)
	}
	if cfg.Persistence.Cache.Redis.TTLStr != "" {
		cfg.Persistence.Cache.Redis.TTL, _ = time.ParseDuration(cfg.Persistence.Cache.Redis.TTLStr)
	}

	// Elite: Push pricing and budget data to llm package to avoid circular imports
	llm.SetGlobalBudget(cfg.LLM.MaxBudgetUSD)
	for name, model := range cfg.Models {
		if model.PromptPrice > 0 || model.CompletionPrice > 0 {
			llm.SetModelPricing(name, llm.ModelPricing{
				PromptPricePerToken:     model.PromptPrice,
				CompletionPricePerToken: model.CompletionPrice,
			})
		}
	}

	return cfg
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
