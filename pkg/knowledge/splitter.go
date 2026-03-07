package knowledge

import (
	"strings"
)

// TokenSplitter breaks massive document strings into manageable chunks
// so they can be securely embedded and loaded into Vector context windows.
type TokenSplitter struct {
	ChunkSize    int
	ChunkOverlap int
}

func NewTokenSplitter(chunkSize, chunkOverlap int) *TokenSplitter {
	if chunkSize <= 0 {
		chunkSize = 1000 // Default to standard OpenAI token sizing maxes
	}
	if chunkOverlap < 0 {
		chunkOverlap = 100 // Default 10% overlap to preserve sentence context
	}
	return &TokenSplitter{
		ChunkSize:    chunkSize,
		ChunkOverlap: chunkOverlap,
	}
}

// SplitText assumes 1 word ≈ 1 token for basic Go architectural extraction.
func (ts *TokenSplitter) SplitText(text string) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var chunks []string
	
	// If the entire document is smaller than the chunk size, just return it.
	if len(words) <= ts.ChunkSize {
		return []string{strings.Join(words, " ")}
	}

	for i := 0; i < len(words); i += (ts.ChunkSize - ts.ChunkOverlap) {
		end := i + ts.ChunkSize
		if end > len(words) {
			end = len(words)
		}

		chunk := strings.Join(words[i:end], " ")
		chunks = append(chunks, chunk)
		
		if end == len(words) {
			break
		}
	}

	return chunks
}
