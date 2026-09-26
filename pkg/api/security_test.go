package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/events"
)

func fp(token string) string { return tokenFingerprint(token) }

func TestSessionOwnerIsolation(t *testing.T) {
	s := NewServer()
	ownerA, ownerB := fp("tok-A"), fp("tok-B")
	if err := s.persistSessionStart("sess-1", ownerA); err != nil {
		t.Fatalf("persist: %v", err)
	}
	if _, err := s.loadSessionState("sess-1", ownerA); err != nil {
		t.Fatalf("owner read: %v", err)
	}
	if _, err := s.loadSessionState("sess-1", ownerB); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("cross-owner read: got %v, want not-found", err)
	}
}

func TestIdempotencyLedger(t *testing.T) {
	s := NewServer()
	if _, dup := s.checkIdem("k1", fp("A")); dup {
		t.Fatal("unseen key reported dup")
	}
	s.storeIdem("k1", fp("A"), "s1")
	e, dup := s.checkIdem("k1", fp("A"))
	if !dup || e.done || e.sessionID != "s1" {
		t.Fatalf("stored entry wrong: %+v %v", e, dup)
	}
	if _, dup := s.checkIdem("k1", fp("B")); dup {
		t.Fatal("cross-owner key visible")
	}
	s.finishIdem("s1", "completed")
	e, dup = s.checkIdem("k1", fp("A"))
	if !dup || !e.done || e.status != "completed" {
		t.Fatalf("finished entry wrong: %+v %v", e, dup)
	}
}

func TestMultiTokenAuth(t *testing.T) {
	t.Setenv("API_AUTH_TOKEN", "")
	t.Setenv("API_AUTH_TOKENS", "tok-A, tok-B")
	s := NewServer()
	for _, tok := range []string{"tok-A", "tok-B"} {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("token %s: got %d", tok, w.Code)
		}
	}
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	req.Header.Set("Authorization", "Bearer nope")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("bad token: got %d, want 401", w.Code)
	}
}

func TestSSEUnknownSession404(t *testing.T) {
	t.Setenv("API_AUTH_TOKEN", "tok-A")
	t.Setenv("API_AUTH_TOKENS", "")
	s := NewServer()
	req := httptest.NewRequest("GET", "/api/v1/stream/ghost", nil)
	req.Header.Set("Authorization", "Bearer tok-A")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown stream: got %d, want 404", w.Code)
	}
}

func TestSSECrossOwner404(t *testing.T) {
	t.Setenv("API_AUTH_TOKENS", "tok-A, tok-B")
	t.Setenv("API_AUTH_TOKEN", "")
	s := NewServer()
	if err := s.persistSessionStart("sess-x", fp("tok-A")); err != nil {
		t.Fatalf("persist: %v", err)
	}
	req := httptest.NewRequest("GET", "/api/v1/stream/sess-x", nil)
	req.Header.Set("Authorization", "Bearer tok-B")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("cross-owner stream: got %d, want 404", w.Code)
	}
}

func TestPassSSEEvent(t *testing.T) {
	if !passSSEEvent("s1", events.Event{SessionID: "s1"}) {
		t.Fatal("own session dropped")
	}
	if passSSEEvent("s1", events.Event{SessionID: "s2"}) {
		t.Fatal("foreign session passed")
	}
	if passSSEEvent("s1", events.Event{}) {
		t.Fatal("untagged event passed")
	}
}
