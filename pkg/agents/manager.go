package agents

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/protocols"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/flows"
)


// ManagerAgent is a specialized agent that orchestrates other agents.
// It handles task delegation, validation, and result aggregation.
type ManagerAgent struct {
	Agent
	ManagedAgents []core.Agent
	A2AClient    *protocols.A2AClient
	Discovery    *protocols.AgentDiscovery
}

// NewManagerAgent creates a new manager agent with default delegation capabilities.
func NewManagerAgent(model llm.Client, agents []core.Agent) *ManagerAgent {
	return &ManagerAgent{
		Agent: Agent{
			Role:      "Manager",
			Goal:      "Efficiently delegate tasks to the best suited agents and aggregate their results into a final answer.",
			Backstory: "You are an expert project manager with deep understanding of team capabilities.",
			LLM:       model,
			Verbose:   true,
		},
		ManagedAgents: agents,
		A2AClient:     protocols.NewA2AClient(""), // Default client
		Discovery:     protocols.NewAgentDiscovery(protocols.GlobalA2ARegistry),
	}
}

func (m *ManagerAgent) DelegateTask(ctx context.Context, taskDescription string) (core.Agent, error) {
	agentRoles := "LOCAL AGENTS:\n"
	for _, a := range m.ManagedAgents {
		agentRoles += fmt.Sprintf("- %s: %s\n", a.GetRole(), a.GetGoal())
	}

	remoteCards := m.Discovery.Registry.ListAll()
	if len(remoteCards) > 0 {
		agentRoles += "\nREMOTE AGENTS (AVAILABLE VIA NETWORK):\n"
		for _, card := range remoteCards {
			if card.ID == m.A2AID { continue } // Skip self
			agentRoles += fmt.Sprintf("- %s: %s (Remote ID: %s)\n", card.Role, card.Description, card.ID)
		}
	}

	prompt := fmt.Sprintf(`Given the following task:
"%s"

And the following available agents:
%s

Which agent is BEST suited to perform this task? 
Respond ONLY with the name of the 'Role' of the agent. If it is a remote agent, respond with its 'Role' exactly as listed.`, taskDescription, agentRoles)

	messages := []llm.Message{
		{Role: "system", Content: m.Backstory},
		{Role: "user", Content: prompt},
	}

	if m.Verbose {
		slog.Info("Manager deciding on delegation (Distributed Mode)", slog.String("task", taskDescription))
	}

	response, err := m.LLM.Generate(ctx, messages, nil)
	if err != nil {
		return nil, fmt.Errorf("manager failed to generate delegation decision: %w", err)
	}

	chosenRole := strings.TrimSpace(response)
	
	// 1. Check local agents first
	for _, a := range m.ManagedAgents {
		if strings.Contains(strings.ToLower(chosenRole), strings.ToLower(a.GetRole())) {
			if m.Verbose {
				slog.Info("Manager delegated task locally", slog.String("agent", a.GetRole()))
			}
			return a, nil
		}
	}

	// 2. Check remote agents
	for _, card := range remoteCards {
		if strings.Contains(strings.ToLower(chosenRole), strings.ToLower(card.Role)) {
			if m.Verbose {
				slog.Info("Manager delegated task REMOTELY", slog.String("agent", card.Role), slog.String("endpoint", card.Endpoint))
			}
			return &protocols.RemoteAgentAdapter{
				Card:   card,
				Client: m.A2AClient,
				FromID: m.A2AID,
			}, nil
		}
	}

	return nil, fmt.Errorf("manager could not find agent with role: %s", chosenRole)
}

func (m *ManagerAgent) DelegateParallelTasks(ctx context.Context, taskDescriptions []string) (map[string]interface{}, error) {
	if m.Verbose {
		slog.Info("🚀 Manager initiating Parallel Scaling", slog.Int("task_count", len(taskDescriptions)))
	}

	// 1. Create a dynamic flow
	flow := flows.NewFlow("manager-parallel", "parallel-root", &flows.BaseState{Data: make(map[string]interface{})})
	
	parallelNode := &flows.FlowNode{
		ID:   "parallel-root",
		Type: flows.NodeParallel,
		Next: []string{"reduce-node"},
	}
	
	for i, task := range taskDescriptions {
		branchID := fmt.Sprintf("task-%d", i)
		parallelNode.ParallelBranches = append(parallelNode.ParallelBranches, branchID)
		
		taskCopy := task
		flow.AddNode(&flows.FlowNode{
			ID:   branchID,
			Type: flows.NodeStep,
			Action: func(ctx context.Context, s flows.State) (flows.State, error) {
				agent, err := m.DelegateTask(ctx, taskCopy)
				if err != nil {
					return s, err
				}
				res, err := agent.Execute(ctx, taskCopy, nil)
				if bState, ok := s.(*flows.BaseState); ok {
					bState.Data[branchID] = res
				}
				return s, nil
			},
		})
	}
	
	flow.AddNode(parallelNode)
	
	// Reduce node to aggregate results
	flow.AddNode(&flows.FlowNode{
		ID:   "reduce-node",
		Type: flows.NodeReduce,
		Merge: func(states []flows.State) flows.State {
			masterState := &flows.BaseState{Data: make(map[string]interface{})}
			for _, s := range states {
				if b, ok := s.(*flows.BaseState); ok {
					for k, v := range b.Data {
						masterState.Data[k] = v
					}
				}
			}
			return masterState
		},
	})
	
	// 2. Execute flow
	engine := &flows.Engine{}
	err := engine.Run(ctx, flow)
	if err != nil {
		return nil, err
	}
	
	if bState, ok := flow.State.(*flows.BaseState); ok {
		return bState.Data, nil
	}
	
	return nil, fmt.Errorf("failed to retrieve state from flow")
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

