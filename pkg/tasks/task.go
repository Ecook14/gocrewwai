package tasks

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	//"encoding/json"
	"time"
	"github.com/Ecook14/gocrewwai/pkg/agents"
	crewErrors "github.com/Ecook14/gocrewwai/pkg/errors"
	"github.com/Ecook14/gocrewwai/pkg/guardrails"
	"github.com/Ecook14/gocrewwai/pkg/events"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
	"github.com/Ecook14/gocrewwai/pkg/tools"
	"path/filepath"
)

// TaskConfig defines the parameters for creating a new Task in a declarative style.
type TaskConfig struct {
	Name               string
	Description        string
	ExpectedOutput     string
	Agent              *agents.Agent
	AgentRole          string // Optional: For late binding
	Tools              []tools.Tool
	AsyncExecution     bool
	OutputFile         string
	CreateDirectory    bool
	OutputJSON         interface{} // Struct to unmarshal the result into
	Markdown           bool
	OutputSchema       string      // JSON schema for validation
	MaxRetries         int
	GuardrailMaxRetries int
	Context            []*Task
	HumanInput         bool
	Guardrails         []guardrails.Guardrail
	CallbackOnComplete func(result interface{})
	Timeout            time.Duration
}

// Task translates the `class Task` python abstraction into idiomatic Go.
type Task struct {
	Name           string `json:"name,omitempty"`
	Description    string `json:"description"`
	ExpectedOutput string `json:"expected_output"`
	Agent          *agents.Agent `json:"-"`
	AgentRole      string `json:"agent_role"` // For late binding, especially from UI
	Tools          []tools.Tool `json:"-"`
	AsyncExecution bool `json:"-"`
	OutputFile     string `json:"-"` // Path to save the final task output (.md, .json, etc.)
	CreateDirectory bool  `json:"-"`

	// Output Formatting
	OutputJSON   interface{} `json:"-"`
	Markdown     bool        `json:"-"`
	OutputPydan  interface{} `json:"-"` // Deprecated
	OutputSchema string      `json:"-"` // JSON Schema string
	MaxRetries   int         `json:"-"` // Retries for schema validation failures
	GuardrailMaxRetries int  `json:"-"` 

	// Execution Tracking
	Processed bool        `json:"processed"`
	Failed    bool        `json:"failed"`
	Error     error       `json:"-"`
	Output    interface{} `json:"output"`

	// Advanced Quality-of-Life Mappings
	Context    []*Task `json:"-"` // Strict outputs to pipe into this task's prompt
	HumanInput bool    `json:"-"` // Blocks CLI execution for mid-flight approval/feedback

	// Guardrails validate task output before marking it as complete.
	Guardrails []guardrails.Guardrail `json:"-"`

	// CallbackOnComplete fires after the task completes successfully.
	CallbackOnComplete func(result interface{}) `json:"-"`

	// Dependencies define explicit graph edges for DAG orchestration.
	Dependencies []*Task `json:"-"`

	// Elite Tier: State Machine & Cyclic Logic
	// OutputCondition returns a key used to select the next task from NextPaths.
	OutputCondition func(result interface{}) string  `json:"-"`
	
	// NextPaths maps condition keys to the successor tasks.
	NextPaths map[string]*Task `json:"-"`

	// MaxCycles limits how many times this task can be re-executed in a cycle.
	MaxCycles int `json:"-"`
	
	// Timeout enforces a maximum duration for the task execution.
	Timeout time.Duration `json:"timeout"`

	// Internal tracking
	CycleCount int `json:"-"`
}

// New creates a new Task using a declarative configuration struct.
func New(cfg TaskConfig) *Task {
	return &Task{
		Name:               cfg.Name,
		Description:        cfg.Description,
		ExpectedOutput:     cfg.ExpectedOutput,
		Agent:              cfg.Agent,
		AgentRole:          cfg.AgentRole,
		Tools:              cfg.Tools,
		AsyncExecution:     cfg.AsyncExecution,
		OutputFile:         cfg.OutputFile,
		CreateDirectory:    cfg.CreateDirectory,
		OutputJSON:         cfg.OutputJSON,
		Markdown:           cfg.Markdown,
		OutputSchema:       cfg.OutputSchema,
		MaxRetries:         cfg.MaxRetries,
		GuardrailMaxRetries: cfg.GuardrailMaxRetries,
		Context:            cfg.Context,
		HumanInput:         cfg.HumanInput,
		Guardrails:         cfg.Guardrails,
		CallbackOnComplete: cfg.CallbackOnComplete,
		Timeout:            cfg.Timeout,
	}
}

