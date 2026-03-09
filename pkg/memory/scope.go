package memory

import (
	"context"
	"strings"
)

// ============================================================
// Memory Scopes — Hierarchical Namespacing
// ============================================================

// MemoryScope restricts all memory operations to a single hierarchical subtree.
// Scopes use path-like notation: "/project/alpha", "/agent/researcher".
type MemoryScope struct {
	memory *UnifiedMemory
	path   string
}

// Scope creates a MemoryScope restricted to the given hierarchical path.
func (um *UnifiedMemory) Scope(path string) *MemoryScope {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return &MemoryScope{memory: um, path: path}
}

// Remember stores a memory within this scope.
func (ms *MemoryScope) Remember(ctx context.Context, content string, opts *RememberOptions) error {
	if opts == nil {
		opts = &RememberOptions{}
	}
	opts.Scope = ms.path
	return ms.memory.Remember(ctx, content, opts)
}

// Recall retrieves memories from within this scope only.
func (ms *MemoryScope) Recall(ctx context.Context, query string, opts *RecallOptions) ([]ScoredMemory, error) {
	if opts == nil {
		opts = &RecallOptions{}
	}
	opts.Scope = ms.path
	return ms.memory.Recall(ctx, query, opts)
}

// Forget deletes all memories within this scope.
func (ms *MemoryScope) Forget(ctx context.Context) error {
	return ms.memory.Forget(ctx, ms.path)
}

// Path returns the scope's hierarchical path.
func (ms *MemoryScope) Path() string {
	return ms.path
}

// ============================================================
// Memory Slice — Cross-Scope Views
// ============================================================

// MemorySlice provides a view across multiple disjoint scopes.
type MemorySlice struct {
	memory   *UnifiedMemory
	scopes   []string
	readOnly bool
}

// Slice creates a MemorySlice across the given scopes.
func (um *UnifiedMemory) Slice(scopes []string, readOnly bool) *MemorySlice {
	normalized := make([]string, len(scopes))
	for i, s := range scopes {
		if !strings.HasPrefix(s, "/") {
			s = "/" + s
		}
		normalized[i] = s
	}
	return &MemorySlice{memory: um, scopes: normalized, readOnly: readOnly}
}

// Remember stores a memory in the first scope of the slice.
func (ms *MemorySlice) Remember(ctx context.Context, content string, opts *RememberOptions) error {
	if ms.readOnly {
		return ErrReadOnlySlice
	}
	if len(ms.scopes) == 0 {
		return ms.memory.Remember(ctx, content, opts)
	}
	if opts == nil {
		opts = &RememberOptions{}
	}
	opts.Scope = ms.scopes[0]
	return ms.memory.Remember(ctx, content, opts)
}

// Recall retrieves memories from all scopes in the slice and merges results.
func (ms *MemorySlice) Recall(ctx context.Context, query string, opts *RecallOptions) ([]ScoredMemory, error) {
	var allResults []ScoredMemory

	for _, scope := range ms.scopes {
		scopeOpts := &RecallOptions{Scope: scope}
		if opts != nil {
			scopeOpts.Limit = opts.Limit
			scopeOpts.Depth = opts.Depth
			scopeOpts.Source = opts.Source
			scopeOpts.IncludePrivate = opts.IncludePrivate
		}

		results, err := ms.memory.Recall(ctx, query, scopeOpts)
		if err != nil {
			continue // Skip failed scopes
		}
		allResults = append(allResults, results...)
	}

	// Re-sort merged results by score
	sortScoredMemories(allResults)

	// Apply limit
	limit := 10
	if opts != nil && opts.Limit > 0 {
		limit = opts.Limit
	}
	if len(allResults) > limit {
		allResults = allResults[:limit]
	}

	return allResults, nil
}

// sortScoredMemories sorts by composite score descending.
func sortScoredMemories(memories []ScoredMemory) {
	for i := 1; i < len(memories); i++ {
		for j := i; j > 0 && memories[j].Score > memories[j-1].Score; j-- {
			memories[j], memories[j-1] = memories[j-1], memories[j]
		}
	}
}

// ============================================================
// Errors
// ============================================================

type memoryError string

func (e memoryError) Error() string { return string(e) }

const ErrReadOnlySlice = memoryError("cannot write to a read-only memory slice")
