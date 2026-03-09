package agents_test

import (
	"context"
	"testing"
	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/testutil"
)

func TestErgonomicAgentInitialization(t *testing.T) {
	mockLLM := &testutil.MockLLM{
		Response: "Test response",
	}

	// Declarative initialization matching the proposed "Python-like" style
	agent := agents.New(agents.AgentConfig{
		Role:      "Senior Data Scientist",
		Goal:      "Analyze data",
		Backstory: "Expert in statistics",
		LLM:       mockLLM,
		Verbose:   true,
	})

	if agent.Role != "Senior Data Scientist" {
		t.Errorf("Expected role 'Senior Data Scientist', got %s", agent.Role)
	}
	if agent.Goal != "Analyze data" {
		t.Errorf("Expected goal 'Analyze data', got %s", agent.Goal)
	}
	if agent.Verbose != true {
		t.Errorf("Expected verbose true, got false")
	}

	// Verify it can execute
	ctx := context.Background()
	_, err := agent.Execute(ctx, "Hello", nil)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}
}

func TestAgentBuilderNewFields(t *testing.T) {
	mockLLM := &testutil.MockLLM{}
	
	agent := agents.NewAgentBuilder().
		Role("Test").
		SystemTemplate("Custom System").
		PromptTemplate("Custom Prompt {input}").
		Build()

	if agent.SystemTemplate != "Custom System" {
		t.Errorf("Expected SystemTemplate 'Custom System', got %s", agent.SystemTemplate)
	}
	if agent.PromptTemplate != "Custom Prompt {input}" {
		t.Errorf("Expected PromptTemplate 'Custom Prompt {input}', got %s", agent.PromptTemplate)
	}
}
