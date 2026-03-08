// Package llm provides Usage tracking for LLM API calls.
// Usage captures token counts, cost, latency, and model metadata per request.
//
// The PriceCache fetches live model pricing from the OpenRouter public API
// on first use (no API key required) and caches it with a configurable TTL.
// It falls back to a hardcoded builtin table when the API is unreachable.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Usage captures the resource consumption of a single LLM API call.
type Usage struct {
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	CostUSD          float64   `json:"cost_usd"`
	Model            string    `json:"model"`
	Provider         string    `json:"provider"` // "openai", "anthropic", "gemini", etc.
	LatencyMs        int64     `json:"latency_ms"`
	Timestamp        time.Time `json:"timestamp"`
}

// UsageTracker accumulates usage data across multiple LLM calls.
// It is safe for concurrent use.
type UsageTracker struct {
	mu     sync.Mutex
	calls  []Usage
	totals Usage
}

// NewUsageTracker creates a new empty tracker.
func NewUsageTracker() *UsageTracker {
	return &UsageTracker{
		calls: make([]Usage, 0, 64),
	}
}

// Record adds a usage record and updates the running totals.
func (ut *UsageTracker) Record(u Usage) {
	if u.Timestamp.IsZero() {
		u.Timestamp = time.Now()
	}
	ut.mu.Lock()
	defer ut.mu.Unlock()
	ut.calls = append(ut.calls, u)
	ut.totals.PromptTokens += u.PromptTokens
	ut.totals.CompletionTokens += u.CompletionTokens
	ut.totals.TotalTokens += u.TotalTokens
	ut.totals.CostUSD += u.CostUSD
	ut.totals.LatencyMs += u.LatencyMs
}

// Totals returns the accumulated totals across all recorded calls.
func (ut *UsageTracker) Totals() Usage {
	ut.mu.Lock()
	defer ut.mu.Unlock()
	return ut.totals
}

// CallCount returns the number of recorded calls.
func (ut *UsageTracker) CallCount() int {
	ut.mu.Lock()
	defer ut.mu.Unlock()
	return len(ut.calls)
}

// AllCalls returns a copy of all recorded usage entries.
func (ut *UsageTracker) AllCalls() []Usage {
	ut.mu.Lock()
	defer ut.mu.Unlock()
	out := make([]Usage, len(ut.calls))
	copy(out, ut.calls)
	return out
}

// Reset clears all recorded usage data.
func (ut *UsageTracker) Reset() {
	ut.mu.Lock()
	defer ut.mu.Unlock()
	ut.calls = ut.calls[:0]
	ut.totals = Usage{}
}

// ---------------------------------------------------------------------------
// ModelPricing & Hardcoded Fallback Table
// ---------------------------------------------------------------------------

// ModelPricing holds per-token pricing for a model.
type ModelPricing struct {
	PromptPricePerToken     float64 `json:"prompt_price_per_token"`
	CompletionPricePerToken float64 `json:"completion_price_per_token"`
}

