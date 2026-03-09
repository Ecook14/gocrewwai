package tools

import "context"

// ArgSchema describes a single input argument for a tool.
type ArgSchema struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "string", "int", "bool", "float", etc.
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// Tool represents the base capability interface that all tools must implement.
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input map[string]interface{}) (string, error)
	// RequiresReview returns true if this tool should pause for human approval.
	RequiresReview() bool
	// ArgsSchema returns the input argument schema for the tool.
	// Returning nil means no schema validation is enforced.
	ArgsSchema() []ArgSchema
	// CacheFunction returns a custom cache key for a given input, or "" to skip caching.
	// If nil/empty, the default caching mechanism is used.
	CacheFunction(input map[string]interface{}) string
}

// ToolAsync extends Tool with asynchronous execution support.
type ToolAsync interface {
	Tool
	ExecuteAsync(ctx context.Context, input map[string]interface{}) (<-chan string, <-chan error)
}

// BaseTool provides a reusable foundation for custom tool implementations.
type BaseTool struct {
	NameValue        string
	DescriptionValue string
	Schema           []ArgSchema // Optional: Define args schema
}

func (t BaseTool) Name() string {
	return t.NameValue
}

func (t BaseTool) Description() string {
	return t.DescriptionValue
}

func (t BaseTool) RequiresReview() bool {
	return false
}

// ArgsSchema returns the tool's input argument schema.
func (t BaseTool) ArgsSchema() []ArgSchema {
	return t.Schema
}

// CacheFunction returns "" by default (use standard caching).
func (t BaseTool) CacheFunction(input map[string]interface{}) string {
	return ""
}
