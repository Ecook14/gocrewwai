package llm

import (
	"context"
	//"fmt"
	"log/slog"
	"strings"
	//"time"
)

// FailoverClient is a middleware that handles automatic fallback to a secondary
// model/provider when the primary model fails (e.g., rate limited, quota exceeded).
type FailoverClient struct {
	Primary   Client
	Secondary Client
	Logger    *slog.Logger
}

// NewFailoverClient creates a new client with failover capabilities.
func NewFailoverClient(primary, secondary Client, logger *slog.Logger) *FailoverClient {
	if logger == nil {
		logger = slog.Default()
	}
	return &FailoverClient{
		Primary:   primary,
		Secondary: secondary,
		Logger:    logger,
	}
}

// Generate attempts the primary client first, then falls back to secondary on failure.
func (c *FailoverClient) Generate(ctx context.Context, messages []Message, options GenerateOptions) (string, error) {
	if err := CheckBudget(); err != nil {
		return "", err
	}
	result, err := c.Primary.Generate(ctx, messages, options)
	if err != nil && c.isFailoverRequired(err) {
		c.Logger.Warn("Primary LLM failed, triggering failover to secondary", "error", err, "primary_model", options.Model)
		return c.Secondary.Generate(ctx, messages, options)
	}
	return result, err
}

// GenerateWithUsage attempts failover for usage-tracked calls.
func (c *FailoverClient) GenerateWithUsage(ctx context.Context, messages []Message, options GenerateOptions) (string, *Usage, error) {
	if err := CheckBudget(); err != nil {
		return "", nil, err
	}
	result, usage, err := c.Primary.GenerateWithUsage(ctx, messages, options)
	if err != nil && c.isFailoverRequired(err) {
		c.Logger.Warn("Primary LLM failed, triggering failover to secondary", "error", err, "primary_model", options.Model)
		return c.Secondary.GenerateWithUsage(ctx, messages, options)
	}
	return result, usage, err
}

// GenerateStructured attempts failover for structured calls.
func (c *FailoverClient) GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options GenerateOptions) (interface{}, error) {
	if err := CheckBudget(); err != nil {
		return nil, err
	}
	result, err := c.Primary.GenerateStructured(ctx, messages, schema, options)
	if err != nil && c.isFailoverRequired(err) {
		c.Logger.Warn("Primary LLM failed, triggering failover to secondary", "error", err, "primary_model", options.Model)
		return c.Secondary.GenerateStructured(ctx, messages, schema, options)
	}
	return result, err
}

// StreamGenerate attempts failover for streaming.
func (c *FailoverClient) StreamGenerate(ctx context.Context, messages []Message, options GenerateOptions) (<-chan string, error) {
	if err := CheckBudget(); err != nil {
		return nil, err
	}
	ch, err := c.Primary.StreamGenerate(ctx, messages, options)
	if err != nil && c.isFailoverRequired(err) {
		c.Logger.Warn("Primary LLM stream failed to start, triggering failover to secondary", "error", err, "primary_model", options.Model)
		return c.Secondary.StreamGenerate(ctx, messages, options)
	}
	return ch, err
}

// isFailoverRequired determines if the error warrants a fallback attempt.
func (c *FailoverClient) isFailoverRequired(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// Failover on rate limits, quota issues, and server-side errors
	return (strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "quota exceeded") ||
		strings.Contains(msg, "limit exceeded") ||
		strings.Contains(msg, "500") ||
		strings.Contains(msg, "unavailable") ||
		strings.Contains(msg, "overloaded") ||
		strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "connection refused")) && !strings.Contains(msg, "budget exceeded")
}
