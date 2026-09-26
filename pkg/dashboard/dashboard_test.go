package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestDashboardAuthOpenWithoutToken(t *testing.T) {
	t.Setenv("DASHBOARD_AUTH_TOKEN", "")
	t.Setenv("API_AUTH_TOKEN", "")
	req := httptest.NewRequest(http.MethodPost, "/api/stop", nil)
	rec := httptest.NewRecorder()
	dashboardAuth(okHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("no-token dev default: got %d, want 200", rec.Code)
	}
}

func TestDashboardAuthRejectsBadToken(t *testing.T) {
	t.Setenv("DASHBOARD_AUTH_TOKEN", "secret-1")
	req := httptest.NewRequest(http.MethodPost, "/api/stop", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	dashboardAuth(okHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad token: got %d, want 401", rec.Code)
	}
}

func TestDashboardAuthAcceptsGoodToken(t *testing.T) {
	t.Setenv("DASHBOARD_AUTH_TOKEN", "secret-1")
	req := httptest.NewRequest(http.MethodPost, "/api/stop", nil)
	req.Header.Set("Authorization", "Bearer secret-1")
	rec := httptest.NewRecorder()
	dashboardAuth(okHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("good token: got %d, want 200", rec.Code)
	}
}

func TestDashboardAuthFallsBackToAPIToken(t *testing.T) {
	t.Setenv("DASHBOARD_AUTH_TOKEN", "")
	t.Setenv("API_AUTH_TOKEN", "api-secret")
	req := httptest.NewRequest(http.MethodPost, "/api/config", nil)
	req.Header.Set("Authorization", "Bearer api-secret")
	rec := httptest.NewRecorder()
	dashboardAuth(okHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("API_AUTH_TOKEN fallback: got %d, want 200", rec.Code)
	}
}

func TestDashboardAuthSkipsNonAPIPaths(t *testing.T) {
	t.Setenv("DASHBOARD_AUTH_TOKEN", "secret-1")
	for _, path := range []string{"/ws", "/web-ui/index.html", "/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		dashboardAuth(okHandler()).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s without token: got %d, want 200", path, rec.Code)
		}
	}
}

func TestCheckOriginSameOrigin(t *testing.T) {
	t.Setenv("DASHBOARD_ALLOWED_ORIGINS", "")
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Host = "localhost:8080"
	req.Header.Set("Origin", "http://localhost:8080")
	if !upgrader.CheckOrigin(req) {
		t.Fatal("same-origin request rejected")
	}
}

func TestCheckOriginCrossSiteDenied(t *testing.T) {
	t.Setenv("DASHBOARD_ALLOWED_ORIGINS", "")
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Host = "localhost:8080"
	req.Header.Set("Origin", "https://evil.example")
	if upgrader.CheckOrigin(req) {
		t.Fatal("cross-site request allowed without allowlist")
	}
}

func TestCheckOriginNoHeaderAllowed(t *testing.T) {
	t.Setenv("DASHBOARD_ALLOWED_ORIGINS", "")
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	if !upgrader.CheckOrigin(req) {
		t.Fatal("non-browser request without Origin rejected")
	}
}

func TestCheckOriginAllowlist(t *testing.T) {
	t.Setenv("DASHBOARD_ALLOWED_ORIGINS", "https://app.example.com")
	allowed := httptest.NewRequest(http.MethodGet, "/ws", nil)
	allowed.Header.Set("Origin", "https://app.example.com")
	if !upgrader.CheckOrigin(allowed) {
		t.Fatal("allowlisted origin rejected")
	}
	denied := httptest.NewRequest(http.MethodGet, "/ws", nil)
	denied.Header.Set("Origin", "https://other.example.com")
	if upgrader.CheckOrigin(denied) {
		t.Fatal("non-allowlisted origin allowed")
	}
}
