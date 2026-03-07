package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

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
	APIKey     string
	client     *openai.Client
	HTTPClient *http.Client
}

func NewOpenAIClient(apiKey string) *OpenAIClient {
	// Elite Pattern: Robust networking with explicit timeouts.
	// Prevents hung processes during heavy multimodal transfers.
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	config := openai.DefaultConfig(apiKey)
	config.HTTPClient = httpClient

	return &OpenAIClient{
		APIKey:     apiKey,
		client:     openai.NewClientWithConfig(config),
		HTTPClient: httpClient,
	}
}

// Generate implements basic message generation with Multimodal (Vision) support.
func (c *OpenAIClient) Generate(ctx context.Context, messages []Message, options map[string]interface{}) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("OpenAI API Key is required")
	}

	var oaiMessages []openai.ChatCompletionMessage
	for _, m := range messages {
		if len(m.Images) > 0 {
			parts := []openai.ChatMessagePart{
				{Type: openai.ChatMessagePartTypeText, Text: m.Content},
			}
			for _, img := range m.Images {
				parts = append(parts, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeImageURL,
					ImageURL: &openai.ChatMessageImageURL{
						URL:    img,
						Detail: openai.ImageURLDetailAuto,
					},
				})
			}
			oaiMessages = append(oaiMessages, openai.ChatCompletionMessage{
				Role:         m.Role,
				MultiContent: parts,
			})
		} else {
			oaiMessages = append(oaiMessages, openai.ChatCompletionMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}

	model := openai.GPT4o
	if options != nil && options["model"] != nil {
		model = options["model"].(string)
	}

	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: oaiMessages,
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("ChatCompletion error: %v", err)
	}

	return resp.Choices[0].Message.Content, nil
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

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ChatCompletion unstructured error: %v", err)
	}
	rawJSON := resp.Choices[0].Message.Content
	
	err = json.Unmarshal([]byte(rawJSON), schema)
	if err != nil {
		return nil, fmt.Errorf("failed to extract schema: %w\nRaw Output: %s", err, rawJSON)
	}

	return schema, nil
}

// StreamGenerate provides real-time token output via a channel for OpenAI.
func (c *OpenAIClient) StreamGenerate(ctx context.Context, messages []Message, options map[string]interface{}) (<-chan string, error) {
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

	model := openai.GPT4o
	if options != nil && options["model"] != nil {
		model = options["model"].(string)
	}

	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: oaiMessages,
		Stream:   true,
	}

	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Stream error: %v", err)
	}

	ch := make(chan string)
	go func() {
		defer close(ch)
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if err != nil {
				return
			}
			if len(response.Choices) > 0 {
				ch <- response.Choices[0].Delta.Content
			}
		}
	}()

	return ch, nil
}



// GenerateSpeech converts text to audio using OpenAI's TTS.
func (c *OpenAIClient) GenerateSpeech(ctx context.Context, text string, options map[string]interface{}) ([]byte, error) {
	model := openai.TTSModel1
	voice := openai.VoiceAlloy
	if options != nil {
		if m, ok := options["model"].(openai.SpeechModel); ok {
			model = m
		}
		if v, ok := options["voice"].(openai.SpeechVoice); ok {
			voice = v
		}
	}

	req := openai.CreateSpeechRequest{
		Model: model,
		Input: text,
		Voice: voice,
	}

	resp, err := c.client.CreateSpeech(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	return io.ReadAll(resp)
}

// TranscribeSpeech converts audio to text using Whisper.
func (c *OpenAIClient) TranscribeSpeech(ctx context.Context, audio []byte, options map[string]interface{}) (string, error) {
	// Actual Implementation: uses multipart writer to send audio to OpenAI Whisper API.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "audio.mp3")
	if err != nil {
		return "", err
	}
	if _, err := part.Write(audio); err != nil {
		return "", err
	}

	if err := writer.WriteField("model", "whisper-1"); err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/audio/transcriptions", body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("whisper transcription failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Text, nil
}
