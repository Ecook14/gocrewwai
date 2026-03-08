package telemetry

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetrics_RecordLLMCall(t *testing.T) {
	m := NewMetrics()
	m.RecordLLMCall("gpt-4o", 150*time.Millisecond, nil)
	m.RecordLLMCall("gpt-4o", 250*time.Millisecond, nil)
	m.RecordLLMCall("claude-3", 100*time.Millisecond, io.ErrUnexpectedEOF)

	snap := m.Snapshot()
	if snap.LLMCallsTotal["gpt-4o"] != 2 {
		t.Errorf("Expected 2 gpt-4o calls, got %d", snap.LLMCallsTotal["gpt-4o"])
	}
	if snap.LLMCallsTotal["claude-3"] != 1 {
		t.Errorf("Expected 1 claude-3 call, got %d", snap.LLMCallsTotal["claude-3"])
	}
}

func TestMetrics_RecordTokens(t *testing.T) {
	m := NewMetrics()
	m.RecordTokens("gpt-4o", 100, 50)
	m.RecordTokens("gpt-4o", 200, 75)

	snap := m.Snapshot()
	if snap.PromptTokensTotal["gpt-4o"] != 300 {
		t.Errorf("Expected 300 prompt tokens, got %d", snap.PromptTokensTotal["gpt-4o"])
	}
	if snap.CompletionTokensTotal["gpt-4o"] != 125 {
		t.Errorf("Expected 125 completion tokens, got %d", snap.CompletionTokensTotal["gpt-4o"])
	}
}

func TestMetrics_ActiveAgents(t *testing.T) {
	m := NewMetrics()
	m.AgentStarted()
	m.AgentStarted()
	m.AgentStopped()

	snap := m.Snapshot()
	if snap.ActiveAgents != 1 {
		t.Errorf("Expected 1 active agent, got %d", snap.ActiveAgents)
	}
	if snap.AgentsCreated != 2 {
		t.Errorf("Expected 2 created agents, got %d", snap.AgentsCreated)
	}
}

func TestMetrics_Handler(t *testing.T) {
	m := NewMetrics()
	m.RecordLLMCall("gpt-4o", 100*time.Millisecond, nil)
	m.RecordTokens("gpt-4o", 50, 25)
	m.RecordCost("gpt-4o", 0.001)
	m.RecordTaskExecution("research", time.Second, nil)
	m.RecordToolCall("search", 500*time.Millisecond, nil)
	m.AgentStarted()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)

	body := w.Body.String()
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	if !strings.Contains(body, "crew_go_llm_calls_total") {
		t.Error("Expected LLM calls metric")
	}
	if !strings.Contains(body, "crew_go_prompt_tokens_total") {
		t.Error("Expected token metric")
	}
	if !strings.Contains(body, "crew_go_active_agents") {
		t.Error("Expected active agents metric")
	}
	if !strings.Contains(body, "crew_go_cost_usd_total") {
		t.Error("Expected cost metric")
	}
}

func TestMetrics_Snapshot(t *testing.T) {
	m := NewMetrics()
	m.RecordCost("gpt-4o", 0.05)
	snap := m.Snapshot()

	if snap.CostTotalUSD["gpt-4o"] != 0.05 {
		t.Errorf("Expected 0.05 cost, got %f", snap.CostTotalUSD["gpt-4o"])
	}
	if snap.UptimeSeconds < 0 {
		t.Error("Uptime should be positive")
	}
}

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LogConfig{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: &buf,
	})

	logger.Info("test message", slog.String("key", "value"))

	if !strings.Contains(buf.String(), "test message") {
		t.Error("Expected log output to contain 'test message'")
	}
}

func TestAgentLogger(t *testing.T) {
	var buf bytes.Buffer
	parent := NewLogger(LogConfig{Format: "json", Output: &buf})
	agentLog := AgentLogger(parent, "researcher", "find papers")

	agentLog.Info("working")
	output := buf.String()
	if !strings.Contains(output, "researcher") {
		t.Error("Expected agent_role in log")
	}
}

func TestAuditLogger(t *testing.T) {
	var buf bytes.Buffer
	audit := NewAuditLogger(&buf, 100)

	err := audit.Log(AuditEntry{
		EventType: "llm_call",
		AgentRole: "researcher",
		Action:    "Generate",
		Input:     "test prompt",
		Output:    "test response",
		Success:   true,
	})
	if err != nil {
		t.Fatalf("Audit log failed: %v", err)
	}

	var entry AuditEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse audit entry: %v", err)
	}
	if entry.EventType != "llm_call" {
		t.Errorf("Expected event_type 'llm_call', got %s", entry.EventType)
	}
}

func TestAuditLogger_Truncation(t *testing.T) {
	var buf bytes.Buffer
	audit := NewAuditLogger(&buf, 10)

	_ = audit.Log(AuditEntry{
		Input: "this is a very long input string that should be truncated",
	})

	if !strings.Contains(buf.String(), "truncated") {
		t.Error("Expected input to be truncated")
	}
}

func TestGlobalMetrics(t *testing.T) {
	m1 := GlobalMetrics()
	m2 := GlobalMetrics()
	if m1 != m2 {
		t.Error("GlobalMetrics should return singleton")
	}
}
