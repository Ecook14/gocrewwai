package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/testutil"
)

// Field-application kickoff tests: several accepted request fields were
// silently dropped on the floor (accepted by binding, never used). These
// tests prove each field reaches the execution it describes by inspecting
// the prompts recorded by the stub LLM after the run completes.

// fieldTestServer builds an authenticated server with a stub provider and
// returns the mock so tests can assert on the prompts the crew saw.
func fieldTestServer(t *testing.T) (*Server, *testutil.MockClient) {
	t.Helper()
	t.Setenv("API_AUTH_TOKEN", "tok-test")
	t.Setenv("API_AUTH_TOKENS", "")
	t.Setenv("ALLOW_INSECURE_DEV", "1")
	t.Setenv("OPENAI_API_KEY", "fake-test-key")
	mock, restore := stubLLMClient(t)
	t.Cleanup(restore)
	s := NewServer()
	t.Cleanup(s.Shutdown)
	return s, mock
}

// recordedPrompts concatenates every message content the stub saw across
// Generate and GenerateWithUsage calls.
func recordedPrompts(mock *testutil.MockClient) string {
	var sb strings.Builder
	calls := append(mock.CallsForMethod("Generate"), mock.CallsForMethod("GenerateWithUsage")...)
	for _, c := range calls {
		for _, msg := range c.Messages {
			sb.WriteString("[" + msg.Role + "] " + msg.Content + "\n")
		}
	}
	return sb.String()
}

