package llm

import (
	"context"
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
	tracker.Record(Usage{
		PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300,
		CostUSD: 0.010, Model: "gpt-4o", Provider: "openai", LatencyMs: 500,
	})

	if tracker.CallCount() != 2 {
		t.Errorf("Expected 2 calls, got %d", tracker.CallCount())
	}
	totals := tracker.Totals()
	if totals.PromptTokens != 300 {
		t.Errorf("Expected 300 prompt tokens, got %d", totals.PromptTokens)
	}
	if totals.CompletionTokens != 150 {
		t.Errorf("Expected 150 completion tokens, got %d", totals.CompletionTokens)
	}
	if totals.TotalTokens != 450 {
		t.Errorf("Expected 450 total tokens, got %d", totals.TotalTokens)
	}
	if totals.LatencyMs != 750 {
		t.Errorf("Expected 750ms latency, got %d", totals.LatencyMs)
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
	u := Usage{PromptTokens: 1000, CompletionTokens: 500, Model: "gpt-4o"}
	cost := CalculateCostStatic(u)
	// gpt-4o: 1000 * 0.0000025 + 500 * 0.00001 = 0.0025 + 0.005 = 0.0075
	if cost < 0.0074 || cost > 0.0076 {
		t.Errorf("Expected cost ~0.0075, got %f", cost)
	}

	// Unknown model → 0
	if CalculateCostStatic(Usage{PromptTokens: 1000, Model: "no-such-model"}) != 0 {
		t.Error("Expected 0 cost for unknown model")
	}
}

// --- PriceCache Tests ---

func TestPriceCache_BuiltinFallback(t *testing.T) {
	cache := NewPriceCache(PriceCacheConfig{
		CacheTTL: 0, // never fetch — use builtins only
	})
	p, ok := cache.GetPricing("gpt-4o")
	if !ok {
		t.Fatal("Expected builtin pricing for gpt-4o")
	}
	if p.PromptPricePerToken == 0 {
		t.Error("Expected non-zero prompt price for gpt-4o")
	}
}

func TestPriceCache_CustomPricing(t *testing.T) {
	cache := NewPriceCache(PriceCacheConfig{
		CacheTTL: 0,
		CustomPricing: map[string]ModelPricing{
			"my-private-model": {PromptPricePerToken: 0.001, CompletionPricePerToken: 0.002},
		},
	})
	p, ok := cache.GetPricing("my-private-model")
	if !ok {
		t.Fatal("Expected custom pricing")
	}
	if p.PromptPricePerToken != 0.001 {
		t.Errorf("Expected 0.001, got %f", p.PromptPricePerToken)
	}
}

func TestPriceCache_SetPricing(t *testing.T) {
	cache := NewPriceCache(PriceCacheConfig{CacheTTL: 0})
	cache.SetPricing("dynamic-model", ModelPricing{
		PromptPricePerToken: 0.05, CompletionPricePerToken: 0.10,
	})
	// 100 * 0.05 + 50 * 0.10 = 5.0 + 5.0 = 10.0
	cost := cache.CalculateCost(Usage{
		PromptTokens: 100, CompletionTokens: 50, Model: "dynamic-model",
	})
	if cost != 10.0 {
		t.Errorf("Expected cost 10.0, got %f", cost)
	}
}

func TestPriceCache_UnknownModel(t *testing.T) {
	cache := NewPriceCache(PriceCacheConfig{CacheTTL: 0})
	cost := cache.CalculateCost(Usage{
		PromptTokens: 1000, CompletionTokens: 500, Model: "no-such-model-xyz",
	})
	if cost != 0 {
		t.Errorf("Expected 0 cost for unknown model, got %f", cost)
	}
}

func TestPriceCache_LiveRefresh(t *testing.T) {
	// Mock OpenRouter API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":      "openai/gpt-4o",
					"pricing": map[string]string{"prompt": "0.0000025", "completion": "0.00001"},
				},
				{
					"id":      "test-provider/test-model",
					"pricing": map[string]string{"prompt": "0.001", "completion": "0.002"},
				},
			},
		})
	}))
	defer server.Close()

	cache := NewPriceCache(PriceCacheConfig{
		APIEndpoint:  server.URL,
		FetchTimeout: 5 * time.Second,
		CacheTTL:     1 * time.Hour,
	})

	// Force refresh
	if err := cache.Refresh(); err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}

	// Short name should work
	p, ok := cache.GetPricing("test-model")
	if !ok {
		t.Fatal("Expected pricing for test-model after refresh")
	}
	if p.PromptPricePerToken != 0.001 {
		t.Errorf("Expected 0.001, got %f", p.PromptPricePerToken)
	}

	// Full ID should also work
	p2, ok := cache.GetPricing("test-provider/test-model")
	if !ok {
		t.Fatal("Expected pricing for full ID")
	}
	if p2.CompletionPricePerToken != 0.002 {
		t.Errorf("Expected 0.002, got %f", p2.CompletionPricePerToken)
	}

	// LastFetch should be set
	if cache.LastFetch().IsZero() {
		t.Error("Expected LastFetch to be set")
	}
}

