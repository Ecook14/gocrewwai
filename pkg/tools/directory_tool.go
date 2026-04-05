package tools

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DirectoryTool provides high-performance directory tree traversal and file mapping.
// It is strictly controlled by config.json parameters.
type DirectoryTool struct {
	BaseTool
	RootPath     string
	MaxDepth     int
	AllowAbsolute bool
}

// NewDirectoryTool creates a new directory traversal tool.
func NewDirectoryTool(rootPath string, maxDepth int, allowAbsolute bool) *DirectoryTool {
	t := &DirectoryTool{
		BaseTool: BaseTool{
			NameValue:        "DirectoryTool",
			DescriptionValue: "List files and directories in a given path. Input: {'path': 'string'}.",
		},
		RootPath:      rootPath,
		MaxDepth:      maxDepth,
		AllowAbsolute: allowAbsolute,
	}

	if t.RootPath == "" {
		t.RootPath = "."
	}
	if t.MaxDepth <= 0 {
		t.MaxDepth = 5
	}

	return t
}

func (t *DirectoryTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	relPath, _ := input["path"].(string)
	if relPath == "" {
		relPath = "."
	}

	// Security: Resolve and validate path
	fullPath := relPath
	if !filepath.IsAbs(relPath) || !t.AllowAbsolute {
		fullPath = filepath.Join(t.RootPath, relPath)
	}

	// Double check we are not escaping root if requested
	absRoot, _ := filepath.Abs(t.RootPath)
	absTarget, _ := filepath.Abs(fullPath)
	if !t.AllowAbsolute && !strings.HasPrefix(absTarget, absRoot) {
		return "", fmt.Errorf("access denied: path escapes root directory")
	}

	var results []string
	err := filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate depth
		rel, _ := filepath.Rel(fullPath, path)
		depth := strings.Count(rel, string(os.PathSeparator))
		if rel == "." {
			depth = 0
		}

		if depth > t.MaxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		info, _ := d.Info()
		entryType := "file"
		if d.IsDir() {
			entryType = "dir"
		}

		results = append(results, fmt.Sprintf("[%s] %s (%d bytes)", entryType, rel, info.Size()))
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("directory list failed: %w", err)
	}

	if len(results) == 0 {
		return "Directory is empty.", nil
	}

	return strings.Join(results, "\n"), nil
}
