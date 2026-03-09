package tools

import (
	"context"

	"github.com/Ecook14/gocrewwai/pkg/protocols"
)

// ---------------------------------------------------------------------------
// MCP ↔ Crew-GO Tool Bridge
// ---------------------------------------------------------------------------

// WrapToolForMCP converts a Crew-GO Tool into an MCP tool definition and handler.
// This allows any Crew-GO tool to be served via an MCP server.
func WrapToolForMCP(tool Tool) (protocols.MCPToolDefinition, protocols.MCPToolHandler) {
	def := protocols.MCPToolDefinition{
		Name:        tool.Name(),
		Description: tool.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type":        "object",
					"description": "Tool input parameters",
				},
			},
		},
	}

	handler := func(ctx context.Context, params map[string]interface{}) (*protocols.MCPToolResult, error) {
		result, err := tool.Execute(ctx, params)
		if err != nil {
			return &protocols.MCPToolResult{
				Content: []protocols.MCPContent{{Type: "text", Text: err.Error()}},
				IsError: true,
			}, nil
		}
		return &protocols.MCPToolResult{
			Content: []protocols.MCPContent{{Type: "text", Text: result}},
		}, nil
	}

	return def, handler
}

// WrapMCPToolForGocrew wraps an MCP tool (via client) as a Gocrew Tool interface.
// This allows remote MCP tools to be used by Gocrew agents.
func WrapMCPToolForCrewGo(client *protocols.MCPClient, toolDef protocols.MCPToolDefinition) Tool {
	return &mcpToolAdapter{
		client: client,
		def:    toolDef,
	}
}

type mcpToolAdapter struct {
	client *protocols.MCPClient
	def    protocols.MCPToolDefinition
}

func (t *mcpToolAdapter) Name() string        { return "mcp:" + t.def.Name }
func (t *mcpToolAdapter) Description() string { return t.def.Description }
func (t *mcpToolAdapter) RequiresReview() bool { return false }

func (t *mcpToolAdapter) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	result, err := t.client.CallTool(ctx, protocols.MCPToolCall{
		Name:   t.def.Name,
		Params: input,
	})
	if err != nil {
		return "", err
	}

	// Concatenate text content
	var text string
	for _, c := range result.Content {
		if c.Type == "text" {
			text += c.Text
		}
	}
	return text, nil
}

// ArgsSchema returns nil for MCP tools as schema is managed by the remote server.
func (t *mcpToolAdapter) ArgsSchema() []ArgSchema { return nil }

// CacheFunction returns "" by default for MCP tools.
func (t *mcpToolAdapter) CacheFunction(input map[string]interface{}) string { return "" }

// RegisterAllToolsOnMCPServer registers all Gocrew tools from a registry onto an MCP server.
func RegisterAllToolsOnMCPServer(mcpServer *protocols.MCPServer, registry *ToolRegistry) {
	for _, tool := range registry.List() {
		def, handler := WrapToolForMCP(tool)
		mcpServer.RegisterTool(def, handler)
	}
}
