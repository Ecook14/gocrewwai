package llm

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

// testClient is a minimal mock for middleware testing.
type testClient struct {
	delay    time.Duration
	response string
	err      error
	calls    int
	mu       sync.Mutex
}

func (tc *testClient) Generate(ctx context.Context, messages []Message, options map[string]interface{}) (string, error) {
	tc.mu.Lock()
	tc.calls++
	tc.mu.Unlock()
	if tc.delay > 0 {
		select {
		case <-time.After(tc.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return tc.response, tc.err
}

func (tc *testClient) GenerateWithUsage(ctx context.Context, messages []Message, options map[string]interface{}) (string, *Usage, error) {
	resp, err := tc.Generate(ctx, messages, options)
	return resp, &Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}, err
}

func (tc *testClient) GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options map[string]interface{}) (interface{}, error) {
	return tc.Generate(ctx, messages, options)
}

func (tc *testClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2}, tc.err
}

func (tc *testClient) StreamGenerate(ctx context.Context, messages []Message, options map[string]interface{}) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- tc.response
	close(ch)
	return ch, tc.err
}

func (tc *testClient) GenerateSpeech(ctx context.Context, text string, options map[string]interface{}) ([]byte, error) {
	return []byte("audio"), tc.err
}

func (tc *testClient) TranscribeSpeech(ctx context.Context, audio []byte, options map[string]interface{}) (string, error) {
	return tc.response, tc.err
}

func (tc *testClient) callCount() int {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	return tc.calls
}

func TestMiddleware_Passthrough(t *testing.T) {
	inner := &testClient{response: "hello"}
	client := WrapClient(inner)

	result, err := client.Generate(context.Background(), []Message{{Role: "user", Content: "test"}}, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result != "hello" {
		t.Errorf("Expected 'hello', got %q", result)
	}
}

func TestMiddleware_Timeout(t *testing.T) {
	inner := &testClient{response: "slow", delay: 500 * time.Millisecond}
	client := WrapClient(inner, WithTimeout(50*time.Millisecond))

	_, err := client.Generate(context.Background(), []Message{{Role: "user", Content: "test"}}, nil)
	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestMiddleware_TimeoutSuccess(t *testing.T) {
	inner := &testClient{response: "fast", delay: 10 * time.Millisecond}
	client := WrapClient(inner, WithTimeout(1*time.Second))

	result, err := client.Generate(context.Background(), []Message{{Role: "user", Content: "test"}}, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result != "fast" {
		t.Errorf("Expected 'fast', got %q", result)
	}
}

func TestMiddleware_RateLimit(t *testing.T) {
	inner := &testClient{response: "ok"}
	// Allow 3 requests per second
	client := WrapClient(inner, WithRateLimit(3, time.Second))

	ctx := context.Background()
	start := time.Now()

	// First 3 should be instant
	for i := 0; i < 3; i++ {
		_, err := client.Generate(ctx, []Message{{Role: "user", Content: "test"}}, nil)
		if err != nil {
			t.Fatalf("Call %d failed: %v", i, err)
		}
	}

	// 4th should be delayed
	_, err := client.Generate(ctx, []Message{{Role: "user", Content: "test"}}, nil)
	if err != nil {
		t.Fatalf("4th call failed: %v", err)
	}

	elapsed := time.Since(start)
	// Should take at least ~300ms for the 4th request (1/3 second refill)
	if elapsed < 200*time.Millisecond {
		t.Errorf("Expected delay from rate limiting, only took %v", elapsed)
	}
}

func TestMiddleware_RateLimitContextCancel(t *testing.T) {
	inner := &testClient{response: "ok"}
	// Very restrictive: 1 per minute
	client := WrapClient(inner, WithRateLimit(1, time.Minute))

	ctx := context.Background()
	// Use up the single token
	_, _ = client.Generate(ctx, []Message{{Role: "user", Content: "first"}}, nil)

	// Second call with short deadline should fail
	ctx2, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	_, err := client.Generate(ctx2, []Message{{Role: "user", Content: "second"}}, nil)
	if err == nil {
		t.Error("Expected rate limit context cancellation error")
	}
}

func TestMiddleware_Logging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	inner := &testClient{response: "logged-response"}
	client := WrapClient(inner, WithLogging(logger))

	_, _ = client.Generate(context.Background(),
		[]Message{{Role: "user", Content: "hello world"}},
		map[string]interface{}{"model": "gpt-4o"},
	)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "Generate") {
		t.Error("Expected log to contain method name 'Generate'")
	}
	if !strings.Contains(logOutput, "gpt-4o") {
		t.Error("Expected log to contain model name 'gpt-4o'")
	}
	if !strings.Contains(logOutput, "LLM call") {
		t.Error("Expected log to contain 'LLM call'")
	}
}

func TestMiddleware_LoggingWithUsage(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	inner := &testClient{response: "response"}
	client := WrapClient(inner, WithLogging(logger))

	_, _, _ = client.GenerateWithUsage(context.Background(),
		[]Message{{Role: "user", Content: "test"}}, nil)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "prompt_tokens") {
		t.Error("Expected log to contain 'prompt_tokens'")
	}
	if !strings.Contains(logOutput, "completion_tokens") {
		t.Error("Expected log to contain 'completion_tokens'")
	}
}

