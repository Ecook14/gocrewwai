package core

import (
	"context"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/llm"
)

func TestAgentInterface(t *testing.T) {
	agent := &MockAgent{}
	if agent.GetRole() != "test" {
		t.Errorf("Expected role 'test', got '%s'", agent.GetRole())
	}
	if agent.GetGoal() != "test goal" {
		t.Errorf("Expected goal 'test goal', got '%s'", agent.GetGoal())
	}
}

func TestAgentExecute(t *testing.T) {
	agent := &MockAgent{}
	result, err := agent.Execute(context.Background(), "test input", llm.GenerateOptions{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result != "result" {
		t.Errorf("Expected 'result', got '%v'", result)
	}
}

func TestSessionManager(t *testing.T) {
	sm, err := InitSessionManager("sqlite", ":memory:")
	if err != nil {
		t.Skipf("SessionManager skipped: %v", err)
	}
	if sm == nil {
		t.Fatal("Expected non-nil session manager")
	}
}

type MockAgent struct{}

func (m *MockAgent) Execute(ctx context.Context, taskInput string, options llm.GenerateOptions) (interface{}, error) {
	return "result", nil
}
func (m *MockAgent) GetRole() string { return "test" }
func (m *MockAgent) GetGoal() string { return "test goal" }
