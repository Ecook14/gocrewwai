package memory

import (
	"context"
	"testing"
)

func TestInMemCosineStoreAddAndSearch(t *testing.T) {
	store := NewInMemCosineStore()
	ctx := context.Background()

	// Add items with known vectors
	item1 := &MemoryItem{
		ID:     "item1",
		Text:   "Go programming language",
		Vector: []float32{1.0, 0.0, 0.0},
	}
	item2 := &MemoryItem{
		ID:     "item2",
		Text:   "Python programming language",
		Vector: []float32{0.0, 1.0, 0.0},
	}
	item3 := &MemoryItem{
		ID:     "item3",
		Text:   "Go concurrency patterns",
		Vector: []float32{0.9, 0.1, 0.0},
	}

	if err := store.Add(ctx, item1); err != nil {
		t.Fatalf("Add item1 failed: %v", err)
	}
	if err := store.Add(ctx, item2); err != nil {
		t.Fatalf("Add item2 failed: %v", err)
	}
	if err := store.Add(ctx, item3); err != nil {
		t.Fatalf("Add item3 failed: %v", err)
	}

	// Search with a query vector close to item1 and item3
	queryVec := []float32{0.95, 0.05, 0.0}
	results, err := store.Search(ctx, queryVec, 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// The closest match should be item3 (0.9, 0.1, 0.0) or item1 (1.0, 0.0, 0.0)
	// based on cosine similarity with query (0.95, 0.05, 0.0)
	foundIDs := map[string]bool{}
	for _, r := range results {
		foundIDs[r.ID] = true
	}
	if !foundIDs["item1"] || !foundIDs["item3"] {
		t.Errorf("expected item1 and item3 in results, got %v", results)
	}
}

func TestCosineSimilarityCalculation(t *testing.T) {
	// Identical vectors should have similarity ≈ 1.0
	sim, err := CosineSimilarity([]float32{1, 0, 0}, []float32{1, 0, 0})
	if err != nil {
		t.Fatalf("cosineSimilarity failed: %v", err)
	}
	if sim < 0.99 {
		t.Errorf("identical vectors should have similarity ≈ 1.0, got %f", sim)
	}

	// Orthogonal vectors should have similarity ≈ 0.0
	sim, err = CosineSimilarity([]float32{1, 0, 0}, []float32{0, 1, 0})
	if err != nil {
		t.Fatalf("cosineSimilarity failed: %v", err)
	}
	if sim > 0.01 {
		t.Errorf("orthogonal vectors should have similarity ≈ 0.0, got %f", sim)
	}
}

func TestEmptyStore(t *testing.T) {
	store := NewInMemCosineStore()
	ctx := context.Background()

	results, err := store.Search(ctx, []float32{1, 0, 0}, 5)
	if err != nil {
		t.Fatalf("Search on empty store failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results from empty store, got %d", len(results))
	}
}

func TestMismatchedVectors(t *testing.T) {
	store := NewInMemCosineStore()
	ctx := context.Background()

	// Add item with 3D vector
	store.Add(ctx, &MemoryItem{
		ID:     "item1",
		Text:   "3D item",
		Vector: []float32{1.0, 0.0, 0.0},
	})

	// Search with 2D vector — should skip mismatched items
	results, err := store.Search(ctx, []float32{1.0, 0.0}, 5)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results due to vector mismatch, got %d", len(results))
	}
}

func TestCosineSimilarityMismatchedLengths(t *testing.T) {
	_, err := CosineSimilarity([]float32{1, 0}, []float32{1, 0, 0})
	if err == nil {
		t.Error("expected error for mismatched vector lengths, got nil")
	}
}
