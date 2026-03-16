package llm

import (
	"context"
	"net/http"
	"os"
	"time"
	"fmt"
	"github.com/sashabaranov/go-openai"
)

// OpenRouterClient leverages the OpenAI-compatible API of OpenRouter.
type OpenRouterClient struct {
	*OpenAIClient
	DefaultModel          string
	DefaultEmbeddingModel string
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
		Timeout: 300 * time.Second,
		Transport: &retryRoundTripper{
			next: &openRouterHeaderRoundTripper{
				next:   http.DefaultTransport,
				APIKey: apiKey,
			},
			maxRetries:   5,
			providerName: "OpenRouter",
		},
	}
	config.HTTPClient = httpClient

	c := &OpenRouterClient{
		OpenAIClient: &OpenAIClient{
			APIKey:     apiKey,
			client:     openai.NewClientWithConfig(config),
			HTTPClient: httpClient,
		},
		DefaultModel:          model,
		DefaultEmbeddingModel: os.Getenv("OPENROUTER_EMBEDDING_MODEL"),
	}
	if c.DefaultEmbeddingModel == "" {
		c.DefaultEmbeddingModel = "nvidia/llama-nemotron-embed-vl-1b-v2:free"
	}
	return c
}

// openRouterHeaderRoundTripper adds required OpenRouter headers
type openRouterHeaderRoundTripper struct {
	next   http.RoundTripper
	APIKey string
}

func (r *openRouterHeaderRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+r.APIKey)
	req.Header.Set("HTTP-Referer", "https://gocrewwai.com")
	req.Header.Set("X-OpenRouter-Title", "Gocrewwai Framework")
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

// GenerateEmbedding overrides the base OpenAI embedding to use OpenRouter-specific free models.
func (c *OpenRouterClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(c.DefaultEmbeddingModel),
	}

	resp, err := c.OpenAIClient.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned from OpenRouter")
	}

	return resp.Data[0].Embedding, nil
}
// WithBaseURL allows reconfiguring the client's endpoint.
func (c *OpenRouterClient) WithBaseURL(url string) *OpenRouterClient {
	c.OpenAIClient.WithBaseURL(url)
	return c
}
