package tools

import (
	"context"
	"fmt"
	"log/slog"
)

// FeedbackProvider defines an interface for soliciting input.
type FeedbackProvider interface {
	Ask(question string) (string, error)
}

// AskQuestionTool is a built-in tool that allows agents to ask questions to each other
// or explicitly solicit user feedback if configured.
type AskQuestionTool struct {
	BaseTool
	Verbose bool
}

func NewAskQuestionTool() *AskQuestionTool {
	return &AskQuestionTool{
		BaseTool: BaseTool{
			NameValue:        "Ask Question",
			DescriptionValue: "Useful to ask a question to another agent or request explicit feedback.",
		},
		Verbose: false,
	}
}


func (t *AskQuestionTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	question, ok := input["question"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'question' argument")
	}

	if t.Verbose {
		slog.Info("Tool [Ask Question]: Executing with question", slog.String("question", question))
	}

	// Advanced Level: Try to get a feedback provider from context
	if provider, ok := ctx.Value("feedback_provider").(FeedbackProvider); ok {
		return provider.Ask(question)
	}

	// Default fallback for the demo
	return "No feedback provider configured. Received response to: " + question, nil
}

func (t *AskQuestionTool) RequiresReview() bool { return false }
