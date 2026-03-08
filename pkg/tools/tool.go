package tools

import "context"

// Tool represents the base capability interface that all tools must implement.
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input map[string]interface{}) (string, error)
	// RequiresReview returns true if this tool should pause for human approval.
	RequiresReview() bool
}

// BaseTool provides a reusable foundation for custom tool implementations.
type BaseTool struct {
	NameValue        string
	DescriptionValue string
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
