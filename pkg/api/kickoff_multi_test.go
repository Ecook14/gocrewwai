package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// Multi-agent kickoff: the backend engine (crew.NewCrew) has always accepted
// N agents and M tasks, but POST /api/v1/crews/kickoff only exposed one of
// each. These tests pin the extended contract: optional agents[]/tasks[]
// arrays alongside the unchanged flat single-agent fields.

func multiTestServer(t *testing.T) *Server {
	t.Helper()
	t.Setenv("API_AUTH_TOKEN", "tok-test")
	t.Setenv("API_AUTH_TOKENS", "")
	t.Setenv("ALLOW_INSECURE_DEV", "1")
	restore := stubLLMClient(t)
	t.Cleanup(restore)
	s := NewServer()
	t.Cleanup(s.Shutdown)
	return s
}

func postKickoff(t *testing.T, s *Server, body string, idemKey string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/v1/crews/kickoff", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok-test")
	req.Header.Set("Content-Type", "application/json")
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)
	return rec
}

func TestKickoff_MultiAgent_TwoAgentsTwoTasks(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "fake-test-key")
	s := multiTestServer(t)
	body := `{"session_id":"multi-1","crew_process":"sequential",
		"agents":[
			{"role":"Researcher","goal":"gather","backstory":"b","model":"gpt-4o"},
			{"role":"Writer","goal":"write","backstory":"b","model":"gpt-4o"}],
		"tasks":[
			{"description":"research x","agent_role":"Researcher"},
			{"description":"write about x","agent_role":"Writer","expected_output":"an article"}]}`
	rec := postKickoff(t, s, body, "multi-key-1")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["session_id"] != "multi-1" {
		t.Fatalf("session_id = %q", resp["session_id"])
	}
}

func TestKickoff_MultiAgent_TaskUnknownAgent(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "fake-test-key")
	s := multiTestServer(t)
	body := `{"session_id":"multi-2",
		"agents":[{"role":"Researcher","model":"gpt-4o"}],
		"tasks":[{"description":"do x","agent_role":"Ghost"}]}`
	rec := postKickoff(t, s, body, "multi-key-2")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKickoff_MultiAgent_TasksWithoutAgents(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "fake-test-key")
	s := multiTestServer(t)
	body := `{"session_id":"multi-3",
		"tasks":[{"description":"do x","agent_role":"Nobody"}]}`
	rec := postKickoff(t, s, body, "multi-key-3")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKickoff_MultiAgent_FlatPlusArrays(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "fake-test-key")
	s := multiTestServer(t)
	body := `{"session_id":"multi-4","agent_role":"R","task_description":"t",
		"agents":[{"role":"R","model":"gpt-4o"}],
		"tasks":[{"description":"t","agent_role":"R"}]}`
	rec := postKickoff(t, s, body, "multi-key-4")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for ambiguous flat+arrays, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKickoff_MultiAgent_TooManyAgents(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "fake-test-key")
	s := multiTestServer(t)
	var sb strings.Builder
	sb.WriteString(`{"session_id":"multi-5","agents":[`)
	for i := 0; i < 11; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `{"role":"R%d","model":"gpt-4o"}`, i)
	}
	sb.WriteString(`],"tasks":[{"description":"t","agent_role":"R0"}]}`)
	rec := postKickoff(t, s, sb.String(), "multi-key-5")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for 11 agents, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKickoff_MultiAgent_TooManyTasks(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "fake-test-key")
	s := multiTestServer(t)
	var sb strings.Builder
	sb.WriteString(`{"session_id":"multi-6","agents":[{"role":"R","model":"gpt-4o"}],"tasks":[`)
	for i := 0; i < 33; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `{"description":"task %d","agent_role":"R"}`, i)
	}
	sb.WriteString(`]}`)
	rec := postKickoff(t, s, sb.String(), "multi-key-6")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for 33 tasks, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKickoff_MultiAgent_ModelWithoutKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	s := multiTestServer(t)
	body := `{"session_id":"multi-7",
		"agents":[{"role":"R","model":"gpt-4o"}],
		"tasks":[{"description":"t","agent_role":"R"}]}`
	rec := postKickoff(t, s, body, "multi-key-7")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKickoff_MultiAgent_AmbiguousTaskNoRole(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "fake-test-key")
	s := multiTestServer(t)
	body := `{"session_id":"multi-8",
		"agents":[{"role":"A","model":"gpt-4o"},{"role":"B","model":"gpt-4o"}],
		"tasks":[{"description":"t"}]}`
	rec := postKickoff(t, s, body, "multi-key-8")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for role-less task with 2 agents, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKickoff_MultiAgent_SingleAgentDefaultWiring(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "fake-test-key")
	s := multiTestServer(t)
	body := `{"session_id":"multi-9",
		"agents":[{"role":"Solo","model":"gpt-4o"}],
		"tasks":[{"description":"t"}]}`
	rec := postKickoff(t, s, body, "multi-key-9")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKickoff_MultiAgent_DuplicateRoles(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "fake-test-key")
	s := multiTestServer(t)
	body := `{"session_id":"multi-11",
		"agents":[{"role":"R","model":"gpt-4o"},{"role":"R","model":"gpt-4o"}],
		"tasks":[{"description":"t","agent_role":"R"}]}`
	rec := postKickoff(t, s, body, "multi-key-11")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for duplicate roles, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKickoff_MultiAgent_PerAgentKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	s := multiTestServer(t)
	body := `{"session_id":"multi-10",
		"agents":[{"role":"R","model":"gpt-4o","api_key":"per-agent-key"}],
		"tasks":[{"description":"t","agent_role":"R"}]}`
	rec := postKickoff(t, s, body, "multi-key-10")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 with per-agent key, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestKickoff_MultiAgent_ConcurrentSameKey: N concurrent multi-agent
