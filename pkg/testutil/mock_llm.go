// Package testutil provides shared testing utilities for the Crew-GO framework.
// It contains a fully configurable mock LLM client, mock tools, and assertion helpers
// used across all package test suites.
package testutil

import (
	"context"
	"fmt"
	"sync"

	"github.com/Ecook14/gocrewwai/pkg/llm"
)

// MockCall records a single invocation to any method on MockClient.
type MockCall struct {
	Method   string
	Messages []llm.Message
	Options  map[string]interface{}
	Schema   interface{}
	Text     string // for embedding calls
}

// MockClient is a configurable mock implementing the full llm.Client interface.
// All function fields are optional — nil fields return sensible defaults.
type MockClient struct {
	mu    sync.Mutex
	Calls []MockCall

	// Configurable response functions
	GenerateFunc           func(ctx context.Context, msgs []llm.Message, opts map[string]interface{}) (string, error)
	GenerateStructuredFunc func(ctx context.Context, msgs []llm.Message, schema interface{}, opts map[string]interface{}) (interface{}, error)
	EmbeddingFunc          func(ctx context.Context, text string) ([]float32, error)
	StreamFunc             func(ctx context.Context, msgs []llm.Message, opts map[string]interface{}) (<-chan string, error)
	SpeechFunc             func(ctx context.Context, text string, opts map[string]interface{}) ([]byte, error)
	TranscribeFunc         func(ctx context.Context, audio []byte, opts map[string]interface{}) (string, error)

	// Usage tracking: if non-nil, returned from GenerateWithUsage
	UsagePerCall *llm.Usage
}

// record appends a call record (thread-safe).
func (m *MockClient) record(call MockCall) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, call)
}

// CallCount returns the total number of recorded calls.
func (m *MockClient) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Calls)
}

// CallsForMethod returns only the calls matching the given method name.
func (m *MockClient) CallsForMethod(method string) []MockCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []MockCall
	for _, c := range m.Calls {
		if c.Method == method {
			result = append(result, c)
		}
	}
	return result
}

// Reset clears all recorded calls.
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = nil
}

// Generate implements llm.Client.
func (m *MockClient) Generate(ctx context.Context, messages []llm.Message, options map[string]interface{}) (string, error) {
	m.record(MockCall{Method: "Generate", Messages: messages, Options: options})
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, messages, options)
	}
	return "mock response", nil
}

// GenerateWithUsage implements llm.Client — returns both text and usage data.
func (m *MockClient) GenerateWithUsage(ctx context.Context, messages []llm.Message, options map[string]interface{}) (string, *llm.Usage, error) {
	m.record(MockCall{Method: "GenerateWithUsage", Messages: messages, Options: options})

	var text string
	var err error
	if m.GenerateFunc != nil {
		text, err = m.GenerateFunc(ctx, messages, options)
	} else {
		text = "mock response"
	}

	usage := m.UsagePerCall
	if usage == nil {
		usage = &llm.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
			Model:            "mock-model",
		}
	}
	return text, usage, err
}

// GenerateStructured implements llm.Client.
func (m *MockClient) GenerateStructured(ctx context.Context, messages []llm.Message, schema interface{}, options map[string]interface{}) (interface{}, error) {
	m.record(MockCall{Method: "GenerateStructured", Messages: messages, Schema: schema, Options: options})
	if m.GenerateStructuredFunc != nil {
		return m.GenerateStructuredFunc(ctx, messages, schema, options)
	}
	return schema, nil
}

// GenerateEmbedding implements llm.Client.
func (m *MockClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	m.record(MockCall{Method: "GenerateEmbedding", Text: text})
	if m.EmbeddingFunc != nil {
		return m.EmbeddingFunc(ctx, text)
	}
	// Return a deterministic 8-dimensional vector based on text length
	dim := 8
	vec := make([]float32, dim)
	for i := range vec {
		vec[i] = float32(len(text)%10) * 0.1 * float32(i+1)
	}
	return vec, nil
}

// StreamGenerate implements llm.Client.
func (m *MockClient) StreamGenerate(ctx context.Context, messages []llm.Message, options map[string]interface{}) (<-chan string, error) {
	m.record(MockCall{Method: "StreamGenerate", Messages: messages, Options: options})
	if m.StreamFunc != nil {
		return m.StreamFunc(ctx, messages, options)
	}
	ch := make(chan string, 3)
	go func() {
		defer close(ch)
		ch <- "mock "
		ch <- "streamed "
		ch <- "response"
	}()
	return ch, nil
}

// GenerateSpeech implements llm.Client.
func (m *MockClient) GenerateSpeech(ctx context.Context, text string, options map[string]interface{}) ([]byte, error) {
	m.record(MockCall{Method: "GenerateSpeech", Text: text, Options: options})
	if m.SpeechFunc != nil {
		return m.SpeechFunc(ctx, text, options)
	}
	return []byte("mock-audio-bytes"), nil
}

// TranscribeSpeech implements llm.Client.
func (m *MockClient) TranscribeSpeech(ctx context.Context, audio []byte, options map[string]interface{}) (string, error) {
	m.record(MockCall{Method: "TranscribeSpeech", Options: options})
	if m.TranscribeFunc != nil {
		return m.TranscribeFunc(ctx, audio, options)
	}
	return "mock transcription", nil
}

// --- Mock Tool ---

// MockTool is a configurable mock implementing the tools.Tool interface.
type MockTool struct {
	NameValue        string
	DescriptionValue string
	ReviewRequired   bool
	ExecuteFunc      func(ctx context.Context, input map[string]interface{}) (string, error)
}

func (t *MockTool) Name() string        { return t.NameValue }
func (t *MockTool) Description() string { return t.DescriptionValue }
func (t *MockTool) RequiresReview() bool { return t.ReviewRequired }

func (t *MockTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	if t.ExecuteFunc != nil {
		return t.ExecuteFunc(ctx, input)
	}
	return fmt.Sprintf("mock tool '%s' executed", t.NameValue), nil
}

// --- Convenience Factories ---

// NewSimpleMock creates a MockClient that always returns the given text.
func NewSimpleMock(response string) *MockClient {
	return &MockClient{
		GenerateFunc: func(ctx context.Context, msgs []llm.Message, opts map[string]interface{}) (string, error) {
			return response, nil
		},
	}
}

// NewSequenceMock creates a MockClient that returns responses in order.
// After all responses are consumed, it returns the last one repeatedly.
func NewSequenceMock(responses ...string) *MockClient {
	idx := 0
	var mu sync.Mutex
	return &MockClient{
		GenerateFunc: func(ctx context.Context, msgs []llm.Message, opts map[string]interface{}) (string, error) {
			mu.Lock()
			defer mu.Unlock()
			if idx < len(responses) {
				r := responses[idx]
				idx++
				return r, nil
			}
			return responses[len(responses)-1], nil
		},
	}
}

// NewErrorMock creates a MockClient that always returns the given error.
func NewErrorMock(err error) *MockClient {
	return &MockClient{
		GenerateFunc: func(ctx context.Context, msgs []llm.Message, opts map[string]interface{}) (string, error) {
			return "", err
		},
	}
}
