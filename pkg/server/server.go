// Copyright 2026 Ecook14
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package server provides the production HTTP server for gocrewwai
// with health checks, metrics, rate limiting, auth, and admin endpoints.
//
// Usage:
//
//	srv := server.New(
//	    server.WithAddr(":9090"),
//	    server.WithMetrics(telemetry.GlobalMetrics()),
//	    server.WithAPIKeys([]string{"sk-..."}),
//	    server.WithRateLimit(100, time.Second),
//	)
//	srv.ListenAndServe() // blocks until shutdown signal
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/telemetry"
)

// Server provides an HTTP server with health, metrics, rate limiting,
// authentication, and admin endpoints.
type Server struct {
	addr            string
	metrics         *telemetry.Metrics
	logger          *slog.Logger
	apiKeys         []string
	rateLimit       int
	rateLimitWin    time.Duration
	rateLimiters    map[string]*tokenBucket
	mux             *http.ServeMux
	httpServer      *http.Server
	ready           atomic.Bool
	checks          map[string]HealthCheck
	mu              sync.RWMutex
	onShutdown      []func()
	shutdownTimeout time.Duration
}

// tokenBucket implements a simple token bucket rate limiter.
type tokenBucket struct {
	mu         sync.Mutex
	tokens     int
	lastRefill time.Time
	rate       int
	window     time.Duration
}

func newTokenBucket(rate int, window time.Duration) *tokenBucket {
	return &tokenBucket{
		tokens:     rate,
		lastRefill: time.Now(),
		rate:       rate,
		window:     window,
	}
}

func (tb *tokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	tb.lastRefill = now

	windows := int(elapsed / tb.window)
	if windows > 0 {
		tb.tokens = min(tb.rate, tb.tokens+windows*tb.rate)
	}

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	return false
}

// HealthCheck is a function that returns an error if the check fails.
type HealthCheck func(ctx context.Context) error

// Option configures the server.
type Option func(*Server)

// WithAddr sets the listen address.
func WithAddr(addr string) Option {
	return func(s *Server) { s.addr = addr }
}

// WithMetrics sets the metrics provider.
func WithMetrics(m *telemetry.Metrics) Option {
	return func(s *Server) { s.metrics = m }
}

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) Option {
	return func(s *Server) { s.logger = l }
}

// WithAPIKeys sets the allowed API keys for authentication.
// When set, all requests must include a valid X-API-Key header.
func WithAPIKeys(keys []string) Option {
	return func(s *Server) { s.apiKeys = keys }
}

// WithRateLimit sets rate limiting: rate requests per window.
// When rate <= 0, rate limiting is disabled.
func WithRateLimit(rate int, window time.Duration) Option {
	return func(s *Server) {
		s.rateLimit = rate
		s.rateLimitWin = window
	}
}

// WithShutdownTimeout sets the timeout for graceful shutdown.
// Default is 15 seconds.
func WithShutdownTimeout(d time.Duration) Option {
	return func(s *Server) { s.shutdownTimeout = d }
}

// New creates a new server with the given options.
func New(opts ...Option) *Server {
	s := &Server{
		addr:            ":9090",
		checks:          make(map[string]HealthCheck),
		mux:             http.NewServeMux(),
		logger:          slog.Default(),
		rateLimiters:    make(map[string]*tokenBucket),
		shutdownTimeout: 15 * time.Second,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.registerRoutes()
	return s
}

// RegisterHealthCheck registers a health check with the given name.
func (s *Server) RegisterHealthCheck(name string, check HealthCheck) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checks[name] = check
}

// SetReady sets the readiness status.
func (s *Server) SetReady(ready bool) {
	s.ready.Store(ready)
}

// IsReady returns the current readiness status.
func (s *Server) IsReady() bool {
	return s.ready.Load()
}

// OnShutdown registers a callback to be called on graceful shutdown.
func (s *Server) OnShutdown(fn func()) {
	s.onShutdown = append(s.onShutdown, fn)
}

