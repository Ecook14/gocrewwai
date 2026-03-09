package tasks

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/llm"
)

type mockAgent struct {
	executeFunc func(taskInput string) (interface{}, error)
}

func (m *mockAgent) Execute(ctx context.Context, taskInput string, options map[string]interface{}) (interface{}, error) {
	return m.executeFunc(taskInput)
}

func (m *mockAgent) GetRole() string { return "Mock Agent" }

func TestTaskExecute(t *testing.T) {
	mockLLM := &mockLLMClient{
		generateFunc: func(messages []llm.Message) (string, error) {
			return "Task Output", nil
		},
	}

	agent := &agents.Agent{
		Role: "Tester",
		LLM:  mockLLM,
	}

	task := &Task{
		Description: "Perform test",
		Agent:       agent,
	}

	res, err := task.Execute(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if res != "Task Output" {
		t.Errorf("Expected 'Task Output', got %v", res)
	}

	if !task.Processed {
		t.Errorf("Expected task to be marked as processed")
	}
}

func TestTask_JSONMarshaling(t *testing.T) {
	task := &Task{
		Description: "Marshal Task",
		CallbackOnComplete: func(result interface{}) {
			// Do nothing
		},
		OutputCondition: func(result interface{}) string {
			return "success"
		},
	}

	_, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task to JSON: %v. This likely means a function field is missing a `json:\"-\"` tag.", err)
	}
}

type mockLLMClient struct {
	llm.Client
	generateFunc func(messages []llm.Message) (string, error)
}

func (m *mockLLMClient) Generate(ctx context.Context, messages []llm.Message, options map[string]interface{}) (string, error) {
	return m.generateFunc(messages)
}
