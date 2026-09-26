package tools

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Ecook14/gocrewwai/pkg/utils"
)

// CSVReadTool allows agents to read and parse CSV files.
type CSVReadTool struct {
	BaseTool
	Chroot string
}

var _ Tool = (*CSVReadTool)(nil)

func NewCSVReadTool(chroot string) *CSVReadTool {
	return &CSVReadTool{
		BaseTool: BaseTool{
			NameValue:        "CSVReadTool",
			DescriptionValue: "Reads and parses a CSV file, returning its contents as structured text. Input requires 'file_path' as a string.",
		},
		Chroot: chroot,
	}
}

func (t *CSVReadTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
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

	f, err := os.Open(safePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1 // variable number of fields

	var lines []string
	lineNum := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("CSV parse error at line %d: %w", lineNum+1, err)
		}
		lineNum++
		lines = append(lines, fmt.Sprintf("Row %d: %s", lineNum, strings.Join(record, " | ")))
	}

	if len(lines) == 0 {
		return "(empty CSV)", nil
	}
	return strings.Join(lines, "\n"), nil
}

func (t *CSVReadTool) CacheFunction(input map[string]interface{}) string {
	if p, ok := input["file_path"].(string); ok && p != "" {
		return "CSVReadTool:" + p
	}
	return ""
}

func (t *CSVReadTool) ArgsSchema() []ArgSchema {
	return []ArgSchema{
		{Name: "file_path", Type: "string", Description: "Path to the CSV file to read", Required: true},
	}
}
