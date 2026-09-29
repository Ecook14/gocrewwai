package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/version"
)

func TestServerNew(t *testing.T) {
	s := NewServer()
	if s == nil {
		t.Fatal("Expected non-nil server")
	}
}

func TestServerRun(t *testing.T) {
	s := NewServer()
	go func() {
		_ = s.Run(":0")
	}()
}

func TestServerHealthReportsVersion(t *testing.T) {
	t.Setenv("API_AUTH_TOKEN", "test-token")
	s := NewServer()

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", w.Code, http.StatusOK)
	}
	var body struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body.Version != version.Display() {
		t.Fatalf("health version = %q, want %q", body.Version, version.Display())
	}
}

func TestServerSetupRoutes(t *testing.T) {
	token := "test-token"
	os.Setenv("API_AUTH_TOKEN", token)
	s := NewServer()
	os.Setenv("API_AUTH_TOKEN", "")

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}
