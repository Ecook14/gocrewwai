package tasks

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"encoding/json"
	"github.com/Ecook14/crewai-go/pkg/agents"
	crewErrors "github.com/Ecook14/crewai-go/pkg/errors"
	"github.com/Ecook14/crewai-go/pkg/guardrails"
	"github.com/Ecook14/crewai-go/pkg/telemetry"
	"github.com/Ecook14/crewai-go/pkg/tools"
)

// Task translates the `class Task` python abstraction into idiomatic Go.
type Task struct {
	Description    string
	ExpectedOutput string
	Agent          *agents.Agent
	Tools          []tools.Tool
	AsyncExecution bool
	OutputFile     string      // Path to save the final task output (.md, .json, etc.)

	// Output Formatting
	OutputJSON   bool
	OutputPydan  interface{} // Deprecated: use OutputSchema
	OutputSchema interface{} // Target Go struct for validation
	MaxRetries   int         // Retries for schema validation failures

	// Execution Tracking
	Processed bool
	Output    interface{}

	// Advanced Quality-of-Life Mappings
	Context    []*Task // Strict outputs to pipe into this task's prompt
	HumanInput bool    // Blocks CLI execution for mid-flight approval/feedback

	// Guardrails validate task output before marking it as complete.
	Guardrails []guardrails.Guardrail

	// CallbackOnComplete fires after the task completes successfully.
	CallbackOnComplete func(result interface{})

	// Dependencies define explicit graph edges for DAG orchestration.
	Dependencies []*Task

	// Elite Tier: State Machine & Cyclic Logic
	// OutputCondition returns a key used to select the next task from NextPaths.
	OutputCondition func(result interface{}) string 
	
	// NextPaths maps condition keys to the successor tasks.
	NextPaths map[string]*Task

	// MaxCycles limits how many times this task can be re-executed in a cycle.
	MaxCycles int
	
	// Internal tracking
	CycleCount int
}

