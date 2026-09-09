package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/tools"
)

func TestFileReadToolName(t *testing.T) {
	tool := tools.NewFileReadTool(t.TempDir())
	if tool.Name() != "FileReadTool" {
		t.Errorf("expected name 'FileReadTool', got '%s'", tool.Name())
	}
}

func TestFileReadToolMissingPath(t *testing.T) {
	tool := tools.NewFileReadTool(t.TempDir())
	_, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing file_path")
	}
}

func TestFileReadToolExecute(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := "hello world"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	tool := tools.NewFileReadTool(tmpDir)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"file_path": tmpFile,
	})
	if err != nil {
		t.Fatalf("FileReadTool.Execute failed: %v", err)
	}
	if result != content {
		t.Errorf("expected '%s', got '%s'", content, result)
	}
}

func TestFileWriteToolName(t *testing.T) {
	tool := tools.NewFileWriteTool(t.TempDir())
	if tool.Name() != "FileWriteTool" {
		t.Errorf("expected name 'FileWriteTool', got '%s'", tool.Name())
	}
}

func TestFileWriteToolMissingFields(t *testing.T) {
	tool := tools.NewFileWriteTool(t.TempDir())
	_, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing fields")
	}
}

func TestFileWriteToolExecute(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "output.txt")
	tool := tools.NewFileWriteTool(tmpDir)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"file_path": tmpFile,
		"content":   "written content",
	})
	if err != nil {
		t.Fatalf("FileWriteTool.Execute failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result message")
	}
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "written content" {
		t.Errorf("expected 'written content', got '%s'", string(data))
	}
}

func TestCodeInterpreterToolName(t *testing.T) {
	tool := tools.NewCodeInterpreterTool()
	if tool.Name() != "CodeInterpreterTool" {
		t.Errorf("expected name 'CodeInterpreterTool', got '%s'", tool.Name())
	}
}

func TestGoogleSheetsToolName(t *testing.T) {
	tool := tools.NewGoogleSheetsTool("")
	if tool.Name() != "GoogleSheetsTool" {
		t.Errorf("expected name 'GoogleSheetsTool', got '%s'", tool.Name())
	}
}

func TestLinearToolName(t *testing.T) {
	tool := tools.NewLinearTool("")
	if tool.Name() != "LinearTool" {
		t.Errorf("expected name 'LinearTool', got '%s'", tool.Name())
	}
}

func TestTwilioToolName(t *testing.T) {
	tool := tools.NewTwilioTool("", "")
	if tool.Name() != "TwilioTool" {
		t.Errorf("expected name 'TwilioTool', got '%s'", tool.Name())
	}
}

func TestBraveSearchToolName(t *testing.T) {
	tool := tools.NewBraveSearchTool("")
	if tool.Name() != "BraveSearchTool" {
		t.Errorf("expected name 'BraveSearchTool', got '%s'", tool.Name())
	}
}

func TestToolExecuteWithBadAction(t *testing.T) {
	tool := tools.NewGoogleSheetsTool("")
	_, err := tool.Execute(context.Background(), map[string]interface{}{"action": "bad"})
	if err == nil {
		t.Error("expected error for bad action")
	}
}

func TestGoogleSheetsToolDescription(t *testing.T) {
	tool := tools.NewGoogleSheetsTool("")
	expected := "Interacts with Google Sheets."
	if tool.Description()[:len(expected)] != expected {
		t.Errorf("expected description starting with '%s', got '%s'", expected, tool.Description())
	}
}

func TestFileEditToolExecute(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_edit.txt")
	content := "Line 1\nLine 2\nLine 3"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	tool := tools.NewFileEditTool(tmpDir)
	input := map[string]interface{}{
		"file_path":       tmpFile,
		"target_text":     "Line 2",
		"replacement_text": "Updated Line 2",
	}
	_, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("FileEditTool.Execute failed: %v", err)
	}
	newData, _ := os.ReadFile(tmpFile)
	if string(newData) == content {
		t.Error("expected file to be updated")
	}
}

func TestFileEditToolMissingFields(t *testing.T) {
	tool := tools.NewFileEditTool(t.TempDir())
	_, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing fields")
	}
}
