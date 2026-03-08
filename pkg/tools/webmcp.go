package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Ecook14/gocrewwai/pkg/protocols"
)

// WebMCPTool discovers and dynamically executes tools hosted via WebMCP.
type WebMCPTool struct {
	BaseTool
	client *protocols.WebMCPClient
}

func NewWebMCPTool() *WebMCPTool {
	return &WebMCPTool{
		client: protocols.NewWebMCPClient(),
	}
}

func (t *WebMCPTool) Name() string { return "WebMCPTool" }

func (t *WebMCPTool) Description() string {
	return "Interacts with remote WebMCP servers. Requires 'url' parameter to discover tools on a web page, and optionally 'tool_name' and 'tool_params' (JSON string) to execute a specific discovered tool."
}

func (t *WebMCPTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	url, ok := input["url"].(string)
	if !ok || url == "" {
		return "", fmt.Errorf("webmcp: missing 'url' parameter")
	}

	// 1. Discover the tools on the page
	declarations, err := t.client.DiscoverTools(ctx, url)
	if err != nil {
		return "", fmt.Errorf("webmcp: failed to discover tools: %w", err)
	}

	if len(declarations) == 0 {
		return "No WebMCP tools found on this page.", nil
	}

	toolNameInter, hasTool := input["tool_name"]
	if !hasTool {
		// Just return discovery
		out, _ := json.MarshalIndent(declarations, "", "  ")
		return fmt.Sprintf("Discovered %d tools on %s:\n%s", len(declarations), url, string(out)), nil
	}

	toolName, ok := toolNameInter.(string)
	if !ok {
		return "", fmt.Errorf("webmcp: 'tool_name' must be a string")
	}

	// 2. Find the requested tool
	var targetTool *protocols.WebMCPToolDeclaration
	for _, d := range declarations {
		if d.Name == toolName {
			targetTool = &d
			break
		}
	}

	if targetTool == nil {
		return "", fmt.Errorf("webmcp: tool '%s' not found on %s", toolName, url)
	}

	// 3. Execute
	var params map[string]interface{}
	if pStr, ok := input["tool_params"].(string); ok && pStr != "" {
		if err := json.Unmarshal([]byte(pStr), &params); err != nil {
			return "", fmt.Errorf("webmcp: failed to parse tool_params JSON: %w", err)
		}
	}

	result, err := t.client.ExecuteTool(ctx, *targetTool, params)
	if err != nil {
		return "", fmt.Errorf("webmcp: execution failed: %w", err)
	}

	return string(result), nil
}

func (t *WebMCPTool) RequiresReview() bool { return true }
