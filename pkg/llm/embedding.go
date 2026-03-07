package llm

import (
	"context"
	"fmt"
	"github.com/sashabaranov/go-openai"
)

// GenerateEmbedding handles translating raw text context into mathematical vectors.
// Maps to Python `litellm.embedding()` functionality natively.
func (c *OpenAIClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API Key is required for embeddings")
	}

	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.AdaEmbeddingV2,
	}

	resp, err := c.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("embedding creation failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned from OpenAI")
	}

	return resp.Data[0].Embedding, nil
}
