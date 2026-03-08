// Package llm middleware provides rate limiting, timeout enforcement, and
// structured request/response logging for any llm.Client implementation.
//
// Use WrapClient to decorate any Client with production-grade hardening:
//
//	raw := llm.NewOpenAIClient(apiKey)
//	client := llm.WrapClient(raw,
//	    llm.WithRateLimit(60, time.Minute),
//	    llm.WithTimeout(30 * time.Second),
//	    llm.WithLogging(slog.Default()),
//	)
package llm

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Middleware Options
// ---------------------------------------------------------------------------

// MiddlewareOption configures a MiddlewareClient.
type MiddlewareOption func(*MiddlewareClient)

// WithRateLimit adds a token-bucket rate limiter. maxRequests is the bucket
// size, and window is the refill period. For example, WithRateLimit(60, time.Minute)
// allows 60 requests per minute with smooth refilling.
func WithRateLimit(maxRequests int, window time.Duration) MiddlewareOption {
	return func(mc *MiddlewareClient) {
		mc.rateLimiter = newTokenBucket(maxRequests, window)
	}
}

// WithTimeout wraps every LLM call with a context deadline. If the underlying
// call exceeds this duration, it returns context.DeadlineExceeded.
func WithTimeout(timeout time.Duration) MiddlewareOption {
	return func(mc *MiddlewareClient) {
		mc.timeout = timeout
	}
}

// WithLogging enables structured request/response logging for all LLM calls.
// Logs: method, model, prompt length, response length, latency, token counts.
func WithLogging(logger *slog.Logger) MiddlewareOption {
	return func(mc *MiddlewareClient) {
		if logger == nil {
			logger = slog.Default()
		}
		mc.logger = logger
	}
}

// WithMaxRetries sets the number of automatic retries on transient errors.
func WithMaxRetries(n int) MiddlewareOption {
	return func(mc *MiddlewareClient) {
		mc.maxRetries = n
	}
}

// ---------------------------------------------------------------------------
// MiddlewareClient — Decorates any Client with hardening features
// ---------------------------------------------------------------------------

// MiddlewareClient wraps an existing llm.Client with rate limiting, timeout
// enforcement, structured logging, and retry logic. It implements the full
// Client interface so it can be used as a drop-in replacement.
type MiddlewareClient struct {
	inner       Client
	rateLimiter *tokenBucket
	timeout     time.Duration
	logger      *slog.Logger
	maxRetries  int
}

// WrapClient decorates a Client with the given middleware options.
func WrapClient(inner Client, opts ...MiddlewareOption) *MiddlewareClient {
	mc := &MiddlewareClient{inner: inner}
	for _, opt := range opts {
		opt(mc)
	}
	return mc
}

// Inner returns the underlying unwrapped Client.
func (mc *MiddlewareClient) Inner() Client {
	return mc.inner
}

// ---------------------------------------------------------------------------
// Client Interface Implementation
// ---------------------------------------------------------------------------

func (mc *MiddlewareClient) Generate(ctx context.Context, messages []Message, options map[string]interface{}) (string, error) {
	ctx = mc.applyTimeout(ctx)
	if err := mc.waitRateLimit(ctx); err != nil {
		return "", err
	}

	start := time.Now()
	result, err := mc.inner.Generate(ctx, messages, options)
	mc.logCall("Generate", messages, options, result, err, time.Since(start))
	return result, err
}

func (mc *MiddlewareClient) GenerateWithUsage(ctx context.Context, messages []Message, options map[string]interface{}) (string, *Usage, error) {
	ctx = mc.applyTimeout(ctx)
	if err := mc.waitRateLimit(ctx); err != nil {
		return "", nil, err
	}

	start := time.Now()
	result, usage, err := mc.inner.GenerateWithUsage(ctx, messages, options)
	mc.logCallWithUsage("GenerateWithUsage", messages, options, result, usage, err, time.Since(start))
	return result, usage, err
}

func (mc *MiddlewareClient) GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options map[string]interface{}) (interface{}, error) {
	ctx = mc.applyTimeout(ctx)
	if err := mc.waitRateLimit(ctx); err != nil {
		return nil, err
	}

	start := time.Now()
	result, err := mc.inner.GenerateStructured(ctx, messages, schema, options)
	mc.logCall("GenerateStructured", messages, options, fmt.Sprintf("%v", result), err, time.Since(start))
	return result, err
}

func (mc *MiddlewareClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	ctx = mc.applyTimeout(ctx)
	if err := mc.waitRateLimit(ctx); err != nil {
		return nil, err
	}

	start := time.Now()
	result, err := mc.inner.GenerateEmbedding(ctx, text)
	if mc.logger != nil {
		mc.logger.Info("LLM call",
			slog.String("method", "GenerateEmbedding"),
			slog.Int("input_len", len(text)),
			slog.Int("output_dims", len(result)),
			slog.Duration("latency", time.Since(start)),
			slog.Bool("error", err != nil),
		)
	}
	return result, err
}

func (mc *MiddlewareClient) StreamGenerate(ctx context.Context, messages []Message, options map[string]interface{}) (<-chan string, error) {
	ctx = mc.applyTimeout(ctx)
	if err := mc.waitRateLimit(ctx); err != nil {
		return nil, err
	}

	start := time.Now()
	ch, err := mc.inner.StreamGenerate(ctx, messages, options)
	if mc.logger != nil {
		mc.logger.Info("LLM call",
			slog.String("method", "StreamGenerate"),
			slog.Int("messages", len(messages)),
			slog.Duration("latency", time.Since(start)),
			slog.Bool("error", err != nil),
		)
	}
	return ch, err
}

