package llm

import (
	"context"
)

// DefaultsClient decorates any Client, injecting default generation options
// (temperature, max tokens, stop sequences) when the per-call options leave
// them unset. Explicit per-call values always win. No-op when defaults are zero.
type DefaultsClient struct {
	inner    Client
	defaults GenerateOptions
}

// WithDefaults wraps client so unset generation options fall back to defaults.
func WithDefaults(client Client, defaults GenerateOptions) *DefaultsClient {
	return &DefaultsClient{inner: client, defaults: defaults}
}

// Unwrap returns the underlying client.
func (d *DefaultsClient) Unwrap() Client { return d.inner }

func (d *DefaultsClient) merged(o GenerateOptions) GenerateOptions {
	if o.Temperature == 0 {
		o.Temperature = d.defaults.Temperature
	}
	if o.MaxTokens == 0 {
		o.MaxTokens = d.defaults.MaxTokens
	}
	if len(o.Stop) == 0 {
		o.Stop = d.defaults.Stop
	}
	if o.Model == "" {
		o.Model = d.defaults.Model
	}
	return o
}

func (d *DefaultsClient) Generate(ctx context.Context, messages []Message, options GenerateOptions) (string, error) {
	return d.inner.Generate(ctx, messages, d.merged(options))
}

func (d *DefaultsClient) GenerateWithUsage(ctx context.Context, messages []Message, options GenerateOptions) (string, *Usage, error) {
	return d.inner.GenerateWithUsage(ctx, messages, d.merged(options))
}

func (d *DefaultsClient) GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options GenerateOptions) (interface{}, error) {
	return d.inner.GenerateStructured(ctx, messages, schema, d.merged(options))
}

func (d *DefaultsClient) StreamGenerate(ctx context.Context, messages []Message, options GenerateOptions) (<-chan string, error) {
	return d.inner.StreamGenerate(ctx, messages, d.merged(options))
}
