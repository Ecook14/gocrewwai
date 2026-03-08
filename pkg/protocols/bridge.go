package protocols

import (
	"context"

	"github.com/Ecook14/crewai-go/pkg/tools"
)

// ---------------------------------------------------------------------------
// MCP ↔ Crew-GO Tool Bridge
// ---------------------------------------------------------------------------

// WrapToolForMCP converts a Crew-GO Tool into an MCP tool definition and handler.
// This allows any Crew-GO tool to be served via an MCP server.
func WrapToolForMCP(tool tools.Tool) (MCPToolDefinition, MCPToolHandler) {
	def := MCPToolDefinition{
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

	handler := func(ctx context.Context, params map[string]interface{}) (*MCPToolResult, error) {
		result, err := tool.Execute(ctx, params)
		if err != nil {
			return &MCPToolResult{
				Content: []MCPContent{{Type: "text", Text: err.Error()}},
				IsError: true,
			}, nil
		}
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: result}},
		}, nil
	}

	return def, handler
}

// WrapMCPToolForCrewGo wraps an MCP tool (via client) as a Crew-GO Tool interface.
// This allows remote MCP tools to be used by Crew-GO agents.
func WrapMCPToolForCrewGo(client *MCPClient, toolDef MCPToolDefinition) tools.Tool {
	return &mcpToolAdapter{
		client: client,
		def:    toolDef,
	}
}

type mcpToolAdapter struct {
	client *MCPClient
	def    MCPToolDefinition
}

func (t *mcpToolAdapter) Name() string        { return "mcp:" + t.def.Name }
func (t *mcpToolAdapter) Description() string { return t.def.Description }
func (t *mcpToolAdapter) RequiresReview() bool { return false }

func (t *mcpToolAdapter) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	result, err := t.client.CallTool(ctx, MCPToolCall{
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

// RegisterAllToolsOnMCPServer registers all Crew-GO tools from a registry onto an MCP server.
func RegisterAllToolsOnMCPServer(mcpServer *MCPServer, registry *tools.ToolRegistry) {
	for _, tool := range registry.List() {
		def, handler := WrapToolForMCP(tool)
		mcpServer.RegisterTool(def, handler)
	}
}
