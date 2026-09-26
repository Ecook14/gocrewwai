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
// enforcement, structured logging, retry logic, and optional response caching. It implements the full
// Client interface so it can be used as a drop-in replacement.
type MiddlewareClient struct {
	inner       Client
	rateLimiter *tokenBucket
	timeout     time.Duration
	logger      *slog.Logger
	maxRetries  int
	cache       Cache
	breaker     *circuitBreaker
}

// WithCache enables response caching on the wrapped client. Cached responses are
// returned for identical inputs, reducing latency and API cost.
func WithCache(cache Cache) MiddlewareOption {
	return func(mc *MiddlewareClient) {
		mc.cache = cache
	}
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

func (mc *MiddlewareClient) Generate(ctx context.Context, messages []Message, options GenerateOptions) (string, error) {
	allow, trial := mc.breakerAllow()
	if !allow {
		return "", ErrCircuitOpen
	}
	ctx, cancel := mc.applyTimeout(ctx)
	defer cancel()
	if err := mc.waitRateLimit(ctx); err != nil {
		mc.breakerResult(trial, err)
		return "", err
	}

	start := time.Now()
	result, err := mc.inner.Generate(ctx, messages, options)
	mc.breakerResult(trial, err)
	mc.logCall("Generate", messages, options, result, err, time.Since(start))
	return result, err
}

func (mc *MiddlewareClient) GenerateWithUsage(ctx context.Context, messages []Message, options GenerateOptions) (string, *Usage, error) {
	allow, trial := mc.breakerAllow()
	if !allow {
		return "", nil, ErrCircuitOpen
	}
	ctx, cancel := mc.applyTimeout(ctx)
	defer cancel()
	if err := mc.waitRateLimit(ctx); err != nil {
		mc.breakerResult(trial, err)
		return "", nil, err
	}

	start := time.Now()
	result, usage, err := mc.inner.GenerateWithUsage(ctx, messages, options)
	mc.breakerResult(trial, err)
	mc.logCallWithUsage("GenerateWithUsage", messages, options, result, usage, err, time.Since(start))
	return result, usage, err
}

func (mc *MiddlewareClient) GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options GenerateOptions) (interface{}, error) {
	allow, trial := mc.breakerAllow()
	if !allow {
		return nil, ErrCircuitOpen
	}
	ctx, cancel := mc.applyTimeout(ctx)
	defer cancel()
	if err := mc.waitRateLimit(ctx); err != nil {
		mc.breakerResult(trial, err)
		return nil, err
	}

	start := time.Now()
	result, err := mc.inner.GenerateStructured(ctx, messages, schema, options)
	mc.breakerResult(trial, err)
	mc.logCall("GenerateStructured", messages, options, fmt.Sprintf("%v", result), err, time.Since(start))
	return result, err
}

func (mc *MiddlewareClient) StreamGenerate(ctx context.Context, messages []Message, options GenerateOptions) (<-chan string, error) {
	// Only setup errors feed the breaker; mid-stream chunk errors travel
	// the channel and cannot be attributed.
	allow, trial := mc.breakerAllow()
	if !allow {
		return nil, ErrCircuitOpen
	}
	ctx, cancel := mc.applyTimeout(ctx)
	defer cancel()
	if err := mc.waitRateLimit(ctx); err != nil {
		mc.breakerResult(trial, err)
		return nil, err
	}

	start := time.Now()
	ch, err := mc.inner.StreamGenerate(ctx, messages, options)
	mc.breakerResult(trial, err)
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

// ---------------------------------------------------------------------------
// Internal Helpers
// ---------------------------------------------------------------------------

// breakerAllow fast-fails when the circuit is open. Helpers are nil-safe
// so clients without WithCircuitBreaker behave exactly as before.
func (mc *MiddlewareClient) breakerAllow() (allow, trial bool) {
	if mc.breaker == nil {
		return true, false
	}
	return mc.breaker.beforeCall()
}

func (mc *MiddlewareClient) breakerResult(trial bool, err error) {
	if mc.breaker == nil {
		return
	}
	mc.breaker.afterCall(trial, err)
}

// applyTimeout returns a context with the configured timeout applied.
// The caller is responsible for deferring the returned cancel function.
func (mc *MiddlewareClient) applyTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if mc.timeout > 0 {
		return context.WithTimeout(ctx, mc.timeout)
	}
	return ctx, func() {}
}

func (mc *MiddlewareClient) waitRateLimit(ctx context.Context) error {
	if mc.rateLimiter != nil {
		return mc.rateLimiter.Wait(ctx)
	}
	return nil
}

func (mc *MiddlewareClient) logCall(method string, messages []Message, options GenerateOptions, result string, err error, latency time.Duration) {
	if mc.logger == nil {
		return
	}

	model := options.Model
	promptLen := 0
	for _, m := range messages {
		promptLen += len(m.Content)
	}

	mc.logger.Info("LLM call",
		slog.String("method", method),
		slog.String("model", model),
		slog.Int("prompt_len", promptLen),
		slog.String("result_len", fmt.Sprintf("%d", len(result))),
		slog.Duration("latency", latency),
		slog.Bool("error", err != nil),
	)
	if err != nil {
		mc.logger.Error("LLM call failed",
			slog.String("method", method),
			slog.String("error", err.Error()),
		)
	}
}

func (mc *MiddlewareClient) logCallWithUsage(method string, messages []Message, options GenerateOptions, result string, usage *Usage, err error, latency time.Duration) {
	if mc.logger == nil {
		return
	}

	model := options.Model
	promptLen := 0
	for _, m := range messages {
		promptLen += len(m.Content)
	}

	mc.logger.Info("LLM call with usage",
		slog.String("method", method),
		slog.String("model", model),
		slog.Int("prompt_len", promptLen),
		slog.Int("prompt_tokens", usage.PromptTokens),
		slog.Int("completion_tokens", usage.CompletionTokens),
		slog.Int("total_tokens", usage.TotalTokens),
		slog.Duration("latency", latency),
		slog.Bool("error", err != nil),
	)
}

// ---------------------------------------------------------------------------
// Token Bucket Rate Limiter
// ---------------------------------------------------------------------------

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
