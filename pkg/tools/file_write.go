package tools

import (
	"context"
	"fmt"
	"os"
)

// FileWriteTool allows agents to write or overwrite contents to a local file.
type FileWriteTool struct {
	BaseTool
	Options map[string]interface{}
}

func NewFileWriteTool() *FileWriteTool {
	return &FileWriteTool{
		BaseTool: BaseTool{
			NameValue:        "FileWriteTool",
			DescriptionValue: "Writes content to a specified file. Input requires 'file_path' as a string and 'content' as a string.",
		},
	}
}


func (t *FileWriteTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	pathRaw, okPath := input["file_path"]
	contentRaw, okContent := input["content"]

	if !okPath || !okContent {
		return "", fmt.Errorf("missing 'file_path' or 'content' in input")
	}

	path, ok := pathRaw.(string)
	if !ok {
		return "", fmt.Errorf("'file_path' must be a string")
	}

	content, ok := contentRaw.(string)
	if !ok {
		return "", fmt.Errorf("'content' must be a string")
	}

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write to file '%s': %w", path, err)
	}

	return fmt.Sprintf("Successfully wrote to %s", path), nil
}

func (t *FileWriteTool) RequiresReview() bool { return false }
