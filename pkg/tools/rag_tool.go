package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Ecook14/gocrewwai/pkg/memory"
)

// NativeRAGTool provides Retrieval-Augmented Generation capabilities.
// It indexes local files and allows agents to search through them.
type NativeRAGTool struct {
	Memory *memory.LongTermMemory
	Dir    string
}

func NewNativeRAGTool(mem *memory.LongTermMemory, dir string) *NativeRAGTool {
	return &NativeRAGTool{
		Memory: mem,
		Dir:    dir,
	}
}

func (t *NativeRAGTool) Name() string { return "NativeRAGTool" }

func (t *NativeRAGTool) Description() string {
	return "Search through local documentation and knowledge bases. Input requires 'query' (the search term). " +
		"Returns the most relevant snippets from the indexed files."
}

// Ingest indexes all files in the provided directory.
func (t *NativeRAGTool) Ingest(ctx context.Context) error {
	return filepath.Walk(t.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return t.Memory.Save(ctx, string(content), map[string]interface{}{
			"source": info.Name(),
			"path":   path,
		})
	})
}

func (t *NativeRAGTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	queryRaw, ok := input["query"]
	if !ok {
		return "", fmt.Errorf("missing 'query' in input")
	}
	query, ok := queryRaw.(string)
	if !ok {
		return "", fmt.Errorf("'query' must be a string")
	}

	items, err := t.Memory.Search(ctx, query, 3)
	if err != nil {
		return "", fmt.Errorf("RAG search failed: %w", err)
	}

	if len(items) == 0 {
		return "No relevant information found in the knowledge base.", nil
	}

	var result string
	for i, item := range items {
		result += fmt.Sprintf("[%d] Source: %v\nContent: %s\n\n", i+1, item.Metadata["source"], item.Text)
	}

	return result, nil
}

func (t *NativeRAGTool) RequiresReview() bool { return false }
