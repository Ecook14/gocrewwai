package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/tools"
)

func TestFileEditTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_edit.txt")
	content := "Line 1\nLine 2\nLine 3"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile)

	tool := tools.NewFileEditTool(tmpDir)
	input := map[string]interface{}{
		"file_path":       tmpFile,
		"target_text":     "Line 2",
		"replacement_text": "Updated Line 2",
	}

	_, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("tool execution failed: %v", err)
	}

	newData, _ := os.ReadFile(tmpFile)
	if string(newData) == content {
		t.Error("expected file to be updated")
	}
}

func TestFileEditTool_MissingFields(t *testing.T) {
	tool := tools.NewFileEditTool(t.TempDir())
	_, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing fields")
	}
}
