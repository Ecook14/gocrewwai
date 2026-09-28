package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func mintTestJWT(t *testing.T, secret, sub string) string {
	t.Helper()
	hb, _ := json.Marshal(map[string]any{"alg": "HS256", "typ": "JWT"})
	pb, _ := json.Marshal(map[string]any{"sub": sub, "exp": time.Now().Add(time.Hour).Unix()})
	h, p := base64.RawURLEncoding.EncodeToString(hb), base64.RawURLEncoding.EncodeToString(pb)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(h + "." + p))
	return h + "." + p + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func TestAuthMiddleware_JWTBearer(t *testing.T) {
	t.Setenv("JWT_SECRET", "server-jwt-test-secret")
	s := New(WithAPIKeys([]string{"valid-key-123"}))
	handler := s.buildHandler()

	req := httptest.NewRequest("GET", "/info", nil)
	req.Header.Set("Authorization", "Bearer "+mintTestJWT(t, "server-jwt-test-secret", "alice"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("valid JWT request returned %d, want 200", rec.Code)
	}
}

func TestAuthMiddleware_JWTWrongSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "server-jwt-test-secret")
	s := New(WithAPIKeys([]string{"valid-key-123"}))
	handler := s.buildHandler()

	req := httptest.NewRequest("GET", "/info", nil)
	req.Header.Set("Authorization", "Bearer "+mintTestJWT(t, "other-secret", "alice"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong-secret JWT request returned %d, want 401", rec.Code)
	}
}

func TestAuthMiddleware_JWTUnset(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	s := New(WithAPIKeys([]string{"valid-key-123"}))
	handler := s.buildHandler()

	req := httptest.NewRequest("GET", "/info", nil)
	req.Header.Set("Authorization", "Bearer "+"anything.at.all")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("JWT without configured secret returned %d, want 401", rec.Code)
	}
}
