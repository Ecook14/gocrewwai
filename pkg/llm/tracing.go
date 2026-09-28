package llm

import (
	"context"

	"github.com/Ecook14/gocrewwai/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

// TracedClient decorates any Client with OpenTelemetry spans, so all six
// providers (openai, anthropic, gemini, groq, openrouter, ollama) get uniform
// observability without per-provider edits. Spans record only model name and
// message count — never prompt content. No-op unless telemetry is enabled.
type TracedClient struct {
	inner Client
	name  string
}

// NewTracedClient wraps client with tracing spans under the given name.
func NewTracedClient(name string, client Client) *TracedClient {
	return &TracedClient{inner: client, name: name}
}

// Unwrap returns the underlying client.
func (t *TracedClient) Unwrap() Client { return t.inner }

func (t *TracedClient) Generate(ctx context.Context, messages []Message, options GenerateOptions) (string, error) {
	ctx, span := telemetry.StartSpan(ctx, "llm."+t.name+".Generate")
	if span != nil {
		span.SetAttributes(
			attribute.String("llm.provider", t.name),
			attribute.String("llm.model", options.Model),
			attribute.Int("llm.message_count", len(messages)),
		)
		defer span.End()
	}
	return t.inner.Generate(ctx, messages, options)
}

func (t *TracedClient) GenerateWithUsage(ctx context.Context, messages []Message, options GenerateOptions) (string, *Usage, error) {
	ctx, span := telemetry.StartSpan(ctx, "llm."+t.name+".GenerateWithUsage")
	if span != nil {
		span.SetAttributes(
			attribute.String("llm.provider", t.name),
			attribute.String("llm.model", options.Model),
			attribute.Int("llm.message_count", len(messages)),
		)
		defer span.End()
	}
	return t.inner.GenerateWithUsage(ctx, messages, options)
}

func (t *TracedClient) GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options GenerateOptions) (interface{}, error) {
	ctx, span := telemetry.StartSpan(ctx, "llm."+t.name+".GenerateStructured")
	if span != nil {
		span.SetAttributes(
			attribute.String("llm.provider", t.name),
			attribute.String("llm.model", options.Model),
		)
		defer span.End()
	}
	return t.inner.GenerateStructured(ctx, messages, schema, options)
}

func (t *TracedClient) StreamGenerate(ctx context.Context, messages []Message, options GenerateOptions) (<-chan string, error) {
	ctx, span := telemetry.StartSpan(ctx, "llm."+t.name+".StreamGenerate")
	if span != nil {
		span.SetAttributes(
			attribute.String("llm.provider", t.name),
			attribute.String("llm.model", options.Model),
		)
		defer span.End()
	}
	return t.inner.StreamGenerate(ctx, messages, options)
}
