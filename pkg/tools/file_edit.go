package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Ecook14/gocrewwai/pkg/utils"
)

// FileEditTool allows agents to edit file contents via search-and-replace.
type FileEditTool struct {
	BaseTool
	Chroot string
}

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

	// Security: Validate path against chroot
	safePath, err := utils.ValidatePath(filePath, t.Chroot)
	if err != nil {
		return "", err
	}

	// 1. Read file
	data, err := os.ReadFile(safePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	content := string(data)

	// 2. Find block
	start, end, ok := utils.FindMatchingBlock(content, targetText)
	if !ok {
		return "", fmt.Errorf("target_text not found in %s (no match found even with heuristic passes)", filePath)
	}

	// 3. Replace
	updated := content[:start] + replacementText + content[end:]

	// 4. Write back
	err = os.WriteFile(safePath, []byte(updated), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully updated %s. Applied patch to block starting at byte %d.", filepath.Base(safePath), start), nil
}