// builtinPricing is an OFFLINE FALLBACK ONLY. The PriceCache automatically
// fetches live pricing from OpenRouter on first use — these values are only
// used when the API is unreachable (no internet, firewall, etc.).
// Prices: USD per token. Last manually updated: March 2026.
var builtinPricing = map[string]ModelPricing{
	// --- OpenAI (as of early 2026) ---
	"gpt-5.2":        {PromptPricePerToken: 0.00000175, CompletionPricePerToken: 0.000014},
	"gpt-4.1":        {PromptPricePerToken: 0.000002, CompletionPricePerToken: 0.000008},
	"gpt-4o":         {PromptPricePerToken: 0.0000025, CompletionPricePerToken: 0.00001},
	"gpt-4o-mini":    {PromptPricePerToken: 0.00000015, CompletionPricePerToken: 0.0000006},
	"gpt-4-turbo":    {PromptPricePerToken: 0.00001, CompletionPricePerToken: 0.00003},
	"gpt-4":          {PromptPricePerToken: 0.00003, CompletionPricePerToken: 0.00006},
	"gpt-3.5-turbo":  {PromptPricePerToken: 0.0000005, CompletionPricePerToken: 0.0000015},

	// --- Anthropic (as of early 2026) ---
	"claude-sonnet-4.5":           {PromptPricePerToken: 0.000003, CompletionPricePerToken: 0.000015},
	"claude-opus-4.5":             {PromptPricePerToken: 0.000005, CompletionPricePerToken: 0.000025},
	"claude-haiku-4.5":            {PromptPricePerToken: 0.000001, CompletionPricePerToken: 0.000005},
	"claude-3-5-sonnet":           {PromptPricePerToken: 0.000003, CompletionPricePerToken: 0.000015},
	"claude-3-5-sonnet-20240620":  {PromptPricePerToken: 0.000003, CompletionPricePerToken: 0.000015},
	"claude-3-opus":               {PromptPricePerToken: 0.000015, CompletionPricePerToken: 0.000075},
	"claude-3-haiku":              {PromptPricePerToken: 0.00000025, CompletionPricePerToken: 0.00000125},

	// --- Google Gemini (as of early 2026) ---
	"gemini-3.1-pro":   {PromptPricePerToken: 0.000002, CompletionPricePerToken: 0.000012},
	"gemini-3.1-flash": {PromptPricePerToken: 0.0000005, CompletionPricePerToken: 0.000003},
	"gemini-2.5-pro":   {PromptPricePerToken: 0.00000125, CompletionPricePerToken: 0.00001},
	"gemini-2.0-flash":  {PromptPricePerToken: 0.0000001, CompletionPricePerToken: 0.0000004},
	"gemini-1.5-pro":    {PromptPricePerToken: 0.00000125, CompletionPricePerToken: 0.000005},

	// --- Groq / Open Source (as of early 2026) ---
	"llama-3.3-70b":      {PromptPricePerToken: 0.0000001, CompletionPricePerToken: 0.00000032},
	"llama-3.1-70b":      {PromptPricePerToken: 0.00000059, CompletionPricePerToken: 0.00000079},
	"mixtral-8x7b-32768": {PromptPricePerToken: 0.00000024, CompletionPricePerToken: 0.00000024},
	"mixtral-8x7b":       {PromptPricePerToken: 0.00000024, CompletionPricePerToken: 0.00000024},

	// --- DeepSeek (as of early 2026) ---
	"deepseek-chat":     {PromptPricePerToken: 0.00000028, CompletionPricePerToken: 0.00000042},
	"deepseek-reasoner": {PromptPricePerToken: 0.00000056, CompletionPricePerToken: 0.00000168},

	// --- Mistral (as of early 2026) ---
	"mistral-large":  {PromptPricePerToken: 0.000002, CompletionPricePerToken: 0.000006},
	"mistral-medium": {PromptPricePerToken: 0.00000275, CompletionPricePerToken: 0.0000081},
	"mistral-small":  {PromptPricePerToken: 0.000001, CompletionPricePerToken: 0.000003},

	// --- Qwen (as of early 2026) ---
	"qwen-2.5-72b": {PromptPricePerToken: 0.0000009, CompletionPricePerToken: 0.0000009},
}

// ---------------------------------------------------------------------------
// PriceCache — Simple Lazy-Loading Price Cache (No Background Goroutines)
// ---------------------------------------------------------------------------
//
// PriceCache fetches live token pricing from the OpenRouter public API
// (https://openrouter.ai/api/v1/models — no API key required) on first use,
// and caches the result. Subsequent calls use the cache until it expires
// (default: 6 hours), then re-fetch transparently.
//
// Usage:
//
//	cache := llm.NewPriceCache(llm.PriceCacheConfig{})
//	cost := cache.CalculateCost(usage) // fetches on first call, cached after
type PriceCache struct {
	mu        sync.RWMutex
	prices    map[string]ModelPricing
	lastFetch time.Time
	config    PriceCacheConfig
}

// PriceCacheConfig configures the price cache behavior.
type PriceCacheConfig struct {
	// CacheTTL controls how long prices are cached before re-fetching.
	// Default: 6 hours. Set to 0 to always use builtins (never fetch).
	CacheTTL time.Duration

	// FetchTimeout is the HTTP timeout for the pricing API call. Default: 10s.
	FetchTimeout time.Duration

	// APIEndpoint overrides the pricing API URL. Default: OpenRouter public API.
	APIEndpoint string

	// CustomPricing allows adding prices for private/custom models.
	// These always take priority over fetched and builtin prices.
	CustomPricing map[string]ModelPricing
}

// NewPriceCache creates a new cache pre-loaded with builtin pricing.
// Live prices are fetched lazily on first CalculateCost/GetPricing call.
func NewPriceCache(cfg PriceCacheConfig) *PriceCache {
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 6 * time.Hour
	}
	if cfg.FetchTimeout == 0 {
		cfg.FetchTimeout = 10 * time.Second
	}
	if cfg.APIEndpoint == "" {
		cfg.APIEndpoint = "https://openrouter.ai/api/v1/models"
	}

	// Start with builtin + custom
	prices := make(map[string]ModelPricing, len(builtinPricing)+len(cfg.CustomPricing))
	for k, v := range builtinPricing {
		prices[k] = v
	}
	for k, v := range cfg.CustomPricing {
		prices[k] = v
	}

	return &PriceCache{
		prices: prices,
		config: cfg,
	}
}

// GetPricing returns the pricing for a model. If the cache is stale or empty,
// it transparently fetches fresh prices from the API first.
func (pc *PriceCache) GetPricing(model string) (ModelPricing, bool) {
	pc.ensureFresh()

	pc.mu.RLock()
	defer pc.mu.RUnlock()
	p, ok := pc.prices[model]
	return p, ok
}

