package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/Ecook14/gocrewwai/pkg/utils"
)

// JSONParseTool validates and formats JSON, returning pretty-printed output
// or error details for malformed input.
type JSONParseTool struct {
	BaseTool
	Chroot string
	once   sync.Once
	cache  string
}

func NewJSONParseTool(chroot string) *JSONParseTool {
	return &JSONParseTool{
		BaseTool: BaseTool{
			NameValue:        "JSONParseTool",
			DescriptionValue: "Parses a JSON file, validates structure, and returns pretty-printed output. Input requires 'file_path' as a string.",
		},
		Chroot: chroot,
	}
}

func (t *JSONParseTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	pathRaw, ok := input["file_path"]
	if !ok {
		return "", fmt.Errorf("missing 'file_path' in input")
	}
	path, ok := pathRaw.(string)
	if !ok {
		return "", fmt.Errorf("'file_path' must be a string")
	}

	safePath, err := utils.ValidatePath(path, t.Chroot)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Validate JSON
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// Pretty-print
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}
	return strings.TrimSpace(buf.String()), nil
}

func (t *JSONParseTool) CacheFunction(input map[string]interface{}) string {
	return ""
}

func (t *JSONParseTool) ArgsSchema() []ArgSchema {
	return []ArgSchema{
		{Name: "file_path", Type: "string", Description: "Path to the JSON file to parse", Required: true},
	}
}

var _ Tool = (*JSONParseTool)(nil)
