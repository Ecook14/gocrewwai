package llm

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/sashabaranov/go-openai"
)

// GroqClient leverages the OpenAI-compatible API of Groq.
type GroqClient struct {
	*OpenAIClient
	DefaultModel string
}

// NewGroqClient creates a client configured for Groq.
func NewGroqClient(apiKey, model string) *GroqClient {
	if apiKey == "" {
		apiKey = os.Getenv("GROQ_API_KEY")
	}
	if model == "" {
		model = os.Getenv("GROQ_MODEL")
	}
	if model == "" {
		model = "mixtral-8x7b-32768" // Fast and reliable default
	}

	// OpenAI-compatible config for Groq
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://api.groq.com/openai/v1"

	httpClient := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &retryRoundTripper{
			next:       http.DefaultTransport,
			maxRetries: 3,
		},
	}
	config.HTTPClient = httpClient

	return &GroqClient{
		OpenAIClient: &OpenAIClient{
			APIKey:     apiKey,
			client:     openai.NewClientWithConfig(config),
			HTTPClient: httpClient,
		},
		DefaultModel: model,
	}
}

// Generate overrides the base OpenAI Generate to inject the default model if not provided.
func (c *GroqClient) Generate(ctx context.Context, messages []Message, options map[string]interface{}) (string, error) {
	if options == nil {
		options = make(map[string]interface{})
	}
	if options["model"] == nil {
		options["model"] = c.DefaultModel
	}
	return c.OpenAIClient.Generate(ctx, messages, options)
}

// GenerateStructured overrides the base OpenAI GenerateStructured to inject the default model if not provided.
func (c *GroqClient) GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options map[string]interface{}) (interface{}, error) {
	if options == nil {
		options = make(map[string]interface{})
	}
	if options["model"] == nil {
		options["model"] = c.DefaultModel
	}
	return c.OpenAIClient.GenerateStructured(ctx, messages, schema, options)
}

// GenerateWithUsage overrides the base to inject the default model and provider.
func (c *GroqClient) GenerateWithUsage(ctx context.Context, messages []Message, options map[string]interface{}) (string, *Usage, error) {
	if options == nil {
		options = make(map[string]interface{})
	}
	if options["model"] == nil {
		options["model"] = c.DefaultModel
	}
	text, usage, err := c.OpenAIClient.GenerateWithUsage(ctx, messages, options)
	if usage != nil {
		usage.Provider = "groq"
	}
	return text, usage, err
}

// StreamGenerate overrides the base OpenAI StreamGenerate to inject the default model if not provided.
func (c *GroqClient) StreamGenerate(ctx context.Context, messages []Message, options map[string]interface{}) (<-chan string, error) {
	if options == nil {
		options = make(map[string]interface{})
	}
	if options["model"] == nil {
		options["model"] = c.DefaultModel
	}
	return c.OpenAIClient.StreamGenerate(ctx, messages, options)
}
