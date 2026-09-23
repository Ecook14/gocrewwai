package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// stderrExporter is a simple span exporter that writes spans to stderr.
// Used as the default when OTLP is not configured, providing visibility
// without requiring an external collector.
type stderrExporter struct{}

func (e *stderrExporter) ExportSpans(ctx context.Context, ss []sdktrace.ReadOnlySpan) error {
	for _, s := range ss {
		fmt.Fprintf(os.Stderr, "[trace] %s %s %s\n", s.SpanContext().TraceID(), s.Name(), s.SpanContext().SpanID())
	}
	return nil
}

func (e *stderrExporter) Shutdown(ctx context.Context) error { return nil }

var (
	Tracer  = otel.Tracer("crew-go")
	Enabled bool
)

// TelemetryConfig defines settings for OpenTelemetry initialization.
type TelemetryConfig struct {
	Enabled           bool
	ServiceName       string
	Exporter          string // "otlp", "stderr"
	OTLPCollectorAddr string // OTLP gRPC collector address (e.g. "localhost:4317")
	SamplingRate      float64
	PrometheusEnabled bool
	PrometheusPort    int
}

// DefaultTelemetryConfig returns sensible OpenTelemetry defaults.
func DefaultTelemetryConfig() TelemetryConfig {
	return TelemetryConfig{
		Enabled:           true,
		Exporter:          "stderr",
		OTLPCollectorAddr: "",
		SamplingRate:      1.0,
	}
}

// InitTelemetry initializes OpenTelemetry based on provided settings.
// When Exporter is "otlp" and OTLPCollectorAddr is set, traces are sent to
// the OTLP collector over gRPC. When the collector is unreachable, the
// function returns an error rather than silently falling back to stderr.
func InitTelemetry(cfg TelemetryConfig) (*sdktrace.TracerProvider, error) {
	if !cfg.Enabled {
		Enabled = false
		return nil, nil
	}
	Enabled = true

	if cfg.ServiceName == "" {
		cfg.ServiceName = os.Getenv("OTEL_SERVICE_NAME")
		if cfg.ServiceName == "" {
			cfg.ServiceName = "gocrewwai"
		}
	}

	var exporter sdktrace.SpanExporter
	var err error

	switch cfg.Exporter {
	case "otlp":
		exporter, err = newOTLPExporter(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
	default:
		exporter = &stderrExporter{}
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

	if cfg.PrometheusEnabled {
		go startPrometheusServer(cfg.PrometheusPort)
	}

	return tp, nil
}

// newOTLPExporter creates a gRPC OTLP trace exporter connected to the
// collector address from config. It does not fall back to stderr when
// the collector is unreachable.
func newOTLPExporter(cfg TelemetryConfig) (sdktrace.SpanExporter, error) {
	addr := cfg.OTLPCollectorAddr
	if addr == "" {
		addr = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}
	if addr == "" {
		addr = "localhost:4317"
	}

	clientOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithInsecure(), // Use TLS in production via WithTLSCredentials
	}

	if cfg.OTLPCollectorAddr != "" {
		clientOpts = append(clientOpts, otlptracegrpc.WithEndpoint(addr))
	}

	return otlptracegrpc.New(context.Background(), clientOpts...)
}

func startPrometheusServer(port int) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", GlobalMetrics().Handler())

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Prometheus metrics server starting on %s/metrics\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Printf("Warning: Prometheus server failed: %v\n", err)
	}
}

// StartSpan is a high-level helper to start a span with context.
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if !Enabled {
		return ctx, trace.SpanFromContext(ctx)
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