// drainAndRead shuts the server down (waiting for the background crew) and
// returns the session status.
func drainAndRead(t *testing.T, s *Server, sessionID string) string {
	t.Helper()
	s.Shutdown()
	req := httptest.NewRequest("GET", "/api/v1/sessions/"+sessionID, nil)
	req.Header.Set("Authorization", "Bearer tok-test")
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("session read: got %d: %s", rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func TestKickoff_FlatExpectedOutputInPrompt(t *testing.T) {
	s, mock := fieldTestServer(t)
	body := `{"session_id":"fields-1","agent_role":"R","agent_goal":"g",
		"agent_backstory":"b","agent_model":"gpt-4o","task_description":"do the thing",
		"task_expected_output":"UNIQUE_EXPECTED_MARKER_123"}`
	rec := postKickoff(t, s, body, "fields-key-1")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	drainAndRead(t, s, "fields-1")
	if got := recordedPrompts(mock); !strings.Contains(got, "UNIQUE_EXPECTED_MARKER_123") {
		t.Fatalf("task_expected_output never reached the LLM prompt:\n%s", got)
	}
}

func TestKickoff_FlatSystemPromptInMessages(t *testing.T) {
	s, mock := fieldTestServer(t)
	body := `{"session_id":"fields-2","agent_role":"R","agent_goal":"g",
		"agent_backstory":"b","agent_model":"gpt-4o","task_description":"do the thing",
		"agent_system_prompt":"UNIQUE_SYSTEM_MARKER_456"}`
	rec := postKickoff(t, s, body, "fields-key-2")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	drainAndRead(t, s, "fields-2")
	got := recordedPrompts(mock)
	if !strings.Contains(got, "UNIQUE_SYSTEM_MARKER_456") {
		t.Fatalf("agent_system_prompt never reached the LLM messages:\n%s", got)
	}
	if !strings.Contains(got, "[system]") {
		t.Fatalf("expected a system-role message, got:\n%s", got)
	}
}

func TestKickoff_FlatToolsJSONTool(t *testing.T) {
	s, _ := fieldTestServer(t)
	body := `{"session_id":"fields-3","agent_role":"R","agent_goal":"g",
		"agent_backstory":"b","agent_model":"gpt-4o","task_description":"do the thing",
		"task_tools":["JSONTool"]}`
	rec := postKickoff(t, s, body, "fields-key-3")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKickoff_FlatToolsShellRejected(t *testing.T) {
	s, _ := fieldTestServer(t)
	body := `{"session_id":"fields-4","agent_role":"R","agent_goal":"g",
		"agent_backstory":"b","agent_model":"gpt-4o","task_description":"do the thing",
		"task_tools":["ShellTool"]}`
	rec := postKickoff(t, s, body, "fields-key-4")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for review-gated ShellTool, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKickoff_FlatToolsUnknownRejected(t *testing.T) {
	s, _ := fieldTestServer(t)
	body := `{"session_id":"fields-5","agent_role":"R","agent_goal":"g",
		"agent_backstory":"b","agent_model":"gpt-4o","task_description":"do the thing",
		"task_tools":["NoSuchTool"]}`
	rec := postKickoff(t, s, body, "fields-key-5")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown tool, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKickoff_MultiToolsPromptExpected(t *testing.T) {
	s, mock := fieldTestServer(t)
	body := `{"session_id":"fields-6",
		"agents":[{"role":"R","goal":"g","backstory":"b","model":"gpt-4o",
			"system_prompt":"UNIQUE_MULTI_SYSTEM_789"}],
		"tasks":[{"description":"do the thing","agent_role":"R",
			"expected_output":"UNIQUE_MULTI_EXPECTED_012","tools":["RegexTool"]}]}`
	rec := postKickoff(t, s, body, "fields-key-6")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	drainAndRead(t, s, "fields-6")
	got := recordedPrompts(mock)
	for _, marker := range []string{"UNIQUE_MULTI_SYSTEM_789", "UNIQUE_MULTI_EXPECTED_012"} {
		if !strings.Contains(got, marker) {
			t.Fatalf("marker %q missing from prompts:\n%s", marker, got)
		}
	}
}

func TestKickoff_StateMachineHyphen(t *testing.T) {
	s, _ := fieldTestServer(t)
	body := `{"session_id":"fields-7","agent_role":"R","agent_goal":"g",
		"agent_backstory":"b","agent_model":"gpt-4o","task_description":"do the thing",
		"crew_process":"state-machine"}`
	rec := postKickoff(t, s, body, "fields-key-7")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	got := drainAndRead(t, s, "fields-7")
	if !strings.Contains(got, `"completed"`) {
		t.Fatalf("state-machine run did not complete: %s", got)
	}
}

func TestResolveTaskTools(t *testing.T) {
	got, apiErr := resolveTaskTools([]string{"JSONTool", "RegexTool"})
	if apiErr != nil {
		t.Fatalf("safe tools rejected: %s", apiErr.message)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(got))
	}
	if _, apiErr := resolveTaskTools(nil); apiErr != nil {
		t.Fatalf("empty tools rejected: %s", apiErr.message)
	}
	for _, tc := range []struct {
		name  string
		tools []string
	}{
		{"review-gated shell", []string{"ShellTool"}},
		{"review-gated file write", []string{"FileWriteTool"}},
		{"review-gated http", []string{"HTTPTool"}},
		{"unknown", []string{"NoSuchTool"}},
		{"mixed safe and gated", []string{"JSONTool", "ShellTool"}},
	} {
		if _, apiErr := resolveTaskTools(tc.tools); apiErr == nil {
			t.Errorf("%s: expected rejection, got none", tc.name)
		} else if apiErr.status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", tc.name, apiErr.status)
		}
	}
}

// TestShutdown_Idempotent: Shutdown closes the shutdown channel, so calling
// it twice (explicit drain in a test plus Cleanup, or double SIGTERM in
// prod) must not panic with close-of-closed-channel.
func TestShutdown_Idempotent(t *testing.T) {
	t.Setenv("API_AUTH_TOKEN", "tok-test")
	t.Setenv("API_AUTH_TOKENS", "")
	s := NewServer()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("second Shutdown panicked: %v", r)
		}
	}()
	s.Shutdown()
	s.Shutdown()
}

// TestNormalizeCrewProcess pins the validation contract, including the
// hyphenated alias. Without the normalization branch, "state-machine" would
// pass validation and then fail at runtime with ErrUnsupportedProcess.
func TestNormalizeCrewProcess(t *testing.T) {
	for _, tc := range []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", "", false},
		{"sequential", "sequential", false},
		{"hierarchical", "hierarchical", false},
		{"consensual", "consensual", false},
		{"graph", "graph", false},
		{"reflective", "reflective", false},
		{"state_machine", "state_machine", false},
		{"state-machine", "state_machine", false},
		{"bogus", "", true},
		{"SEQUENTIAL", "", true},
	} {
		got, apiErr := normalizeCrewProcess(tc.in)
		if tc.wantErr {
			if apiErr == nil {
				t.Errorf("normalizeCrewProcess(%q): expected rejection, got %q", tc.in, got)
			} else if apiErr.status != http.StatusBadRequest {
				t.Errorf("normalizeCrewProcess(%q): status = %d, want 400", tc.in, apiErr.status)
			}
			continue
		}
		if apiErr != nil {
			t.Errorf("normalizeCrewProcess(%q): unexpected rejection: %s", tc.in, apiErr.message)
			continue
		}
		if got != tc.want {
			t.Errorf("normalizeCrewProcess(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestKickoff_RejectedRequestReleasesReservation: a request rejected during
// validation (400/503) must release its idempotency reservation, so a
// corrected retry with the same key starts fresh instead of hanging at 409.
// Pre-fix, every post-reserve rejection leaked the reservation.
func TestKickoff_RejectedRequestReleasesReservation(t *testing.T) {
	s, _ := fieldTestServer(t)
	const key = "fix-and-retry-key"

	bad := `{"session_id":"fields-retry-1","agent_role":"R","agent_goal":"g",
		"agent_backstory":"b","agent_model":"gpt-4o","task_description":"t",
		"task_tools":["ShellTool"]}`
	if rec := postKickoff(t, s, bad, key); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	good := `{"session_id":"fields-retry-2","agent_role":"R","agent_goal":"g",
		"agent_backstory":"b","agent_model":"gpt-4o","task_description":"t"}`
	if rec := postKickoff(t, s, good, key); rec.Code != http.StatusAccepted {
		t.Fatalf("retry with same key after 400: got %d, want 202: %s", rec.Code, rec.Body.String())
	}
}
