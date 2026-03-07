package llm

import "context"

// Message represents a general message structure used in LLM communication.
type Message struct {
	Role    string
	Content string
}

// Client represents the base capabilities for language model generation.
type Client interface {
	// Generate is the core unstructured mapping block
	Generate(ctx context.Context, messages []Message, options map[string]interface{}) (string, error)

	// GenerateStructured pulls responses explicitly as populated JSON mapped into `schema`
	GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options map[string]interface{}) (interface{}, error)

	// GenerateEmbedding forces the text snippet into an ML dimensional vector representations
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}
