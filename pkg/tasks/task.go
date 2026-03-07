package tasks

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Ecook14/crewai-go/pkg/agents"
	"github.com/Ecook14/crewai-go/pkg/tools"
)

// Task translates the `class Task` python abstraction into idiomatic Go.
type Task struct {
	Description    string
	ExpectedOutput string
	Agent          *agents.Agent
	Tools          []tools.Tool
	AsyncExecution bool

	// Output Formatting
	OutputJSON  bool
	OutputPydan interface{} // Map to struct tags in Go

	// Execution Tracking
	Processed bool
	Output    interface{}

	// Advanced Quality-of-Life Mappings
	Context    []*Task // Strict outputs to pipe into this task's prompt
	HumanInput bool    // Blocks CLI execution for mid-flight approval/feedback
}

// Execute kicks off the Task lifecycle utilizing the bound Agent.
func (t *Task) Execute(ctx context.Context) (interface{}, error) {
	if t.Agent == nil {
		return nil, context.Canceled // Require agent mapping
	}

	baseDescription := t.Description

	// 1. Process Task Dependency Contexts (Inject prior task outputs)
	if len(t.Context) > 0 {
		baseDescription += "\n\nCRITICAL CONTEXT FROM PREVIOUS TASKS:\n"
		for i, ctxTask := range t.Context {
			if ctxTask.Processed && ctxTask.Output != nil {
				baseDescription += fmt.Sprintf("--- Context Source %d ---\n%v\n", i+1, ctxTask.Output)
			}
		}
		baseDescription += "--------------------------\n"
	}

	// 2. Process Human-in-the-Loop (HITL) blocking
	if t.HumanInput {
		fmt.Printf("\n[🤖 HITL PAUSE] Agent '%s' is about to execute task:\n%s\n", t.Agent.Role, baseDescription)
		fmt.Print("Please provide feedback or press Enter to approve as-is: ")

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err == nil {
			input = strings.TrimSpace(input)
			if input != "" {
				baseDescription += fmt.Sprintf("\n\nHUMAN FEEDBACK OVERRIDE: %s", input)
				fmt.Println("[✅ Feedback Injected]")
			} else {
				fmt.Println("[✅ Approved]")
			}
		}
	}

	options := make(map[string]interface{})
	if t.OutputPydan != nil {
		options["schema"] = t.OutputPydan
	}

	result, err := t.Agent.Execute(ctx, baseDescription, options)
	if err == nil {
		t.Processed = true
		t.Output = result
	}
	return result, err
}