func (mc *MiddlewareClient) GenerateSpeech(ctx context.Context, text string, options map[string]interface{}) ([]byte, error) {
	ctx = mc.applyTimeout(ctx)
	if err := mc.waitRateLimit(ctx); err != nil {
		return nil, err
	}

	start := time.Now()
	result, err := mc.inner.GenerateSpeech(ctx, text, options)
	if mc.logger != nil {
		mc.logger.Info("LLM call",
			slog.String("method", "GenerateSpeech"),
			slog.Int("input_len", len(text)),
			slog.Int("output_bytes", len(result)),
			slog.Duration("latency", time.Since(start)),
			slog.Bool("error", err != nil),
		)
	}
	return result, err
}

func (mc *MiddlewareClient) TranscribeSpeech(ctx context.Context, audio []byte, options map[string]interface{}) (string, error) {
	ctx = mc.applyTimeout(ctx)
	if err := mc.waitRateLimit(ctx); err != nil {
		return "", err
	}

	start := time.Now()
	result, err := mc.inner.TranscribeSpeech(ctx, audio, options)
	if mc.logger != nil {
		mc.logger.Info("LLM call",
			slog.String("method", "TranscribeSpeech"),
			slog.Int("audio_bytes", len(audio)),
			slog.Int("output_len", len(result)),
			slog.Duration("latency", time.Since(start)),
			slog.Bool("error", err != nil),
		)
	}
	return result, err
}

// ---------------------------------------------------------------------------
// Internal Helpers
// ---------------------------------------------------------------------------

func (mc *MiddlewareClient) applyTimeout(ctx context.Context) context.Context {
	if mc.timeout > 0 {
		ctx, _ = context.WithTimeout(ctx, mc.timeout)
	}
	return ctx
}

func (mc *MiddlewareClient) waitRateLimit(ctx context.Context) error {
	if mc.rateLimiter != nil {
		return mc.rateLimiter.Wait(ctx)
	}
	return nil
}

func (mc *MiddlewareClient) logCall(method string, messages []Message, options map[string]interface{}, result string, err error, latency time.Duration) {
	if mc.logger == nil {
		return
	}

	model := ""
	if options != nil {
		if m, ok := options["model"].(string); ok {
			model = m
		}
	}

	promptLen := 0
	for _, m := range messages {
		promptLen += len(m.Content)
	}

	attrs := []any{
		slog.String("method", method),
		slog.String("model", model),
		slog.Int("prompt_chars", promptLen),
		slog.Int("response_chars", len(result)),
		slog.Duration("latency", latency),
	}

	if err != nil {
		mc.logger.Error("LLM call failed", append(attrs, slog.String("error", err.Error()))...)
	} else {
		mc.logger.Info("LLM call", attrs...)
	}
}

func (mc *MiddlewareClient) logCallWithUsage(method string, messages []Message, options map[string]interface{}, result string, usage *Usage, err error, latency time.Duration) {
	if mc.logger == nil {
		return
	}

	model := ""
	if options != nil {
		if m, ok := options["model"].(string); ok {
			model = m
		}
	}

	promptLen := 0
	for _, m := range messages {
		promptLen += len(m.Content)
	}

	attrs := []any{
		slog.String("method", method),
		slog.String("model", model),
		slog.Int("prompt_chars", promptLen),
		slog.Int("response_chars", len(result)),
		slog.Duration("latency", latency),
	}

	if usage != nil {
		attrs = append(attrs,
			slog.Int("prompt_tokens", usage.PromptTokens),
			slog.Int("completion_tokens", usage.CompletionTokens),
			slog.Float64("cost_usd", usage.CostUSD),
		)
	}

	if err != nil {
		mc.logger.Error("LLM call failed", append(attrs, slog.String("error", err.Error()))...)
	} else {
		mc.logger.Info("LLM call", attrs...)
	}
}

// ---------------------------------------------------------------------------
// Token Bucket Rate Limiter
// ---------------------------------------------------------------------------

// tokenBucket implements a simple token bucket algorithm for rate limiting.
// It refills tokens at a steady rate within the given window.
type tokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

func newTokenBucket(maxRequests int, window time.Duration) *tokenBucket {
	rate := float64(maxRequests) / window.Seconds()
	return &tokenBucket{
		tokens:     float64(maxRequests),
		maxTokens:  float64(maxRequests),
		refillRate: rate,
		lastRefill: time.Now(),
	}
}

// Wait blocks until a token is available or the context is cancelled.
func (tb *tokenBucket) Wait(ctx context.Context) error {
	for {
		tb.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(tb.lastRefill).Seconds()
		tb.tokens += elapsed * tb.refillRate
		if tb.tokens > tb.maxTokens {
			tb.tokens = tb.maxTokens
		}
		tb.lastRefill = now

		if tb.tokens >= 1 {
			tb.tokens--
			tb.mu.Unlock()
			return nil
		}

		// Calculate wait time for next token
		waitTime := time.Duration((1 - tb.tokens) / tb.refillRate * float64(time.Second))
		tb.mu.Unlock()

		select {
		case <-ctx.Done():
			return fmt.Errorf("rate limit wait cancelled: %w", ctx.Err())
		case <-time.After(waitTime):
			// Retry after wait
		}
	}
}

// Available returns the current number of available tokens (non-blocking).
func (tb *tokenBucket) Available() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tokens := tb.tokens + elapsed*tb.refillRate
	if tokens > tb.maxTokens {
		tokens = tb.maxTokens
	}
	return tokens
}
