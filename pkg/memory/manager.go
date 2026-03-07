package memory

import "context"

// MemoryRecord represents a single unit of saved memory.
type MemoryRecord struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Manager defines the interface handling contextual info (UnifiedMemory).
type Manager interface {
	Save(ctx context.Context, record MemoryRecord) error
	Search(ctx context.Context, query string, limit int) ([]MemoryRecord, error)
	Delete(ctx context.Context, id string) error
}
