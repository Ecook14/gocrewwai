package knowledge

import (
	"context"
	"fmt"
)

// ============================================================
// Knowledge Source Types
// ============================================================

// Source is the interface that all knowledge sources must implement.
type Source interface {
	// Type returns the source type identifier.
	Type() string
	// Load returns the raw content to be ingested.
	Load(ctx context.Context) (string, error)
	// Identifier returns a unique key for this source (for dedup).
	Identifier() string
}

// StringSource provides knowledge from raw string content.
type StringSource struct {
	Content string
	Label   string
}

func (s *StringSource) Type() string                           { return "string" }
func (s *StringSource) Identifier() string                     { return "string:" + s.Label }
func (s *StringSource) Load(ctx context.Context) (string, error) { return s.Content, nil }

// TextFileSource provides knowledge from .txt files.
type TextFileSource struct {
	FilePaths []string
}

func (s *TextFileSource) Type() string       { return "text_file" }
func (s *TextFileSource) Identifier() string { return fmt.Sprintf("text_file:%v", s.FilePaths) }
func (s *TextFileSource) Load(ctx context.Context) (string, error) {
	return "", fmt.Errorf("use IngestionEngine.IngestFile for file-based sources")
}

// PDFSource provides knowledge from .pdf files.
type PDFSource struct {
	FilePaths []string
}

func (s *PDFSource) Type() string       { return "pdf" }
func (s *PDFSource) Identifier() string { return fmt.Sprintf("pdf:%v", s.FilePaths) }
func (s *PDFSource) Load(ctx context.Context) (string, error) {
	return "", fmt.Errorf("use IngestionEngine.IngestPDF for PDF sources")
}

// CSVSource provides knowledge from .csv files.
type CSVSource struct {
	FilePaths []string
}

func (s *CSVSource) Type() string       { return "csv" }
func (s *CSVSource) Identifier() string { return fmt.Sprintf("csv:%v", s.FilePaths) }
func (s *CSVSource) Load(ctx context.Context) (string, error) {
	return "", fmt.Errorf("use IngestionEngine.IngestCSV for CSV sources")
}

// JSONSource provides knowledge from .json files.
type JSONSource struct {
	FilePaths []string
}

func (s *JSONSource) Type() string       { return "json" }
func (s *JSONSource) Identifier() string { return fmt.Sprintf("json:%v", s.FilePaths) }
func (s *JSONSource) Load(ctx context.Context) (string, error) {
	return "", fmt.Errorf("use IngestionEngine.IngestJSON for JSON sources")
}

// URLSource provides knowledge from web pages.
type URLSource struct {
	URLs []string
}

func (s *URLSource) Type() string       { return "url" }
func (s *URLSource) Identifier() string { return fmt.Sprintf("url:%v", s.URLs) }
func (s *URLSource) Load(ctx context.Context) (string, error) {
	return "", fmt.Errorf("use IngestionEngine.IngestURL for URL sources")
}

// ============================================================
// Knowledge Config
// ============================================================

// Config controls knowledge retrieval behavior.
type Config struct {
	ResultsLimit   int     `yaml:"results_limit" json:"results_limit"`       // Default: 3
	ScoreThreshold float64 `yaml:"score_threshold" json:"score_threshold"`   // Default: 0.35
	CollectionName string  `yaml:"collection_name" json:"collection_name"`   // Default: "knowledge"
}

// DefaultConfig returns the default knowledge configuration.
func DefaultConfig() Config {
	return Config{
		ResultsLimit:   3,
		ScoreThreshold: 0.35,
		CollectionName: "knowledge",
	}
}
