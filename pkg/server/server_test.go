package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/telemetry"
)

func TestNewServerDefaults(t *testing.T) {
	s := New()
	if s.addr != ":9090" {
		t.Errorf("default addr = %q, want %q", s.addr, ":9090")
	}
	if s.IsReady() {
		t.Error("server should not be ready initially")
	}
}

func TestServer_SetReady(t *testing.T) {
	s := New()
	if s.IsReady() {
		t.Error("server should not be ready initially")
	}
	s.SetReady(true)
	if !s.IsReady() {
		t.Error("server should be ready after SetReady(true)")
	}
	s.SetReady(false)
	if s.IsReady() {
		t.Error("server should not be ready after SetReady(false)")
	}
}

func TestServer_RegisterHealthCheck(t *testing.T) {
	s := New()
	checkCalled := false
	healthy := func(ctx context.Context) error {
		checkCalled = true
		return nil
	}
	s.RegisterHealthCheck("test-check", healthy)
	s.mu.RLock()
	_, ok := s.checks["test-check"]
	s.mu.RUnlock()
	if !ok {
		t.Error("health check should be registered")
	}

	// Invoke the health endpoint to actually call the check
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)
	if !checkCalled {
		t.Error("health check should be callable")
	}
}

func TestHealthEndpoint_AllHealthy(t *testing.T) {
	s := New()

	checkPassed := false
	s.RegisterHealthCheck("passing", func(ctx context.Context) error {
		checkPassed = true
		return nil
	})

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/healthz returned %d, want 200", rec.Code)
	}
	if !checkPassed {
		t.Error("health check was not called")
	}

	var result map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result["passing"] != "healthy" {
		t.Errorf("passing check status = %q, want %q", result["passing"], "healthy")
	}
}

func TestHealthEndpoint_WithFailingCheck(t *testing.T) {
	s := New()

	s.RegisterHealthCheck("failing", func(ctx context.Context) error {
		return fmt.Errorf("something wrong")
	})

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/healthz with failing check returned %d, want 503", rec.Code)
	}

	var result map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result["failing"] == "" {
		t.Error("failing check should have status")
	}
}

func TestHealthEndpoint_Alias(t *testing.T) {
	s := New()

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/health returned %d, want 200", rec.Code)
	}
}

func TestReadyEndpoint_Live(t *testing.T) {
	s := New()
	s.SetReady(true)

	req := httptest.NewRequest("GET", "/readyz", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/readyz returned %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ready") {
		t.Errorf("/readyz body = %q, want to contain 'ready'", rec.Body.String())
	}
}

func TestReadyEndpoint_NotReady(t *testing.T) {
	s := New()

	req := httptest.NewRequest("GET", "/readyz", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/readyz not ready returned %d, want 503", rec.Code)
	}
}

func TestReadyEndpoint_AfterSetReady(t *testing.T) {
	s := New()
	s.SetReady(true)

	req := httptest.NewRequest("GET", "/readyz", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/readyz ready returned %d, want 200", rec.Code)
	}
}

func TestReadyEndpoint_Alias(t *testing.T) {
	s := New()
	s.SetReady(true)

	req := httptest.NewRequest("GET", "/ready", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/ready alias returned %d, want 200", rec.Code)
	}
}

func TestMetricsEndpoint_WithoutMetrics(t *testing.T) {
	s := New()

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("/metrics without metrics returned %d, want 404", rec.Code)
	}
}

func TestMetricsEndpoint_WithMetrics(t *testing.T) {
	m := telemetry.NewMetrics()
	s := New(WithMetrics(m))

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/metrics returned %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "crew_go_") {
		t.Error("/metrics should contain crew_go_ prefix")
	}
}

func TestInfoEndpoint(t *testing.T) {
	s := New()

	req := httptest.NewRequest("GET", "/info", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/info returned %d, want 200", rec.Code)
	}

	var info map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if info["service"] != "gocrew" {
		t.Errorf("service = %v, want gocrew", info["service"])
	}
	if info["version"] != "1.0.0-beta.1" {
		t.Errorf("version = %v, want 1.0.0-beta.1", info["version"])
	}
}

// ========== Rate Limiting Tests (using buildHandler for middleware) ==========

