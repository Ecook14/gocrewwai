package memory

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/Ecook14/gocrewwai/pkg/llm"
)

// LongTermMemory handles cross-execution persistent memory.
// It uses a vector-backed store for semantic retrieval.
type LongTermMemory struct {
	Store     Store
	Embedder  llm.Client // Used to generate vectors for searches
}

func NewLongTermMemory(store Store, embedder llm.Client) *LongTermMemory {
	return &LongTermMemory{
		Store:    store,
		Embedder: embedder,
	}
}

func (l *LongTermMemory) Save(ctx context.Context, text string, metadata map[string]interface{}) error {
	if l.Embedder == nil {
		return fmt.Errorf("embedder is required for long term memory")
	}

	embedder, ok := l.Embedder.(llm.Embedder)
	if !ok {
		return fmt.Errorf("configured LLM does not support text embeddings")
	}

	vector, err := embedder.GenerateEmbedding(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Robust ID generation
	hasher := fnv.New64a()
	hasher.Write([]byte(text))
	item := &MemoryItem{
		ID:       fmt.Sprintf("ltm_%x", hasher.Sum64()),
		Text:     text,
		Vector:   vector,
		Metadata: metadata,
	}

	return l.Store.Add(ctx, item)
}

func (l *LongTermMemory) Search(ctx context.Context, query string, limit int) ([]*MemoryItem, error) {
	if l.Embedder == nil {
		return nil, fmt.Errorf("embedder is required for long term memory search")
	}

	embedder, ok := l.Embedder.(llm.Embedder)
	if !ok {
		return nil, fmt.Errorf("configured LLM does not support text embeddings")
	}

	queryVector, err := embedder.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	return l.Store.Search(ctx, queryVector, limit)
}
