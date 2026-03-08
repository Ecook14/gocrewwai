package memory

import (
	"context"
	"testing"

	"github.com/Ecook14/crewai-go/pkg/llm"
)

func TestShortTermMemory(t *testing.T) {
	stm := NewShortTermMemory()
	ctx := context.Background()

	item := &MemoryItem{
		ID:   "1",
		Text: "Hello",
	}

	if err := stm.Add(ctx, item); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	results, err := stm.Search(ctx, "Hello", 1)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 || results[0].Text != "Hello" {
		t.Errorf("Unexpected search results: %v", results)
	}
}

type mockLLM struct {
	llm.Client
	embedFunc func(text string) ([]float32, error)
}

func (m *mockLLM) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return m.embedFunc(text)
}

func TestLongTermMemory(t *testing.T) {
	mockStore := &mockSimpleStore{}
	mockLLM := &mockLLM{
		embedFunc: func(text string) ([]float32, error) {
			return []float32{0.1}, nil
		},
	}

	ltm := NewLongTermMemory(mockStore, mockLLM)
	ctx := context.Background()

	if err := ltm.Save(ctx, "Long term info", map[string]interface{}{"id": "lt1"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	results, err := ltm.Search(ctx, "info", 1)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 || results[0].Text != "Long term info" {
		t.Errorf("Unexpected search results: %v", results)
	}
}

type mockSimpleStore struct {
	Store
	items []*MemoryItem
}

func (m *mockSimpleStore) Add(ctx context.Context, item *MemoryItem) error {
	m.items = append(m.items, item)
	return nil
}

func (m *mockSimpleStore) Search(ctx context.Context, vector []float32, limit int) ([]*MemoryItem, error) {
	return m.items, nil
}
