package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// AskHumanTool allows an agent to request information or approval from a human.
type AskHumanTool struct {
	BaseTool
	Enabled bool
	// In/Out back the interactive prompt. Nil means os.Stdin/os.Stdout,
	// so CLI behavior is unchanged; tests inject buffers.
	In  io.Reader
	Out io.Writer
}

// WithHumanInput sets a custom input stream for the prompt.
func WithHumanInput(r io.Reader) func(*AskHumanTool) {
	return func(t *AskHumanTool) { t.In = r }
}

// WithHumanOutput sets a custom output stream for the prompt.
func WithHumanOutput(w io.Writer) func(*AskHumanTool) {
	return func(t *AskHumanTool) { t.Out = w }
}

func NewAskHumanTool(enabled bool, opts ...func(*AskHumanTool)) *AskHumanTool {
	t := &AskHumanTool{Enabled: enabled}
	for _, opt := range opts {
		opt(t)
	}
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

func (t *AskHumanTool) humanIn() io.Reader {
	if t.In != nil {
		return t.In
	}
	return os.Stdin
}

func (t *AskHumanTool) humanOut() io.Writer {
	if t.Out != nil {
		return t.Out
	}
	return os.Stdout
}

func (t *AskHumanTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	if !t.Enabled {
		return "", fmt.Errorf("Human-in-the-loop (HITL) is currently disabled")
	}

	prompt, _ := input["question"].(string)
	if prompt == "" {
		prompt, _ = input["prompt"].(string)
	}

	fmt.Fprintf(t.humanOut(), "\n🤔 AGENT IS ASKING: %s\n", prompt)
	fmt.Fprint(t.humanOut(), "👤 YOUR RESPONSE: ")

	reader := bufio.NewReader(t.humanIn())
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read human input: %w", err)
	}

	return strings.TrimSpace(response), nil
}
