package knowledge

import (
	"context"
	"os"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/memory"
	"strings"
)

func TestTokenSplitter(t *testing.T) {
	splitter := NewTokenSplitter(5, 1)
	text := "one two three four five six seven eight nine ten"
	chunks := splitter.SplitText(text)

	if len(chunks) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(chunks))
	}

	// Chunk 1: one two three four five
	// Chunk 2: five six seven eight nine (starts at index 5-1=4)
	// Chunk 3: nine ten (starts at index 9-1=8)
	
	if chunks[0] != "one two three four five" {
		t.Errorf("Unexpected chunk 0: %s", chunks[0])
	}
}

type mockLLM struct {
	llm.Client
	embedFunc func(text string) ([]float32, error)
}

func (m *mockLLM) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return m.embedFunc(text)
}

type mockStore struct {
	memory.Store
	items []*memory.MemoryItem
}

func (m *mockStore) Add(ctx context.Context, item *memory.MemoryItem) error {
	m.items = append(m.items, item)
	return nil
}

func (m *mockStore) BulkAdd(ctx context.Context, items []*memory.MemoryItem) error {
	m.items = append(m.items, items...)
	return nil
}

func TestIngestionEngine(t *testing.T) {
	tmpFile := "test_ingest.txt"
	content := "This is a test file for RAG ingestion. It should be split into multiple chunks."
	os.WriteFile(tmpFile, []byte(content), 0644)
	defer os.Remove(tmpFile)

	mockLLM := &mockLLM{
		embedFunc: func(text string) ([]float32, error) {
			return []float32{0.1, 0.2}, nil
		},
	}
	mockStore := &mockStore{}
	splitter := NewTokenSplitter(5, 0)
	
	ie := &IngestionEngine{
		Store:    mockStore,
		LLM:      mockLLM,
		Splitter: splitter,
	}

	err := ie.IngestFile(context.Background(), tmpFile)
	if err != nil {
		t.Fatalf("IngestFile failed: %v", err)
	}

	if len(mockStore.items) == 0 {
		t.Errorf("No items ingested into store")
	}
}

func TestIngestDirectory(t *testing.T) {
	tmpDir := "./test_ingest_dir"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	os.WriteFile(tmpDir+"/file1.md", []byte("content 1"), 0644)
	os.WriteFile(tmpDir+"/file2.txt", []byte("content 2"), 0644)
	os.WriteFile(tmpDir+"/file3.jpg", []byte("binary"), 0644) // Should be ignored

	mockLLM := &mockLLM{
		embedFunc: func(text string) ([]float32, error) {
			return []float32{0.1}, nil
		},
	}
	mockStore := &mockStore{}
	ie := &IngestionEngine{
		Store:    mockStore,
		LLM:      mockLLM,
		Splitter: NewTokenSplitter(10, 0),
	}

	err := ie.IngestDirectory(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("IngestDirectory failed: %v", err)
	}

	// Should have 2 items (file1 and file2)
	if len(mockStore.items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(mockStore.items))
	}
}

type mockSearchStore struct {
	memory.Store
}

func (m *mockSearchStore) Search(ctx context.Context, vector []float32, limit int) ([]*memory.MemoryItem, error) {
	return []*memory.MemoryItem{
		{Text: "Result 1"},
	}, nil
}

func TestKnowledgeSearchTool(t *testing.T) {
	store := &mockSearchStore{}
	mockLLM := &mockLLM{
		embedFunc: func(text string) ([]float32, error) {
			return []float32{1.0}, nil
		},
	}
	tool := NewKnowledgeSearchTool(store, mockLLM)

	res, err := tool.Execute(context.Background(), map[string]interface{}{"query": "test"})
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	if !strings.Contains(res, "Result 1") {
		t.Errorf("Expected result not found in output: %s", res)
	}
}
