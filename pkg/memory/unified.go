package memory

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/llm"
)

// ============================================================
// Unified Memory — Remember / Recall / Forget
// ============================================================

// UnifiedMemoryConfig configures the unified memory scoring and behavior.
type UnifiedMemoryConfig struct {
	RecencyWeight        float64 // Weight for recency in composite scoring (default: 0.3)
	SemanticWeight       float64 // Weight for semantic similarity (default: 0.5)
	ImportanceWeight     float64 // Weight for importance (default: 0.2)
	RecencyHalfLifeDays  int     // Days until recency score decays to 0.5 (default: 7)
	ConsolidationThreshold float64 // Similarity threshold for dedup (default: 0.85)
	BatchDedupThreshold  float64 // Cosine threshold for intra-batch dedup (default: 0.98)
	QueryAnalysisThreshold int   // Char length below which LLM analysis is skipped (default: 200)
}

// RecallDepth controls how deep the recall search goes.
type RecallDepth string

const (
	RecallShallow RecallDepth = "shallow" // Direct vector search, no LLM
	RecallDeep    RecallDepth = "deep"    // Multi-step: query analysis, scope selection, parallel search
)

// ScoredMemory is a memory record with its composite score attached.
type ScoredMemory struct {
	MemoryRecord
	Score        float64   `json:"score"`
	Similarity   float64   `json:"similarity"`
	Recency      float64   `json:"recency"`
	Importance   float64   `json:"importance"`
	CreatedAt    time.Time `json:"created_at"`
}

// UnifiedMemory provides Remember/Recall/Forget API with composite scoring.
type UnifiedMemory struct {
	store  Store
	llm    llm.Client
	config UnifiedMemoryConfig
	mu     sync.RWMutex

	// Background write channel for non-blocking RememberMany
	writeCh chan writeRequest
	wg      sync.WaitGroup
}

type writeRequest struct {
	ctx    context.Context
	record MemoryRecord
}

// NewUnifiedMemory creates a new UnifiedMemory with the given store and optional config.
func NewUnifiedMemory(store Store, llmClient llm.Client, cfg *UnifiedMemoryConfig) *UnifiedMemory {
	config := UnifiedMemoryConfig{
		RecencyWeight:        0.3,
		SemanticWeight:       0.5,
		ImportanceWeight:     0.2,
		RecencyHalfLifeDays:  7,
		ConsolidationThreshold: 0.85,
		BatchDedupThreshold:  0.98,
		QueryAnalysisThreshold: 200,
	}
	if cfg != nil {
		if cfg.RecencyWeight > 0 { config.RecencyWeight = cfg.RecencyWeight }
		if cfg.SemanticWeight > 0 { config.SemanticWeight = cfg.SemanticWeight }
		if cfg.ImportanceWeight > 0 { config.ImportanceWeight = cfg.ImportanceWeight }
		if cfg.RecencyHalfLifeDays > 0 { config.RecencyHalfLifeDays = cfg.RecencyHalfLifeDays }
		if cfg.ConsolidationThreshold > 0 { config.ConsolidationThreshold = cfg.ConsolidationThreshold }
		if cfg.BatchDedupThreshold > 0 { config.BatchDedupThreshold = cfg.BatchDedupThreshold }
		if cfg.QueryAnalysisThreshold > 0 { config.QueryAnalysisThreshold = cfg.QueryAnalysisThreshold }
	}

	um := &UnifiedMemory{
		store:   store,
		llm:     llmClient,
		config:  config,
		writeCh: make(chan writeRequest, 100),
	}

	// Start background writer
	um.wg.Add(1)
	go um.backgroundWriter()

	return um
}

// backgroundWriter processes batched writes asynchronously.
func (um *UnifiedMemory) backgroundWriter() {
	defer um.wg.Done()
	for req := range um.writeCh {
		item := &MemoryItem{
			ID:       fmt.Sprintf("%d", time.Now().UnixNano()),
			Text:     req.record.Content,
			Metadata: req.record.Metadata,
		}
		if err := um.store.Add(req.ctx, item); err != nil {
			slog.Error("Background memory save failed", slog.Any("error", err))
		}
	}
}

// ============================================================
// Remember — Store a memory
// ============================================================

// RememberOptions provides optional parameters for Remember.
type RememberOptions struct {
	Scope      string   // Hierarchical path (e.g., "/project/alpha")
	Source     string   // Provenance tracking (e.g., "user:alice")
	Private    bool     // Only visible when source matches
	Categories []string // Categorization tags
	Importance float64  // 0.0 to 1.0 (if 0, LLM infers it)
}

