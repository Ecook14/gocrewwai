package llm

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/sashabaranov/go-openai"
)

// OpenRouterClient leverages the OpenAI-compatible API of OpenRouter.
type OpenRouterClient struct {
	*OpenAIClient
	DefaultModel string
}

// NewOpenRouterClient creates a client configured for OpenRouter.
func NewOpenRouterClient(apiKey, model string) *OpenRouterClient {
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if model == "" {
		model = os.Getenv("OPENROUTER_MODEL")
	}
	if model == "" {
		model = "openrouter/free" // Reliable catch-all for free models
	}

	// OpenAI-compatible config for OpenRouter
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://openrouter.ai/api/v1"

	httpClient := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &retryRoundTripper{
			next: &openRouterHeaderRoundTripper{
				next: http.DefaultTransport,
			},
			maxRetries: 3,
		},
	}
	config.HTTPClient = httpClient

	return &OpenRouterClient{
		OpenAIClient: &OpenAIClient{
			APIKey:     apiKey,
			client:     openai.NewClientWithConfig(config),
			HTTPClient: httpClient,
		},
		DefaultModel: model,
	}
}

// openRouterHeaderRoundTripper adds required OpenRouter headers
type openRouterHeaderRoundTripper struct {
	next http.RoundTripper
}

func (r *openRouterHeaderRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("HTTP-Referer", "https://github.com/Ecook14/crewai-go")
	req.Header.Set("X-OpenRouter-Title", "Crew-GO Framework")
	return r.next.RoundTrip(req)
}

// Generate overrides the base OpenAI Generate to inject the default model if not provided.
func (c *OpenRouterClient) Generate(ctx context.Context, messages []Message, options map[string]interface{}) (string, error) {
	if options == nil {
		options = make(map[string]interface{})
	}
	if options["model"] == nil {
		options["model"] = c.DefaultModel
	}
	return c.OpenAIClient.Generate(ctx, messages, options)
}

// GenerateStructured overrides the base OpenAI GenerateStructured to inject the default model if not provided.
func (c *OpenRouterClient) GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options map[string]interface{}) (interface{}, error) {
	if options == nil {
		options = make(map[string]interface{})
	}
	if options["model"] == nil {
		options["model"] = c.DefaultModel
	}
	return c.OpenAIClient.GenerateStructured(ctx, messages, schema, options)
}

// GenerateWithUsage overrides the base to inject the default model and provider.
func (c *OpenRouterClient) GenerateWithUsage(ctx context.Context, messages []Message, options map[string]interface{}) (string, *Usage, error) {
	if options == nil {
		options = make(map[string]interface{})
	}
	if options["model"] == nil {
		options["model"] = c.DefaultModel
	}
	text, usage, err := c.OpenAIClient.GenerateWithUsage(ctx, messages, options)
	if usage != nil {
		usage.Provider = "openrouter"
	}
	return text, usage, err
}

// StreamGenerate overrides the base OpenAI StreamGenerate to inject the default model if not provided.
func (c *OpenRouterClient) StreamGenerate(ctx context.Context, messages []Message, options map[string]interface{}) (<-chan string, error) {
	if options == nil {
		options = make(map[string]interface{})
	}
	if options["model"] == nil {
		options["model"] = c.DefaultModel
	}
	return c.OpenAIClient.StreamGenerate(ctx, messages, options)
}
