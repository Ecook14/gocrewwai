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

// GeminiClient provides a native implementation for Google Gemini.
type GeminiClient struct {
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

// NewGeminiClient creates a new client for Google Gemini.
func NewGeminiClient(apiKey, model string) *GeminiClient {
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	if model == "" {
		model = os.Getenv("GOOGLE_MODEL")
	}
	if model == "" {
		model = "gemini-1.5-pro"
	}

	return &GeminiClient{
		APIKey: apiKey,
		Model:  model,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Generate implements basic message generation for Gemini.
func (c *GeminiClient) Generate(ctx context.Context, messages []Message, options map[string]interface{}) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("google Gemini API Key is required")
	}

	model := c.Model
	if options != nil && options["model"] != nil {
		model = options["model"].(string)
	}

	// Map internal Message to Gemini Content
	type geminiPart struct {
		Text string `json:"text"`
	}
	type geminiContent struct {
		Role  string       `json:"role,omitempty"`
		Parts []geminiPart `json:"parts"`
	}

	var geminiContents []geminiContent
	systemInstruction := ""

	for _, m := range messages {
		role := m.Role
		if role == "system" {
			systemInstruction = m.Content
			continue
		}
		if role == "assistant" {
			role = "model"
		}
		geminiContents = append(geminiContents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	reqPayload := map[string]interface{}{
		"contents": geminiContents,
	}
	if systemInstruction != "" {
		reqPayload["system_instruction"] = map[string]interface{}{
			"parts": []geminiPart{{Text: systemInstruction}},
		}
	}

	reqBody, _ := json.Marshal(reqPayload)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, c.APIKey)

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gemini api error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		return result.Candidates[0].Content.Parts[0].Text, nil
	}
	return "", fmt.Errorf("gemini returned empty content")
}

// GenerateWithUsage implements Client — returns both text and usage data.
func (c *GeminiClient) GenerateWithUsage(ctx context.Context, messages []Message, options map[string]interface{}) (string, *Usage, error) {
	start := time.Now()
	text, err := c.Generate(ctx, messages, options)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return "", nil, err
	}

	model := c.Model
	if options != nil && options["model"] != nil {
		model = options["model"].(string)
	}

	// Approximate token count (Gemini API provides usageMetadata but we don't capture it
	// from Generate — use word-based estimation as a fallback)
	promptTokens := 0
	for _, m := range messages {
		promptTokens += len(strings.Fields(m.Content))
	}
	completionTokens := len(strings.Fields(text))

	usage := &Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		Model:            model,
		Provider:         "gemini",
		LatencyMs:        latency,
	}
	usage.CostUSD = CalculateCost(*usage)

	return text, usage, nil
}

// GenerateStructured handles strict JSON extraction via Gemini.
func (c *GeminiClient) GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options map[string]interface{}) (interface{}, error) {
	// Gemini supports JSON mode via generation configuration.
	// For simplicity, we use the same system prompt pattern as Anthropic.
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
	} else if idx := strings.Index(cleanJSON, "{"); idx != -1 {
		// Fallback for cases where it's not in a code block
		cleanJSON = cleanJSON[idx:]
		if lastIdx := strings.LastIndex(cleanJSON, "}"); lastIdx != -1 {
			cleanJSON = cleanJSON[:lastIdx+1]
		}
	}

	if err := json.Unmarshal([]byte(strings.TrimSpace(cleanJSON)), schema); err != nil {
		return nil, fmt.Errorf("failed to extract schema: %w\nRaw Output: %s", err, responseText)
	}

	return schema, nil
}

// GenerateEmbedding (Stub - Gemini supports these natively, but we'll stick to text-gen for now)
func (c *GeminiClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("gemini embeddings not yet implemented in this wrapper")
}

// StreamGenerate handles real-time token output.
func (c *GeminiClient) StreamGenerate(ctx context.Context, messages []Message, options map[string]interface{}) (<-chan string, error) {
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
func (c *GeminiClient) GenerateSpeech(ctx context.Context, text string, options map[string]interface{}) ([]byte, error) {
	return nil, fmt.Errorf("gemini does not support TTS in this direct api wrapper")
}
func (c *GeminiClient) TranscribeSpeech(ctx context.Context, audio []byte, options map[string]interface{}) (string, error) {
	return "", fmt.Errorf("gemini does not support STT in this direct api wrapper")
}
