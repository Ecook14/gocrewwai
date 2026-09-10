package api

import (
	"net/http"
	"net/http/httptest"
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
	s := NewServer()
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}
