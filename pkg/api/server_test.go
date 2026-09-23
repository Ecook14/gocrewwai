package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
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