func TestRateLimiting_Enabled(t *testing.T) {
	s := New(WithRateLimit(5, time.Minute))
	s.resetRateLimiters()

	handler := s.buildHandler()

	// First request should succeed
	req := httptest.NewRequest("GET", "/info", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("first request returned %d, want 200", rec.Code)
	}

	// Exhaust the rate limit
	s.mu.Lock()
	tb, _ := s.rateLimiters["127.0.0.1"]
	if tb != nil {
		tb.tokens = 0
	}
	s.mu.Unlock()

	// Subsequent request should be rate limited
	req2 := httptest.NewRequest("GET", "/info", nil)
	req2.RemoteAddr = "127.0.0.1:12346"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("rate limited request returned %d, want 429", rec2.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp); err == nil {
		if resp["error"] != "rate limit exceeded" {
			t.Errorf("error message = %q, want 'rate limit exceeded'", resp["error"])
		}
	}
}

func TestRateLimiting_Disabled(t *testing.T) {
	s := New(WithRateLimit(-1, time.Second))

	handler := s.buildHandler()

	req := httptest.NewRequest("GET", "/info", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("request with rate limiting disabled returned %d, want 200", rec.Code)
	}
}

func TestRateLimiting_PerIP(t *testing.T) {
	s := New(WithRateLimit(1, time.Minute))
	s.resetRateLimiters()

	handler := s.buildHandler()

	// First IP: exhaust its limit
	req1 := httptest.NewRequest("GET", "/info", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// Exhaust the one token
	s.mu.Lock()
	tb, _ := s.rateLimiters["192.168.1.1"]
	if tb != nil {
		tb.tokens = 0
	}
	s.mu.Unlock()

	// Second request from same IP should be limited
	req2 := httptest.NewRequest("GET", "/info", nil)
	req2.RemoteAddr = "192.168.1.1:12346"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("same IP second request returned %d, want 429", rec2.Code)
	}

	// Different IP should still have tokens
	req3 := httptest.NewRequest("GET", "/info", nil)
	req3.RemoteAddr = "192.168.1.2:12347"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Errorf("different IP request returned %d, want 200", rec3.Code)
	}
}

func TestRateLimiting_XForwardedFor(t *testing.T) {
	s := New(WithRateLimit(1, time.Minute))
	s.resetRateLimiters()

	handler := s.buildHandler()

	// First request
	req := httptest.NewRequest("GET", "/info", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Exhaust
	s.mu.Lock()
	tb, _ := s.rateLimiters["10.0.0.1"]
	if tb != nil {
		tb.tokens = 0
	}
	s.mu.Unlock()

	// Second request from same X-Forwarded-For should be limited
	req2 := httptest.NewRequest("GET", "/info", nil)
	req2.Header.Set("X-Forwarded-For", "10.0.0.1")
	req2.RemoteAddr = "10.0.0.1:12346"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("X-Forwarded-For rate limit returned %d, want 429", rec2.Code)
	}
}

// ========== Auth Middleware Tests ==========

func TestAuthMiddleware_Enabled_WithValidKey(t *testing.T) {
	s := New(WithAPIKeys([]string{"valid-key-123"}))

	handler := s.buildHandler()

	req := httptest.NewRequest("GET", "/info", nil)
	req.Header.Set("X-API-Key", "valid-key-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("valid API key request returned %d, want 200", rec.Code)
	}
}

