package llm

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sashabaranov/go-openai"
)

// OpenAIOptions wraps configuration for the OpenAI generator.
type OpenAIOptions struct {
	Model       string
	Temperature float32
	MaxTokens   int
}

// OpenAIClient implements the Client interface for OpenAI connectivity.
type OpenAIClient struct {
	APIKey string
	client *openai.Client
}

func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		APIKey: apiKey,
		client: openai.NewClient(apiKey),
	}
}

// Generate implements basic message generation.
func (c *OpenAIClient) Generate(ctx context.Context, messages []Message, options map[string]interface{}) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("OpenAI API Key is required")
	}

	var oaiMessages []openai.ChatCompletionMessage
	for _, m := range messages {
		oaiMessages = append(oaiMessages, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	model := openai.GPT4o
	if options != nil && options["model"] != nil {
		model = options["model"].(string)
	}

	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: oaiMessages,
	}

	// Phase 18: MD5 LLM Response Caching
	hashStr := getCacheHash(oaiMessages, model)
	cacheFile := filepath.Join(os.TempDir(), "crew_cache_"+hashStr+".txt")
	if cached, err := os.ReadFile(cacheFile); err == nil {
		return string(cached), nil // Cache HIT
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("ChatCompletion error: %v", err)
	}

	result := resp.Choices[0].Message.Content
	_ = os.WriteFile(cacheFile, []byte(result), 0644) // Save cache MISS

	return result, nil
}

// GenerateStructured implements generation with structured JSON schema outputs.
func (c *OpenAIClient) GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options map[string]interface{}) (interface{}, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API Key is required")
	}

	var oaiMessages []openai.ChatCompletionMessage
	for _, m := range messages {
		oaiMessages = append(oaiMessages, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	// For simple schema matching, tell OpenAI to return JSON object
	oaiMessages = append(oaiMessages, openai.ChatCompletionMessage{
		Role:    "system",
		Content: "You must return your output precisely in valid JSON format matching the requested structure.",
	})

	model := openai.GPT4o
	if options != nil && options["model"] != nil {
		model = options["model"].(string)
	}

	req := openai.ChatCompletionRequest{
		Model:          model,
		Messages:       oaiMessages,
		ResponseFormat: &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject},
	}

	// Phase 18: MD5 Structured LLM Caching
	hashStr := getCacheHash(oaiMessages, model+"_structured")
	cacheFile := filepath.Join(os.TempDir(), "crew_cache_"+hashStr+".json")
	var rawJSON string

	if cached, err := os.ReadFile(cacheFile); err == nil {
		rawJSON = string(cached)
	} else {
		resp, err := c.client.CreateChatCompletion(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("ChatCompletion unstructured error: %v", err)
		}
		rawJSON = resp.Choices[0].Message.Content
		_ = os.WriteFile(cacheFile, []byte(rawJSON), 0644)
	}
	
	err := json.Unmarshal([]byte(rawJSON), schema)
	if err != nil {
		return nil, fmt.Errorf("failed to extract schema: %w\nRaw Output: %s", err, rawJSON)
	}

	return schema, nil
}

// getCacheHash generates a unique MD5 signature for an LLM request
func getCacheHash(messages []openai.ChatCompletionMessage, model string) string {
	payload := struct {
		Model    string
		Messages []openai.ChatCompletionMessage
	}{model, messages}
	data, _ := json.Marshal(payload)
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}
