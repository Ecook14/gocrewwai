package knowledge

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/memory"
	"github.com/ledongthuc/pdf"
)

// IngestionEngine takes physical files, chunks them, and saves them into the Agent's Vector Memory.
type IngestionEngine struct {
	Store    memory.Store
	LLM      llm.Client // Used purely for the vector Translation
	Splitter *TokenSplitter
}

// IngestDirectory reads all supported files in a directory and loads them.
// Supported: .md, .txt, .csv, .json, .jsonl
func (ie *IngestionEngine) IngestDirectory(ctx context.Context, dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	supportedExts := map[string]bool{
		".md": true, ".txt": true, ".csv": true,
		".json": true, ".jsonl": true,
		".pdf": true, ".docx": true,
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if supportedExts[ext] {
			path := filepath.Join(dirPath, entry.Name())
			if err := ie.IngestFile(ctx, path); err != nil {
				return err
			}
		}
	}
	return nil
}

// IngestFile detects format by extension and ingests accordingly.
func (ie *IngestionEngine) IngestFile(ctx context.Context, filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".csv":
		return ie.IngestCSV(ctx, filePath)
	case ".json":
		return ie.IngestJSON(ctx, filePath)
	case ".jsonl":
		return ie.IngestJSONL(ctx, filePath)
	case ".pdf":
		return ie.IngestPDF(ctx, filePath)
	case ".docx":
		return ie.IngestDocx(ctx, filePath)
	default:
		// .md, .txt, and any other text format
		return ie.IngestText(ctx, filePath)
	}
}

// IngestText reads a plain text file, chunks it, embeds, and stores.
func (ie *IngestionEngine) IngestText(ctx context.Context, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	chunks := ie.Splitter.SplitText(string(data))
	return ie.storeChunks(ctx, filePath, chunks)
}

// IngestCSV reads a CSV file and converts each row to a text chunk.
// First row is treated as headers; each subsequent row becomes
// "header1: value1, header2: value2, ..." for semantic embedding.
func (ie *IngestionEngine) IngestCSV(ctx context.Context, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open CSV: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV headers: %w", err)
	}

	var chunks []string
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // Skip malformed rows
		}

		// Convert row to "key: value" pairs
		var parts []string
		for i, val := range row {
			if i < len(headers) {
				parts = append(parts, fmt.Sprintf("%s: %s", headers[i], val))
			}
		}
		chunks = append(chunks, strings.Join(parts, ", "))
	}

	return ie.storeChunks(ctx, filePath, chunks)
}

// IngestJSON reads a JSON file. If the root is an array, each element
// becomes a chunk. If it's an object, the entire document is chunked as text.
func (ie *IngestionEngine) IngestJSON(ctx context.Context, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read JSON: %w", err)
	}

	var chunks []string

	// Try array first
	var arr []interface{}
	if json.Unmarshal(data, &arr) == nil {
		for _, item := range arr {
			itemJSON, _ := json.Marshal(item)
			chunks = append(chunks, string(itemJSON))
		}
	} else {
		// Single object or other — chunk as text
		chunks = ie.Splitter.SplitText(string(data))
	}

	return ie.storeChunks(ctx, filePath, chunks)
}

// IngestJSONL reads a JSON Lines file (one JSON object per line).
func (ie *IngestionEngine) IngestJSONL(ctx context.Context, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read JSONL: %w", err)
	}

	var chunks []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		chunks = append(chunks, line)
	}

	return ie.storeChunks(ctx, filePath, chunks)
}

// IngestPDF extracts plain text from a PDF document using ledongthuc/pdf.
func (ie *IngestionEngine) IngestPDF(ctx context.Context, filePath string) error {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open pdf: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		return fmt.Errorf("failed to extract pdf text: %w", err)
	}
	
	buf.ReadFrom(b)
	content := buf.String()

	chunks := ie.Splitter.SplitText(content)
	return ie.storeChunks(ctx, filePath, chunks)
}

// IngestDocx extracts plain text natively from a Word Document's zipped XML structure.
func (ie *IngestionEngine) IngestDocx(ctx context.Context, filePath string) error {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return fmt.Errorf("failed to open docx: %w", err)
	}
	defer r.Close()

	var docXML []byte
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			buf := new(bytes.Buffer)
			_, _ = io.Copy(buf, rc)
			rc.Close()
			docXML = buf.Bytes()
			break
		}
	}

	if docXML == nil {
		return fmt.Errorf("word/document.xml not found inside docx archive")
	}

	// Stream through XML and extract raw CharData (the actual text)
	var buf bytes.Buffer
	decoder := xml.NewDecoder(bytes.NewReader(docXML))
	for {
		t, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to parse docx xml: %w", err)
		}
		if cd, ok := t.(xml.CharData); ok {
			buf.Write(cd)
			buf.WriteString(" ")
		}
	}

	// Docx text often needs basic spacing cleanup
	content := stripHTMLTags(buf.String())
	chunks := ie.Splitter.SplitText(content)
	return ie.storeChunks(ctx, filePath, chunks)
}

// IngestURL fetches content from a URL, extracts text, and ingests it.
// For HTML pages, basic tag stripping is applied. For JSON endpoints,
// the response is parsed as JSON.
func (ie *IngestionEngine) IngestURL(ctx context.Context, url string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Gocrew/1.0 Knowledge Ingestion")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("URL returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	content := string(body)

	// Basic HTML tag stripping for web pages
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		content = stripHTMLTags(content)
	}

	chunks := ie.Splitter.SplitText(content)
	return ie.storeChunks(ctx, url, chunks)
}

// IngestRawText ingests a raw text string directly (no file needed).
// Useful for programmatic ingestion of API responses, database dumps, etc.
func (ie *IngestionEngine) IngestRawText(ctx context.Context, source, text string) error {
	chunks := ie.Splitter.SplitText(text)
	return ie.storeChunks(ctx, source, chunks)
}

// storeChunks embeds and stores text chunks with source metadata.
func (ie *IngestionEngine) storeChunks(ctx context.Context, source string, chunks []string) error {
	items := make([]*memory.MemoryItem, 0, len(chunks))

	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			continue
		}

		embedder, ok := ie.LLM.(llm.Embedder)
		if !ok {
			return fmt.Errorf("configured LLM does not support text embeddings, cannot ingest")
		}

		vector, err := embedder.GenerateEmbedding(ctx, chunk)
		if err != nil {
			return fmt.Errorf("failed to embed chunk %d from %s: %w", i, source, err)
		}

		hasher := fnv.New64a()
		hasher.Write([]byte(source))
		hasher.Write([]byte(fmt.Sprintf("_%d", i)))

		items = append(items, &memory.MemoryItem{
			ID:     fmt.Sprintf("doc_%x_%d", hasher.Sum64(), i),
			Text:   chunk,
			Vector: vector,
			Metadata: map[string]interface{}{
				"source":     source,
				"chunk":      i,
				"total":      len(chunks),
				"ingested_at": time.Now().Format(time.RFC3339),
			},
			CreatedAt: time.Now(),
		})
	}

	if len(items) == 0 {
		return nil
	}

	// Use BulkAdd for efficiency
	return ie.Store.BulkAdd(ctx, items)
}

// stripHTMLTags performs basic HTML tag removal for plain text extraction.
func stripHTMLTags(html string) string {
	var result strings.Builder
	inTag := false
	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
			result.WriteRune(' ')
		case !inTag:
			result.WriteRune(r)
		}
	}
	// Collapse multiple spaces
	text := result.String()
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}
	return strings.TrimSpace(text)
}