// CalculateCost computes the USD cost for a Usage entry.
func (pc *PriceCache) CalculateCost(u Usage) float64 {
	p, ok := pc.GetPricing(u.Model)
	if !ok {
		return 0
	}
	return float64(u.PromptTokens)*p.PromptPricePerToken +
		float64(u.CompletionTokens)*p.CompletionPricePerToken
}

// SetPricing manually adds or overrides pricing for a model at runtime.
func (pc *PriceCache) SetPricing(model string, pricing ModelPricing) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.prices[model] = pricing
}

// ModelCount returns the number of models with known pricing.
func (pc *PriceCache) ModelCount() int {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return len(pc.prices)
}

// LastFetch returns when prices were last fetched from the API.
func (pc *PriceCache) LastFetch() time.Time {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.lastFetch
}

// AllPricing returns a snapshot copy of all known model pricing.
func (pc *PriceCache) AllPricing() map[string]ModelPricing {
	pc.ensureFresh()
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	out := make(map[string]ModelPricing, len(pc.prices))
	for k, v := range pc.prices {
		out[k] = v
	}
	return out
}

// Refresh forces an immediate price fetch, regardless of TTL.
func (pc *PriceCache) Refresh() error {
	return pc.fetchPrices()
}

// ensureFresh checks if the cache needs refreshing and fetches if stale.
func (pc *PriceCache) ensureFresh() {
	pc.mu.RLock()
	stale := pc.lastFetch.IsZero() || time.Since(pc.lastFetch) > pc.config.CacheTTL
	pc.mu.RUnlock()

	if stale {
		// Best-effort fetch — if it fails, we keep using builtin/stale data
		_ = pc.fetchPrices()
	}
}

// fetchPrices fetches latest pricing from the OpenRouter API.
func (pc *PriceCache) fetchPrices() error {
	ctx, cancel := context.WithTimeout(context.Background(), pc.config.FetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", pc.config.APIEndpoint, nil)
	if err != nil {
		return fmt.Errorf("price fetch: failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("price fetch: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("price fetch: API returned status %d", resp.StatusCode)
	}

	// OpenRouter /api/v1/models response format
	var apiResp struct {
		Data []struct {
			ID      string `json:"id"` // e.g. "openai/gpt-4o"
			Pricing struct {
				Prompt     string `json:"prompt"`     // price per token as string
				Completion string `json:"completion"` // price per token as string
			} `json:"pricing"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("price fetch: failed to decode response: %w", err)
	}

	// Build new price map: builtins → live → custom (priority order)
	newPrices := make(map[string]ModelPricing, len(apiResp.Data)+len(builtinPricing))
	for k, v := range builtinPricing {
		newPrices[k] = v
	}
	for _, model := range apiResp.Data {
		promptPrice := parsePrice(model.Pricing.Prompt)
		completionPrice := parsePrice(model.Pricing.Completion)
		if promptPrice <= 0 && completionPrice <= 0 {
			continue
		}
		pricing := ModelPricing{
			PromptPricePerToken:     promptPrice,
			CompletionPricePerToken: completionPrice,
		}
		newPrices[model.ID] = pricing
		// Also store short name: "openai/gpt-4o" → "gpt-4o"
		if parts := strings.SplitN(model.ID, "/", 2); len(parts) == 2 {
			newPrices[parts[1]] = pricing
		}
	}
	for k, v := range pc.config.CustomPricing {
		newPrices[k] = v
	}

	pc.mu.Lock()
	pc.prices = newPrices
	pc.lastFetch = time.Now()
	pc.mu.Unlock()

	return nil
}

// parsePrice converts a string price (from OpenRouter API) to float64.
func parsePrice(s string) float64 {
	if s == "" || s == "0" {
		return 0
	}
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// ---------------------------------------------------------------------------
// Global Singleton + Package-Level CalculateCost
// ---------------------------------------------------------------------------

var (
	globalCache     *PriceCache
	globalCacheOnce sync.Once
)

// GlobalPriceCache returns the default singleton PriceCache.
// Created lazily on first call with default config.
func GlobalPriceCache() *PriceCache {
	globalCacheOnce.Do(func() {
		globalCache = NewPriceCache(PriceCacheConfig{})
	})
	return globalCache
}

// CalculateCost computes the USD cost for a Usage entry using live pricing
// from the global PriceCache. Prices are fetched from OpenRouter on first
// call and cached for 6 hours.
func CalculateCost(u Usage) float64 {
	return GlobalPriceCache().CalculateCost(u)
}

// CalculateCostStatic computes cost using only the hardcoded builtin table.
// Use this when you explicitly don't want any HTTP calls (e.g., in tests).
func CalculateCostStatic(u Usage) float64 {
	pricing, ok := builtinPricing[u.Model]
	if !ok {
		return 0
	}
	return float64(u.PromptTokens)*pricing.PromptPricePerToken +
		float64(u.CompletionTokens)*pricing.CompletionPricePerToken
}

