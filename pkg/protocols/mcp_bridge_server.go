package protocols

import (
	"context"
	"fmt"
	"github.com/Ecook14/gocrewwai/pkg/core"
)

// AgentMCPBridge provides helpers to expose Gocrew Agents as MCP Tools.
type AgentMCPBridge struct {
	Server *MCPServer
}

func NewAgentMCPBridge(server *MCPServer) *AgentMCPBridge {
	return &AgentMCPBridge{Server: server}
}

// ExposeAgent registers an agent's execution capability as an MCP tool.
func (b *AgentMCPBridge) ExposeAgent(agent core.Agent, role string, goal string) {
	name := fmt.Sprintf("ask_%s", sanitizeName(role))
	
	def := MCPToolDefinition{
		Name:        name,
		Description: fmt.Sprintf("Consult the %s agent. Goal: %s", role, goal),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task": map[string]interface{}{
					"type":        "string",
					"description": "The specific task or question for the agent.",
				},
			},
			"required": []string{"task"},
		},
	}

	b.Server.RegisterTool(def, func(ctx context.Context, params map[string]interface{}) (*MCPToolResult, error) {
		task, _ := params["task"].(string)
		if task == "" {
			return nil, fmt.Errorf("task parameter is required")
		}

		// Execute the agent
		result, err := agent.Execute(ctx, task, nil)
		if err != nil {
			return nil, err
		}

		return &MCPToolResult{
			Content: []MCPContent{
				{Type: "text", Text: fmt.Sprintf("%v", result)},
			},
		}, nil
	})
}

func sanitizeName(name string) string {
	// Simple sanitizer for MCP tool names
	return name // In practice, replace spaces with underscores, etc.
}