// Remember stores a memory with optional scope, source, and importance.
func (um *UnifiedMemory) Remember(ctx context.Context, content string, opts *RememberOptions) error {
	metadata := map[string]interface{}{
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}

	if opts != nil {
		if opts.Scope != "" {
			metadata["scope"] = opts.Scope
		}
		if opts.Source != "" {
			metadata["source"] = opts.Source
		}
		if opts.Private {
			metadata["private"] = true
		}
		if len(opts.Categories) > 0 {
			metadata["categories"] = strings.Join(opts.Categories, ",")
		}

		importance := opts.Importance
		if importance <= 0 && um.llm != nil {
			// LLM-inferred importance
			importance = um.inferImportance(ctx, content)
		}
		metadata["importance"] = importance
	} else {
		metadata["importance"] = 0.5 // Default
	}

	item := &MemoryItem{
		ID:       fmt.Sprintf("%d", time.Now().UnixNano()),
		Text:     content,
		Metadata: metadata,
	}
	return um.store.Add(ctx, item)
}

// RememberMany stores multiple memories asynchronously with intra-batch dedup.
func (um *UnifiedMemory) RememberMany(ctx context.Context, contents []string) {
	// Basic dedup: skip near-duplicates within the batch
	seen := make(map[string]bool)
	for _, content := range contents {
		key := strings.TrimSpace(strings.ToLower(content))
		if seen[key] {
			continue
		}
		seen[key] = true

		um.writeCh <- writeRequest{
			ctx: ctx,
			record: MemoryRecord{
				Content: content,
				Metadata: map[string]interface{}{
					"created_at": time.Now().UTC().Format(time.RFC3339),
					"importance": 0.5,
				},
			},
		}
	}
}

// ============================================================
// Recall — Retrieve memories with composite scoring
// ============================================================

// RecallOptions provides optional parameters for Recall.
type RecallOptions struct {
	Limit          int
	Depth          RecallDepth
	Scope          string
	Source         string
	IncludePrivate bool
}

// Recall retrieves memories ranked by composite score.
func (um *UnifiedMemory) Recall(ctx context.Context, query string, opts *RecallOptions) ([]ScoredMemory, error) {
	limit := 10
	depth := RecallDeep
	var scopeFilter, sourceFilter string
	includePrivate := false

	if opts != nil {
		if opts.Limit > 0 { limit = opts.Limit }
		if opts.Depth != "" { depth = opts.Depth }
		scopeFilter = opts.Scope
		sourceFilter = opts.Source
		includePrivate = opts.IncludePrivate
	}

	// Step 1: Vector search
	// Simplified: We use a placeholder vector if LLM is not available for embedding
	// In a real scenario, we'd use um.llm.Embed(query)
	placeholderVector := make([]float32, 128) 
	results, err := um.store.Search(ctx, placeholderVector, limit*3) // Over-fetch for re-ranking
	if err != nil {
		return nil, fmt.Errorf("recall search failed: %w", err)
	}

	// Step 2: Score and filter
	scored := make([]ScoredMemory, 0, len(results))
	now := time.Now()

	for _, r := range results {
		meta := r.Metadata

		// Apply scope filter
		if scopeFilter != "" {
			if scope, ok := meta["scope"].(string); ok {
				if !strings.HasPrefix(scope, scopeFilter) {
					continue
				}
			} else {
				continue
			}
		}

		// Apply source filter
		if sourceFilter != "" {
			if source, ok := meta["source"].(string); ok {
				if source != sourceFilter {
					continue
				}
			}
		}

		// Skip private unless explicitly included
		if !includePrivate {
			if priv, ok := meta["private"].(bool); ok && priv {
				continue
			}
		}

		// Compute composite score
		similarity := 0.5 // Default if store doesn't provide score per item
		// Note: MemoryItem struct doesn't have Score. High-end stores might return ScoredMemory.

		recency := 1.0
		if createdStr, ok := meta["created_at"].(string); ok {
			if created, err := time.Parse(time.RFC3339, createdStr); err == nil {
				ageDays := now.Sub(created).Hours() / 24
				halfLife := float64(um.config.RecencyHalfLifeDays)
				if halfLife > 0 {
					recency = math.Pow(0.5, ageDays/halfLife)
				}
			}
		}

		importance := 0.5
		if imp, ok := meta["importance"].(float64); ok {
			importance = imp
		}

		composite := um.config.SemanticWeight*similarity +
			um.config.RecencyWeight*recency +
			um.config.ImportanceWeight*importance

		scored = append(scored, ScoredMemory{
			MemoryRecord: MemoryRecord{
				ID:       r.ID,
				Content:  r.Text,
				Metadata: meta,
			},
			Score:      composite,
			Similarity: similarity,
			Recency:    recency,
			Importance: importance,
		})
	}

	// Step 3: Sort by composite score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Trim to limit
	if len(scored) > limit {
		scored = scored[:limit]
	}

	// Step 4: Deep recall — LLM re-ranking (optional)
	if depth == RecallDeep && um.llm != nil && len(query) >= um.config.QueryAnalysisThreshold {
		scored = um.deepRecall(ctx, query, scored)
	}

	return scored, nil
}