// kickoffs sharing one Idempotency-Key but naming different sessions must
// collapse onto exactly one execution. Losers see 409 while the winner is
// still running, or 200 with the stored result if the winner already
// finished (the stub client completes fast, so both occur); in all cases
// every response must reference the winner's session, proving no duplicate
// crew ever executed.
func TestKickoff_MultiAgent_ConcurrentSameKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "fake-test-key")
	s := multiTestServer(t)

	const n = 8
	var wg sync.WaitGroup
	type outcome struct {
		code int
		body string
	}
	outcomes := make([]outcome, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			body := fmt.Sprintf(`{"session_id":"multi-race-%d",
				"agents":[{"role":"R%d","model":"gpt-4o"},{"role":"W%d","model":"gpt-4o"}],
				"tasks":[{"description":"t","agent_role":"R%d"}]}`, idx, idx, idx, idx)
			rec := postKickoff(t, s, body, "multi-race-key")
			outcomes[idx] = outcome{rec.Code, rec.Body.String()}
		}(i)
	}
	wg.Wait()

	var winner string
	accepted := 0
	for _, o := range outcomes {
		if o.code != http.StatusAccepted {
			continue
		}
		accepted++
		var resp map[string]any
		if err := json.Unmarshal([]byte(o.body), &resp); err != nil {
			t.Fatalf("decode 202 body: %v", err)
		}
		sid, _ := resp["session_id"].(string)
		winner = sid
	}
	if accepted != 1 {
		t.Fatalf("expected exactly 1 accepted multi run, got %d", accepted)
	}
	for i, o := range outcomes {
		if o.code == http.StatusAccepted {
			continue
		}
		if o.code != http.StatusConflict && o.code != http.StatusOK {
			t.Fatalf("request %d: unexpected status %d, want 202, 409, or 200-replay", i, o.code)
		}
		var resp map[string]any
		if err := json.Unmarshal([]byte(o.body), &resp); err != nil {
			t.Fatalf("request %d: decode %d body: %v", i, o.code, err)
		}
		// 409 carries the running session; 200 carries the stored result.
		// Both must name the winner's session.
		sid, _ := resp["session_id"].(string)
		if sid != winner {
			t.Fatalf("request %d: collapsed onto %q, want winner %q (duplicate execution)",
				i, sid, winner)
		}
	}
}
