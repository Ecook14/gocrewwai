package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestKickoff_SemSaturationReleasesIdem: when kickoffSem is saturated, the
// Idempotency-Key reservation must be released so a follow-up retry with
// the same key (after the queue drains) is accepted. Pre-fix this fails
// because the 429 path never called releaseIdem.
func TestKickoff_SemSaturationReleasesIdem(t *testing.T) {
	t.Setenv("API_AUTH_TOKEN", "tok-test")
	t.Setenv("API_AUTH_TOKENS", "")
	t.Setenv("ALLOW_INSECURE_DEV", "1")
	t.Setenv("OPENAI_API_KEY", "fake-test-key")

	// Stub the provider so the accepted (202) kickoff never makes a real
	// network call. Without this the crew goroutine dials api.openai.com,
	// gets a 401, and retries — slow, flaky, and outbound traffic in CI.
	restore := stubLLMClient(t)
	defer restore()

	// Saturate kickoffSem under the production mutex so the in-flight
	// goroutine from the rejected request observes the swap consistently.
	prevSem := kickoffSem
	kickoffSemMu.Lock()
	kickoffSem = make(chan struct{}, 1)
	kickoffSem <- struct{}{}
	kickoffSemMu.Unlock()

	s := NewServer()
	t.Cleanup(func() {
		// Wait for the server to drain in-flight kickoff goroutines so
		// the global kickoffSem is no longer being read; otherwise the
		// next test's handleKickoff may observe a torn variable.
		s.Shutdown()
		kickoffSemMu.Lock()
		kickoffSem = prevSem
		kickoffSemMu.Unlock()
	})

	// First call: rejected with 429 (sem saturated).
	body := `{"session_id":"sess-A","agent_role":"r","agent_goal":"g","agent_backstory":"b","agent_model":"gpt-4o","task_description":"x"}`
	idemKey := "saturation-test-key"
	req := httptest.NewRequest("POST", "/api/v1/crews/kickoff", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok-test")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idemKey)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on saturated sem, got %d", rec.Code)
	}

	// Drain the slot we filled so the next request is accepted. Read the
	// global under the lock so we take from the same channel handleKickoff
	// acquires from.
	kickoffSemMu.RLock()
	sem := kickoffSem
	kickoffSemMu.RUnlock()
	<-sem

	// Second call: SAME Idempotency-Key must be accepted now (reservation released).
	body2 := `{"session_id":"sess-B","agent_role":"r","agent_goal":"g","agent_backstory":"b","agent_model":"gpt-4o","task_description":"y"}`
	req2 := httptest.NewRequest("POST", "/api/v1/crews/kickoff", strings.NewReader(body2))
	req2.Header.Set("Authorization", "Bearer tok-test")
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", idemKey)
	rec2 := httptest.NewRecorder()
	s.router.ServeHTTP(rec2, req2)
	if rec2.Code == http.StatusConflict {
		t.Fatalf("Idempotency-Key leaked across 429: replay returned 409 (%s)", rec2.Body.String())
	}
	if rec2.Code != http.StatusAccepted {
		t.Fatalf("expected 202 after sem drained, got %d (%s)", rec2.Code, rec2.Body.String())
	}
}
