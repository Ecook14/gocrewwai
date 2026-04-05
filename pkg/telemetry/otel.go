package telemetry

import (
	"context"
	"fmt"
	//"io"
	"net/http"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	Tracer  = otel.Tracer("crew-go")
	Enabled bool
)

// TelemetryConfig defines settings for OpenTelemetry initialization.
type TelemetryConfig struct {
	Enabled      bool
	ServiceName  string
	Exporter     string
	SamplingRate float64
	PrometheusEnabled bool
	PrometheusPort    int
}

// InitTelemetry initializes OpenTelemetry based on provided settings.
func InitTelemetry(cfg TelemetryConfig) (*sdktrace.TracerProvider, error) {
	if !cfg.Enabled {
		Enabled = false
		return nil, nil
	}
	Enabled = true

	var exporter sdktrace.SpanExporter
	var err error

	// Select exporter based on config
	switch cfg.Exporter {
	case "otlp":
		// Note: In a real production environment, you would use OTLP.
		// For now, we remain Zero-Dependency compatible by defaulting to structured stdout.
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	default:
		exporter, err = stdouttrace.New(stdouttrace.WithWriter(os.Stdout))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create telemetry exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(cfg.ServiceName),
		)),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRate)),
	)

	otel.SetTracerProvider(tp)

	// Initialize Prometheus if enabled
	if cfg.PrometheusEnabled {
		go startPrometheusServer(cfg.PrometheusPort)
	}

	return tp, nil
}

func startPrometheusServer(port int) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", GlobalMetrics().Handler())
	
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("📊 Prometheus metrics server starting on %s/metrics\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Printf("Warning: Prometheus server failed: %v\n", err)
	}
}

// StartSpan is a high-level helper to start a span with context.
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if !Enabled {
		return ctx, trace.SpanFromContext(ctx) // Return no-op or current
	}
	return Tracer.Start(ctx, name)
}

// GetSpan returns the current span from the context.
func GetSpan(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// WithSpan executes a function within a named span.
func WithSpan(ctx context.Context, name string, fn func(context.Context) error) error {
	ctx, span := StartSpan(ctx, name)
	if span != nil {
		defer span.End()
	}
	return fn(ctx)
}