func TestAuthMiddleware_Enabled_WithInvalidKey(t *testing.T) {
	s := New(WithAPIKeys([]string{"valid-key-123"}))

	handler := s.buildHandler()

	req := httptest.NewRequest("GET", "/info", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("invalid API key request returned %d, want 401", rec.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err == nil {
		if !strings.Contains(resp["error"], "unauthorized") {
			t.Errorf("error message = %q, should contain 'unauthorized'", resp["error"])
		}
	}
}

func TestAuthMiddleware_Enabled_MissingKey(t *testing.T) {
	s := New(WithAPIKeys([]string{"valid-key-123"}))

	handler := s.buildHandler()

	req := httptest.NewRequest("GET", "/info", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("missing API key request returned %d, want 401", rec.Code)
	}
}

func TestAuthMiddleware_SkipHealthEndpoints(t *testing.T) {
	s := New(WithAPIKeys([]string{"valid-key-123"}))
	s.SetReady(true)

	handler := s.buildHandler()

	endpoints := []string{"/healthz", "/health", "/readyz", "/ready"}
	for _, ep := range endpoints {
		req := httptest.NewRequest("GET", ep, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("%s without API key returned %d, want 200 (should skip auth)", ep, rec.Code)
		}
	}
}

func TestAuthMiddleware_Disabled(t *testing.T) {
	s := New()

	handler := s.buildHandler()

	req := httptest.NewRequest("GET", "/info", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should work without API key when auth is disabled
	if rec.Code != http.StatusOK {
		t.Errorf("request without auth returned %d, want 200", rec.Code)
	}
}

func TestAuthMiddleware_MultipleKeys(t *testing.T) {
	s := New(WithAPIKeys([]string{"key-1", "key-2", "key-3"}))

	handler := s.buildHandler()

	for _, key := range []string{"key-1", "key-2", "key-3"} {
		req := httptest.NewRequest("GET", "/info", nil)
		req.Header.Set("X-API-Key", key)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("key %q returned %d, want 200", key, rec.Code)
		}
	}
}

// ========== pprof Tests ==========

func TestPprofEndpoints_WithoutAuth(t *testing.T) {
	s := New()

	handler := s.buildHandler()

	req := httptest.NewRequest("GET", "/debug/pprof/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// pprof should be accessible without auth when API keys not set
	if rec.Code != http.StatusOK {
		t.Errorf("/debug/pprof/ returned %d, want 200", rec.Code)
	}
}

func TestPprofEndpoints_WithAuth_WithoutKey(t *testing.T) {
	s := New(WithAPIKeys([]string{"admin-key"}))

	handler := s.buildHandler()

	req := httptest.NewRequest("GET", "/debug/pprof/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("/debug/pprof/ without key returned %d, want 401", rec.Code)
	}
}

func TestPprofEndpoints_WithAuth_WithValidKey(t *testing.T) {
	s := New(WithAPIKeys([]string{"admin-key"}))

	handler := s.buildHandler()

	req := httptest.NewRequest("GET", "/debug/pprof/", nil)
	req.Header.Set("X-API-Key", "admin-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/debug/pprof/ with valid key returned %d, want 200", rec.Code)
	}
}

// ========== Logging Middleware Tests ==========

func TestLoggingMiddleware_CapturesStatus(t *testing.T) {
	s := New()

	handler := s.buildHandler()

	req := httptest.NewRequest("GET", "/info", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("logging middleware request returned %d, want 200", rec.Code)
	}
}

func TestLoggingMiddleware_Captures404(t *testing.T) {
	s := New()

	handler := s.buildHandler()

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("404 request returned %d, want 404", rec.Code)
	}
}

// ========== Combined Middleware Tests ==========

func TestCombined_RateLimitAndAuth(t *testing.T) {
	s := New(
		WithAPIKeys([]string{"secret-key"}),
		WithRateLimit(10, time.Minute),
	)
	s.resetRateLimiters()

	handler := s.buildHandler()

	// Valid request should pass both
	req := httptest.NewRequest("GET", "/info", nil)
	req.Header.Set("X-API-Key", "secret-key")
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("combined middleware returned %d, want 200", rec.Code)
	}

	// Exhaust rate limit
	s.mu.Lock()
	tb, _ := s.rateLimiters["127.0.0.1"]
	if tb != nil {
		tb.tokens = 0
	}
	s.mu.Unlock()

	// Even with valid key, should be rate limited
	req2 := httptest.NewRequest("GET", "/info", nil)
	req2.Header.Set("X-API-Key", "secret-key")
	req2.RemoteAddr = "127.0.0.1:12346"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("rate limited even with valid key returned %d, want 429", rec2.Code)
	}
}

func TestCombined_AuthThenRateLimit(t *testing.T) {
	s := New(
		WithAPIKeys([]string{"secret-key"}),
		WithRateLimit(10, time.Minute),
	)
	s.resetRateLimiters()

	handler := s.buildHandler()

	// Exhaust rate limit first
	s.mu.Lock()
	tb := newTokenBucket(10, time.Minute)
	tb.tokens = 0
	s.rateLimiters["127.0.0.1"] = tb
	s.mu.Unlock()

	// Request without API key should get 429 (rate limit checked before auth in middleware chain)
	req := httptest.NewRequest("GET", "/info", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("rate limit should be checked before auth, got %d", rec.Code)
	}
}

// ========== Graceful Shutdown Tests ==========

