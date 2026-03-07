package tools

import "context"

// Tool represents the base capability interface that all tools must implement.
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input map[string]interface{}) (string, error)
}
