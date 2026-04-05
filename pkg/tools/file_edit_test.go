package tools_test

import (
	"context"
	"os"
	"testing"
	"github.com/Ecook14/gocrewwai/pkg/tools"
)

func TestFileEditTool_Execute(t *testing.T) {
	tmpFile := "test_edit.txt"
	content := "Line 1\nLine 2\nLine 3"
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create tmp file: %v", err)
	}
	defer os.Remove(tmpFile)

	tool := tools.NewFileEditTool()
	input := map[string]interface{}{
		"file_path":        tmpFile,
		"target_text":      "Line 2",
		"replacement_text": "Updated Line 2",
	}

	res, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("tool execution failed: %v", err)
	}

	if res == "" {
		t.Error("expected non-empty result message")
	}

	newData, _ := os.ReadFile(tmpFile)
	newContent := string(newData)
	expected := "Line 1\nUpdated Line 2\nLine 3"
	if newContent != expected {
		t.Errorf("unexpected content: %s, want %s", newContent, expected)
	}
}
