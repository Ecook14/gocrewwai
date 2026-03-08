package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// AnthropicClient provides a native implementation for Anthropic Claude.
type AnthropicClient struct {
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

// NewAnthropicClient creates a new client for Anthropic.
func NewAnthropicClient(apiKey, model string) *AnthropicClient {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if model == "" {
		model = os.Getenv("ANTHROPIC_MODEL")
	}
	if model == "" {
		model = "claude-3-5-sonnet-20240620"
	}

	return &AnthropicClient{
		APIKey: apiKey,
		Model:  model,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Generate implements basic message generation for Claude.
func (c *AnthropicClient) Generate(ctx context.Context, messages []Message, options map[string]interface{}) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("anthropic API Key is required")
	}

	model := c.Model
	if options != nil && options["model"] != nil {
		model = options["model"].(string)
	}

	// Map internal Message to Anthropic Message
	type anthropicMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var anthropicMessages []anthropicMsg

	systemPrompt := ""
	for _, m := range messages {
		if m.Role == "system" {
			systemPrompt = m.Content
			continue
		}
		anthropicMessages = append(anthropicMessages, anthropicMsg{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":      model,
		"system":     systemPrompt,
		"messages":   anthropicMessages,
		"max_tokens": 4096,
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("anthropic api error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Content) > 0 {
		return result.Content[0].Text, nil
	}
	return "", fmt.Errorf("anthropic returned empty content")
}

// GenerateWithUsage implements Client — returns both text and token usage data.
func (c *AnthropicClient) GenerateWithUsage(ctx context.Context, messages []Message, options map[string]interface{}) (string, *Usage, error) {
	if c.APIKey == "" {
		return "", nil, fmt.Errorf("anthropic API Key is required")
	}

	model := c.Model
	if options != nil && options["model"] != nil {
		model = options["model"].(string)
	}

	type anthropicMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var anthropicMessages []anthropicMsg

	systemPrompt := ""
	for _, m := range messages {
		if m.Role == "system" {
			systemPrompt = m.Content
			continue
		}
		anthropicMessages = append(anthropicMessages, anthropicMsg{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":      model,
		"system":     systemPrompt,
		"messages":   anthropicMessages,
		"max_tokens": 4096,
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	start := time.Now()
	resp, err := c.HTTPClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("anthropic api error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", nil, err
	}

	usage := &Usage{
		PromptTokens:     result.Usage.InputTokens,
		CompletionTokens: result.Usage.OutputTokens,
		TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
		Model:            model,
		Provider:         "anthropic",
		LatencyMs:        latency,
	}
	usage.CostUSD = CalculateCost(*usage)

	if len(result.Content) > 0 {
		return result.Content[0].Text, usage, nil
	}
	return "", usage, fmt.Errorf("anthropic returned empty content")
}

// GenerateStructured handles strict JSON extraction via Anthropic.
func (c *AnthropicClient) GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options map[string]interface{}) (interface{}, error) {
	// Anthropic works best with system prompts for structured data.
	messages = append(messages, Message{
		Role:    "system",
		Content: "You must return your output precisely in valid JSON format matching the requested structure.",
	})

	responseText, err := c.Generate(ctx, messages, options)
	if err != nil {
		return nil, err
	}

	// Extract JSON if it's wrapped in markdown blocks
	cleanJSON := responseText
	if idx := strings.Index(cleanJSON, "```json"); idx != -1 {
		cleanJSON = cleanJSON[idx+7:]
		if endIdx := strings.Index(cleanJSON, "```"); endIdx != -1 {
			cleanJSON = cleanJSON[:endIdx]
		}
	}

	if err := json.Unmarshal([]byte(strings.TrimSpace(cleanJSON)), schema); err != nil {
		return nil, fmt.Errorf("failed to extract schema: %w\nRaw Output: %s", err, responseText)
	}

	return schema, nil
}

// GenerateEmbedding (Stub)
func (c *AnthropicClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("anthropic native embeddings not supported; use voyage or openai")
}

// StreamGenerate handles real-time token output.
func (c *AnthropicClient) StreamGenerate(ctx context.Context, messages []Message, options map[string]interface{}) (<-chan string, error) {
	// For simplicity, implement synchronous fallback or a simplified SSE stream in the future.
	ch := make(chan string)
	go func() {
		defer close(ch)
		res, err := c.Generate(ctx, messages, options)
		if err == nil {
			ch <- res
		}
	}()
	return ch, nil
}

// Multi-modal stubs for interface compliance
func (c *AnthropicClient) GenerateSpeech(ctx context.Context, text string, options map[string]interface{}) ([]byte, error) {
	return nil, fmt.Errorf("anthropic does not support TTS")
}
func (c *AnthropicClient) TranscribeSpeech(ctx context.Context, audio []byte, options map[string]interface{}) (string, error) {
	return "", fmt.Errorf("anthropic does not support STT")
}
