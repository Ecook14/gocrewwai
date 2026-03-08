package tools

import (
	"context"
	"fmt"
	"os"
)

// FileReadTool allows agents to read the contents of a local file.
type FileReadTool struct {
	BaseTool
}

func NewFileReadTool() *FileReadTool {
	return &FileReadTool{
		BaseTool: BaseTool{
			NameValue:        "FileReadTool",
			DescriptionValue: "Reads the content of a specified file. Input requires 'file_path' as a string.",
		},
	}
}


func (t *FileReadTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	pathRaw, ok := input["file_path"]
	if !ok {
		return "", fmt.Errorf("missing 'file_path' in input")
	}

	path, ok := pathRaw.(string)
	if !ok {
		return "", fmt.Errorf("'file_path' must be a string")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file '%s': %w", path, err)
	}

	return string(data), nil
}

func (t *FileReadTool) RequiresReview() bool { return false }
