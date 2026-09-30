package tools

import (
	"context"
	"fmt"

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

	// Security: root-confined write — the create/truncate itself is bound
	// to the chroot, so leaf and ancestor symlinks cannot redirect it
	// outside (no validate-then-open race).
	if err := utils.WriteFileInRoot(t.Chroot, path, []byte(content), 0644); err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully wrote to %s", path), nil
}

func (t *FileWriteTool) RequiresReview() bool { return true }
func (t *FileWriteTool) Name() string         { return t.BaseTool.NameValue }
func (t *FileWriteTool) Description() string {
	return "Writes or overwrites the contents of a local file at the given path. DANGEROUS: this tool overwrites files on disk. Input: {'file_path': 'string', 'content': 'string'}. Path is validated against the chroot directory — files outside the chroot cannot be written."
}

var _ Tool = (*FileWriteTool)(nil)
