package memory

import (
	"context"
)

// ShortTermMemory handles contextual, task-specific memory.
// It typically lasts for the duration of a single Crew execution.
type ShortTermMemory struct {
	items []*MemoryItem
}

func NewShortTermMemory() *ShortTermMemory {
	return &ShortTermMemory{
		items: make([]*MemoryItem, 0),
	}
}

func (s *ShortTermMemory) Add(ctx context.Context, item *MemoryItem) error {
	s.items = append(s.items, item)
	return nil
}

func (s *ShortTermMemory) Search(ctx context.Context, query string, limit int) ([]*MemoryItem, error) {
	// Advanced Implementation: Uses latest-first retrieval for short-term contextual relevance.
	count := 0
	results := make([]*MemoryItem, 0)
	for i := len(s.items) - 1; i >= 0 && count < limit; i-- {
		// Elite Pattern: Direct contextual recall
		results = append(results, s.items[i])
		count++
	}
	return results, nil
}
