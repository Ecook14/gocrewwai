package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileReadToolExecute(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(tmpFile, []byte("hello world"), 0644)

	tool := NewFileReadTool()

	if tool.Name() != "FileReadTool" {
		t.Errorf("expected name 'FileReadTool', got '%s'", tool.Name())
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"file_path": tmpFile,
	})
	if err != nil {
		t.Fatalf("FileReadTool.Execute failed: %v", err)
	}

	if result != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", result)
	}
}

func TestFileReadToolMissingPath(t *testing.T) {
	tool := NewFileReadTool()
	_, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing file_path, got nil")
	}
}

func TestFileWriteToolExecute(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "output.txt")

	tool := NewFileWriteTool()

	if tool.Name() != "FileWriteTool" {
		t.Errorf("expected name 'FileWriteTool', got '%s'", tool.Name())
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"file_path": tmpFile,
		"content":   "written content",
	})
	if err != nil {
		t.Fatalf("FileWriteTool.Execute failed: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}

	// Verify the file was written
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(data) != "written content" {
		t.Errorf("expected 'written content', got '%s'", string(data))
	}
}

func TestFileWriteToolMissingInput(t *testing.T) {
	tool := NewFileWriteTool()
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"file_path": "/tmp/test.txt",
	})
	if err == nil {
		t.Error("expected error for missing content, got nil")
	}
}

func TestAskQuestionToolExecute(t *testing.T) {
	tool := NewAskQuestionTool()

	if tool.Name() != "Ask Question" {
		t.Errorf("expected name 'Ask Question', got '%s'", tool.Name())
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"question": "What is Go?",
	})
	if err != nil {
		t.Fatalf("AskQuestionTool.Execute failed: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestAskQuestionToolMissingInput(t *testing.T) {
	tool := NewAskQuestionTool()
	_, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing question, got nil")
	}
}

func TestSearchWebToolName(t *testing.T) {
	tool := NewSearchWebTool()
	if tool.Name() != "SearchWebTool" {
		t.Errorf("expected name 'SearchWebTool', got '%s'", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestSearchWebToolMissingQuery(t *testing.T) {
	tool := NewSearchWebTool()
	_, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing query, got nil")
	}
}

func TestCodeInterpreterToolName(t *testing.T) {
	tool := NewCodeInterpreterTool()
	if tool.Name() != "CodeInterpreterTool" {
		t.Errorf("expected name 'CodeInterpreterTool', got '%s'", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestCodeInterpreterToolMissingInput(t *testing.T) {
	tool := NewCodeInterpreterTool()

	// Missing language
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"code": "fmt.Println(\"hi\")",
	})
	if err == nil {
		t.Error("expected error for missing language, got nil")
	}

	// Missing code
	_, err = tool.Execute(context.Background(), map[string]interface{}{
		"language": "go",
	})
	if err == nil {
		t.Error("expected error for missing code, got nil")
	}
}

func TestCodeInterpreterToolUnsupportedLanguage(t *testing.T) {
	tool := NewCodeInterpreterTool()
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"language": "rust",
		"code":     "fn main() {}",
	})
	if err == nil {
		t.Error("expected error for unsupported language, got nil")
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<b>bold</b>", "bold"},
		{"no tags", "no tags"},
		{"<a href='url'>link</a> text", "link text"},
		{"", ""},
	}

	for _, tt := range tests {
		result := stripHTMLTags(tt.input)
		if result != tt.expected {
			t.Errorf("stripHTMLTags(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestCalculatorToolExecute(t *testing.T) {
	tool := NewCalculatorTool()
	
	tests := []struct {
		expr     string
		expected string
	}{
		{"2 + 2", "4"},
		{"10 / 2", "5"},
		{"10 * 5", "50"},
	}

	for _, tt := range tests {
		res, err := tool.Execute(context.Background(), map[string]interface{}{"expression": tt.expr})
		if err != nil {
			t.Errorf("expr %s failed: %v", tt.expr, err)
		}
		if !strings.Contains(res, tt.expected) {
			t.Errorf("expr %s expected %s, got %s", tt.expr, tt.expected, res)
		}
	}
}
