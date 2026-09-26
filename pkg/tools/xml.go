package tools

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/utils"
)

// XMLReadTool reads and parses XML files, returning formatted text.
type XMLReadTool struct {
	BaseTool
	Chroot string
}

func NewXMLReadTool(chroot string) *XMLReadTool {
	return &XMLReadTool{
		BaseTool: BaseTool{
			NameValue:        "XMLReadTool",
			DescriptionValue: "Reads and extracts text content from an XML file. Input requires 'file_path' as a string.",
		},
		Chroot: chroot,
	}
}

func (t *XMLReadTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
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

	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(data); err != nil {
		// Fall back to raw content if not well-formed XML
		return string(data), nil
	}
	return buf.String(), nil
}

func (t *XMLReadTool) ArgsSchema() []ArgSchema {
	return []ArgSchema{
		{Name: "file_path", Type: "string", Description: "Path to the XML file to read", Required: true},
	}
}
func (t *XMLReadTool) CacheFunction(input map[string]interface{}) string {
	if p, ok := input["file_path"].(string); ok && p != "" {
		return "XMLReadTool:" + p
	}
	return ""
}

var _ Tool = (*XMLReadTool)(nil)
