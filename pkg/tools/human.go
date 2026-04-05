package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

// AskHumanTool allows an agent to request information or approval from a human.
type AskHumanTool struct {
	BaseTool
	Enabled bool
}

func NewAskHumanTool(enabled bool) *AskHumanTool {
	t := &AskHumanTool{Enabled: enabled}
	t.NameValue = "AskHuman"
	t.DescriptionValue = "Use this tool to ask a human for missing information, clarification, or approval before proceeding with a critical action. Input should be a clear question."
	t.Schema = []ArgSchema{
		{
			Name:        "question",
			Type:        "string",
			Description: "The question or prompt to ask the human.",
			Required:    true,
		},
	}
	return t
}



func (t *AskHumanTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	if !t.Enabled {
		return "", fmt.Errorf("Human-in-the-loop (HITL) is currently disabled")
	}

	prompt, _ := input["question"].(string)
	if prompt == "" {
		prompt, _ = input["prompt"].(string)
	}

	fmt.Printf("\n🤔 AGENT IS ASKING: %s\n", prompt)
	fmt.Print("👤 YOUR RESPONSE: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read human input: %w", err)
	}

	return strings.TrimSpace(response), nil
}
