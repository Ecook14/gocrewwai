package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Ecook14/gocrewwai/pkg/utils"
	"gopkg.in/yaml.v3"
)

// YAMLReadTool reads and formats YAML files.
type YAMLReadTool struct {
	BaseTool
	Chroot string
}

func NewYAMLReadTool(chroot string) *YAMLReadTool {
	return &YAMLReadTool{
		BaseTool: BaseTool{
			NameValue:        "YAMLReadTool",
			DescriptionValue: "Reads and extracts text content from a YAML file. Input requires 'file_path' as a string.",
		},
		Chroot: chroot,
	}
}

func (t *YAMLReadTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
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

	// Validate and reformat YAML
	var parsed interface{}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("invalid YAML: %w", err)
	}

	var out strings.Builder
	if err := yaml.NewEncoder(&out).Encode(parsed); err != nil {
		return "", fmt.Errorf("failed to format YAML: %w", err)
	}
	return strings.TrimSpace(out.String()), nil
}

func (t *YAMLReadTool) ArgsSchema() []ArgSchema {
	return []ArgSchema{
		{Name: "file_path", Type: "string", Description: "Path to the YAML file to read", Required: true},
	}
}
func (t *YAMLReadTool) CacheFunction(input map[string]interface{}) string { return "" }
func (t *YAMLReadTool) Name() string                                      { return t.BaseTool.NameValue }
func (t *YAMLReadTool) Description() string                               { return t.BaseTool.DescriptionValue }
