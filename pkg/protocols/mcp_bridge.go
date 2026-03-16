package protocols

import (
	"context"
	"fmt"
	"github.com/Ecook14/gocrewwai/pkg/tools"
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

// WrapMCPToolForCrewGo wraps an MCP tool (via client) as a Gocrew Tool interface.
// This allows remote MCP tools to be used by Gocrew agents.
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

// ArgsSchema returns the translated schema from the remote MCP server.
func (t *mcpToolAdapter) ArgsSchema() []tools.ArgSchema {
	if t.def.InputSchema == nil {
		return nil
	}
	
	properties, ok := t.def.InputSchema["properties"].(map[string]interface{})
	if !ok {
		return nil
	}

	required, _ := t.def.InputSchema["required"].([]interface{})
	isRequired := func(name string) bool {
		for _, r := range required {
			if s, ok := r.(string); ok && s == name {
				return true
			}
		}
		return false
	}

	var schema []tools.ArgSchema
	for name, details := range properties {
		prop, _ := details.(map[string]interface{})
		arg := tools.ArgSchema{
			Name:        name,
			Description: fmt.Sprintf("%v", prop["description"]),
			Required:    isRequired(name),
			Type:        fmt.Sprintf("%v", prop["type"]),
		}
		schema = append(schema, arg)
	}
	return schema
}

// CacheFunction returns "" by default for MCP tools.
func (t *mcpToolAdapter) CacheFunction(input map[string]interface{}) string { return "" }

// WrapMCPResource wraps a discovered MCP resource into a specialized reader tool.
func WrapMCPResource(client *MCPClient, def MCPResourceDefinition) tools.Tool {
	return &mcpResourceAdapter{
		client: client,
		def:    def,
	}
}

type mcpResourceAdapter struct {
	client *MCPClient
	def    MCPResourceDefinition
}

func (t *mcpResourceAdapter) Name() string {
	return fmt.Sprintf("read_%s", t.def.Name)
}

func (t *mcpResourceAdapter) Description() string {
	return fmt.Sprintf("Reads the content of the resource: %s. Use this to inspect static data, logs, or schemas.", t.def.Description)
}

func (t *mcpResourceAdapter) RequiresReview() bool { return false }

func (t *mcpResourceAdapter) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	contents, err := t.client.ReadResource(ctx, t.def.URI)
	if err != nil {
		return "", err
	}

	var result string
	for _, c := range contents {
		if c.Text != "" {
			result += c.Text + "\n"
		} else if c.Blob != "" {
			result += "[Binary Data: " + c.Blob + "]\n"
		}
	}
	return result, nil
}

func (t *mcpResourceAdapter) ArgsSchema() []tools.ArgSchema { return nil }
func (t *mcpResourceAdapter) CacheFunction(input map[string]interface{}) string { return "" }

// RegisterAllToolsOnMCPServer registers all Gocrew tools from a registry onto an MCP server.
func RegisterAllToolsOnMCPServer(mcpServer *MCPServer, registry *tools.ToolRegistry) {
	for _, tool := range registry.List() {
		def, handler := WrapToolForMCP(tool)
		mcpServer.RegisterTool(def, handler)
	}
}