func TestMiddleware_Inner(t *testing.T) {
	inner := &testClient{response: "test"}
	client := WrapClient(inner)
	if client.Inner() != inner {
		t.Error("Inner() should return the original client")
	}
}

func TestMiddleware_AllMethods(t *testing.T) {
	inner := &testClient{response: "ok"}
	client := WrapClient(inner)
	ctx := context.Background()

	// Verify all methods work through middleware
	if _, err := client.Generate(ctx, nil, nil); err != nil {
		t.Errorf("Generate failed: %v", err)
	}
	if _, _, err := client.GenerateWithUsage(ctx, nil, nil); err != nil {
		t.Errorf("GenerateWithUsage failed: %v", err)
	}
	if _, err := client.GenerateStructured(ctx, nil, nil, nil); err != nil {
		t.Errorf("GenerateStructured failed: %v", err)
	}
	if _, err := client.GenerateEmbedding(ctx, "test"); err != nil {
		t.Errorf("GenerateEmbedding failed: %v", err)
	}
	if _, err := client.StreamGenerate(ctx, nil, nil); err != nil {
		t.Errorf("StreamGenerate failed: %v", err)
	}
	if _, err := client.GenerateSpeech(ctx, "test", nil); err != nil {
		t.Errorf("GenerateSpeech failed: %v", err)
	}
	if _, err := client.TranscribeSpeech(ctx, nil, nil); err != nil {
		t.Errorf("TranscribeSpeech failed: %v", err)
	}
}

func TestTokenBucket_Available(t *testing.T) {
	tb := newTokenBucket(10, time.Second)
	avail := tb.Available()
	if avail < 9.9 || avail > 10.1 {
		t.Errorf("Expected ~10 available tokens, got %f", avail)
	}

	// Consume one
	_ = tb.Wait(context.Background())
	avail = tb.Available()
	if avail > 9.5 {
		t.Errorf("Expected ~9 available tokens after consuming one, got %f", avail)
	}
}

func TestMiddleware_CombinedOptions(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	inner := &testClient{response: "combined"}
	client := WrapClient(inner,
		WithRateLimit(100, time.Second),
		WithTimeout(5*time.Second),
		WithLogging(logger),
		WithMaxRetries(3),
	)

	result, err := client.Generate(context.Background(),
		[]Message{{Role: "user", Content: "test"}}, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result != "combined" {
		t.Errorf("Expected 'combined', got %q", result)
	}

	// Verify logging happened
	if !strings.Contains(buf.String(), "LLM call") {
		t.Error("Expected logging output")
	}
}
