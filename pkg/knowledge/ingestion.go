package knowledge

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/crewai-go/pkg/llm"
	"github.com/Ecook14/crewai-go/pkg/memory"
)

// IngestionEngine takes physical files, chunks them, and saves them into the Agent's Vector Memory.
type IngestionEngine struct {
	Store    memory.Store
	LLM      llm.Client // Used purely for the vector Translation
	Splitter *TokenSplitter
}

// IngestFile reads a local document and natively loads it to the Agent's memory network.
func (ie *IngestionEngine) IngestFile(ctx context.Context, filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file for RAG ingestion: %w", err)
	}

	content := string(data)
	chunks := ie.Splitter.SplitText(content)

	for i, chunk := range chunks {
		vector, err := ie.LLM.GenerateEmbedding(ctx, chunk)
		if err != nil {
			return fmt.Errorf("failed to embed chunk %d: %w", i, err)
		}

		item := &memory.MemoryItem{
			ID:     fmt.Sprintf("doc_%s_chunk_%d", filepath, i),
			Text:   chunk,
			Vector: vector,
			Metadata: map[string]interface{}{
				"source": filepath,
				"chunk":  i,
			},
		}

		if err := ie.Store.Add(ctx, item); err != nil {
			return fmt.Errorf("failed to save chunk %d to memory: %w", i, err)
		}
	}

	return nil
}
