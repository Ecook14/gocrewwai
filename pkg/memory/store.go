package memory

import (
	"context"
	"time"
)

// MemoryItem represents a single contextual block of agent interaction data.
type MemoryItem struct {
	ID        string                 `json:"id"`
	Text      string                 `json:"text"`
	Vector    []float32              `json:"vector,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at,omitempty"`
	ExpiresAt time.Time              `json:"expires_at,omitempty"` // zero = never expires
}

// Expired returns true if the item has a TTL and it has passed.
func (m *MemoryItem) Expired() bool {
	return !m.ExpiresAt.IsZero() && time.Now().After(m.ExpiresAt)
}

// Store defines the interface for underlying memory backends.
type Store interface {
	// Add inserts a single item into the memory database.
	Add(ctx context.Context, item *MemoryItem) error

	// BulkAdd inserts multiple items in a single batch operation.
	BulkAdd(ctx context.Context, items []*MemoryItem) error

	// Delete removes an item by ID.
	Delete(ctx context.Context, id string) error

	// Search locates the nearest matching MemoryItems to a given query vector.
	Search(ctx context.Context, queryVector []float32, limit int) ([]*MemoryItem, error)

	// Count returns the total number of stored items.
	Count(ctx context.Context) (int, error)
}

