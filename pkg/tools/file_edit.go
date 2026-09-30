package tools

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Ecook14/gocrewwai/pkg/utils"
)

// FileEditTool allows agents to edit file contents via search-and-replace.
type FileEditTool struct {
	BaseTool
	Chroot string
}

var _ Tool = (*FileEditTool)(nil)

func NewFileEditTool(chroot string) *FileEditTool {
	return &FileEditTool{
		BaseTool: BaseTool{
			NameValue:        "FileEditTool",
			DescriptionValue: "Edit file content via search-and-replace strings. Input: {'file_path': 'string', 'search': 'string', 'replace': 'string'}.",
		},
		Chroot: chroot,
	}
}

func (t *FileEditTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	filePath, _ := input["file_path"].(string)
	targetText, _ := input["target_text"].(string)
	replacementText, _ := input["replacement_text"].(string)

	if filePath == "" || targetText == "" {
		return "", fmt.Errorf("file_path and target_text are required")
	}

	// Security: root-confined read (no symlink-redirect race).
	data, err := utils.ReadFileInRoot(t.Chroot, filePath)
	if err != nil {
		return "", err
	}
	content := string(data)

	// 2. Find block
	start, end, ok := utils.FindMatchingBlock(content, targetText)
	if !ok {
		return "", fmt.Errorf("target_text not found in %s (no match found even with heuristic passes)", filePath)
	}

	// 3. Replace
	updated := content[:start] + replacementText + content[end:]

	// 4. Write back through the confined root.
	if err := utils.WriteFileInRoot(t.Chroot, filePath, []byte(updated), 0644); err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully updated %s. Applied patch to block starting at byte %d.", filepath.Base(filePath), start), nil
}

// RequiresReview gates file modification — always human-approved.
func (t *FileEditTool) RequiresReview() bool { return true }