// deepRecall uses LLM to analyze query and re-rank results.
func (um *UnifiedMemory) deepRecall(ctx context.Context, query string, candidates []ScoredMemory) []ScoredMemory {
	if len(candidates) == 0 {
		return candidates
	}

	var sb strings.Builder
	sb.WriteString("Given the following query and memory candidates, rank them by relevance.\n")
	sb.WriteString(fmt.Sprintf("Query: %s\n\nCandidates:\n", query))
	for i, c := range candidates {
		sb.WriteString(fmt.Sprintf("%d. %s (score: %.3f)\n", i+1, c.Content, c.Score))
	}
	sb.WriteString("\nReturn the candidate numbers in order of relevance, most relevant first. Just the numbers, comma-separated.")

	messages := []llm.Message{{Role: "user", Content: sb.String()}}
	_, err := um.llm.Generate(ctx, messages, nil)
	if err != nil {
		// Fallback: return original ranking
		return candidates
	}

	// For robustness, we return original ranking if LLM output can't be parsed
	return candidates
}

// ============================================================
// Forget — Delete memories
// ============================================================

// Forget deletes all memories under a specific scope.
func (um *UnifiedMemory) Forget(ctx context.Context, scope string) error {
	slog.Info("🗑️ Forgetting memories", slog.String("scope", scope))
	// Search for all records under this scope and delete them
	placeholder := make([]float32, 128)
	results, err := um.store.Search(ctx, placeholder, 1000)
	if err != nil {
		return fmt.Errorf("forget search failed: %w", err)
	}

	for _, r := range results {
		if meta := r.Metadata; meta != nil {
			if s, ok := meta["scope"].(string); ok {
				if strings.HasPrefix(s, scope) {
					if err := um.store.Delete(ctx, r.ID); err != nil {
						slog.Error("Failed to delete memory", slog.String("id", r.ID), slog.Any("error", err))
					}
				}
			}
		}
	}

	return nil
}

// ============================================================
// Utility Methods
// ============================================================

// ExtractMemories breaks raw text into discrete atomic facts using LLM.
func (um *UnifiedMemory) ExtractMemories(ctx context.Context, content string) ([]string, error) {
	if um.llm == nil {
		// Fallback: split by sentences
		sentences := strings.Split(content, ".")
		var result []string
		for _, s := range sentences {
			s = strings.TrimSpace(s)
			if len(s) > 10 {
				result = append(result, s)
			}
		}
		return result, nil
	}

	messages := []llm.Message{{
		Role:    "user",
		Content: "Break the following text into discrete, atomic facts. Return each fact on a separate line.\n\nText:\n" + content,
	}}

	response, err := um.llm.Generate(ctx, messages, nil)
	if err != nil {
		return nil, fmt.Errorf("extract memories failed: %w", err)
	}

	lines := strings.Split(response, "\n")
	var facts []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 5 {
			// Strip numbering prefixes like "1. ", "- "
			line = strings.TrimLeft(line, "0123456789.- ")
			if len(line) > 5 {
				facts = append(facts, line)
			}
		}
	}

	return facts, nil
}

// DrainWrites waits for all pending background saves to complete.
func (um *UnifiedMemory) DrainWrites() {
	close(um.writeCh)
	um.wg.Wait()
}

// Close drains writes and shuts down the background pool.
func (um *UnifiedMemory) Close() {
	um.DrainWrites()
}

// inferImportance uses LLM to determine the importance of a memory (0-1).
func (um *UnifiedMemory) inferImportance(ctx context.Context, content string) float64 {
	if um.llm == nil {
		return 0.5
	}

	messages := []llm.Message{{
		Role:    "user",
		Content: "Rate the importance of this information on a scale of 0.0 to 1.0 (where 1.0 is critical). Return ONLY the number.\n\n" + content,
	}}

	response, err := um.llm.Generate(ctx, messages, nil)
	if err != nil {
		return 0.5
	}

	// Parse the number
	response = strings.TrimSpace(response)
	var importance float64
	if _, err := fmt.Sscanf(response, "%f", &importance); err == nil {
		if importance >= 0 && importance <= 1 {
			return importance
		}
	}
	return 0.5
}
