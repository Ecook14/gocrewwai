package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/Ecook14/crewai-go/pkg/llm"
	"github.com/Ecook14/crewai-go/pkg/tools"
)

var defaultLogger = slog.New(slog.NewTextHandler(os.Stdout, nil))

// Agent translates the `class Agent` python abstraction into idiomatic Go.
type Agent struct {
	Role      string
	Goal      string
	Backstory string
	Verbose   bool

	LLM   llm.Client
	Tools []tools.Tool

	// Execution context limits
	MaxIter              int
	MaxRetryLimit        int
	MaxRPM               int
	RespectContextWindow bool

	// StepCallback allows developers to hook into the execution loop for UI streaming
	StepCallback func(step map[string]interface{})
}

// Execute handles running a task, converting from `async def execute_task()`.
// This forms the core logic layer for Agent behavior mapping.
// Provides structured outputs if mapped via Options array.
func (a *Agent) Execute(ctx context.Context, taskInput string, options map[string]interface{}) (interface{}, error) {
	if a.LLM == nil {
		return "Task executed successfully by " + a.Role, nil
	}

	if a.Verbose {
		preview := taskInput
		if len(preview) > 50 {
			preview = preview[:47] + "..."
		}
		defaultLogger.Info("Agent executing", slog.String("role", a.Role), slog.String("task", preview))
	}

	if a.StepCallback != nil {
		a.StepCallback(map[string]interface{}{
			"role":  a.Role,
			"phase": "starting",
			"input": taskInput,
		})
	}

	// Format the ReAct Tooling System Prompt
	toolDescriptions := ""
	for _, t := range a.Tools {
		toolDescriptions += fmt.Sprintf("- %s: %s\n", t.Name(), t.Description())
	}

	systemPrompt := fmt.Sprintf(`You are %s. %s
Your goal is: %s

You have access to the following tools:
%s

To use a tool, you MUST reply with a pure JSON object in this exact format:
{"tool": "ToolName", "input": "input payload string"}

Once you have gathered all necessary information and are ready to provide the final answer, do NOT return a tool JSON. Simply output your final answer text natively.`, a.Role, a.Backstory, a.Goal, toolDescriptions)

	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: taskInput},
	}

	// ---------------------------------------------------------
	// PHASE 19: The ReAct Autonomous Tool-Calling Loop Engine
	// ---------------------------------------------------------
	maxLoops := a.MaxIter
	if maxLoops <= 0 {
		maxLoops = 15 // Default safety guard against infinite loops
	}

	for i := 0; i < maxLoops; i++ {

		// Check global task cancellation before expensive LLM hits
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if a.StepCallback != nil {
			a.StepCallback(map[string]interface{}{"role": a.Role, "phase": "thinking", "iteration": i + 1})
		}

		// Hit the LLM
		var response interface{}
		var err error

		if options != nil && options["schema"] != nil {
			// If a strict final schema is requested, we assume this is the final iteration and disable autonomous looping natively
			// (as forcing the LLM to output a struct breaks the {"tool": "x"} abstraction).
			mappedSchema := options["schema"]
			response, err = a.LLM.GenerateStructured(ctx, messages, mappedSchema, options)
			return response, err
		} else {
			response, err = a.LLM.Generate(ctx, messages, options)
		}

		if err != nil {
			return nil, fmt.Errorf("llm generation failed: %w", err)
		}

		responseText, ok := response.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected unstructured non-string LLM output")
		}

		// ReAct Tool Parsing Logics (Looking for the {"tool": ...} block)
		var toolReq struct {
			Tool  string `json:"tool"`
			Input string `json:"input"`
		}

		// Fast-fail JSON parse check to see if the LLM wants a tool or is giving the final answer
		if err := json.Unmarshal([]byte(strings.TrimSpace(responseText)), &toolReq); err == nil && toolReq.Tool != "" {
			var activeTool tools.Tool
			for _, t := range a.Tools {
				if t.Name() == toolReq.Tool {
					activeTool = t
					break
				}
			}

			if activeTool != nil {
				if a.Verbose {
					defaultLogger.Info("🔨 Agent Tool Triggered", slog.String("agent", a.Role), slog.String("tool", toolReq.Tool), slog.String("input", toolReq.Input))
				}

				if a.StepCallback != nil {
					a.StepCallback(map[string]interface{}{"role": a.Role, "phase": "tool_execution", "tool": toolReq.Tool})
				}

				// Execute the physically bound Go interface Tool
				toolResult, toolErr := activeTool.Run(toolReq.Input)
				
				var observation string
				if toolErr != nil {
					observation = fmt.Sprintf("Tool Execution Error: %v", toolErr)
				} else {
					observation = fmt.Sprintf("Observation: %v", toolResult)
				}

				// Append the history so the LLM remembers what it did!
				messages = append(messages, llm.Message{Role: "assistant", Content: responseText})
				messages = append(messages, llm.Message{Role: "user", Content: observation})
				continue // LOOP BACK AROUND 🔄
			} else {
				// The LLM hallucinated a tool name that doesn't exist
				messages = append(messages, llm.Message{Role: "assistant", Content: responseText})
				messages = append(messages, llm.Message{Role: "user", Content: fmt.Sprintf("Observation Error: The tool '%s' does not exist. Available tools are: \n%s", toolReq.Tool, toolDescriptions)})
				continue
			}
		}

		// If the JSON parsing fails, or no tool was requested, we assume the LLM has output the final human-readable answer.
		if a.Verbose {
			defaultLogger.Info("✅ Agent loop finalized", slog.String("agent", a.Role))
		}
		
		return responseText, nil
	}

	return nil, fmt.Errorf("agent '%s' hit max iteration limit (%d) without providing a final answer", a.Role, maxLoops)
}
