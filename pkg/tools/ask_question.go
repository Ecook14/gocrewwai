package tools

import (
	"context"
	"fmt"
)

// AskQuestionTool is a built-in tool that allows agents to ask questions to each other
// or explicitly solicit user feedback if configured.
type AskQuestionTool struct {
	Verbose bool
}

func NewAskQuestionTool() *AskQuestionTool {
	return &AskQuestionTool{Verbose: false}
}

func (t *AskQuestionTool) Name() string {
	return "Ask Question"
}

func (t *AskQuestionTool) Description() string {
	return "Useful to ask a question to another agent or request explicit feedback."
}

func (t *AskQuestionTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	question, ok := input["question"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'question' argument")
	}

	if t.Verbose {
		fmt.Printf("Tool [Ask Question]: Executing with question: %s\n", question)
	}

	// In a complete implementation, this would route back through the Crew orchestrator
	// via a shared context or channel to query a target agent.
	return "Received response to: " + question, nil
}
