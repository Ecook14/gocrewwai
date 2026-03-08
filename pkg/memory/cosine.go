package memory

import (
	"context"
	"sort"
	"sync"
)

// InMemCosineStore is a basic reference implementation of `Store`.
// It acts as the default ShortTerm memory, keeping vectors in standard RAM slices and checking distances.
type InMemCosineStore struct {
	mu    sync.RWMutex
	items []*MemoryItem
}

func NewInMemCosineStore() *InMemCosineStore {
	return &InMemCosineStore{
		items: make([]*MemoryItem, 0),
	}
}

// Add appends the memory item locally.
func (s *InMemCosineStore) Add(ctx context.Context, item *MemoryItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, item)
	return nil
}

// Search calculates cosine similarity against the entire memory graph and sorts the hits.
func (s *InMemCosineStore) Search(ctx context.Context, queryVector []float32, limit int) ([]*MemoryItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type scoredItem struct {
		item  *MemoryItem
		score float32
	}

	var results []scoredItem

	for _, t := range s.items {
		if len(t.Vector) != len(queryVector) {
			continue // Skip mismatched embeddings
		}
		
		sim, err := CosineSimilarity(queryVector, t.Vector)
		if err != nil {
			return nil, err
		}

		results = append(results, scoredItem{item: t, score: sim})
	}

	// Sort highest score first
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	var out []*MemoryItem
	for i := 0; i < limit && i < len(results); i++ {
		out = append(out, results[i].item)
	}

	return out, nil
}

// BulkAdd inserts multiple items.
func (s *InMemCosineStore) BulkAdd(ctx context.Context, items []*MemoryItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, items...)
	return nil
}

// Delete removes an item by ID.
func (s *InMemCosineStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.items {
		if item.ID == id {
			s.items = append(s.items[:i], s.items[i+1:]...)
			return nil
		}
	}
	return nil
}

// Count returns the number of stored items.
func (s *InMemCosineStore) Count(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items), nil
}

// Close is a no-op for in-memory stores.
func (s *InMemCosineStore) Close() error {
	return nil
}
