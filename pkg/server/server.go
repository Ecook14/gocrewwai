package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Ecook14/crewai-go/pkg/telemetry"
)

// ---------------------------------------------------------------------------
// Production Server — Health Checks, Metrics, Graceful Shutdown
// ---------------------------------------------------------------------------
//
// Usage:
//
//	srv := server.New(
//	    server.WithAddr(":9090"),
//	    server.WithMetrics(telemetry.GlobalMetrics()),
//	)
//	srv.ListenAndServe() // blocks until shutdown signal

// Server provides an HTTP server with health, metrics, and admin endpoints.
type Server struct {
	addr      string
	metrics   *telemetry.Metrics
	logger    *slog.Logger
	mux       *http.ServeMux
	ready     atomic.Bool
	checks    map[string]HealthCheck
	mu        sync.RWMutex
	onShutdown []func()
}

// HealthCheck is a function that returns an error if the check fails.
type HealthCheck func(ctx context.Context) error

// Option configures the server.
type Option func(*Server)

// WithAddr sets the listen address.
func WithAddr(addr string) Option {
	return func(s *Server) { s.addr = addr }
}

// WithMetrics attaches a metrics collector.
func WithMetrics(m *telemetry.Metrics) Option {
	return func(s *Server) { s.metrics = m }
}

// WithLogger sets the server logger.
func WithLogger(l *slog.Logger) Option {
	return func(s *Server) { s.logger = l }
}

// New creates a production server.
func New(opts ...Option) *Server {
	s := &Server{
		addr:    ":9090",
		logger:  slog.Default(),
		mux:     http.NewServeMux(),
		checks:  make(map[string]HealthCheck),
	}
	for _, opt := range opts {
		opt(s)
	}

	s.registerRoutes()
	return s
}

// RegisterHealthCheck adds a named health check.
func (s *Server) RegisterHealthCheck(name string, check HealthCheck) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checks[name] = check
}

// SetReady marks the server as ready to accept traffic.
func (s *Server) SetReady(ready bool) {
	s.ready.Store(ready)
}

// OnShutdown registers a function to call during graceful shutdown.
func (s *Server) OnShutdown(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onShutdown = append(s.onShutdown, fn)
}

func (s *Server) registerRoutes() {
	// Health check — always returns 200 if process is alive
	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		checks := s.checks
		s.mu.RUnlock()

		results := make(map[string]string)
		allHealthy := true

		for name, check := range checks {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			if err := check(ctx); err != nil {
				results[name] = fmt.Sprintf("unhealthy: %s", err.Error())
				allHealthy = false
			} else {
				results[name] = "healthy"
			}
			cancel()
		}

		if len(results) == 0 {
			results["server"] = "healthy"
		}

		w.Header().Set("Content-Type", "application/json")
		if allHealthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(results)
	})

	// Readiness — returns 200 only when explicitly set ready
	s.mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if s.ready.Load() {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "ready")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, "not ready")
		}
	})

	// Metrics endpoint
	if s.metrics != nil {
		s.mux.Handle("/metrics", s.metrics.Handler())
	}

	// Info endpoint
	s.mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		info := map[string]interface{}{
			"service": "crew-go",
			"version": "1.0.0",
			"go":      "1.22+",
		}
		if s.metrics != nil {
			snap := s.metrics.Snapshot()
			info["uptime_seconds"] = snap.UptimeSeconds
			info["active_agents"] = snap.ActiveAgents
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})
}

// ListenAndServe starts the HTTP server with graceful shutdown on SIGINT/SIGTERM.
func (s *Server) ListenAndServe() error {
	httpServer := &http.Server{
		Addr:         s.addr,
		Handler:      s.mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh

		s.logger.Info("Shutdown signal received", slog.String("signal", sig.String()))
		s.ready.Store(false)

		// Run shutdown hooks
		s.mu.RLock()
		hooks := s.onShutdown
		s.mu.RUnlock()
		for _, fn := range hooks {
			fn()
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			s.logger.Error("Shutdown error", slog.Any("error", err))
		}
		close(done)
	}()

	s.ready.Store(true)
	s.logger.Info("Server starting", slog.String("addr", s.addr))

	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	<-done
	s.logger.Info("Server stopped gracefully")
	return nil
}

// Mux returns the underlying ServeMux for adding custom routes.
func (s *Server) Mux() *http.ServeMux {
	return s.mux
}
