package llm

import "context"

// Message represents a general message structure used in LLM communication.
type Message struct {
	Role    string
	Content string
	Images  []string // URLs or base64-encoded image data
}

// GenerateOptions configures a single LLM request.
type GenerateOptions struct {
	Model       string                 `json:"model"`
	Temperature float32                `json:"temperature"`
	MaxTokens   int                    `json:"max_tokens"`
	Stop        []string               `json:"stop,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"` // Provider-specific extensions
}

// Client represents the base capabilities for language model generation.
type Client interface {
	// Generate is the core unstructured mapping block
	Generate(ctx context.Context, messages []Message, options GenerateOptions) (string, error)

	// GenerateWithUsage is like Generate but also returns token usage and cost data.
	GenerateWithUsage(ctx context.Context, messages []Message, options GenerateOptions) (string, *Usage, error)

	// GenerateStructured pulls responses explicitly as populated JSON mapped into `schema`
	GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options GenerateOptions) (interface{}, error)

	// StreamGenerate provides real-time token output via a channel
	StreamGenerate(ctx context.Context, messages []Message, options GenerateOptions) (<-chan string, error)
}

// Embedder represents models capable of generating embeddings.
type Embedder interface {
	// GenerateEmbedding forces the text snippet into an ML dimensional vector representations
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// AudioGenerator represents models capable of dealing with audio.
type AudioGenerator interface {
	// GenerateSpeech converts text to audio bytes (TTS).
	GenerateSpeech(ctx context.Context, text string, options map[string]interface{}) ([]byte, error)

	// TranscribeSpeech converts audio bytes to text (STT).
	TranscribeSpeech(ctx context.Context, audio []byte, options map[string]interface{}) (string, error)
}

// ExtractStructured securely types the outcome of a structured generation request,
// bridging the un-typed `interface{}` boundary of the Client.
func ExtractStructured[T any](ctx context.Context, client Client, messages []Message, options map[string]interface{}) (*T, error) {
	var target T
	genOptions := GenerateOptions{
		Extra: options,
	}
	if model, ok := options["model"].(string); ok {
		genOptions.Model = model
	}
	if temp, ok := options["temperature"].(float64); ok {
		genOptions.Temperature = float32(temp)
	}

	_, err := client.GenerateStructured(ctx, messages, &target, genOptions)
	if err != nil {
		return nil, err
	}
	return &target, nil
}