// Start starts the server in a goroutine. Use Shutdown to stop it.
func (s *Server) Start() error {
	handler := s.buildHandler()
	httpServer := &http.Server{
		Addr:         s.addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	s.httpServer = httpServer
	return httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() error {
	// Always fire shutdown callbacks, even if httpServer is nil
	for _, fn := range s.onShutdown {
		fn()
	}
	if s.httpServer == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// buildHandler constructs the middleware chain.
// Middleware is applied in this order (outer to inner):
//   logging -> rateLimit -> auth -> routes
// This means rate limiting is checked first, then auth, then the route handler.
func (s *Server) buildHandler() http.Handler {
	handler := s.copyRoutes()

	if len(s.apiKeys) > 0 {
		handler = s.authMiddleware(handler)
	}
	if s.rateLimit > 0 {
		handler = s.rateLimitMiddleware(handler)
	}
	handler = s.loggingMiddleware(handler)
	return handler
}

// copyRoutes returns a handler with all routes copied from s.mux.
// This is needed so middleware can wrap without affecting the original mux.
func (s *Server) copyRoutes() http.Handler {
	mux := http.NewServeMux()
	
	// Copy health endpoints
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/health", s.handleHealth)
	
	// Copy readiness endpoints
	mux.HandleFunc("/readyz", s.handleReadiness)
	mux.HandleFunc("/ready", s.handleReadiness)
	
	// Copy metrics
	if s.metrics != nil {
		mux.Handle("/metrics", s.metrics.Handler())
	}
	
	// Copy info
	mux.HandleFunc("/info", s.handleInfo)
	
	// Copy pprof endpoints
	s.copyPprofRoutes(mux)
	
	return mux
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/healthz", s.handleHealth)
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/readyz", s.handleReadiness)
	s.mux.HandleFunc("/ready", s.handleReadiness)
	
	if s.metrics != nil {
		s.mux.Handle("/metrics", s.metrics.Handler())
	}
	
	s.mux.HandleFunc("/info", s.handleInfo)
	s.registerPprofRoutes()
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
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
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	if s.ready.Load() {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ready")
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "not ready")
	}
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"service": "gocrew",
		"version": "1.0.0-beta.1",
		"go":      "1.23+",
	}
	if s.metrics != nil {
		snap := s.metrics.Snapshot()
		info["uptime_seconds"] = snap.UptimeSeconds
		info["active_agents"] = snap.ActiveAgents
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (s *Server) registerPprofRoutes() {
	s.mux.HandleFunc("/debug/pprof/", func(w http.ResponseWriter, r *http.Request) {
		if len(s.apiKeys) > 0 && !s.validateAPIKey(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		pprof.Index(w, r)
	})
	s.mux.HandleFunc("/debug/pprof/cmdline", func(w http.ResponseWriter, r *http.Request) {
		if len(s.apiKeys) > 0 && !s.validateAPIKey(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		pprof.Cmdline(w, r)
	})
	s.mux.HandleFunc("/debug/pprof/profile", func(w http.ResponseWriter, r *http.Request) {
		if len(s.apiKeys) > 0 && !s.validateAPIKey(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		pprof.Profile(w, r)
	})
	s.mux.HandleFunc("/debug/pprof/symbol", func(w http.ResponseWriter, r *http.Request) {
		if len(s.apiKeys) > 0 && !s.validateAPIKey(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		pprof.Symbol(w, r)
	})
	s.mux.HandleFunc("/debug/pprof/trace", func(w http.ResponseWriter, r *http.Request) {
		if len(s.apiKeys) > 0 && !s.validateAPIKey(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		pprof.Trace(w, r)
	})
}

func (s *Server) copyPprofRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/debug/pprof/", func(w http.ResponseWriter, r *http.Request) {
		if len(s.apiKeys) > 0 && !s.validateAPIKey(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		pprof.Index(w, r)
	})
	mux.HandleFunc("/debug/pprof/cmdline", func(w http.ResponseWriter, r *http.Request) {
		if len(s.apiKeys) > 0 && !s.validateAPIKey(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		pprof.Cmdline(w, r)
	})
	mux.HandleFunc("/debug/pprof/profile", func(w http.ResponseWriter, r *http.Request) {
		if len(s.apiKeys) > 0 && !s.validateAPIKey(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		pprof.Profile(w, r)
	})
	mux.HandleFunc("/debug/pprof/symbol", func(w http.ResponseWriter, r *http.Request) {
		if len(s.apiKeys) > 0 && !s.validateAPIKey(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		pprof.Symbol(w, r)
	})
	mux.HandleFunc("/debug/pprof/trace", func(w http.ResponseWriter, r *http.Request) {
		if len(s.apiKeys) > 0 && !s.validateAPIKey(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		pprof.Trace(w, r)
	})
}

// validateAPIKey checks if the request has a valid API key.
func (s *Server) validateAPIKey(r *http.Request) bool {
	key := r.Header.Get("X-API-Key")
	if key == "" {
		return false
	}
	for _, k := range s.apiKeys {
		if k == key {
			return true
		}
	}
	return false
}

// rateLimitMiddleware wraps handler with per-IP rate limiting.
func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := s.getClientIP(r)
		tb, ok := s.rateLimiters[ip]
		if !ok {
			tb = newTokenBucket(s.rateLimit, s.rateLimitWin)
			s.mu.Lock()
			s.rateLimiters[ip] = tb
			s.mu.Unlock()
		}

		if !tb.Allow() {
			s.logger.Warn("rate limit exceeded", "ip", ip, "path", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error":       "rate limit exceeded",
				"retry_after": s.rateLimitWin.String(),
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts the client IP from the request.
func (s *Server) getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// authMiddleware wraps handler with API key authentication.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/healthz" || path == "/health" || path == "/readyz" || path == "/ready" {
			next.ServeHTTP(w, r)
			return
		}

		if !s.validateAPIKey(r) {
			s.logger.Warn("unauthorized request", "path", path, "ip", s.getClientIP(r))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "unauthorized: invalid or missing API key",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware wraps handler with structured request logging.
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapper := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)

		s.logger.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", s.getClientIP(r),
			"status", wrapper.statusCode,
			"duration_ms", duration.Milliseconds(),
			"user_agent", r.UserAgent(),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// ListenAndServe starts the HTTP server with graceful shutdown on SIGINT/SIGTERM.
func (s *Server) ListenAndServe() error {
	handler := s.buildHandler()

	httpServer := &http.Server{
		Addr:         s.addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	s.httpServer = httpServer

	done := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		s.logger.Info("shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			s.logger.Error("server shutdown failed", "error", err)
		}
		for _, fn := range s.onShutdown {
			fn()
		}
		close(done)
	}()

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	<-done
	return nil
}

// Mux returns the underlying ServeMux for advanced routing.
func (s *Server) Mux() *http.ServeMux {
	return s.mux
}

// Test-only helpers

func (s *Server) getRateLimitState(ip string) (int, bool) {
	tb, ok := s.rateLimiters[ip]
	if !ok {
		return 0, false
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.tokens, true
}

func (s *Server) resetRateLimiters() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rateLimiters = make(map[string]*tokenBucket)
}
