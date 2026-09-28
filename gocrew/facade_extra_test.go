package gocrew

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/webhook"
)

// TestFacadeExtra_Coverage pins every extended-facade constructor to its pkg
// implementation. If a pkg symbol is renamed, this fails instead of drifting.
func TestFacadeExtra_Coverage(t *testing.T) {
	if m := NewMetrics(); m == nil {
		t.Error("NewMetrics nil")
	}
	if cfg := DefaultTelemetryConfig(); cfg.ServiceName == "" && !cfg.Enabled {
		t.Error("DefaultTelemetryConfig zero")
	}
	if c := NewMCPClient("https://example.com/mcp"); c == nil {
		t.Error("NewMCPClient nil")
	}
	if c := NewA2AClient("tok"); c == nil {
		t.Error("NewA2AClient nil")
	}
	if b := NewAgentCardBuilder(); b == nil {
		t.Error("NewAgentCardBuilder nil")
	}
	if m := NewCrewCheckpointManager(t.TempDir()); m == nil {
		t.Error("NewCrewCheckpointManager nil")
	}
	if s := NewConversationStore(); s == nil {
		t.Error("NewConversationStore nil")
	}
	if s := NewEntityStore(); s == nil {
		t.Error("NewEntityStore nil")
	}
	if sp := NewTokenSplitter(500, 50); sp == nil {
		t.Error("NewTokenSplitter nil")
	}
	if g := NewJSONGuardrail(); g == nil {
		t.Error("NewJSONGuardrail nil")
	}
	if g := NewXMLGuardrail("root"); g == nil {
		t.Error("NewXMLGuardrail nil")
	}
	if g := NewCSVGuardrail(2); g == nil {
		t.Error("NewCSVGuardrail nil")
	}
	if _, err := NewRegexGuardrail(`^x`); err != nil {
		t.Errorf("NewRegexGuardrail: %v", err)
	}
	if ks := NewKnowledgeSearchTool(nil, nil); ks == nil {
		t.Error("NewKnowledgeSearchTool nil")
	}
	if gf := NewGraphFlow("g", nil); gf == nil {
		t.Error("NewGraphFlow nil")
	}
	if v := NewJWTValidator("s"); v == nil {
		t.Error("NewJWTValidator nil")
	}
	os.Setenv("GOCREW_ALLOW_PRIVATE_URLS", "1")
	defer os.Unsetenv("GOCREW_ALLOW_PRIVATE_URLS")
	if n, err := NewWebhookNotifier("http://127.0.0.1:9/hook", "s"); err != nil || n == nil {
		t.Errorf("NewWebhookNotifier: %v", err)
	}
	if err := VerifyWebhookSignature("s", []byte("b"), "sha256="+webhook.Sign("s", []byte("b")), "", 0); err != nil {
		t.Errorf("VerifyWebhookSignature: %v", err)
	}
	if s := DefaultInputSanitizer(); s == nil {
		t.Error("DefaultInputSanitizer nil")
	}
	if e := NewFlowEngine(nil); e == nil {
		t.Error("NewFlowEngine nil")
	}
	if langs := SupportedLanguages(); len(langs) < 2 {
		t.Errorf("SupportedLanguages = %v, want en+es", langs)
	}
	if u := NewLLMUsageTracker(); u == nil {
		t.Error("NewLLMUsageTracker nil")
	}
	wrapped := WrapLLMClient(NewOpenAI("k", "m"),
		LLMWithRateLimit(10, time.Minute),
		LLMWithTimeout(time.Second),
		LLMWithMaxRetries(2),
		LLMWithCircuitBreaker(3, time.Second))
	if wrapped == nil {
		t.Error("WrapLLMClient nil")
	}
	traced := NewTracedLLMClient("test", NewOpenAI("k", "m"))
	if traced == nil || traced.Unwrap() == nil {
		t.Error("NewTracedLLMClient nil")
	}
	ctx, end := StartSpan(context.Background(), "test")
	end()
	_ = ctx
}
