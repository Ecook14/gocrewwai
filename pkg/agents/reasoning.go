package agents

import (
	"context"
	"log/slog"
	"strings"

	"github.com/Ecook14/gocrewwai/pkg/events"
	"github.com/Ecook14/gocrewwai/pkg/llm"
)

// reasoningPlan represents the structured output of the reasoning phase.
type reasoningPlan struct {
	Understanding string   `json:"understanding"`
	Steps         []string `json:"steps"`
	Challenges    string   `json:"challenges"`
	Tools         []string `json:"tools"`
	Outcome       string   `json:"outcome"`
	Ready         bool     `json:"ready"`
}

// runReasoningLoop implements the Reflect-Evaluate-Refine lifecycle.
func (a *Agent) runReasoningLoop(ctx context.Context, messages []llm.Message, options map[string]interface{}) (string, error) {
	events.GlobalBus.Publish(events.Event{
		Type:      events.AgentReasoningStarted,
		Source:    a.Role,
	})

	maxAttempts := a.MaxReasoningAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3 // Default safety fallback
	}

	var finalPlan string
	currentMessages := append([]llm.Message{}, messages...)

	for i := 0; i < maxAttempts; i++ {
		// 1. Reflect & Plan
		prompt := a.I18N.Process(a.I18N.Retrieve("reasoning", "create_plan_prompt"), map[string]string{
			"role":            a.Role,
			"backstory":       a.Backstory,
			"goal":            a.Goal,
			"description":     messages[len(messages)-1].Content, // Last user message is the task description
			"expected_output": "",                                // Handled via task-level prompts usually
			"tools":           "",                                // Handled via system prompt usually
		})

		currentMessages = append(currentMessages, llm.Message{Role: "user", Content: prompt})
		
		response, err := a.LLM.Generate(ctx, currentMessages, llm.MapToOptions(options))
		if err != nil {
			return "", err
		}

		// 2. Evaluate
		isReady := strings.Contains(strings.ToUpper(response), "READY: I AM READY") || 
		           strings.Contains(strings.ToUpper(response), "READY: YES") ||
		           strings.Contains(strings.ToUpper(response), "STATUS: YES")

		if isReady {
			finalPlan = response
			if a.Verbose {
				defaultLogger.Info("🧠 Agent reasoning loop: READY", slog.String("role", a.Role), slog.Int("attempt", i+1))
			}
			break
		}

		// 3. Refine
		if a.Verbose {
			defaultLogger.Info("🧠 Agent reasoning loop: REFINING", slog.String("role", a.Role), slog.Int("attempt", i+1))
		}
		
		refineMsg := a.I18N.Process(a.I18N.Retrieve("reasoning", "refine_plan_prompt"), map[string]string{
			"role":         a.Role,
			"backstory":    a.Backstory,
			"goal":         a.Goal,
			"current_plan": response,
		})
		currentMessages = append(currentMessages, llm.Message{Role: "assistant", Content: response})
		currentMessages = append(currentMessages, llm.Message{Role: "user", Content: refineMsg})
		
		finalPlan = response // Fallback if loop ends
	}

	events.GlobalBus.Publish(events.Event{
		Type:      events.AgentReasoningCompleted,
		Source:    a.Role,
		Payload: map[string]interface{}{
			"plan": finalPlan,
		},
	})

	return finalPlan, nil
}
