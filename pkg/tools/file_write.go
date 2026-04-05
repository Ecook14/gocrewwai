package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/utils"
)

// FileWriteTool allows agents to write or overwrite contents to a local file.
type FileWriteTool struct {
	BaseTool
	Chroot string
}

func NewFileWriteTool(chroot string) *FileWriteTool {
	return &FileWriteTool{
		BaseTool: BaseTool{
			NameValue:        "FileWriteTool",
			DescriptionValue: "Write or overwrite contents of a file. Input: {'file_path': 'string', 'content': 'string'}.",
		},
		Chroot: chroot,
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

	// Security: Validate path against chroot
	safePath, err := utils.ValidatePath(path, t.Chroot)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(safePath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write to file '%s': %w", path, err)
	}

	return fmt.Sprintf("Successfully wrote to %s", path), nil
}

func (t *FileWriteTool) RequiresReview() bool { return false }
