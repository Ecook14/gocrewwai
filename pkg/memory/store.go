package memory

import "context"

// MemoryItem represents a single contextual block of agent interaction data.
type MemoryItem struct {
	ID       string                 `json:"id"`
	Text     string                 `json:"text"`
	Vector   []float32              `json:"vector,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Store defines the interface for underlying memory backends.
// This abstract interface translates between Python CrewAI's Chroma/LanceDB stores.
type Store interface {
	// Add inserts a single item into the memory database.
	Add(ctx context.Context, item *MemoryItem) error

	// Search locates the nearest matching MemoryItems to a given query vector.
	// Typically implemented via Cosine Similarity in pure Go, or PgVector if scaled out.
	Search(ctx context.Context, queryVector []float32, limit int) ([]*MemoryItem, error)
}
