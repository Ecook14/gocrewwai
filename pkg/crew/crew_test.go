package crew

import (
	"context"
	"strings"
	"testing"

	"github.com/Ecook14/crewai-go/pkg/agents"
	"github.com/Ecook14/crewai-go/pkg/llm"
	"github.com/Ecook14/crewai-go/pkg/tasks"
)

type mockLLM struct {
	llm.Client
	generateFunc func(messages []llm.Message) (string, error)
}

func (m *mockLLM) Generate(ctx context.Context, messages []llm.Message, options map[string]interface{}) (string, error) {
	return m.generateFunc(messages)
}

func TestCrewKickoff_Sequential(t *testing.T) {
	mock := &mockLLM{
		generateFunc: func(messages []llm.Message) (string, error) {
			return "Done", nil
		},
	}

	agent := &agents.Agent{Role: "Worker", LLM: mock}
	task := &tasks.Task{Description: "Job", Agent: agent}

	c := &Crew{
		Agents:  []*agents.Agent{agent},
		Tasks:   []*tasks.Task{task},
		Process: Sequential,
	}

	res, err := c.Kickoff(context.Background())
	if err != nil {
		t.Fatalf("Kickoff failed: %v", err)
	}

	if res != "Done" {
		t.Errorf("Expected 'Done', got %v", res)
	}
}

func TestCrewKickoff_Hierarchical(t *testing.T) {
	// For hierarchical, we need to mock delegation and synthesis.
	mock := &mockLLM{
		generateFunc: func(messages []llm.Message) (string, error) {
			// First call for delegation, second for synthesis
			for _, m := range messages {
				if strings.Contains(m.Content, "BEST suited") {
					return "Worker", nil
				}
				if strings.Contains(m.Content, "Analyze the following") {
					return "Aggregated Report", nil
				}
			}
			return "Worker Output", nil
		},
	}

	agent := &agents.Agent{Role: "Worker", LLM: mock}
	task := &tasks.Task{Description: "Job", Agent: agent}

	c := &Crew{
		Agents:  []*agents.Agent{agent},
		Tasks:   []*tasks.Task{task},
		Process: Hierarchical,
	}

	_, err := c.Kickoff(context.Background())
	if err != nil {
		t.Fatalf("Kickoff failed: %v", err)
	}
}

func TestCrewKickoff_DelegationInjection(t *testing.T) {
	mock := &mockLLM{
		generateFunc: func(messages []llm.Message) (string, error) {
			return "Done", nil
		},
	}

	agent := &agents.Agent{Role: "Worker", LLM: mock, AllowDelegation: true}
	task := &tasks.Task{Description: "Job", Agent: agent}

	c := &Crew{
		Agents:  []*agents.Agent{agent, {Role: "Coworker"}},
		Tasks:   []*tasks.Task{task},
		Process: Sequential,
	}

	_, err := c.Kickoff(context.Background())
	if err != nil {
		t.Fatalf("Kickoff failed: %v", err)
	}

	foundDelegation := false
	for _, tool := range agent.Tools {
		if tool.Name() == "DelegateWork" {
			foundDelegation = true
			break
		}
	}

	if !foundDelegation {
		t.Errorf("Expected DelegateWork tool to be injected")
	}
}

func TestCrewKickoff_Planning(t *testing.T) {
	mock := &mockLLM{
		generateFunc: func(messages []llm.Message) (string, error) {
			for _, m := range messages {
				if strings.Contains(m.Content, "Strategic Manager") {
					return "Plan: Do A then B.", nil
				}
			}
			return "Done", nil
		},
	}

	agent := &agents.Agent{Role: "Worker", LLM: mock}
	task := &tasks.Task{Description: "Job", Agent: agent}

	c := &Crew{
		Agents:   []*agents.Agent{agent},
		Tasks:    []*tasks.Task{task},
		Planning: true,
		Process:  Sequential,
	}

	_, err := c.Kickoff(context.Background())
	if err != nil {
		t.Fatalf("Kickoff failed: %v", err)
	}

	if !strings.Contains(task.Description, "[STRATEGIC PLAN]") {
		t.Error("Expected task description to contain strategic plan")
	}
	if !strings.Contains(task.Description, "Plan: Do A then B.") {
		t.Error("Expected task description to contain the generated plan content")
	}
}
