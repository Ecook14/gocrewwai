package core

import "context"

// Agent defines the minimal interface required for task execution and delegation.
// Moving this to a core package prevents circular dependencies between agents,
// protocols, and delegation packages.
type Agent interface {
	GetRole() string
	GetGoal() string
	GetMaxRPM() int
	SetMaxRPM(int)
	GetUsageMetrics() map[string]int
	GetToolCount() int
	Execute(ctx context.Context, input string, options map[string]interface{}) (interface{}, error)
}
