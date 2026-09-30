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
	BaseURL    string
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
		APIKey:  apiKey,
		Model:   model,
		BaseURL: "https://generativelanguage.googleapis.com/v1beta",
		HTTPClient: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

// WithBaseURL allows reconfiguring the client's endpoint.
func (c *GeminiClient) WithBaseURL(url string) *GeminiClient {
	c.BaseURL = url
	return c
}

type geminiUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Generate implements basic message generation for Gemini.
func (c *GeminiClient) Generate(ctx context.Context, messages []Message, options GenerateOptions) (string, error) {
	text, _, err := c.generateBase(ctx, messages, options)
	if err != nil {
		return "", err
	}
	var promptText strings.Builder
	for _, m := range messages {
		promptText.WriteString(m.Content)
	}
	TrackHeuristicUsage("gemini", c.Model, promptText.String(), text, 0)
	return text, nil
}

// sanitizeProviderError strips any credential material from provider errors
// before they propagate to callers (and remote A2A responses).
func sanitizeProviderError(err error, apiKey string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if apiKey != "" {
		msg = strings.ReplaceAll(msg, apiKey, "[REDACTED]")
	}
	// URLs (with query strings) sometimes appear in transport errors.
	if i := strings.Index(msg, "https://"); i >= 0 {
		if j := strings.Index(msg[i:], " "); j >= 0 {
			msg = msg[:i] + "[url-redacted]" + msg[i+j:]
		} else {
			msg = msg[:i] + "[url-redacted]"
		}
	}
	if msg != err.Error() {
		return fmt.Errorf("%s", msg)
	}
	return err
}

func (c *GeminiClient) generateBase(ctx context.Context, messages []Message, options GenerateOptions) (string, *geminiUsage, error) {
	if c.APIKey == "" {
		return "", nil, fmt.Errorf("google Gemini API Key is required")
	}
	if err := AdmitBudget(); err != nil {
		return "", nil, err
	}

	model := options.Model
	if model == "" {
		model = c.Model
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
	// Credential goes in the provider-supported auth header, never the URL
	// query: Go includes the query in transport-error text, which previously
	// leaked the key into A2A task responses.
	endpoint := fmt.Sprintf("%s/models/%s:generateContent", c.BaseURL, model)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", nil, fmt.Errorf("gemini request: %w", sanitizeProviderError(err, c.APIKey))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", nil, sanitizeProviderError(err, c.APIKey)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("gemini api error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", nil, err
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		usage := &geminiUsage{
			PromptTokens:     result.UsageMetadata.PromptTokenCount,
			CompletionTokens: result.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      result.UsageMetadata.TotalTokenCount,
		}
		return result.Candidates[0].Content.Parts[0].Text, usage, nil
	}
	return "", nil, fmt.Errorf("gemini returned empty content")
}

// GenerateWithUsage implements Client — returns both text and token usage data.
func (c *GeminiClient) GenerateWithUsage(ctx context.Context, messages []Message, options GenerateOptions) (string, *Usage, error) {
	if c.APIKey == "" {
		return "", nil, fmt.Errorf("google Gemini API Key is required")
	}

	model := options.Model
	if model == "" {
		model = c.Model
	}

	start := time.Now()
	text, gUsage, err := c.generateBase(ctx, messages, options)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return "", nil, err
	}

	usage := &Usage{
		Model:     model,
		Provider:  "gemini",
		LatencyMs: latency,
	}

	if gUsage != nil {
		usage.PromptTokens = gUsage.PromptTokens
		usage.CompletionTokens = gUsage.CompletionTokens
		usage.TotalTokens = gUsage.TotalTokens
	}
	usage.CostUSD = CalculateCost(*usage)

	// Record to global tracker
	GlobalTracker().Record(*usage)

	return text, usage, nil
}

// GenerateStructured handles strict JSON extraction via Gemini.
func (c *GeminiClient) GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options GenerateOptions) (interface{}, error) {
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

// StreamGenerate handles real-time token output using Gemini's streaming endpoint.
func (c *GeminiClient) StreamGenerate(ctx context.Context, messages []Message, options GenerateOptions) (<-chan string, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("google Gemini API Key is required")
	}
	if err := AdmitBudget(); err != nil {
		return nil, err
	}

	model := options.Model
	if model == "" {
		model = c.Model
	}

	// Map messages... (similar to Generate)
	var geminiContents []interface{}
	for _, m := range messages {
		geminiContents = append(geminiContents, map[string]interface{}{
			"role":  m.Role,
			"parts": []map[string]interface{}{{"text": m.Content}},
		})
	}

	reqPayload := map[string]interface{}{"contents": geminiContents}
	reqBody, _ := json.Marshal(reqPayload)
	endpoint := fmt.Sprintf("%s/models/%s:streamGenerateContent", c.BaseURL, model)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("gemini request: %w", sanitizeProviderError(err, c.APIKey))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, sanitizeProviderError(err, c.APIKey)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("gemini api error (%d): %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan string)
	go func() {
		defer resp.Body.Close()
		defer close(ch)

		decoder := json.NewDecoder(resp.Body)
		// Gemini returns a JSON array of objects
		if _, err := decoder.Token(); err != nil { // consume '['
			return
		}

		for decoder.More() {
			var chunk struct {
				Candidates []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
			}
			if err := decoder.Decode(&chunk); err != nil {
				return
			}
			for _, cand := range chunk.Candidates {
				for _, part := range cand.Content.Parts {
					ch <- part.Text
				}
			}
		}
	}()

	return ch, nil
}