func TestServer_OnShutdown(t *testing.T) {
	s := New()
	var called bool
	s.OnShutdown(func() { called = true })

	s.mu.Lock()
	if len(s.onShutdown) != 1 {
		t.Fatalf("expected 1 shutdown callback, got %d", len(s.onShutdown))
	}
	s.mu.Unlock()

	// Start server in background
	done := make(chan struct{})
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("ListenAndServe error: %v", err)
		}
		close(done)
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Trigger shutdown via SIGTERM to our own process
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	proc.Signal(syscall.SIGTERM)

	// Wait for shutdown to complete
	select {
	case <-done:
		// OK
	case <-time.After(5 * time.Second):
		t.Error("server did not shut down in time")
	}

	if !called {
		t.Error("shutdown callback should have been called")
	}
}

func TestServer_ShutdownTimeout(t *testing.T) {
	s := New(WithShutdownTimeout(50 * time.Millisecond))

	// Start server in background
	done := make(chan struct{})
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("ListenAndServe error: %v", err)
		}
		close(done)
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Trigger shutdown via SIGTERM
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	proc.Signal(syscall.SIGTERM)

	// Wait for shutdown to complete
	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Error("shutdown did not complete within timeout")
	}
}

func TestServer_MultipleShutdownCallbacks(t *testing.T) {
	s := New()
	var count int
	for i := 0; i < 3; i++ {
		s.OnShutdown(func() { count++ })
	}

	s.mu.Lock()
	if len(s.onShutdown) != 3 {
		t.Fatalf("expected 3 shutdown callbacks, got %d", len(s.onShutdown))
	}
	s.mu.Unlock()

	// Test that callbacks fire when Shutdown is called directly (without server)
	s.Shutdown()
	if count != 3 {
		t.Errorf("expected 3 shutdown callbacks, got %d", count)
	}
}

// ========== getClientIP Tests ==========

func TestGetClientIP_FromRemoteAddr(t *testing.T) {
	s := New()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	ip := s.getClientIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("getClientIP = %q, want %q", ip, "192.168.1.1")
	}
}

func TestGetClientIP_FromXForwardedFor(t *testing.T) {
	s := New()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.2")

	ip := s.getClientIP(req)
	if ip != "203.0.113.5" {
		t.Errorf("getClientIP with X-Forwarded-For = %q, want %q", ip, "203.0.113.5")
	}
}

func TestGetClientIP_FromXRealIP(t *testing.T) {
	s := New()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "198.51.100.7")

	ip := s.getClientIP(req)
	if ip != "198.51.100.7" {
		t.Errorf("getClientIP with X-Real-IP = %q, want %q", ip, "198.51.100.7")
	}
}

// ========== HealthCheck Timeout Tests ==========

func TestHealthCheck_Timeout(t *testing.T) {
	s := New()

	slowCheck := func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}
	s.RegisterHealthCheck("slow", slowCheck)

	req := httptest.NewRequest("GET", "/healthz", nil)
	// Set a very short context timeout
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	// Should return 503 because health check timed out
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/healthz with timeout returned %d, want 503", rec.Code)
	}
}

// ========== Benchmarks ==========

func BenchmarkHealthEndpoint(b *testing.B) {
	s := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/healthz", nil)
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, req)
	}
}

func BenchmarkInfoEndpoint(b *testing.B) {
	s := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/info", nil)
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, req)
	}
}

