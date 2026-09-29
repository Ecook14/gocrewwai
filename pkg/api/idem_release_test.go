package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/crew"
)

// failingCheckpointStore always errors on Save (used to force the
// persistSessionStart error path while keeping the test hermetic).
type failingCheckpointStore struct{}

func (failingCheckpointStore) Save(_ context.Context, _ *crew.Checkpoint) error {
	return errors.New("simulated persist failure")
}
func (failingCheckpointStore) LoadLatest(_ context.Context, _ string) (*crew.Checkpoint, error) {
	return nil, nil
}
func (failingCheckpointStore) LoadByID(_ context.Context, _ string, _ int64) (*crew.Checkpoint, error) {
	return nil, nil
}
func (failingCheckpointStore) ListCheckpoints(_ context.Context, _ string) ([]*crew.Checkpoint, error) {
	return nil, nil
}
func (failingCheckpointStore) Delete(_ context.Context, _ string, _ int64) error {
	return nil
}
func (failingCheckpointStore) Close() error { return nil }

// TestPersistSessionStartReturnsCheckpointError: persistSessionStart must
// surface checkpoint Save failures (was silently logging + returning nil).
func TestPersistSessionStartReturnsCheckpointError(t *testing.T) {
	t.Setenv("API_AUTH_TOKEN", "tok-test")
	t.Setenv("API_AUTH_TOKENS", "")
	t.Setenv("ALLOW_INSECURE_DEV", "1")

	s := NewServer()
	s.checkpointStore = failingCheckpointStore{}

	err := s.persistSessionStart("any-sess", "tok-test")
	if err == nil {
		t.Fatal("persistSessionStart must return error when checkpoint store Save fails")
	}
	// And the in-memory session must be rolled back so a stale map entry
	// isn't left behind when the caller aborts.
	s.mu.RLock()
	_, leaked := s.sessions["any-sess"]
	s.mu.RUnlock()
	if leaked {
		t.Error("in-memory session entry leaked after persist failure (should be deleted)")
	}
}

// TestKickoff_PersistFailureReleasesIdem: when checkpoint persistence fails
// (the only path that returns error from persistSessionStart), the
// Idempotency-Key reservation must be released so a retry succeeds.
func TestKickoff_PersistFailureReleasesIdem(t *testing.T) {
	t.Setenv("API_AUTH_TOKEN", "tok-test")
	t.Setenv("API_AUTH_TOKENS", "")
	t.Setenv("ALLOW_INSECURE_DEV", "1")
	t.Setenv("OPENAI_API_KEY", "fake-test-key")

	// Defensive: the retry below carries agent_model, so a provider client
	// gets constructed. Stub it so no real endpoint is ever contacted.
	_, restore := stubLLMClient(t)
	defer restore()

	s := NewServer()
	s.checkpointStore = failingCheckpointStore{}

	idemKey := "persist-failure-test"
	body := `{"session_id":"persist-fail-session","agent_role":"r","agent_goal":"g","agent_backstory":"b","agent_model":"gpt-4o","task_description":"x"}`

	req := httptest.NewRequest("POST", "/api/v1/crews/kickoff", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok-test")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idemKey)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from forced persist failure, got %d: %s", rec.Code, rec.Body.String())
	}

	// Drain sem via Shutdown so the global isn't read concurrently with
	// the next test's kickoffSem swap.
	s.Shutdown()

	// Retry with a different session_id but the SAME Idempotency-Key.
	// Pre-fix: this returned 409 (leaked reservation).
	body2 := `{"session_id":"persist-fail-session-2","agent_role":"r","agent_goal":"g","agent_backstory":"b","agent_model":"gpt-4o","agent_api_key":"fake-test-key","task_description":"y"}`
	req2 := httptest.NewRequest("POST", "/api/v1/crews/kickoff", strings.NewReader(body2))
	req2.Header.Set("Authorization", "Bearer tok-test")
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", idemKey)
	rec2 := httptest.NewRecorder()
	s.router.ServeHTTP(rec2, req2)
	if rec2.Code == http.StatusConflict {
		t.Fatalf("Idempotency-Key leaked past rejection: %s", rec2.Body.String())
	}
	// We expect a second 500 (persist still fails), not 409. The point is
	// that the ledger no longer holds the reservation.
	if rec2.Code != http.StatusInternalServerError {
		t.Fatalf("expected retry to surface the same persist failure (500), got %d", rec2.Code)
	}
}
