// Package delegation provides the inter-agent task delegation engine for Gocrew.
// This enables agents to delegate sub-tasks to other agents within the same crew,
// mirroring Python CrewAI's agent delegation capabilities.
package delegation

import (
	"context"
	"fmt"
	"strings"
)

// Agent defines the minimal interface required for delegation targets.
// This avoids circular imports with the agents package.
type Agent interface {
	GetRole() string
	Execute(ctx context.Context, taskInput string, options map[string]interface{}) (interface{}, error)
}

// DelegateWorkTool implements tools.Tool and allows an agent to delegate work
// to a coworker agent. This is analogous to Python CrewAI's built-in delegation mechanism.
type DelegateWorkTool struct {
	Coworkers []Agent
}

func NewDelegateWorkTool(coworkers []Agent) *DelegateWorkTool {
	return &DelegateWorkTool{Coworkers: coworkers}
}

func (t *DelegateWorkTool) Name() string { return "DelegateWork" }

func (t *DelegateWorkTool) Description() string {
	roles := make([]string, 0, len(t.Coworkers))
	for _, cw := range t.Coworkers {
		roles = append(roles, cw.GetRole())
	}
	return fmt.Sprintf(
		"Delegate a specific task to one of the following coworkers: [%s]. "+
			"Input requires 'coworker' (the role name) and 'task' (the task description) and 'context' (any helpful context).",
		strings.Join(roles, ", "),
	)
}

func (t *DelegateWorkTool) RequiresReview() bool { return false }

func (t *DelegateWorkTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	coworkerRaw, ok := input["coworker"]
	if !ok {
		return "", fmt.Errorf("missing 'coworker' in input")
	}
	coworkerRole, ok := coworkerRaw.(string)
	if !ok {
		return "", fmt.Errorf("'coworker' must be a string")
	}

	taskRaw, ok := input["task"]
	if !ok {
		return "", fmt.Errorf("missing 'task' in input")
	}
	taskDesc, ok := taskRaw.(string)
	if !ok {
		return "", fmt.Errorf("'task' must be a string")
	}

	// Optional context
	taskContext := ""
	if ctxRaw, ok := input["context"]; ok {
		if ctxStr, ok := ctxRaw.(string); ok {
			taskContext = ctxStr
		}
	}

	// Find the coworker
	var target Agent
	for _, cw := range t.Coworkers {
		if strings.EqualFold(cw.GetRole(), coworkerRole) {
			target = cw
			break
		}
	}

	if target == nil {
		available := make([]string, 0, len(t.Coworkers))
		for _, cw := range t.Coworkers {
			available = append(available, cw.GetRole())
		}
		return "", fmt.Errorf("coworker '%s' not found. Available coworkers: [%s]",
			coworkerRole, strings.Join(available, ", "))
	}

	// Build the delegated task input
	fullInput := taskDesc
	if taskContext != "" {
		fullInput += "\n\nAdditional Context:\n" + taskContext
	}

	result, err := target.Execute(ctx, fullInput, nil)
	if err != nil {
		return "", fmt.Errorf("delegation to '%s' failed: %w", coworkerRole, err)
	}

	return fmt.Sprintf("%v", result), nil
}

// AskQuestionTool implements tools.Tool and allows an agent to ask a question
// to a coworker agent. Similar to DelegateWork but for information gathering.
type AskQuestionTool struct {
	Coworkers []Agent
}

func NewAskQuestionTool(coworkers []Agent) *AskQuestionTool {
	return &AskQuestionTool{Coworkers: coworkers}
}

func (t *AskQuestionTool) Name() string { return "AskQuestion" }

func (t *AskQuestionTool) Description() string {
	roles := make([]string, 0, len(t.Coworkers))
	for _, cw := range t.Coworkers {
		roles = append(roles, cw.GetRole())
	}
	return fmt.Sprintf(
		"Ask a specific question to one of the following coworkers: [%s]. "+
			"Input requires 'coworker' (the role name) and 'question' (the question to ask) and 'context' (any helpful context).",
		strings.Join(roles, ", "),
	)
}

func (t *AskQuestionTool) RequiresReview() bool { return false }

func (t *AskQuestionTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	coworkerRaw, ok := input["coworker"]
	if !ok {
		return "", fmt.Errorf("missing 'coworker' in input")
	}
	coworkerRole, ok := coworkerRaw.(string)
	if !ok {
		return "", fmt.Errorf("'coworker' must be a string")
	}

	questionRaw, ok := input["question"]
	if !ok {
		return "", fmt.Errorf("missing 'question' in input")
	}
	question, ok := questionRaw.(string)
	if !ok {
		return "", fmt.Errorf("'question' must be a string")
	}

	// Optional context
	taskContext := ""
	if ctxRaw, ok := input["context"]; ok {
		if ctxStr, ok := ctxRaw.(string); ok {
			taskContext = ctxStr
		}
	}

	// Find the coworker
	var target Agent
	for _, cw := range t.Coworkers {
		if strings.EqualFold(cw.GetRole(), coworkerRole) {
			target = cw
			break
		}
	}

	if target == nil {
		available := make([]string, 0, len(t.Coworkers))
		for _, cw := range t.Coworkers {
			available = append(available, cw.GetRole())
		}
		return "", fmt.Errorf("coworker '%s' not found. Available coworkers: [%s]",
			coworkerRole, strings.Join(available, ", "))
	}

	fullInput := "I need your help answering the following question: " + question
	if taskContext != "" {
		fullInput += "\n\nAdditional Context:\n" + taskContext
	}

	result, err := target.Execute(ctx, fullInput, nil)
	if err != nil {
		return "", fmt.Errorf("question to '%s' failed: %w", coworkerRole, err)
	}

	return fmt.Sprintf("%v", result), nil
}
