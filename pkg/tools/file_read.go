package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/utils"
)

// FileReadTool allows agents to read the contents of a local file.
type FileReadTool struct {
	BaseTool
	Chroot string
}

func NewFileReadTool(chroot string) *FileReadTool {
	return &FileReadTool{
		BaseTool: BaseTool{
			NameValue:        "FileReadTool",
			DescriptionValue: "Reads the content of a specified file. Input requires 'file_path' as a string.",
		},
		Chroot: chroot,
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

	// Security: Validate path against chroot
	safePath, err := utils.ValidatePath(path, t.Chroot)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file '%s': %w", path, err)
	}

	return string(data), nil
}

func (t *FileReadTool) RequiresReview() bool { return false }
