package telemetry

import (
	"context"
	"fmt"
	"io"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer  = otel.Tracer("gocrewwai")
	Enabled bool // Toggleable by developer
)

// InitTelemetry initializes a basic OpenTelemetry provider.
func InitTelemetry(w io.Writer) (*sdktrace.TracerProvider, error) {
	Enabled = true
	exporter, err := stdouttrace.New(stdouttrace.WithWriter(w))
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("gocrew-app"),
		)),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

// StartSpan is a helper to start a span only if telemetry is enabled.
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if !Enabled {
		return ctx, nil // trace.Span is an interface, nil is fine if not used or use no-op
	}
	return Tracer.Start(ctx, name)
}
