package tools

import (
	"context"
	"testing"
)

func TestBrowserToolName(t *testing.T) {
	tool := NewBrowserTool()
	if tool.Name() != "BrowserControl" {
		t.Errorf("expected name 'BrowserControl', got '%s'", tool.Name())
	}
}

func TestBrowserToolRequiresReview(t *testing.T) {
	tool := NewBrowserTool()
	if !tool.RequiresReview() {
		t.Error("expected BrowserTool to require review")
	}
}

func TestWASMSandboxToolName(t *testing.T) {
	ctx := context.Background()
	tool := NewWASMSandboxTool(ctx)
	if tool.Name() != "WASMSandboxTool" {
		t.Errorf("expected name 'WASMSandboxTool', got '%s'", tool.Name())
	}
}

func TestWASMSandboxToolMissingPath(t *testing.T) {
	ctx := context.Background()
	tool := NewWASMSandboxTool(ctx)
	_, err := tool.Execute(ctx, map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing path, got nil")
	}
}

func TestGitHubToolConfiguration(t *testing.T) {
	// Test without token and without env (assuming GITHUB_TOKEN is not set in test env)
	tool := NewGitHubTool("")
	if tool != nil {
		t.Error("expected NewGitHubTool to return nil when no token or env is provided")
	}

	// Test with explicit token
	tool = NewGitHubTool("test-token")
	if tool == nil {
		t.Fatal("expected NewGitHubTool to return instance with explicit token")
	}
	if tool.Name() != "GitHubTool" {
		t.Errorf("expected name 'GitHubTool', got '%s'", tool.Name())
	}
}

func TestSlackToolConfiguration(t *testing.T) {
	tool := NewSlackTool("")
	if tool != nil {
		t.Error("expected NewSlackTool to return nil when no token or env is provided")
	}

	tool = NewSlackTool("xoxb-test")
	if tool == nil {
		t.Fatal("expected NewSlackTool to return instance")
	}
}

func TestArxivToolName(t *testing.T) {
	tool := NewArxivTool()
	if tool.Name() != "ArxivTool" {
		t.Errorf("expected name 'ArxivTool', got '%s'", tool.Name())
	}
}

func TestArxivToolMissingInput(t *testing.T) {
	tool := NewArxivTool()
	_, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing query")
	}
}
