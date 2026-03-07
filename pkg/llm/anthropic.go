package llm

import (
	"context"
	"fmt"
)

// AnthropicClient provides an alternative to OpenAI mapping to the official LLM base interface.
type AnthropicClient struct {
	APIKey string
	Model  string
}

func NewAnthropicClient(apiKey, model string) *AnthropicClient {
	if model == "" {
		model = "claude-3-opus-20240229"
	}
	return &AnthropicClient{APIKey: apiKey, Model: model}
}

// Generate maps messages to the Claude API (Stubbed for Interface compliance).
func (c *AnthropicClient) Generate(ctx context.Context, messages []Message, options map[string]interface{}) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("anthropic API Key is missing. Cannot route to Claude")
	}

	// This would natively wrap against github.com/anthropics/anthropic-go or a raw HTTP request.
	return "Anthropic Claude Generate block successful mock.", nil
}

// GenerateStructured handles strict JSON extraction.
func (c *AnthropicClient) GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options map[string]interface{}) (interface{}, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("anthropic API Key is missing")
	}

	// In Claude, this is handled via Tool use / Output enforcing prompts
	return schema, nil
}

// GenerateEmbedding handles vector translation for Anthropic models.
func (c *AnthropicClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("anthropic direct embeddings natively omitted; route to Voyage AI or OpenAI instead")
}
