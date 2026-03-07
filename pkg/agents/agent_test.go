package agents

import (
	"context"
	"testing"

	"github.com/Ecook14/crewai-go/pkg/llm"
)

type mockLLM struct{}

func (m *mockLLM) Generate(ctx context.Context, messages []llm.Message, options map[string]interface{}) (string, error) {
	return "Mock Generated Response", nil
}

func (m *mockLLM) GenerateStructured(ctx context.Context, messages []llm.Message, schema interface{}, options map[string]interface{}) (interface{}, error) {
	return nil, nil
}

func TestAgentExecuteWithoutLLM(t *testing.T) {
	agent := Agent{
		Role:      "Tester",
		Goal:      "Validate agent base execution",
		Backstory: "Expert Go engineer validating logic",
	}

	result, err := agent.Execute(context.Background(), "Run integration checks", nil)
	if err != nil {
		t.Fatalf("agent.Execute failed: %v", err)
	}

	expectedResult := "Task executed successfully by Tester"
	if result != expectedResult {
		t.Errorf("expected '%s', got '%s'", expectedResult, result)
	}
}

func TestAgentExecuteWithLLM(t *testing.T) {
	agent := Agent{
		Role:      "Architect",
		Goal:      "Design system",
		Backstory: "Expert",
		LLM:       &mockLLM{},
	}

	result, err := agent.Execute(context.Background(), "Design a cache", nil)
	if err != nil {
		t.Fatalf("agent.Execute failed: %v", err)
	}

	expectedResult := "Mock Generated Response"
	if result != expectedResult {
		t.Errorf("expected '%s', got '%s'", expectedResult, result)
	}
}
