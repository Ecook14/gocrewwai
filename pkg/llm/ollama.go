package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// OllamaClient provides native implementation for local Ollama models.
type OllamaClient struct {
	APIKey  string
	Model   string
	BaseURL string
	client  *http.Client
}

// NewOllamaClient creates a client for local Ollama inference.
func NewOllamaClient(model string, baseURL ...string) *OllamaClient {
	url := "http://localhost:11434"
	if len(baseURL) > 0 && baseURL[0] != "" {
		url = baseURL[0]
	}
	if model == "" {
		model = os.Getenv("OLLAMA_MODEL")
	}
	if model == "" {
		model = "llama3"
	}
	return &OllamaClient{
		Model:   model,
		BaseURL: url,
		client: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

// ollamaMessage represents a single message in the Ollama chat API.
type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ollamaRequest represents the full chat request payload.
type ollamaRequest struct {
	Model      string              `json:"model"`
	Messages   []ollamaMessage     `json:"messages"`
	Stream     bool                `json:"stream"`
	Options    *ollamaOptions      `json:"options,omitempty"`
	Format     string              `json:"format,omitempty"`
	KeepAlive  string              `json:"keep_alive,omitempty"`
}

type ollamaOptions struct {
	Temperature *float32 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"num_predict,omitempty"`
	TopP        *float32 `json:"top_p,omitempty"`
}

type ollamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done      bool   `json:"done"`
	Total     int    `json:"total_duration"`
	Load      int    `json:"load_duration"`
	Evaluate  int    `json:"eval_duration"`
	TokenCount int   `json:"token_count"`
}

// ollamaEmbeddingRequest represents an embedding request.
type ollamaEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// ollamaEmbeddingResponse represents an embedding response.
type ollamaEmbeddingResponse struct {
	Model     string      `json:"model"`
	CreatedAt string      `json:"created_at"`
	Embeddings [][]float32 `json:"embeddings"`
	Total     int         `json:"total_duration"`
}

// Generate implements the Client interface for Ollama.
func (c *OllamaClient) Generate(ctx context.Context, messages []Message, options GenerateOptions) (string, error) {
	req := ollamaRequest{
		Model:    options.Model,
		Messages: toOllamaMessages(messages),
		Stream:   false,
	}
	if options.Temperature > 0 {
		req.Options = &ollamaOptions{Temperature: &options.Temperature}
	}
	if options.MaxTokens > 0 {
		req.Options = &ollamaOptions{MaxTokens: &options.MaxTokens}
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ollama request: %w", err)
	}

	resp, err := c.client.Post(c.url("/api/chat"), "application/json", strings.NewReader(string(payload)))
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var ollResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollResp); err != nil {
		return "", fmt.Errorf("failed to decode ollama response: %w", err)
	}

	return ollResp.Message.Content, nil
}

// GenerateWithUsage implements the Client interface for Ollama with token tracking.
func (c *OllamaClient) GenerateWithUsage(ctx context.Context, messages []Message, options GenerateOptions) (string, *Usage, error) {
	result, err := c.Generate(ctx, messages, options)
	if err != nil {
		return "", nil, err
	}
	return result, &Usage{
		PromptTokens:     options.MaxTokens * 2, // Estimate
		CompletionTokens: len(strings.Split(result, " ")) * 2,
		Model:            options.Model,
		Provider:         "ollama",
		Timestamp:        time.Now(),
	}, nil
}

// GenerateStructured implements the Client interface for Ollama with structured output.
func (c *OllamaClient) GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options GenerateOptions) (interface{}, error) {
	// Ollama supports JSON format via the format parameter
	options.Extra = map[string]interface{}{"response_format": "json_object"}
	result, err := c.Generate(ctx, messages, options)
	if err != nil {
		return nil, err
	}
	// Parse the JSON result
	var resultMap map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultMap); err != nil {
		return nil, fmt.Errorf("failed to parse structured response: %w", err)
	}
	return resultMap, nil
}

// StreamGenerate implements the Client interface for Ollama streaming.
func (c *OllamaClient) StreamGenerate(ctx context.Context, messages []Message, options GenerateOptions) (<-chan string, error) {
	ch := make(chan string, 100)
	req := ollamaRequest{
		Model:    options.Model,
		Messages: toOllamaMessages(messages),
		Stream:   true,
	}

	go func() {
		defer close(ch)
		payload, _ := json.Marshal(req)
		resp, err := c.client.Post(c.url("/api/chat"), "application/json", strings.NewReader(string(payload)))
		if err != nil {
			ch <- "" // Signal error by closing
			return
		}
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for decoder.More() {
			var ollResp ollamaResponse
			if err := decoder.Decode(&ollResp); err != nil {
				return
			}
			if ollResp.Message.Content != "" {
				ch <- ollResp.Message.Content
			}
			if ollResp.Done {
				break
			}
		}
	}()

	return ch, nil
}

// GenerateEmbedding implements the Embedder interface for Ollama.
func (c *OllamaClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	req := ollamaEmbeddingRequest{
		Model: c.Model,
		Input: []string{text},
	}
	payload, _ := json.Marshal(req)

	resp, err := c.client.Post(c.url("/api/embeddings"), "application/json", strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("ollama embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	var embResp ollamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	if len(embResp.Embeddings) > 0 {
		return embResp.Embeddings[0], nil
	}
	return nil, fmt.Errorf("no embeddings returned")
}

// url constructs a full URL for the Ollama API.
func (c *OllamaClient) url(path string) string {
	return fmt.Sprintf("%s%s", c.BaseURL, path)
}

// toOllamaMessages converts our Message slice to Ollama format.
func toOllamaMessages(msgs []Message) []ollamaMessage {
	result := make([]ollamaMessage, len(msgs))
	for i, m := range msgs {
		result[i] = ollamaMessage{Role: m.Role, Content: m.Content}
	}
	return result
}