func TestPriceCache_FailureKeepsBuiltins(t *testing.T) {
	cache := NewPriceCache(PriceCacheConfig{
		APIEndpoint:  "http://localhost:1/nonexistent",
		FetchTimeout: 1 * time.Second,
		CacheTTL:     1 * time.Hour,
	})

	// This will fail silently via ensureFresh
	p, ok := cache.GetPricing("gpt-4o")
	if !ok {
		t.Fatal("Expected builtin pricing to survive failed fetch")
	}
	if p.PromptPricePerToken == 0 {
		t.Error("Expected non-zero price")
	}
}

func TestPriceCache_AutoRefreshOnStale(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
	}))
	defer server.Close()

	cache := NewPriceCache(PriceCacheConfig{
		APIEndpoint:  server.URL,
		CacheTTL:     1 * time.Millisecond, // expire immediately
		FetchTimeout: 2 * time.Second,
	})

	// First call triggers fetch
	cache.GetPricing("gpt-4o")
	firstCount := callCount

	// Wait for cache to expire
	time.Sleep(5 * time.Millisecond)

	// Second call should trigger another fetch
	cache.GetPricing("gpt-4o")
	if callCount <= firstCount {
		t.Errorf("Expected re-fetch after TTL expiry, got %d total calls", callCount)
	}
}

func TestPriceCache_ModelCount(t *testing.T) {
	cache := NewPriceCache(PriceCacheConfig{CacheTTL: 0})
	if cache.ModelCount() < 10 {
		t.Errorf("Expected at least 10 builtin models, got %d", cache.ModelCount())
	}
}

func TestPriceCache_AllPricingIsCopy(t *testing.T) {
	cache := NewPriceCache(PriceCacheConfig{CacheTTL: 0})
	snapshot := cache.AllPricing()
	snapshot["injected"] = ModelPricing{PromptPricePerToken: 999}
	if _, ok := cache.GetPricing("injected"); ok {
		t.Error("AllPricing returned a reference, not a copy")
	}
}

func TestParsePrice(t *testing.T) {
	cases := []struct{ in string; want float64 }{
		{"0.0000025", 0.0000025},
		{"0.001", 0.001},
		{"0", 0},
		{"", 0},
	}
	for _, c := range cases {
		got := parsePrice(c.in)
		if got != c.want {
			t.Errorf("parsePrice(%q) = %f, want %f", c.in, got, c.want)
		}
	}
}

// Verify PriceCache satisfies no-API-key requirement
func TestPriceCache_NoAPIKeyNeeded(t *testing.T) {
	// The OpenRouter /api/v1/models endpoint is public.
	// This test just ensures our cache works without any auth config.
	cache := NewPriceCache(PriceCacheConfig{CacheTTL: 0})
	ctx := context.Background()
	_ = ctx // no auth headers needed
	if cache.ModelCount() == 0 {
		t.Error("Expected builtin models loaded without any API key")
	}
}
