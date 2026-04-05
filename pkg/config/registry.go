package config

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/Ecook14/gocrewwai/pkg/llm"
)

var (
	registryMu sync.RWMutex
	clients    = make(map[string]llm.Client)
)

// GetClient resolves and returns a pre-configured LLM client for the given model name or alias.
// This is moved from pkg/llm to break the import cycle.
func GetClient(modelName string) (llm.Client, error) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if client, ok := clients[modelName]; ok {
		return client, nil
	}

	cfg := Get()
	modelCfg, ok := cfg.Models[modelName]
	
	var providerID, actualModelID string
	if ok {
		providerID = modelCfg.ProviderID
		actualModelID = modelCfg.ModelID
	} else {
		// Fallback to direct detection (e.g. "openai/gpt-4o")
		if strings.Contains(modelName, "/") {
			parts := strings.SplitN(modelName, "/", 2)
			providerID = parts[0]
			actualModelID = parts[1]
		} else {
			// Legacy detection by prefix
			return detectAndCreateClient(modelName)
		}
	}

	client, err := createClientForProvider(providerID, actualModelID)
	if err != nil {
		return nil, err
	}

	// Always wrap with default hardening
	wrapped := llm.WrapClient(client,
		llm.WithMaxRetries(cfg.LLM.MaxRetries),
		llm.WithTimeout(cfg.LLM.Timeout),
	)

	// If a failover model is configured, wrap with FailoverClient
	if cfg.LLM.FailoverModel != "" && cfg.LLM.FailoverModel != modelName {
		secondaryCfg, sOk := cfg.Models[cfg.LLM.FailoverModel]
		if sOk {
			sClient, sErr := createClientForProvider(secondaryCfg.ProviderID, secondaryCfg.ModelID)
			if sErr == nil {
				failover := llm.NewFailoverClient(wrapped, sClient, nil)
				clients[modelName] = failover
				return failover, nil
			}
		}
	}

	clients[modelName] = wrapped
	return wrapped, nil
}

func detectAndCreateClient(model string) (llm.Client, error) {
	lower := strings.ToLower(model)
	if strings.Contains(lower, "gpt-") {
		return createClientForProvider("openai", model)
	}
	if strings.Contains(lower, "claude-") {
		return createClientForProvider("anthropic", model)
	}
	if strings.Contains(lower, "gemini-") {
		return createClientForProvider("google", model)
	}
	return nil, fmt.Errorf("unknown model provider for %s: please define in config.json", model)
}

func createClientForProvider(providerName, model string) (llm.Client, error) {
	cfg := Get()
	p, ok := cfg.Providers[strings.ToLower(providerName)]
	if !ok {
		return nil, fmt.Errorf("provider configuration not found for: %s", providerName)
	}

	apiKey := p.APIKey
	if apiKey == "" {
		apiKey = os.Getenv(strings.ToUpper(providerName) + "_API_KEY")
	}

	switch strings.ToLower(providerName) {
	case "openai":
		client := llm.NewOpenAIClient(apiKey)
		if p.BaseURL != "" {
			client.WithBaseURL(p.BaseURL)
		}
		return client, nil
	case "anthropic":
		client := llm.NewAnthropicClient(apiKey, model)
		if p.BaseURL != "" {
			client.WithBaseURL(p.BaseURL)
		}
		return client, nil
	case "google", "gemini":
		client := llm.NewGeminiClient(apiKey, model)
		if p.BaseURL != "" {
			client.WithBaseURL(p.BaseURL)
		}
		return client, nil
	case "openrouter":
		client := llm.NewOpenRouterClient(apiKey, model)
		if p.BaseURL != "" {
			client.WithBaseURL(p.BaseURL)
		}
		return client, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}
