package llm

import "context"

// Message represents a general message structure used in LLM communication.
type Message struct {
	Role    string
	Content string
	Images  []string // URLs or base64-encoded image data
}

// Client represents the base capabilities for language model generation.
type Client interface {
	// Generate is the core unstructured mapping block
	Generate(ctx context.Context, messages []Message, options map[string]interface{}) (string, error)

	// GenerateWithUsage is like Generate but also returns token usage and cost data.
	// Providers that don't support usage tracking should return a nil Usage.
	GenerateWithUsage(ctx context.Context, messages []Message, options map[string]interface{}) (string, *Usage, error)

	// GenerateStructured pulls responses explicitly as populated JSON mapped into `schema`
	GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options map[string]interface{}) (interface{}, error)

	// GenerateEmbedding forces the text snippet into an ML dimensional vector representations
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)

	// StreamGenerate provides real-time token output via a channel
	StreamGenerate(ctx context.Context, messages []Message, options map[string]interface{}) (<-chan string, error)

	// Elite Tier: Multimodal Speech
	// GenerateSpeech converts text to audio bytes (TTS).
	GenerateSpeech(ctx context.Context, text string, options map[string]interface{}) ([]byte, error)

	// TranscribeSpeech converts audio bytes to text (STT).
	TranscribeSpeech(ctx context.Context, audio []byte, options map[string]interface{}) (string, error)
}

// ExtractStructured securely types the outcome of a structured generation request,
// bridging the un-typed `interface{}` boundary of the Client.
func ExtractStructured[T any](ctx context.Context, client Client, messages []Message, options map[string]interface{}) (*T, error) {
	var target T
	_, err := client.GenerateStructured(ctx, messages, &target, options)
	if err != nil {
		return nil, err
	}
	return &target, nil
}