// Execute kicks off the Task lifecycle utilizing the bound Agent.
func (t *Task) Execute(ctx context.Context) (interface{}, error) {
	if t.Agent == nil {
		return nil, crewErrors.ErrNoAgent
	}

	baseDescription := t.Description

	// Publish telemetry event
	telemetry.GlobalBus.Publish(telemetry.Event{
		Type:      telemetry.EventTaskStarted,
		AgentRole: strings.Clone(t.Agent.Role),
		Payload: map[string]interface{}{
			"description": baseDescription,
		},
	})

	// 1. Append expected output hint
	if t.ExpectedOutput != "" {
		baseDescription += "\n\nEXPECTED OUTPUT FORMAT:\n" + t.ExpectedOutput
	}

	// 2. Process Task Dependency Contexts (Inject prior task outputs)
	if len(t.Context) > 0 {
		baseDescription += "\n\nCRITICAL CONTEXT FROM PREVIOUS TASKS:\n"
		for i, ctxTask := range t.Context {
			if ctxTask.Processed && ctxTask.Output != nil {
				baseDescription += fmt.Sprintf("--- Context Source %d ---\n%v\n", i+1, ctxTask.Output)
			}
		}
		baseDescription += "--------------------------\n"
	}

	// 3. Process Human-in-the-Loop (HITL) blocking
	if t.HumanInput {
		slog.Info("[🤖 HITL PAUSE] Agent is about to execute task", slog.String("role", t.Agent.Role), slog.String("description", baseDescription))
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
	if t.OutputSchema != nil {
		options["schema"] = t.OutputSchema
	} else if t.OutputPydan != nil {
		options["schema"] = t.OutputPydan
	}

	maxRetries := t.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1 // At least 1 attempt
	}

	var result interface{}
	var err error
	validator := &Validator{}

	for i := 0; i < maxRetries; i++ {
		result, err = t.Agent.Execute(ctx, baseDescription, options)
		if err == nil {
			// If we have a schema, and the result is a string, try to repair and unmarshal
			if t.OutputSchema != nil || t.OutputPydan != nil {
				if resultStr, ok := result.(string); ok {
					schema := t.OutputSchema
					if schema == nil {
						schema = t.OutputPydan
					}
					
					repaired := validator.RepairJSON(resultStr)
					validated, vErr := validator.ValidateSchema(repaired, schema)
					if vErr == nil {
						result = validated
						break
					}
					err = vErr // Set error for possible retry
				} else {
					// result is already a struct/map from structured generation
					break
				}
			} else {
				break
			}
		}
		
		if i < maxRetries-1 {
			slog.Warn("[⚠️ Task Retry] Validation failed, retrying", slog.Int("iter", i+1), slog.Int("max", maxRetries), slog.Any("error", err))
			continue
		}
	}

	if err != nil {
		return nil, err
	}

	// 4. Apply task-level guardrails
	if len(t.Guardrails) > 0 {
		if resultStr, ok := result.(string); ok {
			if gErr := guardrails.RunAll(t.Guardrails, resultStr); gErr != nil {
				return nil, fmt.Errorf("%w: %v", crewErrors.ErrGuardrailFailed, gErr)
			}
		}
	}

	t.Processed = true
	t.Output = result

	// 5. Auto-save output to file if specified
	if t.OutputFile != "" {
		var outputBytes []byte
		if t.OutputJSON {
			outputBytes, _ = json.MarshalIndent(result, "", "  ")
		} else {
			outputBytes = []byte(fmt.Sprintf("%v", result))
		}
		
		err := os.WriteFile(t.OutputFile, outputBytes, 0644)
		if err != nil {
			slog.Error("Failed to auto-save task output", slog.String("file", t.OutputFile), slog.Any("error", err))
		} else {
			slog.Info("Task output auto-saved", slog.String("file", t.OutputFile))
		}
	}

	// 4. Fire completion callback
	if t.CallbackOnComplete != nil {
		t.CallbackOnComplete(result)
	}

	// 5. Post-Execution HITL: Review and Edit Result
	if t.HumanInput {
		slog.Info("[🤖 HITL REVIEW] Agent finished task", slog.String("role", t.Agent.Role), slog.Any("result", result))
		fmt.Print("Press Enter to approve, or type 'edit' to modify the output: ")
		
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "edit" {
			fmt.Println("Please enter the new final output (type 'EOF' on a new line to finish):")
			var newOutput strings.Builder
			for {
				line, _ := reader.ReadString('\n')
				if strings.TrimSpace(line) == "EOF" {
					break
				}
				newOutput.WriteString(line)
			}
			result = strings.TrimSpace(newOutput.String())
			t.Output = result
			fmt.Println("[✅ Output Manually Overridden]")
		}
	}

	// Publish telemetry event
	telemetry.GlobalBus.Publish(telemetry.Event{
		Type:      telemetry.EventTaskFinished,
		AgentRole: strings.Clone(t.Agent.Role),
		Payload: map[string]interface{}{
			"result": result,
		},
	})

	return result, nil
}

// GetOutput securely translates the raw interface{} Output into a strongly typed pointer.
// This provides true type-safety for autonomous engine outputs.
func GetOutput[T any](t *Task) (*T, error) {
	if !t.Processed {
		return nil, fmt.Errorf("task has not been processed yet")
	}
	if t.Output == nil {
		return nil, fmt.Errorf("task output is nil")
	}

	// Case 1: The output is exactly *T (from LLM mapping).
	if typed, ok := t.Output.(*T); ok {
		return typed, nil
	}

	// Case 2: The output is exactly T.
	if typed, ok := t.Output.(T); ok {
		return &typed, nil
	}

	// Case 3: Try simple string assertion explicitly if T is string
	if s, ok := t.Output.(string); ok {
		var target T
		if anyTarget, ok := any(&target).(*string); ok {
			*anyTarget = s
			return &target, nil
		}
	}

	return nil, fmt.Errorf("task output is of type %T, expected *%T", t.Output, new(T))
}
