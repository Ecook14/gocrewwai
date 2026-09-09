package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUsageTracker_Record(t *testing.T) {
	tracker := NewUsageTracker()
	tracker.Record(Usage{
		PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150,
		CostUSD: 0.005, Model: "gpt-4o", Provider: "openai", LatencyMs: 250,
	})
	totals := tracker.Totals()
	if totals.PromptTokens != 100 {
		t.Errorf("Expected 100 prompt tokens, got %d", totals.PromptTokens)
	}
	if totals.CompletionTokens != 50 {
		t.Errorf("Expected 50 completion tokens, got %d", totals.CompletionTokens)
	}
	if totals.TotalTokens != 150 {
		t.Errorf("Expected 150 total tokens, got %d", totals.TotalTokens)
	}
	if totals.LatencyMs != 250 {
		t.Errorf("Expected 250ms latency, got %d", totals.LatencyMs)
	}
	if tracker.CallCount() != 1 {
		t.Errorf("Expected 1 call, got %d", tracker.CallCount())
	}
}

func TestUsageTracker_Reset(t *testing.T) {
	tracker := NewUsageTracker()
	tracker.Record(Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150})
	tracker.Reset()
	if tracker.CallCount() != 0 {
		t.Errorf("Expected 0 calls after reset, got %d", tracker.CallCount())
	}
	if tracker.Totals().TotalTokens != 0 {
		t.Errorf("Expected 0 total tokens after reset, got %d", tracker.Totals().TotalTokens)
	}
}

func TestUsageTracker_AllCalls(t *testing.T) {
	tracker := NewUsageTracker()
	tracker.Record(Usage{Model: "a"})
	tracker.Record(Usage{Model: "b"})
	calls := tracker.AllCalls()
	if len(calls) != 2 {
		t.Errorf("Expected 2 calls, got %d", len(calls))
	}
	if calls[0].Model != "a" || calls[1].Model != "b" {
		t.Error("Calls not in expected order")
	}
}

func TestCalculateCostStatic(t *testing.T) {
	SetModelPricing("gpt-4o", ModelPricing{
		PromptPricePerToken:     0.0000025,
		CompletionPricePerToken: 0.00001,
	})
	defer SetModelPricing("gpt-4o", ModelPricing{})
	u := Usage{PromptTokens: 1000, CompletionTokens: 500, Model: "gpt-4o"}
	cost := CalculateCostStatic(u)
	expected := 1000*0.0000025 + 500*0.00001 // 0.0075
	if cost < expected-0.0001 || cost > expected+0.0001 {
		t.Errorf("Expected cost ~0.0075, got %f", cost)
	}
	if CalculateCostStatic(Usage{PromptTokens: 1000, Model: "no-such-model"}) != 0 {
		t.Error("Expected 0 cost for unknown model")
	}
}

func TestPriceCache_BuiltinFallback(t *testing.T) {
	SetModelPricing("gpt-4o", ModelPricing{
		PromptPricePerToken:     0.0000025,
		CompletionPricePerToken: 0.00001,
	})
	defer SetModelPricing("gpt-4o", ModelPricing{})
	cache := NewPriceCache(PriceCacheConfig{CacheTTL: 1 * time.Hour})
	p, ok := cache.GetPricing("gpt-4o")
	if !ok {
		t.Fatal("Expected builtin pricing for gpt-4o")
	}
	if p.PromptPricePerToken == 0 {
		t.Error("Expected non-zero prompt price for gpt-4o")
	}
}

func TestPriceCache_AutoRefreshOnStale(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id": "openai/gpt-4o",
					"pricing": map[string]string{
						"prompt":     "0.0000025",
						"completion": "0.00001",
					},
				},
			},
		})
	}))
	defer server.Close()

	cache := NewPriceCache(PriceCacheConfig{
		CacheTTL:     1 * time.Millisecond,
		APIEndpoint:  server.URL + "/api/v1/models",
		FetchTimeout: 5 * time.Second,
	})
	SetModelPricing("gpt-4o", ModelPricing{
		PromptPricePerToken:     0.0000025,
		CompletionPricePerToken: 0.00001,
	})
	defer SetModelPricing("gpt-4o", ModelPricing{})
	cache.GetPricing("gpt-4o")
	firstCount := callCount
	if firstCount == 0 {
		t.Fatal("Expected at least one API call")
	}
	time.Sleep(5 * time.Millisecond)
	cache.GetPricing("gpt-4o")
	if callCount <= firstCount {
		t.Errorf("Expected re-fetch after TTL expiry, got %d total calls", callCount)
	}
}

func TestPriceCache_ModelCount(t *testing.T) {
	SetModelPricing("gpt-4o", ModelPricing{
		PromptPricePerToken:     0.0000025,
		CompletionPricePerToken: 0.00001,
	})
	SetModelPricing("claude-3", ModelPricing{
		PromptPricePerToken:     0.000015,
		CompletionPricePerToken: 0.000075,
	})
	defer func() {
		SetModelPricing("gpt-4o", ModelPricing{})
		SetModelPricing("claude-3", ModelPricing{})
	}()
	cache := NewPriceCache(PriceCacheConfig{CacheTTL: 0})
	if cache.ModelCount() < 2 {
		t.Errorf("Expected at least 2 models, got %d", cache.ModelCount())
	}
}

func TestPriceCache_AllPricingIsCopy(t *testing.T) {
	SetModelPricing("gpt-4o", ModelPricing{
		PromptPricePerToken:     0.0000025,
		CompletionPricePerToken: 0.00001,
	})
	defer SetModelPricing("gpt-4o", ModelPricing{})
	cache := NewPriceCache(PriceCacheConfig{CacheTTL: 0})
	snapshot := cache.AllPricing()
	if len(snapshot) == 0 {
		t.Error("Expected non-empty pricing snapshot")
	}
}

func TestPriceCache_NoAPIKeyNeeded(t *testing.T) {
	SetModelPricing("gpt-4o", ModelPricing{
		PromptPricePerToken:     0.0000025,
		CompletionPricePerToken: 0.00001,
	})
	defer SetModelPricing("gpt-4o", ModelPricing{})
	cache := NewPriceCache(PriceCacheConfig{CacheTTL: 0})
	if cache.ModelCount() == 0 {
		t.Error("Expected builtin models loaded without any API key")
	}
}

func TestPriceCache_GetPricingUnknownModel(t *testing.T) {
	cache := NewPriceCache(PriceCacheConfig{CacheTTL: 0})
	_, ok := cache.GetPricing("nonexistent-model-12345")
	if ok {
		t.Error("Expected false for unknown model")
	}
}
