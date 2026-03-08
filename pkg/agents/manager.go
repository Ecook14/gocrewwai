package agents

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Ecook14/crewai-go/pkg/llm"
)

// ManagerAgent is a specialized agent that orchestrates other agents.
// It handles task delegation, validation, and result aggregation.
type ManagerAgent struct {
	Agent
	ManagedAgents []*Agent
}

// NewManagerAgent creates a new manager agent with default delegation capabilities.
func NewManagerAgent(model llm.Client, agents []*Agent) *ManagerAgent {
	return &ManagerAgent{
		Agent: Agent{
			Role:      "Manager",
			Goal:      "Efficiently delegate tasks to the best suited agents and aggregate their results into a final answer.",
			Backstory: "You are an expert project manager with deep understanding of team capabilities.",
			LLM:       model,
			Verbose:   true,
		},
		ManagedAgents: agents,
	}
}

func (m *ManagerAgent) DelegateTask(ctx context.Context, taskDescription string) (*Agent, error) {
	agentRoles := ""
	for _, a := range m.ManagedAgents {
		agentRoles += fmt.Sprintf("- %s: %s\n", a.Role, a.Goal)
	}

	prompt := fmt.Sprintf(`Given the following task:
"%s"

And the following available agents:
%s

Which agent is BEST suited to perform this task? 
Respond ONLY with the name of the 'Role' of the agent.`, taskDescription, agentRoles)

	messages := []llm.Message{
		{Role: "system", Content: m.Backstory},
		{Role: "user", Content: prompt},
	}

	if m.Verbose {
		slog.Info("Manager deciding on delegation", slog.String("task", taskDescription))
	}

	response, err := m.LLM.Generate(ctx, messages, nil)
	if err != nil {
		return nil, fmt.Errorf("manager failed to generate delegation decision: %w", err)
	}

	chosenRole := strings.TrimSpace(response)
	for _, a := range m.ManagedAgents {
		if strings.Contains(strings.ToLower(chosenRole), strings.ToLower(a.Role)) {
			if m.Verbose {
				slog.Info("Manager delegated task", slog.String("agent", strings.Clone(a.Role)))
			}
			return a, nil
		}
	}

	return nil, fmt.Errorf("manager could not find agent with role: %s", chosenRole)
}

func (m *ManagerAgent) GeneratePlan(ctx context.Context, tasks_list string) (string, error) {
	prompt := fmt.Sprintf(`You are the Strategic Manager. Given the following list of tasks for the Crew:
%s

Please create a high-level strategic plan. 
Include:
1. Coordination strategy (which tasks depend on which).
2. Key risks or hurdles for the agents.
3. How to ensure the final output is unified and consistent.

Respond with the plan details.`, tasks_list)

	messages := []llm.Message{
		{Role: "system", Content: m.Backstory},
		{Role: "user", Content: prompt},
	}

	if m.Verbose {
		slog.Info("Manager generating strategic plan")
	}

	response, err := m.LLM.Generate(ctx, messages, nil)
	if err != nil {
		return "", fmt.Errorf("manager failed to generate plan: %w", err)
	}

	return strings.TrimSpace(response), nil
}