func BenchmarkRateLimitMiddleware(b *testing.B) {
	s := New(WithRateLimit(1000, time.Minute))
	s.resetRateLimiters()

	handler := s.buildHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/info", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkAuthMiddleware(b *testing.B) {
	s := New(WithAPIKeys([]string{"test-key"}))

	handler := s.buildHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/info", nil)
		req.Header.Set("X-API-Key", "test-key")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// ========== Logger Integration Test ==========

func TestLoggerIntegration(t *testing.T) {
	var logOutput strings.Builder
	logger := slog.New(slog.NewJSONHandler(&logOutput, nil))

	s := New(WithLogger(logger))

	handler := s.buildHandler()

	req := httptest.NewRequest("GET", "/info", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("logger integration returned %d, want 200", rec.Code)
	}

	// Verify logging occurred (log output should contain the request info)
	output := logOutput.String()
	if !strings.Contains(output, `"method":"GET"`) {
		t.Error("logger output should contain method")
	}
	if !strings.Contains(output, `"path":"/info"`) {
		t.Error("logger output should contain path")
	}
}

// ========== Server Build Handler Test ==========

func TestBuildHandler_ReturnsHandler(t *testing.T) {
	s := New()
	handler := s.buildHandler()
	if handler == nil {
		t.Error("buildHandler returned nil")
	}
}

func TestBuildHandler_WithMetrics(t *testing.T) {
	m := telemetry.NewMetrics()
	s := New(WithMetrics(m))
	handler := s.buildHandler()
	if handler == nil {
		t.Error("buildHandler with metrics returned nil")
	}
}

func TestBuildHandler_WithRateLimit(t *testing.T) {
	s := New(WithRateLimit(100, time.Second))
	handler := s.buildHandler()
	if handler == nil {
		t.Error("buildHandler with rate limit returned nil")
	}
}

func TestBuildHandler_WithAuth(t *testing.T) {
	s := New(WithAPIKeys([]string{"key"}))
	handler := s.buildHandler()
	if handler == nil {
		t.Error("buildHandler with auth returned nil")
	}
}

// ========== Copy Routes Test ==========

func TestCopyRoutes_ServesEndpoints(t *testing.T) {
	s := New()
	s.SetReady(true)
	handler := s.copyRoutes()

	tests := []struct {
		path       string
		wantStatus int
	}{
		{"/healthz", http.StatusOK},
		{"/health", http.StatusOK},
		{"/readyz", http.StatusOK},
		{"/ready", http.StatusOK},
		{"/info", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Errorf("%s returned %d, want %d", tt.path, rec.Code, tt.wantStatus)
			}
		})
	}
}

// ========== Request/Response Writer Test ==========

func TestResponseWriter_CapturesStatusCode(t *testing.T) {
	s := New()

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	rec := httptest.NewRecorder()

	// Create a wrapper like the logging middleware does
	wrapper := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	// Serve a 404 directly via mux
	s.mux.ServeHTTP(wrapper, req)

	if wrapper.statusCode != http.StatusNotFound {
		t.Errorf("wrapper captured status %d, want %d", wrapper.statusCode, http.StatusNotFound)
	}
}

// ========== HealthCheck Context Propagation ==========

func TestHealthCheck_ContextPropagation(t *testing.T) {
	s := New()

	ctxCaptured := false
	s.RegisterHealthCheck("ctx-test", func(ctx context.Context) error {
		ctxCaptured = true
		// Verify context has deadline from the handler
		deadline, hasDeadline := ctx.Deadline()
		if !hasDeadline {
			t.Error("health check context should have a deadline")
		}
		_ = deadline
		return nil
	})

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if !ctxCaptured {
		t.Error("health check context was not captured")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("/healthz returned %d, want 200", rec.Code)
	}
}

// ========== Concurrent Rate Limit Access ==========

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	s := New(WithRateLimit(1000, time.Second))
	s.resetRateLimiters()

	handler := s.buildHandler()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/info", nil)
			req.RemoteAddr = fmt.Sprintf("127.0.0.1:%d", 12345+j)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}(i)
	}
	wg.Wait()
	// If we get here without panic, concurrent access is safe
}

// ========== Full Middleware Stack Integration ==========

func TestFullMiddlewareStack(t *testing.T) {
	m := telemetry.NewMetrics()
	s := New(
		WithMetrics(m),
		WithAPIKeys([]string{"prod-key"}),
		WithRateLimit(100, time.Minute),
		WithLogger(slog.Default()),
	)
	s.resetRateLimiters()
	s.SetReady(true)

	handler := s.buildHandler()

	// Test 1: Health endpoint works without auth
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("health endpoint in full stack returned %d, want 200", rec.Code)
	}

	// Test 2: Protected endpoint needs auth
	req2 := httptest.NewRequest("GET", "/info", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("protected endpoint without key returned %d, want 401", rec2.Code)
	}

	// Test 3: Protected endpoint with valid key
	req3 := httptest.NewRequest("GET", "/info", nil)
	req3.Header.Set("X-API-Key", "prod-key")
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Errorf("protected endpoint with key returned %d, want 200", rec3.Code)
	}

	// Test 4: Readiness endpoint
	req4 := httptest.NewRequest("GET", "/readyz", nil)
	rec4 := httptest.NewRecorder()
	handler.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusOK {
		t.Errorf("readiness endpoint in full stack returned %d, want 200", rec4.Code)
	}
}