// Execute kicks off the Task lifecycle utilizing the bound Agent.
func (t *Task) Execute(ctx context.Context) (interface{}, error) {
	if t.Agent == nil {
		return nil, crewErrors.ErrNoAgent
	}

	// 1. Enforce Task-level Timeout
	if t.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.Timeout)
		defer cancel()
	}

	// 2. Start Telemetry Span & Check Pause Status
	ctx, span := telemetry.StartSpan(ctx, "Task.Execute")
	if span != nil {
		defer span.End()
	}

	if err := telemetry.GlobalExecutionController.WaitIfPaused(ctx); err != nil {
		return nil, err
	}

	baseDescription := t.Description

	// Publish system event
	events.GlobalBus.Publish(events.Event{
		Type:   events.TaskStarted,
		Source: t.Name,
		Payload: map[string]interface{}{
			"description": baseDescription,
			"agent":       t.Agent.Role,
		},
	})

	// 1. Append expected output hint
	if t.ExpectedOutput != "" {
		baseDescription += "\n\nEXPECTED OUTPUT FORMAT:\n" + t.ExpectedOutput
	}

	if t.Markdown {
		baseDescription += "\n\nCRITICAL: You MUST format your final answer using valid Markdown syntax (headers, lists, code blocks, etc.)."
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
	if t.OutputSchema != "" {
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

	if t.OutputSchema != "" {
		options["schema"] = t.OutputSchema
	} else if t.OutputJSON != nil {
		options["schema"] = t.OutputJSON
	}

	for i := 0; i < maxRetries; i++ {
		result, err = t.Agent.Execute(ctx, baseDescription, options)
		if err == nil {
			// If we have a target struct (OutputJSON), unmarshal and validate
			if t.OutputJSON != nil {
				if resultStr, ok := result.(string); ok {
					repaired := validator.RepairJSON(resultStr)
					validated, vErr := validator.ValidateSchema(repaired, t.OutputJSON)
					if vErr == nil {
						result = validated
						break
					}
					err = vErr 
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

	// 4. Apply task-level guardrails with retries
	if len(t.Guardrails) > 0 {
		gRetries := t.GuardrailMaxRetries
		if gRetries <= 0 {
			gRetries = 1
		}
		
		for gr := 0; gr < gRetries; gr++ {
			if resultStr, ok := result.(string); ok {
				if gErr := guardrails.RunAll(t.Guardrails, resultStr); gErr != nil {
					if gr < gRetries-1 {
						slog.Warn("[⚠️ Guardrail Retry] Guardrail failed, re-executing task", slog.Int("iter", gr+1), slog.Any("error", gErr))
						result, err = t.Agent.Execute(ctx, baseDescription, options)
						if err != nil {
							return nil, err
						}
						continue
					}
					return nil, fmt.Errorf("%w: %v", crewErrors.ErrGuardrailFailed, gErr)
				}
			}
			break
		}
	}

	t.Processed = true
	t.Output = result

	// 5. Auto-save output to file if specified
	if t.OutputFile != "" {
		if t.CreateDirectory {
			dir := filepath.Dir(t.OutputFile)
			if err := os.MkdirAll(dir, 0755); err != nil {
				slog.Error("Failed to create directory for task output", slog.String("dir", dir), slog.Any("error", err))
			}
		}

		var outputBytes []byte
		if t.OutputJSON != nil {
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

	// Publish system event
	events.GlobalBus.Publish(events.Event{
		Type:   events.TaskCompleted,
		Source: t.Name,
		Payload: map[string]interface{}{
			"result": result,
			"agent":  t.Agent.Role,
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

func (t *Task) GetDescription() string { return t.Description }
func (t *Task) GetAgentRole() string   { return t.AgentRole }
func (t *Task) SetOutput(out interface{}) { t.Output = out }
func (t *Task) SetError(err error)       { t.Error = err }
func (t *Task) SetProcessed(p bool)      { t.Processed = p }
