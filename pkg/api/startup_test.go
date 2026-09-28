package api

import (
	"testing"
)

// TestServerRun_RefusesInsecureDefault proves fail-closed startup: without
// tokens and without ALLOW_INSECURE_DEV=1, Run returns an error instead of
// serving with the well-known default token.
func TestServerRun_RefusesInsecureDefault(t *testing.T) {
	t.Setenv("API_AUTH_TOKEN", "")
	t.Setenv("API_AUTH_TOKENS", "")
	t.Setenv("ALLOW_INSECURE_DEV", "")
	s := NewServer()
	if err := s.Run(":0"); err == nil {
		t.Fatal("expected Run to refuse insecure default token")
	}
}

// TestServerRun_DevOptOut allows the insecure default only with explicit opt-in.
func TestServerRun_DevOptOut(t *testing.T) {
	t.Setenv("API_AUTH_TOKEN", "")
	t.Setenv("API_AUTH_TOKENS", "")
	t.Setenv("ALLOW_INSECURE_DEV", "1")
	s := NewServer()
	if s.startupErr != nil {
		t.Fatalf("expected no startup error with opt-out, got %v", s.startupErr)
	}
}

// TestServerRun_WithToken has no startup error when a real token is set.
func TestServerRun_WithToken(t *testing.T) {
	t.Setenv("API_AUTH_TOKEN", "test-startup-token")
	t.Setenv("API_AUTH_TOKENS", "")
	t.Setenv("ALLOW_INSECURE_DEV", "")
	s := NewServer()
	if s.startupErr != nil {
		t.Fatalf("expected no startup error with token, got %v", s.startupErr)
	}
}
