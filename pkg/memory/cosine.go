package memory

import (
	"context"
	"fmt"
	"math"
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
		
		sim, err := cosineSimilarity(queryVector, t.Vector)
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

func cosineSimilarity(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector lengths do not match: %d != %d", len(a), len(b))
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0, nil // Handle null vectors linearly
	}

	sim := dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
	return sim, nil
}
