package knowledge

import (
	"context"
	"fmt"
	"strings"

	"github.com/Ecook14/crewai-go/pkg/llm"
	"github.com/Ecook14/crewai-go/pkg/memory"
)

// KnowledgeSearchTool allows agents to search specifically within the ingested knowledge base.
type KnowledgeSearchTool struct {
	Store memory.Store
	LLM   llm.Client
}

func NewKnowledgeSearchTool(store memory.Store, llm llm.Client) *KnowledgeSearchTool {
	return &KnowledgeSearchTool{Store: store, LLM: llm}
}

func (t *KnowledgeSearchTool) Name() string { return "KnowledgeSearchTool" }

func (t *KnowledgeSearchTool) Description() string {
	return "Searches the internal knowledge base for relevant information. Input requires 'query' as a string."
}

func (t *KnowledgeSearchTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	query, ok := input["query"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'query' parameter")
	}

	// Generate embedding for the query
	vector, err := t.LLM.GenerateEmbedding(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to generate embedding for knowledge search: %w", err)
	}

	results, err := t.Store.Search(ctx, vector, 3)
	if err != nil {
		return "", fmt.Errorf("failed to search knowledge base: %w", err)
	}

	if len(results) == 0 {
		return "No relevant information found in the knowledge base.", nil
	}

	var sb strings.Builder
	sb.WriteString("Relevant information from internal knowledge base:\n\n")
	for i, res := range results {
		sb.WriteString(fmt.Sprintf("-- Source %d --\n%s\n\n", i+1, res.Text))
	}

	return sb.String(), nil
}

func (t *KnowledgeSearchTool) RequiresReview() bool { return false }
