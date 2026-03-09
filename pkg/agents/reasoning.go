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
		prompt := `Analyze the task and tools provided. Create a detailed reasoning plan in the following structure:
- Understanding of the task: [Your interpretation]
- Key steps: [Numbered list of actions]
- Approach to challenges: [How you will handle edge cases]
- Use of available tools: [Which tools and why]
- Expected outcome: [What success looks like]
- READY status: [YES/NO]`

		currentMessages = append(currentMessages, llm.Message{Role: "user", Content: prompt})
		
		response, err := a.LLM.Generate(ctx, currentMessages, options)
		if err != nil {
			return "", err
		}

		// 2. Evaluate
		isReady := strings.Contains(strings.ToUpper(response), "READY: YES") || 
		           strings.Contains(strings.ToUpper(response), "STATUS: YES") ||
				   strings.Contains(strings.ToUpper(response), "READY status: YES")

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
		
		refineMsg := "Your plan is not yet marked as READY: YES. Please refine the plan, address potential shortcomings, and Ensure you end with 'READY status: YES' when fully prepared."
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
