package telemetry

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc/credentials"

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
		// Record-safe encoding: span names may carry request-controlled
		// text (CWE-117). Strip CR/LF and non-printables so each span
		// occupies exactly one physical log record.
		fmt.Fprintf(os.Stderr, "[trace] %s %s %s\n", s.SpanContext().TraceID(), sanitizeSpanName(s.Name()), s.SpanContext().SpanID())
	}
	return nil
}

// sanitizeSpanName keeps printable runes and replaces record-breaking
// characters so names cannot forge log lines.
func sanitizeSpanName(name string) string {
	const maxLen = 256
	if len(name) > maxLen {
		name = name[:maxLen]
	}
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' {
			return ' '
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
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
//
// SamplingRate is clamped to [0,1]. The stderr exporter only emits span
// names and IDs (no attributes), so it never leaks prompt content.
func InitTelemetry(cfg TelemetryConfig) (*sdktrace.TracerProvider, error) {
	if !cfg.Enabled {
		Enabled = false
		return nil, nil
	}
	Enabled = true

	if cfg.SamplingRate < 0 {
		cfg.SamplingRate = 0
	}
	if cfg.SamplingRate > 1 {
		cfg.SamplingRate = 1
	}

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
//
// Transport security: loopback collectors (localhost/127.0.0.1/::1) use
// insecure gRPC; non-loopback collectors use TLS with system roots unless
// OTEL_EXPORTER_OTLP_INSECURE=1 explicitly opts into plaintext.
func newOTLPExporter(cfg TelemetryConfig) (sdktrace.SpanExporter, error) {
	addr := cfg.OTLPCollectorAddr
	if addr == "" {
		addr = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}
	if addr == "" {
		addr = "localhost:4317"
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	insecure := strings.EqualFold(os.Getenv("OTEL_EXPORTER_OTLP_INSECURE"), "1") ||
		host == "localhost" || host == "127.0.0.1" || host == "::1"

	clientOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(addr),
	}
	if insecure {
		clientOpts = append(clientOpts, otlptracegrpc.WithInsecure())
	} else {
		clientOpts = append(clientOpts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})))
	}

	return otlptracegrpc.New(context.Background(), clientOpts...)
}

func startPrometheusServer(port int) {
	if port <= 0 || port > 65535 {
		fmt.Printf("Warning: invalid Prometheus port %d, metrics server disabled\n", port)
		return
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", GlobalMetrics().Handler())

	// Bind loopback only: metrics must not be exposed on all interfaces
	// without an explicit override.
	bind := os.Getenv("PROMETHEUS_BIND")
	if bind == "" {
		bind = "127.0.0.1"
	}
	addr := fmt.Sprintf("%s:%d", bind, port)
	fmt.Printf("Prometheus metrics server starting on %s/metrics\n", addr)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
